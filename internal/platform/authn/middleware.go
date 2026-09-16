package authn

import (
	"context"
	"net/http"

	"github.com/pietromedeiros/meirinho/internal/platform/httpx"
	"github.com/pietromedeiros/meirinho/internal/platform/logging"
)

type ctxKey struct{}

// Middleware exige um JWT válido e coloca os claims no contexto. Handlers
// protegidos pegam o tenant com Do(ctx) — nunca de um parâmetro de request,
// que o cliente poderia forjar.
func Middleware(v *Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, err := BearerDoRequest(r)
			if err != nil {
				httpx.Fail(w, http.StatusUnauthorized, "nao_autenticado", "token ausente ou malformado")
				return
			}
			claims, err := v.Verificar(raw)
			if err != nil {
				httpx.Fail(w, http.StatusUnauthorized, "nao_autenticado", "token inválido ou expirado")
				return
			}
			ctx := context.WithValue(r.Context(), ctxKey{}, claims)
			ctx = logging.Into(ctx, logging.From(ctx).With("tenant_id", claims.TenantID))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Do devolve os claims autenticados do contexto.
func Do(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(ctxKey{}).(*Claims)
	return c, ok && c != nil
}

// TenantDo é o atalho usado por todo handler protegido.
func TenantDo(ctx context.Context) string {
	if c, ok := Do(ctx); ok {
		return c.TenantID
	}
	return ""
}
