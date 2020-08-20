package middleware

import (
	"net/http"
	"strings"

	"github.com/Bnei-Baruch/gxydb-api/common"
	"github.com/Bnei-Baruch/gxydb-api/pkg/httputil"
)

func MinimalPermissionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if common.Config.SkipPermissions {
			next.ServeHTTP(w, r)
			return
		}

		// skip permissions check on health_check and gateway event handlers
		if r.URL.Path == "/health_check" ||
			r.URL.Path == "/metrics" ||
			r.URL.Path == "/event" ||
			strings.HasPrefix(r.URL.Path, "/protocol") {
			next.ServeHTTP(w, r)
			return
		}

		if !RequestHasRole(r, common.AllRoles...) {
			httputil.NewForbiddenError().Abort(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func RequestHasRole(r *http.Request, roles ...string) bool {
	rCtx, _ := ContextFromRequest(r)

	if rCtx.ServiceUser {
		return true
	}

	if rCtx.IDClaims != nil && rCtx.IDClaims.HasAnyRole(roles...) {
		return true
	}

	return false
}
