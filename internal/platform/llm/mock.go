package llm

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pietromedeiros/meirinho/internal/domain"
	"github.com/pietromedeiros/meirinho/internal/platform/mockdata"
)

// mockAnalyzer devolve resultado plausível sem chamar a API.
//
// É o padrão do compose: `docker compose up` tem que funcionar numa máquina sem
// ANTHROPIC_API_KEY, senão o pipeline inteiro fica intestável localmente. Quem
// tem chave liga ANALYZER=claude.
type mockAnalyzer struct{ atraso time.Duration }

func (m *mockAnalyzer) Nome() string { return "mock" }

func (m *mockAnalyzer) esperar(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(m.atraso):
		return nil
	}
}

func (m *mockAnalyzer) AnalisarAudiencia(ctx context.Context, transcricao string, area domain.AreaDoDireito) (*AnaliseAudiencia, error) {
	if err := m.esperar(ctx); err != nil {
		return nil, err
	}
	// A análise é a do mesmo caso fictício que o transcritor mock escolheu.
	c := mockdata.PorTranscricao(transcricao, area)
	return &AnaliseAudiencia{Resumo: c.Resumo, SugestaoEstrategica: c.Sugestao, PontosCriticos: c.Pontos}, nil
}

// regraMock é uma linha da heurística de classificação do mock.
type regraMock struct {
	termos    []string
	urgencia  domain.Urgencia
	prazo     int // 0 = sem prazo
	confianca float64
	sugestao  string
	motivo    string
}

// A ordem importa: a primeira regra que casa vence, então as que abrem prazo
// vêm antes das genéricas ("sentença publicada" é sentença, não publicação).
var regrasMock = []regraMock{
	// Juntada é ato do cartório registrando uma peça: quem precisa agir já foi
	// intimado por outra movimentação. O laudo é a exceção — abre a fase de
	// manifestação sobre a perícia.
	{[]string{"juntada de laudo"}, domain.UrgenciaMedia, 0, 0.8,
		"Ler o laudo e avaliar a manifestação e a indicação de assistente técnico.",
		"laudo pericial juntado"},
	{[]string{"juntada de"}, domain.UrgenciaBaixa, 0, 0.8,
		"Apenas registrar; não exige providência.",
		"juntada de peça"},
	{[]string{"sentença publicada", "sentença proferida", "julgo procedente", "julgo improcedente"}, domain.UrgenciaAlta, 15, 0.9,
		"Ler a sentença e avaliar apelação ou embargos de declaração (5 dias) com o cliente.",
		"sentença publicada abre prazo recursal"},
	{[]string{"cite-se", "citação"}, domain.UrgenciaAlta, 15, 0.9,
		"Conferir a data da juntada do mandado e preparar a contestação.",
		"citação abre prazo de resposta"},
	{[]string{"embargos"}, domain.UrgenciaAlta, 5, 0.85,
		"Verificar se cabe resposta aos embargos e anotar o prazo de 5 dias.",
		"embargos de declaração opostos"},
	{[]string{"para réplica", "sobre a contestação"}, domain.UrgenciaAlta, 15, 0.9,
		"Preparar a réplica rebatendo as preliminares e os documentos novos.",
		"intimação para réplica"},
	{[]string{"intima", "prazo", "manifeste", "recurso"}, domain.UrgenciaAlta, 15, 0.85,
		"Identificar o ato a praticar e protocolar a manifestação dentro do prazo.",
		"termo indicativo de prazo"},
	{[]string{"audiência"}, domain.UrgenciaMedia, 0, 0.8,
		"Agendar a audiência, avisar o cliente e confirmar as testemunhas.",
		"audiência designada"},
	{[]string{"perícia", "pericial", "laudo"}, domain.UrgenciaMedia, 0, 0.8,
		"Avaliar a indicação de assistente técnico e a formulação de quesitos.",
		"fase pericial"},
	{[]string{"despacho", "decisão", "saneador", "tutela"}, domain.UrgenciaMedia, 0, 0.75,
		"Ler a decisão e verificar se exige providência ou comporta recurso.",
		"ato decisório sem prazo explícito"},
	{[]string{"juntada", "publicado", "distribuído", "autuado", "conclusos", "remetidos"}, domain.UrgenciaBaixa, 0, 0.75,
		"Apenas registrar; não exige providência.",
		"ato cartorário"},
}

var rePrazo = regexp.MustCompile(`prazo (?:de |em )?(\d{1,3}) dias`)

// ClassificarMovimentacao do mock usa heurística de palavra-chave, não sorteio.
// Assim o fallback conservador do classification-worker fica exercitável
// localmente: descrição não reconhecida cai em confiança baixa e vira alerta.
func (m *mockAnalyzer) ClassificarMovimentacao(ctx context.Context, descricao string, area domain.AreaDoDireito) (*ClassificacaoMovimentacao, error) {
	if err := m.esperar(ctx); err != nil {
		return nil, err
	}
	d := strings.ToLower(descricao)
	for _, r := range regrasMock {
		if !contemAlguma(d, r.termos...) {
			continue
		}
		c := &ClassificacaoMovimentacao{
			Urgencia: r.urgencia, Confianca: r.confianca, Sugestao: r.sugestao, Motivo: r.motivo,
		}
		if r.prazo > 0 {
			// O prazo escrito na própria movimentação vale mais que o padrão
			// da regra ("... prazo de 5 dias").
			p := r.prazo
			if m := rePrazo.FindStringSubmatch(d); m != nil {
				if n, err := strconv.Atoi(m[1]); err == nil && n > 0 {
					p = n
				}
			}
			c.PrazoDias = &p
		}
		return c, nil
	}
	// Confiança baixa de propósito: dispara o fallback conservador.
	return &ClassificacaoMovimentacao{
		Urgencia: domain.UrgenciaNenhuma, Confianca: 0.2,
		Sugestao: "Ler a movimentação na íntegra: a classificação automática não a reconheceu.",
		Motivo:   "descrição não reconhecida pela heurística",
	}, nil
}

func contemAlguma(s string, termos ...string) bool {
	for _, t := range termos {
		if strings.Contains(s, t) {
			return true
		}
	}
	return false
}
