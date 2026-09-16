package httpx

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pietromedeiros/meirinho/internal/platform/logging"
)

type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// Observability injeta request_id, loga cada requisição em JSON e converte
// panic em 500 — sem isso um panic derruba o serviço inteiro.
func Observability(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := r.Header.Get("X-Request-Id")
			if reqID == "" {
				reqID = uuid.NewString()
			}
			w.Header().Set("X-Request-Id", reqID)

			l := logger.With("request_id", reqID, "method", r.Method, "path", r.URL.Path)
			ctx := logging.Into(r.Context(), l)

			sw := &statusWriter{ResponseWriter: w}
			start := time.Now()

			defer func() {
				if rec := recover(); rec != nil {
					l.Error("panic no handler", "panic", rec)
					if sw.status == 0 {
						Fail(sw, http.StatusInternalServerError, "internal_error", "erro interno")
					}
				}
				l.Info("http",
					"status", sw.status,
					"bytes", sw.bytes,
					"duration_ms", time.Since(start).Milliseconds(),
				)
			}()

			next.ServeHTTP(sw, r.WithContext(ctx))
		})
	}
}

// Health é o endpoint que o docker-compose usa como healthcheck.
func Health(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
