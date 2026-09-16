// analysis-service: consome audio.transcribed, manda a transcrição para o
// analyzer e publica hearing.analyzed.
//
// É a etapa que transforma texto cru em algo que o advogado usa: resumo,
// sugestão estratégica e pontos críticos. Worker puro, com /health só para o
// compose saber que o processo está vivo entre eventos.
package main

import (
	"context"
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
	pool     *db.Pool
	bus      *events.Bus
	analyzer llm.Analyzer
}

func main() {
	rt, cleanup := service.Bootstrap("analysis-service")
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

	w := &worker{pool: pool, bus: bus, analyzer: llm.Escolher(rt.Log)}
	rt.Log.Info("analyzer selecionado", "analyzer", w.analyzer.Nome())

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/health", httpx.Health)
		_ = rt.ServirHTTP(config.String("PORT", "8083"), mux)
	}()

	if err := bus.Consume(rt.Ctx, domain.StreamAudioTranscribed, "analysis-service", w.processar); err != nil {
		rt.Fatal("consumir audio.transcribed", err)
	}
}

func (w *worker) processar(ctx context.Context, raw []byte) error {
	ev, err := events.Decode[domain.AudioTranscribed](raw)
	if err != nil {
		// Payload corrompido não melhora com retry — confirma e segue.
		return nil
	}

	if err := w.marcarStatus(ctx, ev.TenantID, ev.HearingID, domain.StatusAnalyzing, nil); err != nil {
		return err
	}

	analise, err := w.analyzer.AnalisarAudiencia(ctx, ev.Transcript, ev.Area)
	if err != nil {
		msg := err.Error()
		_ = w.marcarStatus(ctx, ev.TenantID, ev.HearingID, domain.StatusFailed, &msg)
		return err
	}

	if err := w.salvarAnalise(ctx, ev.TenantID, ev.HearingID, analise); err != nil {
		return err
	}

	return w.bus.Publish(ctx, domain.StreamHearingAnalyzed, domain.HearingAnalyzed{
		HearingID:  ev.HearingID,
		TenantID:   ev.TenantID,
		Summary:    analise.Resumo,
		Suggestion: analise.SugestaoEstrategica,
	})
}

func (w *worker) marcarStatus(ctx context.Context, tenantID, id string, s domain.HearingStatus, erro *string) error {
	return w.pool.TenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE hearing.hearings
			   SET status = $2, erro = $3, updated_at = now()
			 WHERE id = $1`, id, string(s), erro)
		return err
	})
}

func (w *worker) salvarAnalise(ctx context.Context, tenantID, id string, a *llm.AnaliseAudiencia) error {
	return w.pool.TenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE hearing.hearings
			   SET summary = $2, suggestion = $3, pontos_criticos = $4,
			       status = 'analyzed', erro = NULL, updated_at = now()
			 WHERE id = $1`, id, a.Resumo, a.SugestaoEstrategica, a.PontosCriticos)
		return err
	})
}
