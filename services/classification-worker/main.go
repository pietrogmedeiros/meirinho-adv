// classification-worker: consome movement.received, classifica a urgência da
// movimentação e publica movement.classified.
//
// O valor do produto está aqui: o advogado não precisa ler cada publicação do
// diário para descobrir o que abre prazo. Worker puro, com /health.
package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/pietromedeiros/meirinho/internal/domain"
	"github.com/pietromedeiros/meirinho/internal/platform/config"
	"github.com/pietromedeiros/meirinho/internal/platform/db"
	"github.com/pietromedeiros/meirinho/internal/platform/events"
	"github.com/pietromedeiros/meirinho/internal/platform/httpx"
	"github.com/pietromedeiros/meirinho/internal/platform/llm"
	"github.com/pietromedeiros/meirinho/internal/platform/service"
)

type worker struct {
	pool *db.Pool
	bus  *events.Bus
	clf  *classificador
}

func main() {
	rt, cleanup := service.Bootstrap("classification-worker")
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

	piso, ok := domain.ParseUrgencia(config.String("CLASSIFY_PISO", "media"))
	if !ok {
		rt.Fatal("configuração", errors.New("CLASSIFY_PISO inválido (alta, media, baixa, nenhuma)"))
	}

	analyzer := llm.Escolher(rt.Log)
	w := &worker{
		pool: pool,
		bus:  bus,
		clf: &classificador{
			analyzer:     analyzer,
			piso:         piso,
			minConfianca: config.Float("CLASSIFY_MIN_CONFIANCA", 0.6),
			tentativas:   config.Int("CLASSIFY_TENTATIVAS", 2),
			log:          rt.Log,
		},
	}
	rt.Log.Info("classificador pronto", "analyzer", analyzer.Nome(),
		"piso", piso, "min_confianca", w.clf.minConfianca)

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/health", httpx.Health)
		_ = rt.ServirHTTP(config.String("PORT", "8087"), mux)
	}()

	if err := bus.Consume(rt.Ctx, domain.StreamMovementReceived, "classification-worker", w.processar); err != nil {
		rt.Fatal("consumir movement.received", err)
	}
}

func (w *worker) processar(ctx context.Context, raw []byte) error {
	ev, err := events.Decode[domain.MovementReceived](raw)
	if err != nil {
		return nil // payload corrompido não melhora com retry
	}

	// Classificar nunca devolve erro: na pior hipótese devolve o piso com
	// fallback marcado, então toda movimentação recebida é resolvida.
	res, fallback := w.clf.Classificar(ctx, ev)

	if err := w.salvar(ctx, ev, res, fallback); err != nil {
		return err
	}

	return w.bus.Publish(ctx, domain.StreamMovementClassified, domain.MovementClassified{
		MovementID:       ev.MovementID,
		ProcessID:        ev.ProcessID,
		TenantID:         ev.TenantID,
		NumeroCNJ:        ev.NumeroCNJ,
		Descricao:        ev.Descricao,
		Urgencia:         res.Urgencia,
		Sugestao:         res.Sugestao,
		PrazoDias:        res.PrazoDias,
		FallbackAplicado: fallback,
	})
}

func (w *worker) salvar(ctx context.Context, ev domain.MovementReceived, res *llm.ClassificacaoMovimentacao, fallback bool) error {
	return w.pool.TenantTx(ctx, ev.TenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE process.movements
			   SET urgencia_classificada = $2, sugestao = $3, prazo_dias = $4, fallback_aplicado = $5
			 WHERE id = $1`,
			ev.MovementID, string(res.Urgencia), res.Sugestao, res.PrazoDias, fallback)
		return err
	})
}
