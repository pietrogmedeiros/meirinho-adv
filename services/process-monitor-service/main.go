// process-monitor-service: dono da carteira de processos monitorados e do
// poller que descobre movimentação nova.
//
// Ele não classifica nada — só transforma "apareceu andamento novo no tribunal"
// em movement.received. Quem lê a urgência é o classification-worker.
package main

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/pietromedeiros/meirinho/internal/domain"
	"github.com/pietromedeiros/meirinho/internal/platform/authn"
	"github.com/pietromedeiros/meirinho/internal/platform/config"
	"github.com/pietromedeiros/meirinho/internal/platform/db"
	"github.com/pietromedeiros/meirinho/internal/platform/events"
	"github.com/pietromedeiros/meirinho/internal/platform/httpx"
	"github.com/pietromedeiros/meirinho/internal/platform/logging"
	"github.com/pietromedeiros/meirinho/internal/platform/service"
)

// Só dígitos: a numeração unificada do CNJ tem 20 deles
// (NNNNNNN-DD.AAAA.J.TR.OOOO). Guardamos normalizado para que o mesmo processo
// digitado com ou sem pontuação não entre duas vezes na carteira.
var reNaoDigito = regexp.MustCompile(`\D`)

type api struct {
	store    *store
	poller   *poller
	provider string
}

func main() {
	rt, cleanup := service.Bootstrap("process-monitor-service")
	defer cleanup()

	pool, err := db.Open(rt.Ctx, config.MustString("DATABASE_URL"))
	if err != nil {
		rt.Fatal("abrir banco", err)
	}
	defer pool.Close()
	if err := pool.Aguardar(rt.Ctx, 60); err != nil {
		rt.Fatal("aguardar postgres", err)
	}

	bus := events.New(config.MustString("REDIS_ADDR"), rt.Log)
	defer bus.Close()
	if err := bus.Aguardar(rt.Ctx, 60); err != nil {
		rt.Fatal("aguardar redis", err)
	}

	prov, err := escolherProvider()
	if err != nil {
		rt.Fatal("provider de processos", err)
	}

	st := &store{pool: pool}
	pl := &poller{
		store:     st,
		bus:       bus,
		provider:  prov,
		intervalo: config.Duration("POLL_INTERVAL", 5*time.Minute),
		log:       rt.Log,
	}
	a := &api{store: st, poller: pl, provider: prov.Nome()}

	rt.Log.Info("monitor iniciado", "provider", prov.Nome(), "intervalo", pl.intervalo.String())
	go pl.Rodar(rt.Ctx)

	verifier := authn.NewVerifier(config.MustString("JWT_SECRET"))

	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(httpx.Observability(rt.Log))
	r.Get("/health", httpx.Health)

	r.Route("/api/processes", func(r chi.Router) {
		r.Use(authn.Middleware(verifier))
		r.Get("/", a.listar)
		r.Post("/", a.criar)
		r.Delete("/{id}", a.arquivar)
		r.Get("/{id}/movements", a.movimentacoes)
	})

	if err := rt.ServirHTTP(config.String("PORT", "8085"), r); err != nil {
		rt.Fatal("servidor http", err)
	}
}

func (a *api) listar(w http.ResponseWriter, r *http.Request) {
	itens, err := a.store.listar(r.Context(), authn.TenantDo(r.Context()))
	if err != nil {
		logging.From(r.Context()).Error("listar processos", "err", err)
		httpx.Fail(w, http.StatusInternalServerError, "erro_interno", "não foi possível listar")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"itens": itens})
}

type novoProcesso struct {
	NumeroCNJ     string `json:"numero_cnj"`
	Titulo        string `json:"titulo"`
	Tribunal      string `json:"tribunal"`
	AreaDoDireito string `json:"area_do_direito"`
}

func (a *api) criar(w http.ResponseWriter, r *http.Request) {
	var req novoProcesso
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "corpo_invalido", "JSON inválido")
		return
	}

	cnj := reNaoDigito.ReplaceAllString(req.NumeroCNJ, "")
	if len(cnj) != 20 {
		httpx.Fail(w, http.StatusBadRequest, "cnj_invalido",
			"numero_cnj deve ter 20 dígitos no padrão NNNNNNN-DD.AAAA.J.TR.OOOO")
		return
	}

	area, ok := domain.ParseArea(req.AreaDoDireito)
	if !ok {
		httpx.Fail(w, http.StatusBadRequest, "area_invalida",
			"área do direito inválida (civel, familia, criminal, trabalhista, administrativo)")
		return
	}

	p, err := a.store.criar(r.Context(), authn.TenantDo(r.Context()), cnj,
		strings.TrimSpace(req.Titulo), strings.TrimSpace(req.Tribunal), area, a.provider)
	switch {
	case errors.Is(err, errDuplicado):
		httpx.Fail(w, http.StatusConflict, "processo_duplicado", "este processo já está na sua carteira")
		return
	case err != nil:
		logging.From(r.Context()).Error("criar processo", "err", err)
		httpx.Fail(w, http.StatusInternalServerError, "erro_interno", "não foi possível cadastrar o processo")
		return
	}
	httpx.JSON(w, http.StatusCreated, p)
}

func (a *api) arquivar(w http.ResponseWriter, r *http.Request) {
	err := a.store.arquivar(r.Context(), authn.TenantDo(r.Context()), chi.URLParam(r, "id"))
	if errors.Is(err, errNaoEncontrado) {
		httpx.Fail(w, http.StatusNotFound, "nao_encontrado", "processo não encontrado")
		return
	}
	if err != nil {
		logging.From(r.Context()).Error("arquivar processo", "err", err)
		httpx.Fail(w, http.StatusInternalServerError, "erro_interno", "não foi possível arquivar")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"status": "arquivado"})
}

func (a *api) movimentacoes(w http.ResponseWriter, r *http.Request) {
	itens, err := a.store.movimentacoes(r.Context(), authn.TenantDo(r.Context()), chi.URLParam(r, "id"))
	if err != nil {
		logging.From(r.Context()).Error("listar movimentações", "err", err)
		httpx.Fail(w, http.StatusInternalServerError, "erro_interno", "não foi possível listar")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"itens": itens})
}
