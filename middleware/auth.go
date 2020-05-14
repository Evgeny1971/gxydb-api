package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc"
	pkgerr "github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"

	"github.com/Bnei-Baruch/gxydb-api/common"
	"github.com/Bnei-Baruch/gxydb-api/pkg/httputil"
)

type Roles struct {
	Roles []string `json:"roles"`
}

type IDTokenClaims struct {
	Acr               string           `json:"acr"`
	AllowedOrigins    []string         `json:"allowed-origins"`
	Aud               interface{}      `json:"aud"`
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

func (c *IDTokenClaims) initRoleMap() {
	if c.rolesMap == nil {
		c.rolesMap = make(map[string]struct{})
		if c.RealmAccess.Roles != nil {
			for _, r := range c.RealmAccess.Roles {
				c.rolesMap[r] = struct{}{}
			}
		}
	}
}

func (c *IDTokenClaims) HasAnyRole(roles ...string) bool {
	c.initRoleMap()
	for _, role := range roles {
		if _, ok := c.rolesMap[role]; ok {
			return true
		}
	}
	return false
}

type OIDCTokenVerifier interface {
	Verify(context.Context, string) (*oidc.IDToken, error)
}

func AuthenticationMiddleware(tokenVerifier OIDCTokenVerifier, gwPwd func(string) (string, bool)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// health_check needs no auth
			if r.URL.Path == "/health_check" {
				next.ServeHTTP(w, r)
				return
			}

			// gateways are using basic auth
			if r.URL.Path == "/event" || r.URL.Path == "/protocol" {
				if common.Config.SkipEventsAuth {
					next.ServeHTTP(w, r)
					return
				}

				username, password, ok := r.BasicAuth()
				if !ok {
					httputil.NewUnauthorizedError(pkgerr.Errorf("no `Authorization` header set")).Abort(w, r)
					return
				}

				hPwd, ok := gwPwd(username)
				if !ok {
					httputil.NewUnauthorizedError(pkgerr.Errorf("unknown gateway: %s", username)).Abort(w, r)
					return
				}

				if err := bcrypt.CompareHashAndPassword([]byte(hPwd), []byte(password)); err != nil {
					httputil.NewUnauthorizedError(pkgerr.Errorf("wrong password: %s", password)).Abort(w, r)
					return
				}

				next.ServeHTTP(w, r)
				return
			}

			// APIs are using a mix of JWT and basic auth
			if common.Config.SkipAuth {
				next.ServeHTTP(w, r)
				return
			}

			// service users are using basic auth
			if username, password, ok := r.BasicAuth(); ok {
				if username != "service" { // TODO: change me ?!
					httputil.NewUnauthorizedError(pkgerr.Errorf("unknown username: %s", username)).Abort(w, r)
					return
				}

				success := false
				for _, pwd := range common.Config.ServicePasswords {
					if err := bcrypt.CompareHashAndPassword([]byte(pwd), []byte(password)); err == nil {
						success = true
						break
					}
				}

				if !success {
					httputil.NewUnauthorizedError(pkgerr.Errorf("wrong password: %s", password)).Abort(w, r)
					return
				}

				rCtx, ok := ContextFromRequest(r)
				if ok {
					rCtx.ServiceUser = true
				}

				next.ServeHTTP(w, r)
				return
			}

			// accounts service users are using JWT
			auth := parseToken(r)
			if auth == "" {
				httputil.NewUnauthorizedError(pkgerr.Errorf("no `Authorization` header set")).Abort(w, r)
				return
			}

			token, err := tokenVerifier.Verify(context.TODO(), auth)
			if err != nil {
				httputil.NewUnauthorizedError(err).Abort(w, r)
				return
			}

			var claims IDTokenClaims
			if err := token.Claims(&claims); err != nil {
				httputil.NewBadRequestError(err, "malformed JWT claims").Abort(w, r)
				return
			}

			rCtx, ok := ContextFromRequest(r)
			if ok {
				rCtx.IDClaims = &claims
			}

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
