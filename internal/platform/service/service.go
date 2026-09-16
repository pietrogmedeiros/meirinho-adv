// Package service tira o boilerplate comum de subir um serviço: logger, otel,
// servidor HTTP com shutdown gracioso e cancelamento por sinal.
package service

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pietromedeiros/meirinho/internal/platform/config"
	"github.com/pietromedeiros/meirinho/internal/platform/logging"
	"github.com/pietromedeiros/meirinho/internal/platform/obs"
	"go.opentelemetry.io/otel/trace"
)

type Runtime struct {
	Nome   string
	Log    *slog.Logger
	Tracer trace.Tracer
	Ctx    context.Context
}

// Bootstrap monta logger + tracing e devolve um contexto que é cancelado no
// primeiro SIGINT/SIGTERM.
func Bootstrap(nome string) (*Runtime, func()) {
	log := logging.New(nome, config.String("LOG_LEVEL", "info"))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	tracer, shutdown := obs.Setup(ctx, nome, config.String("OTEL_EXPORTER_OTLP_ENDPOINT", ""), log)

	cleanup := func() {
		stop()
		ctxSd, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdown(ctxSd)
	}
	return &Runtime{Nome: nome, Log: log, Tracer: tracer, Ctx: ctx}, cleanup
}

// ServirHTTP sobe o servidor e bloqueia até o contexto ser cancelado, então
// drena as conexões abertas antes de sair.
func (r *Runtime) ServirHTTP(porta string, h http.Handler) error {
	srv := &http.Server{
		Addr:              ":" + porta,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		// Upload de áudio pode ser grande e lento: sem timeout de escrita
		// agressivo, mas com limite de leitura no handler.
		WriteTimeout: 5 * time.Minute,
		ReadTimeout:  5 * time.Minute,
	}

	erros := make(chan error, 1)
	go func() {
		r.Log.Info("servidor http ouvindo", "porta", porta)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			erros <- err
		}
	}()

	select {
	case err := <-erros:
		return err
	case <-r.Ctx.Done():
		r.Log.Info("encerrando servidor http")
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	}
}

// Fatal encerra o processo registrando o motivo em JSON.
func (r *Runtime) Fatal(msg string, err error) {
	r.Log.Error(msg, "err", err)
	os.Exit(1)
}
