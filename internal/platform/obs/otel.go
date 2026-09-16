// Package obs liga o OpenTelemetry. Os traces atravessam os serviços via
// contexto HTTP; sem um coletor configurado (OTEL_EXPORTER_OTLP_ENDPOINT
// vazio) o setup vira no-op e não custa nada — útil para rodar local.
package obs

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// Shutdown deve ser chamado no encerramento para drenar spans em buffer.
type Shutdown func(context.Context) error

func Setup(ctx context.Context, service, endpoint string, log *slog.Logger) (trace.Tracer, Shutdown) {
	// Propagação sempre ligada: mesmo sem exportar, o trace-id atravessa os
	// serviços e aparece nos logs estruturados.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))

	if endpoint == "" {
		log.Info("otel desabilitado (sem OTEL_EXPORTER_OTLP_ENDPOINT)")
		return noop.NewTracerProvider().Tracer(service), func(context.Context) error { return nil }
	}

	exp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		// Observabilidade quebrada não pode derrubar o serviço.
		log.Error("otel: exporter indisponível, seguindo sem traces", "err", err)
		return noop.NewTracerProvider().Tracer(service), func(context.Context) error { return nil }
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp, sdktrace.WithBatchTimeout(5*time.Second)),
	)
	otel.SetTracerProvider(tp)
	log.Info("otel habilitado", "endpoint", endpoint)

	return tp.Tracer(service), tp.Shutdown
}
