// transcription-worker: consome audio.uploaded, roda o motor de transcrição
// sobre o arquivo guardado no object storage e publica audio.transcribed.
//
// É um worker puro (sem API pública) além do /health que o compose consulta.
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pietromedeiros/meirinho/internal/domain"
	"github.com/pietromedeiros/meirinho/internal/platform/config"
	"github.com/pietromedeiros/meirinho/internal/platform/db"
	"github.com/pietromedeiros/meirinho/internal/platform/events"
	"github.com/pietromedeiros/meirinho/internal/platform/httpx"
	"github.com/pietromedeiros/meirinho/internal/platform/service"
	"github.com/pietromedeiros/meirinho/internal/platform/storage"
)

type worker struct {
	pool        *db.Pool
	bus         *events.Bus
	storage     *storage.Client
	transcriber Transcriber
}

func main() {
	rt, cleanup := service.Bootstrap("transcription-worker")
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

	st, err := storage.New(rt.Ctx, storage.Config{
		Endpoint:  config.MustString("MINIO_ENDPOINT"),
		AccessKey: config.MustString("MINIO_ACCESS_KEY"),
		SecretKey: config.MustString("MINIO_SECRET_KEY"),
		Bucket:    config.String("MINIO_BUCKET", "meirinho-audio"),
		UseSSL:    config.Bool("MINIO_USE_SSL", false),
		SSE:       config.Bool("MINIO_SSE_ENABLED", false),
	})
	if err != nil {
		rt.Fatal("cliente de storage", err)
	}
	if err := st.GarantirBucket(rt.Ctx, 60); err != nil {
		rt.Fatal("garantir bucket", err)
	}

	w := &worker{
		pool:        pool,
		bus:         bus,
		storage:     st,
		transcriber: escolherTranscriber(),
	}
	rt.Log.Info("motor de transcrição selecionado", "motor", w.transcriber.Nome())

	// /health separado do loop de consumo: o compose precisa saber que o
	// processo está vivo mesmo quando não há evento na fila.
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/health", httpx.Health)
		_ = rt.ServirHTTP(config.String("PORT", "8086"), mux)
	}()

	if err := bus.Consume(rt.Ctx, domain.StreamAudioUploaded, "transcription-worker", w.processar); err != nil {
		rt.Fatal("consumir audio.uploaded", err)
	}
}

// escolherTranscriber decide entre o motor real e o mock. O mock é o padrão
// para que `docker compose up` funcione em qualquer máquina; quem tiver o
// Whisper instalado liga TRANSCRIBER=whisper.
func escolherTranscriber() Transcriber {
	if config.String("TRANSCRIBER", "mock") == "whisper" {
		return &whisperCLI{
			binario: config.String("WHISPER_BIN", "whisper"),
			modelo:  config.String("WHISPER_MODEL", "small"),
			idioma:  config.String("WHISPER_LANG", "pt"),
			timeout: config.Duration("WHISPER_TIMEOUT", 30*time.Minute),
		}
	}
	return &mockTranscriber{atraso: config.Duration("MOCK_TRANSCRIBE_DELAY", 4*time.Second)}
}

func (w *worker) processar(ctx context.Context, raw []byte) error {
	ev, err := events.Decode[domain.AudioUploaded](raw)
	if err != nil {
		// Payload corrompido não melhora com retry — confirma e segue.
		return nil
	}

	if err := w.marcarStatus(ctx, ev.TenantID, ev.HearingID, domain.StatusTranscribing, nil); err != nil {
		return err
	}

	transcricao, err := w.transcrever(ctx, ev.ObjectKey)
	if err != nil {
		msg := err.Error()
		_ = w.marcarStatus(ctx, ev.TenantID, ev.HearingID, domain.StatusFailed, &msg)
		return err
	}

	if err := w.salvarTranscricao(ctx, ev.TenantID, ev.HearingID, transcricao); err != nil {
		return err
	}

	return w.bus.Publish(ctx, domain.StreamAudioTranscribed, domain.AudioTranscribed{
		HearingID:  ev.HearingID,
		TenantID:   ev.TenantID,
		Transcript: transcricao,
		Area:       ev.Area,
	})
}

// transcrever baixa o objeto para um arquivo temporário — os motores de
// transcrição trabalham sobre caminho em disco, não sobre stream.
func (w *worker) transcrever(ctx context.Context, objectKey string) (string, error) {
	obj, err := w.storage.Download(ctx, objectKey)
	if err != nil {
		return "", fmt.Errorf("baixar áudio: %w", err)
	}
	defer obj.Close()

	tmp, err := os.CreateTemp("", "audiencia-*"+filepath.Ext(objectKey))
	if err != nil {
		return "", fmt.Errorf("criar temporário: %w", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := io.Copy(tmp, obj); err != nil {
		tmp.Close()
		return "", fmt.Errorf("gravar temporário: %w", err)
	}
	tmp.Close()

	return w.transcriber.Transcrever(ctx, tmp.Name())
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

func (w *worker) salvarTranscricao(ctx context.Context, tenantID, id, transcricao string) error {
	return w.pool.TenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE hearing.hearings
			   SET transcript = $2, status = 'transcribed', erro = NULL, updated_at = now()
			 WHERE id = $1`, id, transcricao)
		return err
	})
}
