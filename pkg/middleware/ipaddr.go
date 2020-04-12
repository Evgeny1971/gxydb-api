package middleware

import (
	"context"
	"net/http"

	"github.com/Bnei-Baruch/gxydb-api/pkg/httputil"
)

type realIPKey struct{}

func RealIPFromRequest(r *http.Request) (string, bool) {
	if r == nil {
		return "", false
	}
	return RealIPFromCtx(r.Context())
}

func RealIPFromCtx(ctx context.Context) (string, bool) {
	realIP, ok := ctx.Value(realIPKey{}).(string)
	return realIP, ok
}

func RealIPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), realIPKey{}, httputil.GetRealIP(r)))
		next.ServeHTTP(w, r)
	})
}
