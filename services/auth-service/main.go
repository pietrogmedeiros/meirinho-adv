// auth-service: cadastro, login e emissão de JWT.
//
// Cada advogado é um tenant. O token que sai daqui carrega o tenant_id que
// todos os outros serviços usam para escopar dado.
package main

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/pietromedeiros/meirinho/internal/platform/authn"
	"github.com/pietromedeiros/meirinho/internal/platform/config"
	"github.com/pietromedeiros/meirinho/internal/platform/db"
	"github.com/pietromedeiros/meirinho/internal/platform/httpx"
	"github.com/pietromedeiros/meirinho/internal/platform/service"
	"golang.org/x/crypto/bcrypt"
)

var reEmail = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

type api struct {
	store    *store
	issuer   *authn.Issuer
	verifier *authn.Verifier
}

func main() {
	rt, cleanup := service.Bootstrap("auth-service")
	defer cleanup()

	pool, err := db.Open(rt.Ctx, config.MustString("DATABASE_URL"))
	if err != nil {
		rt.Fatal("abrir banco", err)
	}
	defer pool.Close()
	if err := pool.Aguardar(rt.Ctx, 60); err != nil {
		rt.Fatal("aguardar postgres", err)
	}

	segredo := config.MustString("JWT_SECRET")
	a := &api{
		store:    &store{pool: pool},
		issuer:   authn.NewIssuer(segredo, config.Duration("JWT_TTL", 24*time.Hour)),
		verifier: authn.NewVerifier(segredo),
	}

	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(httpx.Observability(rt.Log))
	r.Get("/health", httpx.Health)

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/cadastro", a.cadastro)
		r.Post("/login", a.login)

		r.Group(func(r chi.Router) {
			r.Use(authn.Middleware(a.verifier))
			r.Get("/eu", a.eu)
			r.Patch("/eu", a.atualizarPerfil)
			r.Post("/senha", a.trocarSenha)
			r.Patch("/retencao", a.retencao)
		})
	})

	if err := rt.ServirHTTP(config.String("PORT", "8081"), r); err != nil {
		rt.Fatal("servidor http", err)
	}
}

type reqCadastro struct {
	Nome  string `json:"nome"`
	OAB   string `json:"oab"`
	Email string `json:"email"`
	Senha string `json:"senha"`
}

type respSessao struct {
	Token    string    `json:"token"`
	ExpiraEm time.Time `json:"expira_em"`
	Tenant   *tenant   `json:"tenant"`
}

func (a *api) cadastro(w http.ResponseWriter, r *http.Request) {
	var req reqCadastro
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "payload_invalido", "não foi possível ler o corpo da requisição")
		return
	}

	req.Nome = strings.TrimSpace(req.Nome)
	req.OAB = strings.TrimSpace(req.OAB)
	req.Email = strings.TrimSpace(req.Email)

	switch {
	case req.Nome == "":
		httpx.Fail(w, http.StatusBadRequest, "nome_obrigatorio", "informe seu nome")
		return
	case req.OAB == "":
		httpx.Fail(w, http.StatusBadRequest, "oab_obrigatoria", "informe seu número de OAB")
		return
	case !reEmail.MatchString(req.Email):
		httpx.Fail(w, http.StatusBadRequest, "email_invalido", "informe um e-mail válido")
		return
	case len(req.Senha) < 8:
		httpx.Fail(w, http.StatusBadRequest, "senha_fraca", "a senha precisa ter ao menos 8 caracteres")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Senha), bcrypt.DefaultCost)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "erro_interno", "falha ao processar a senha")
		return
	}

	t, err := a.store.criar(r.Context(), req.Nome, req.OAB, req.Email, string(hash))
	if err != nil {
		if err == errEmailEmUso {
			httpx.Fail(w, http.StatusConflict, "email_em_uso", "já existe uma conta com esse e-mail")
			return
		}
		httpx.Fail(w, http.StatusInternalServerError, "erro_interno", "não foi possível criar a conta")
		return
	}

	a.responderSessao(w, http.StatusCreated, t)
}

type reqLogin struct {
	Email string `json:"email"`
	Senha string `json:"senha"`
}

func (a *api) login(w http.ResponseWriter, r *http.Request) {
	var req reqLogin
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "payload_invalido", "não foi possível ler o corpo da requisição")
		return
	}

	t, hash, err := a.store.porEmail(r.Context(), req.Email)
	if err != nil {
		// Mesma resposta para e-mail inexistente e senha errada: não entregar
		// quais e-mails estão cadastrados.
		httpx.Fail(w, http.StatusUnauthorized, "credenciais_invalidas", "e-mail ou senha incorretos")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Senha)); err != nil {
		httpx.Fail(w, http.StatusUnauthorized, "credenciais_invalidas", "e-mail ou senha incorretos")
		return
	}

	a.responderSessao(w, http.StatusOK, t)
}

func (a *api) eu(w http.ResponseWriter, r *http.Request) {
	t, err := a.store.porID(r.Context(), authn.TenantDo(r.Context()))
	if err != nil {
		httpx.Fail(w, http.StatusNotFound, "nao_encontrado", "conta não encontrada")
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

type reqRetencao struct {
	RetencaoDias int `json:"retencao_dias"`
}

// retencao ajusta por quanto tempo o tenant quer guardar áudio e dado
// processual — a política de retenção exigida pela LGPD.
func (a *api) retencao(w http.ResponseWriter, r *http.Request) {
	var req reqRetencao
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "payload_invalido", "corpo inválido")
		return
	}
	if req.RetencaoDias < 30 || req.RetencaoDias > 3650 {
		httpx.Fail(w, http.StatusBadRequest, "retencao_invalida", "a retenção deve ficar entre 30 e 3650 dias")
		return
	}
	t, err := a.store.atualizarRetencao(r.Context(), authn.TenantDo(r.Context()), req.RetencaoDias)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "erro_interno", "não foi possível atualizar")
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

type reqPerfil struct {
	Nome  string `json:"nome"`
	OAB   string `json:"oab"`
	Email string `json:"email"`
}

func (a *api) atualizarPerfil(w http.ResponseWriter, r *http.Request) {
	var req reqPerfil
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "payload_invalido", "corpo inválido")
		return
	}
	req.Nome = strings.TrimSpace(req.Nome)
	req.OAB = strings.TrimSpace(req.OAB)
	req.Email = strings.TrimSpace(req.Email)
	switch {
	case req.Nome == "":
		httpx.Fail(w, http.StatusBadRequest, "nome_obrigatorio", "informe seu nome")
		return
	case req.OAB == "":
		httpx.Fail(w, http.StatusBadRequest, "oab_obrigatoria", "informe seu número de OAB")
		return
	case !reEmail.MatchString(req.Email):
		httpx.Fail(w, http.StatusBadRequest, "email_invalido", "informe um e-mail válido")
		return
	}

	t, err := a.store.atualizarPerfil(r.Context(), authn.TenantDo(r.Context()), req.Nome, req.OAB, req.Email)
	if errors.Is(err, errEmailEmUso) {
		httpx.Fail(w, http.StatusConflict, "email_em_uso", "já existe uma conta com esse e-mail")
		return
	}
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "erro_interno", "não foi possível atualizar")
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

type reqSenha struct {
	SenhaAtual string `json:"senha_atual"`
	NovaSenha  string `json:"nova_senha"`
}

// trocarSenha exige a senha atual: um token de sessão roubado não pode virar
// tomada de conta. Os tokens já emitidos continuam válidos até expirar — não
// há lista de revogação no MVP.
func (a *api) trocarSenha(w http.ResponseWriter, r *http.Request) {
	var req reqSenha
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "payload_invalido", "corpo inválido")
		return
	}
	if len(req.NovaSenha) < 8 {
		httpx.Fail(w, http.StatusBadRequest, "senha_fraca", "a nova senha precisa ter ao menos 8 caracteres")
		return
	}
	id := authn.TenantDo(r.Context())
	hash, err := a.store.senhaHash(r.Context(), id)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "erro_interno", "não foi possível trocar a senha")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.SenhaAtual)) != nil {
		httpx.Fail(w, http.StatusBadRequest, "senha_atual_incorreta", "a senha atual está incorreta")
		return
	}
	novo, err := bcrypt.GenerateFromPassword([]byte(req.NovaSenha), bcrypt.DefaultCost)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "erro_interno", "falha ao processar a senha")
		return
	}
	if err := a.store.atualizarSenha(r.Context(), id, string(novo)); err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "erro_interno", "não foi possível trocar a senha")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) responderSessao(w http.ResponseWriter, status int, t *tenant) {
	token, exp, err := a.issuer.Emitir(t.ID, t.Email, t.Nome)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "erro_interno", "falha ao emitir sessão")
		return
	}
	httpx.JSON(w, status, respSessao{Token: token, ExpiraEm: exp, Tenant: t})
}
