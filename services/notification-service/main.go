// notification-service: ponta final do pipeline. Consome hearing.analyzed e
// movement.classified e transforma os dois em algo que o advogado vê.
//
// É o único serviço que consome mais de um stream: os dois caminhos do produto
// (audiência e processo) convergem aqui, na caixa de entrada.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

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

// Um resumo inteiro não cabe numa notificação: o corpo é a chamada, o detalhe
// está na tela da audiência.
const maxCorpo = 400

type api struct {
	store       *store
	despachante *despachante
	// Abaixo deste nível a movimentação entra na lista mas não gera alerta —
	// exceto quando o classificador hesitou (fallback), caso em que avisar é
	// sempre a decisão certa.
	urgenciaMinima domain.Urgencia
}

func main() {
	rt, cleanup := service.Bootstrap("notification-service")
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

	minima, ok := domain.ParseUrgencia(config.String("NOTIFY_URGENCIA_MINIMA", "baixa"))
	if !ok {
		rt.Fatal("configuração", errors.New("NOTIFY_URGENCIA_MINIMA inválida (alta, media, baixa, nenhuma)"))
	}

	a := &api{
		store:          &store{pool: pool},
		despachante:    novoDespachante(rt.Log),
		urgenciaMinima: minima,
	}
	rt.Log.Info("notificador pronto", "urgencia_minima", minima, "canais_extras", len(a.despachante.canais))

	// Dois consumidores, um por stream, cada um com seu grupo: uma audiência
	// analisada e uma movimentação classificada são independentes e não devem
	// bloquear uma à outra.
	go func() {
		if err := bus.Consume(rt.Ctx, domain.StreamHearingAnalyzed, "notification-service", a.deAudiencia); err != nil {
			rt.Log.Error("consumir hearing.analyzed", "err", err)
		}
	}()
	go func() {
		if err := bus.Consume(rt.Ctx, domain.StreamMovementClassified, "notification-service", a.deMovimentacao); err != nil {
			rt.Log.Error("consumir movement.classified", "err", err)
		}
	}()

	verifier := authn.NewVerifier(config.MustString("JWT_SECRET"))

	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(httpx.Observability(rt.Log))
	r.Get("/health", httpx.Health)

	r.Route("/api/notifications", func(r chi.Router) {
		r.Use(authn.Middleware(verifier))
		r.Get("/", a.listar)
		r.Get("/resumo", a.resumo)
		r.Post("/lidas", a.marcarTodasLidas)
		r.Post("/{id}/lida", a.marcarLida)
	})

	if err := rt.ServirHTTP(config.String("PORT", "8084"), r); err != nil {
		rt.Fatal("servidor http", err)
	}
}

func (a *api) deAudiencia(ctx context.Context, raw []byte) error {
	ev, err := events.Decode[domain.HearingAnalyzed](raw)
	if err != nil {
		return nil
	}
	id := ev.HearingID
	n, err := a.store.criar(ctx, notificacao{
		TenantID:     ev.TenantID,
		Tipo:         domain.NotifInApp,
		Categoria:    "audiencia",
		Titulo:       "Análise da audiência concluída",
		Corpo:        truncar(ev.Summary, maxCorpo),
		ReferenciaID: &id,
	})
	if err != nil {
		return err
	}
	a.despachante.Despachar(ctx, *n)
	return nil
}

func (a *api) deMovimentacao(ctx context.Context, raw []byte) error {
	ev, err := events.Decode[domain.MovementClassified](raw)
	if err != nil {
		return nil
	}

	// Silenciar o irrelevante é o que mantém o alerta com valor. Mas o
	// fallback fura o filtro: ali a urgência baixa significa "não sei", e não
	// "não importa".
	if !ev.FallbackAplicado && ev.Urgencia.Nivel() < a.urgenciaMinima.Nivel() {
		return a.store.marcarMovimentacaoNotificada(ctx, ev.TenantID, ev.MovementID)
	}

	id := ev.MovementID
	urg := ev.Urgencia
	n, err := a.store.criar(ctx, notificacao{
		TenantID:     ev.TenantID,
		Tipo:         domain.NotifInApp,
		Categoria:    "movimentacao",
		Titulo:       tituloMovimentacao(ev),
		Corpo:        corpoMovimentacao(ev),
		Urgencia:     &urg,
		ReferenciaID: &id,
	})
	if err != nil {
		return err
	}

	a.despachante.Despachar(ctx, *n)
	return a.store.marcarMovimentacaoNotificada(ctx, ev.TenantID, ev.MovementID)
}

func tituloMovimentacao(ev domain.MovementClassified) string {
	if ev.FallbackAplicado {
		return "Movimentação exige leitura manual — " + formatarCNJ(ev.NumeroCNJ)
	}
	switch ev.Urgencia {
	case domain.UrgenciaAlta:
		if ev.PrazoDias != nil {
			return fmt.Sprintf("Prazo de %d dias — %s", *ev.PrazoDias, formatarCNJ(ev.NumeroCNJ))
		}
		return "Movimentação urgente — " + formatarCNJ(ev.NumeroCNJ)
	case domain.UrgenciaMedia:
		return "Movimentação relevante — " + formatarCNJ(ev.NumeroCNJ)
	default:
		return "Nova movimentação — " + formatarCNJ(ev.NumeroCNJ)
	}
}

func corpoMovimentacao(ev domain.MovementClassified) string {
	var sb strings.Builder
	sb.WriteString(truncar(ev.Descricao, maxCorpo))
	if ev.Sugestao != "" {
		sb.WriteString("\n\nSugestão: ")
		sb.WriteString(ev.Sugestao)
	}
	return sb.String()
}

// formatarCNJ devolve o número na máscara que o advogado reconhece; guardamos
// só os 20 dígitos, mas ninguém lê processo sem pontuação.
func formatarCNJ(cnj string) string {
	if len(cnj) != 20 {
		return cnj
	}
	return fmt.Sprintf("%s-%s.%s.%s.%s.%s",
		cnj[0:7], cnj[7:9], cnj[9:13], cnj[13:14], cnj[14:16], cnj[16:20])
}

func truncar(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return strings.TrimSpace(s[:max]) + "…"
}

func (a *api) listar(w http.ResponseWriter, r *http.Request) {
	apenasNaoLidas := r.URL.Query().Get("nao_lidas") == "true"
	itens, err := a.store.listar(r.Context(), authn.TenantDo(r.Context()), apenasNaoLidas)
	if err != nil {
		logging.From(r.Context()).Error("listar notificações", "err", err)
		httpx.Fail(w, http.StatusInternalServerError, "erro_interno", "não foi possível listar")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"itens": itens})
}

func (a *api) resumo(w http.ResponseWriter, r *http.Request) {
	n, err := a.store.naoLidas(r.Context(), authn.TenantDo(r.Context()))
	if err != nil {
		logging.From(r.Context()).Error("contar não lidas", "err", err)
		httpx.Fail(w, http.StatusInternalServerError, "erro_interno", "não foi possível contar")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"nao_lidas": n})
}

func (a *api) marcarLida(w http.ResponseWriter, r *http.Request) {
	err := a.store.marcarLida(r.Context(), authn.TenantDo(r.Context()), chi.URLParam(r, "id"))
	if errors.Is(err, errNaoEncontrada) {
		httpx.Fail(w, http.StatusNotFound, "nao_encontrada", "notificação não encontrada")
		return
	}
	if err != nil {
		logging.From(r.Context()).Error("marcar lida", "err", err)
		httpx.Fail(w, http.StatusInternalServerError, "erro_interno", "não foi possível marcar")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"lida": true})
}

func (a *api) marcarTodasLidas(w http.ResponseWriter, r *http.Request) {
	n, err := a.store.marcarTodasLidas(r.Context(), authn.TenantDo(r.Context()))
	if err != nil {
		logging.From(r.Context()).Error("marcar todas lidas", "err", err)
		httpx.Fail(w, http.StatusInternalServerError, "erro_interno", "não foi possível marcar")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"marcadas": n})
}
