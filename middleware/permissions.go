package middleware

import (
	"net/http"

	"github.com/Bnei-Baruch/gxydb-api/common"
	"github.com/Bnei-Baruch/gxydb-api/pkg/httputil"
)

func MinimalPermissionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if common.Config.SkipPermissions {
			next.ServeHTTP(w, r)
			return
		}

		// skip permissions check on gateway event handlers
		if r.URL.Path == "/event" || r.URL.Path == "/protocol" {
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
