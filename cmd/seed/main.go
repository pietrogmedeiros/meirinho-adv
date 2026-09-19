// Popula uma conta de demonstração pela API pública, como um usuário faria:
// cadastro, processos e upload de audiências. Nada é escrito direto no banco,
// então o que aparece na tela passou pelo pipeline inteiro (transcrição,
// análise, poller, classificação, notificação).
//
// Idempotente: rodar de novo entra na conta existente, ignora processo já
// cadastrado e só envia audiências se a conta ainda não tiver nenhuma.
//
//	docker compose run --rm -e SEED_DEMO=true seed
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/pietromedeiros/meirinho/internal/platform/config"
)

type processoSeed struct {
	CNJ, Titulo, Tribunal, Area string
}

var processos = []processoSeed{
	{"1002345-67.2025.8.26.0100", "Carlos Silva x Banco Horizonte S.A.", "TJSP", "civel"},
	{"0804512-33.2025.8.19.0001", "Condomínio Jardim das Flores x Construtora Prisma", "TJRJ", "civel"},
	{"1011876-20.2025.8.26.0002", "M. S. O. x Ricardo Oliveira (alimentos)", "TJSP", "familia"},
	{"5003321-88.2025.8.13.0024", "Inventário de Helena Castro", "TJMG", "familia"},
	{"1500789-14.2025.8.26.0050", "Justiça Pública x Anderson Lima", "TJSP", "criminal"},
	{"0010456-72.2025.5.02.0031", "Juliana Pereira x Logística Rápida Ltda.", "TRT-2", "trabalhista"},
	{"0020981-05.2025.5.15.0092", "Marcos Souza x Supermercados Bom Preço", "TRT-15", "trabalhista"},
	{"1034567-91.2025.4.03.6100", "Construtora Alvorada x Município de Santa Rita", "TRF-3", "administrativo"},
}

type audienciaSeed struct {
	Titulo, Area, Arquivo string
}

// Os títulos contêm a chave de um caso de internal/platform/mockdata, então o
// transcritor mock devolve a audiência fictícia correspondente.
var audiencias = []audienciaSeed{
	{"Instrução — Silva x Banco Horizonte", "civel", "instrucao-silva-banco.wav"},
	{"Instrução — Condomínio Jardim das Flores (vícios construtivos)", "civel", "instrucao-condominio.wav"},
	{"Conciliação — Beatriz Nogueira x plano de saúde", "civel", "conciliacao-plano-saude.wav"},
	{"Conciliação — alimentos M. S. O.", "familia", "conciliacao-alimentos.wav"},
	{"Inventário de Helena Castro — avaliação do imóvel", "familia", "inventario-helena.wav"},
	{"Instrução — modificação de guarda (Pedro, 9 anos)", "familia", "instrucao-guarda.wav"},
	{"Instrução — Anderson Lima (roubo majorado)", "criminal", "instrucao-anderson-lima.wav"},
	{"Instrução — Rodrigo Almeida (embriaguez ao volante)", "criminal", "instrucao-rodrigo.wav"},
	{"Audiência una — Juliana Pereira (horas extras)", "trabalhista", "una-juliana-pereira.wav"},
	{"Instrução — Marcos Souza (insalubridade)", "trabalhista", "instrucao-marcos-souza.wav"},
	{"Instrução — Construtora Alvorada x Município", "administrativo", "instrucao-alvorada.wav"},
	{"Instrução — Sérgio Mendes (anulação de PAD)", "administrativo", "instrucao-pad.wav"},
}

type cliente struct {
	base  string
	token string
	http  *http.Client
}

func main() {
	c := &cliente{
		base: config.String("SEED_API_URL", "http://gateway"),
		http: &http.Client{Timeout: 30 * time.Second},
	}
	if !config.Bool("SEED_DEMO", true) {
		fmt.Println("seed: SEED_DEMO desligado, nada a fazer")
		return
	}
	email := config.String("SEED_EMAIL", "demo@meirinho.dev")
	senha := config.String("SEED_SENHA", "meirinho123")

	if err := c.aguardar(); err != nil {
		falhar("api indisponível", err)
	}
	if err := c.entrarOuCadastrar(email, senha); err != nil {
		falhar("autenticar", err)
	}

	for _, p := range processos {
		err := c.json("POST", "/api/processes", map[string]string{
			"numero_cnj": p.CNJ, "titulo": p.Titulo, "tribunal": p.Tribunal, "area_do_direito": p.Area,
		}, nil)
		var e *erroHTTP
		switch {
		case errors.As(err, &e) && e.status == http.StatusConflict:
			fmt.Println("  processo já existe:", p.Titulo)
		case err != nil:
			falhar("criar processo "+p.CNJ, err)
		default:
			fmt.Println("  processo cadastrado:", p.Titulo)
		}
	}

	var lista struct {
		Itens []json.RawMessage `json:"itens"`
	}
	if err := c.json("GET", "/api/hearings", nil, &lista); err != nil {
		falhar("listar audiências", err)
	}
	if len(lista.Itens) > 0 {
		fmt.Printf("  conta já tem %d audiência(s); nenhuma enviada\n", len(lista.Itens))
	} else {
		for i, a := range audiencias {
			if err := c.enviarAudio(a, wav(time.Duration(6+i*2)*time.Second)); err != nil {
				falhar("enviar audiência "+a.Titulo, err)
			}
			fmt.Println("  audiência enviada:", a.Titulo)
		}
	}

	fmt.Printf(`
Pronto. Audiências e movimentações são processadas em segundo plano: em
menos de um minuto as análises e a classificação de urgência aparecem.

  Acesse: http://localhost:8080
  E-mail: %s
  Senha:  %s
`, email, senha)
}

// aguardar espera o gateway e cada serviço atrás dele. O /health do gateway
// responde assim que o nginx sobe, antes dos serviços: logo depois de um
// `up -d` ele diz "ok" e as rotas ainda dão 502. Sem token, uma rota protegida
// responde 401 — o que prova que o serviço está de pé.
func (c *cliente) aguardar() error {
	rotas := []string{"/health", "/api/auth/eu", "/api/hearings", "/api/processes", "/api/notifications"}
	var err error
	for i := 0; i < 90; i++ {
		err = nil
		for _, r := range rotas {
			resp, e := c.http.Get(c.base + r)
			if e != nil {
				err = e
				break
			}
			resp.Body.Close()
			if resp.StatusCode >= 500 {
				err = fmt.Errorf("%s: HTTP %d", r, resp.StatusCode)
				break
			}
		}
		if err == nil {
			return nil
		}
		time.Sleep(time.Second)
	}
	return err
}

func (c *cliente) entrarOuCadastrar(email, senha string) error {
	var s struct {
		Token string `json:"token"`
	}
	err := c.json("POST", "/api/auth/login", map[string]string{"email": email, "senha": senha}, &s)
	var e *erroHTTP
	if errors.As(err, &e) && e.status == http.StatusUnauthorized {
		err = c.json("POST", "/api/auth/cadastro", map[string]string{
			"nome": "Dra. Ana Beatriz Costa", "oab": "SP 245.871", "email": email, "senha": senha,
		}, &s)
		if err == nil {
			fmt.Println("conta criada:", email)
		}
	} else if err == nil {
		fmt.Println("conta existente:", email)
	}
	if err != nil {
		return err
	}
	c.token = s.Token
	return nil
}

type erroHTTP struct {
	status int
	corpo  string
}

func (e *erroHTTP) Error() string { return fmt.Sprintf("HTTP %d: %s", e.status, e.corpo) }

func (c *cliente) fazer(req *http.Request, saida any) error {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	corpo, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return &erroHTTP{resp.StatusCode, string(corpo)}
	}
	if saida != nil {
		return json.Unmarshal(corpo, saida)
	}
	return nil
}

func (c *cliente) json(metodo, caminho string, corpo, saida any) error {
	var r io.Reader
	if corpo != nil {
		b, err := json.Marshal(corpo)
		if err != nil {
			return err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(metodo, c.base+caminho, r)
	if err != nil {
		return err
	}
	if corpo != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.fazer(req, saida)
}

func (c *cliente) enviarAudio(a audienciaSeed, audio []byte) error {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("titulo", a.Titulo)
	_ = w.WriteField("area_do_direito", a.Area)
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{fmt.Sprintf(`form-data; name="audio"; filename=%q`, a.Arquivo)}
	h["Content-Type"] = []string{"audio/wav"}
	part, err := w.CreatePart(h)
	if err != nil {
		return err
	}
	if _, err := part.Write(audio); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	req, err := http.NewRequest("POST", c.base+"/api/hearings", &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	return c.fazer(req, nil)
}

// wav gera um tom baixo de 220 Hz, mono, 8 kHz. O conteúdo não importa para o
// transcritor mock; importa que seja áudio válido, para o player da tela tocar.
func wav(d time.Duration) []byte {
	const taxa = 8000
	n := int(d.Seconds() * taxa)
	var b bytes.Buffer
	escrever := func(v any) { _ = binary.Write(&b, binary.LittleEndian, v) }
	b.WriteString("RIFF")
	escrever(uint32(36 + n*2))
	b.WriteString("WAVEfmt ")
	escrever(uint32(16))
	escrever(uint16(1)) // PCM
	escrever(uint16(1)) // mono
	escrever(uint32(taxa))
	escrever(uint32(taxa * 2))
	escrever(uint16(2))
	escrever(uint16(16))
	b.WriteString("data")
	escrever(uint32(n * 2))
	for i := 0; i < n; i++ {
		escrever(int16(1500 * math.Sin(2*math.Pi*220*float64(i)/taxa)))
	}
	return b.Bytes()
}

func falhar(o string, err error) {
	fmt.Fprintf(os.Stderr, "seed: %s: %v\n", o, err)
	os.Exit(1)
}
