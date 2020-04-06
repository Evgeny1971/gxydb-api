package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc"
	"github.com/gorilla/mux"

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
}

func Middleware(tokenVerifier *oidc.IDTokenVerifier) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// Detect client IP
			//ip := httputil.GetRealIP(r)

			// Check if IP is private
			//private, err := isPrivateIP(ip)
			//if err != nil {
			//	respondWithError(w, httputil.StatusBadRequest, err.Error())
			//	return
			//}

			// TODO: IP based authentication simply doesn't work in real life.
			// Get rid of this.
			//allow, err := isAllowedIP(ip)
			//if err != nil {
			//	httputil.RespondWithError(w, http.StatusBadRequest, "Invalid IP")
			//	return
			//} else if allow {
			//	next.ServeHTTP(w, r)
			//	return
			//}

			auth := parseToken(r)
			if auth == "" {
				httputil.RespondWithError(w, http.StatusBadRequest, "Token not found")
				return
			}

			token, err := tokenVerifier.Verify(context.TODO(), auth)
			if err != nil {
				httputil.RespondWithError(w, http.StatusUnauthorized, err.Error())
				return
			}

			var claims IDTokenClaims
			if err := token.Claims(&claims); err != nil {
				httputil.RespondWithError(w, http.StatusBadRequest, err.Error())
				return
			}

			if !checkPermission(claims.RealmAccess.Roles) {
				httputil.RespondWithError(w, http.StatusForbidden, "Access denied")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func checkPermission(roles []string) bool {
	if roles != nil {
		for _, r := range roles {
			if r == "bb_user" {
				return true
			}
		}
	}
	return false
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
