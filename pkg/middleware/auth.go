package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc"

	"github.com/Bnei-Baruch/gxydb-api/pkg/httputil"
)

type Roles struct {
	Roles []string `json:"roles"`
}

type IDTokenClaims struct {
	Acr               string           `json:"acr"`
	AllowedOrigins    []string         `json:"allowed-origins"`
	Aud               string           `json:"aud"`
	AuthTime          int              `json:"auth_time"`
	Azp               string           `json:"azp"`
	Email             string           `json:"email"`
	Exp               int              `json:"exp"`
	FamilyName        string           `json:"family_name"`
	GivenName         string           `json:"given_name"`
	Iat               int              `json:"iat"`
	Iss               string           `json:"iss"`
	Jti               string           `json:"jti"`
	Name              string           `json:"name"`
	Nbf               int              `json:"nbf"`
	Nonce             string           `json:"nonce"`
	PreferredUsername string           `json:"preferred_username"`
	RealmAccess       Roles            `json:"realm_access"`
	ResourceAccess    map[string]Roles `json:"resource_access"`
	SessionState      string           `json:"session_state"`
	Sub               string           `json:"sub"`
	Typ               string           `json:"typ"`

	rolesMap map[string]struct{}
}

func (c *IDTokenClaims) HasRole(role string) bool {
	if c.rolesMap == nil {
		c.rolesMap = make(map[string]struct{})
		if c.RealmAccess.Roles != nil {
			for _, r := range c.RealmAccess.Roles {
				c.rolesMap[r] = struct{}{}
			}
		}
	}

	_, ok := c.rolesMap[role]
	return ok
}

type claimsKey struct{}

func IDClaimsFromRequest(r *http.Request) (*IDTokenClaims, bool) {
	if r == nil {
		return nil, false
	}
	return IDClaimFromCtx(r.Context())
}

func IDClaimFromCtx(ctx context.Context) (*IDTokenClaims, bool) {
	claims, ok := ctx.Value(claimsKey{}).(*IDTokenClaims)
	return claims, ok
}

func AuthenticationMiddleware(tokenVerifier *oidc.IDTokenVerifier, disabled bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if disabled {
				next.ServeHTTP(w, r)
				return
			}

			// We start without auth on these endpoints until we retire gxyd
			if r.URL.Path == "/event" || r.URL.Path == "/protocol" {
				next.ServeHTTP(w, r)
				return
			}

			//// Janus events and protocol room are using basic auth
			//if r.URL.Path == "/event" || r.URL.Path == "/protocol" {
			//	username, password, ok := r.BasicAuth()
			//	if !ok {
			//		httputil.NewBadRequestError(nil, "no `Authorization` header set").Abort(w, r)
			//		return
			//	}
			//
			//	if username != "replace_me" || password != "replace_me" {
			//		httputil.NewUnauthorizedError(errors.New("wrong username password")).Abort(w, r)
			//		return
			//	}
			//
			//	next.ServeHTTP(w, r)
			//	return
			//}

			// APIs are using JWT

			auth := parseToken(r)
			if auth == "" {
				httputil.NewBadRequestError(nil, "no `Authorization` header set").Abort(w, r)
				return
			}

			token, err := tokenVerifier.Verify(context.TODO(), auth)
			if err != nil {
				httputil.NewUnauthorizedError(err).Abort(w, r)
				return
			}

			var claims *IDTokenClaims
			if err := token.Claims(claims); err != nil {
				httputil.NewBadRequestError(err, "malformed JWT claims").Abort(w, r)
				return
			}

			r = r.WithContext(context.WithValue(r.Context(), claimsKey{}, claims))

			next.ServeHTTP(w, r)
		})
	}
}

func parseToken(r *http.Request) string {
	authHeader := strings.Split(strings.TrimSpace(r.Header.Get("Authorization")), " ")
	if len(authHeader) == 2 &&
		strings.ToLower(authHeader[0]) == "bearer" &&
		len(authHeader[1]) > 0 {
		return authHeader[1]
	}
	return ""
}

func isAllowedIP(ipAddr string) (bool, error) {
	ip := net.ParseIP(strings.TrimSpace(ipAddr))
	if ip == nil {
		return false, fmt.Errorf("invalid IP address %s", ipAddr)
	}

	_, lcl, _ := net.ParseCIDR("10.66.0.0/16")
	_, vpn, _ := net.ParseCIDR("172.16.102.0/24")
	return lcl.Contains(ip) || vpn.Contains(ip), nil
}