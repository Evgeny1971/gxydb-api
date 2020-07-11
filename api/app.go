package api

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
	"github.com/rs/zerolog/log"

	"github.com/Bnei-Baruch/gxydb-api/common"
	"github.com/Bnei-Baruch/gxydb-api/domain"
	"github.com/Bnei-Baruch/gxydb-api/middleware"
)

type App struct {
	Router               *mux.Router
	Handler              http.Handler
	DB                   common.DBInterface
	cache                *AppCache
	sessionManager       SessionManager
	gatewayTokensManager *domain.GatewayTokensManager
}

func (a *App) initOidc(issuer string) middleware.OIDCTokenVerifier {
	provider, err := oidc.NewProvider(context.TODO(), issuer)
	if err != nil {
		log.Fatal().Err(err).Msg("oidc.NewProvider")
	}

	return provider.Verifier(&oidc.Config{
		SkipClientIDCheck: true,
	})
}

func (a *App) Initialize() {
	log.Info().Msg("initializing app")

	db, err := sql.Open("postgres", common.Config.DBUrl)
	if err != nil {
		log.Fatal().Err(err).Msg("sql.Open")
	}

	var tokenVerifier middleware.OIDCTokenVerifier
	if !common.Config.SkipAuth {
		tokenVerifier = a.initOidc(common.Config.AccountsUrl)
	}

	a.InitializeWithDeps(db, tokenVerifier)
}

func (a *App) InitializeWithDeps(db common.DBInterface, tokenVerifier middleware.OIDCTokenVerifier) {
	a.DB = db

	a.Router = mux.NewRouter()
	a.initializeRoutes()

	a.cache = new(AppCache)
	if err := a.cache.Init(db); err != nil {
		log.Fatal().Err(err).Msg("initialize app cache")
	}

	// this is declared here to abstract away the cache from auth middleware
	gatewayPwd := func(name string) (string, bool) {
		g, ok := a.cache.gateways.ByName(name)
		if ok {
			return g.EventsPassword, true
		}
		return "", false
	}

	a.sessionManager = NewV1SessionManager(a.DB, a.cache)

	if common.Config.MonitorGatewayTokens {
		a.gatewayTokensManager = domain.NewGatewayTokensManager(a.DB, 3*24*time.Hour)
		a.gatewayTokensManager.AddObserver(a)
		a.gatewayTokensManager.Monitor()
	}

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedHeaders: []string{"Origin", "Accept", "Content-Type", "X-Requested-With", "Authorization"},
	})

	a.Handler = middleware.ContextMiddleware(
		middleware.LoggingMiddleware(
			middleware.RecoveryMiddleware(
				middleware.RealIPMiddleware(
					corsMiddleware.Handler(
						middleware.AuthenticationMiddleware(tokenVerifier, gatewayPwd)(
							middleware.MinimalPermissionMiddleware(
								a.Router)))))))
}

func (a *App) Run() {
	defer a.Shutdown()

	addr := common.Config.ListenAddress
	log.Info().Msgf("app run %s", addr)
	if err := http.ListenAndServe(addr, a.Handler); err != nil {
		log.Fatal().Err(err).Msg("http.ListenAndServe")
	}
}

func (a *App) Shutdown() {
	if a.gatewayTokensManager != nil {
		a.gatewayTokensManager.Close()
	}
	a.cache.Close()
	if err := a.DB.Close(); err != nil {
		log.Error().Err(err).Msg("DB.close")
	}
}

func (a *App) Notify(event interface{}) {
	switch event.(type) {
	case string:
		log.Info().Msgf("processing %s", event)
		switch event.(string) {
		case common.EventGatewayTokensChanged:
			if err := a.cache.gatewayTokens.Reload(a.DB); err != nil {
				log.Error().Err(err).Msg("cache.gatewayTokens.Reload")
			}
		}
	}
}

func (a *App) initializeRoutes() {
	a.Router.HandleFunc("/health_check", a.HealthCheck).Methods("GET")

	// api v1 (current)
	a.Router.HandleFunc("/groups", a.V1ListGroups).Methods("GET")
	a.Router.HandleFunc("/group/{id}", a.V1CreateGroup).Methods("PUT")
	a.Router.HandleFunc("/rooms", a.V1ListRooms).Methods("GET")
	a.Router.HandleFunc("/room/{id}", a.V1GetRoom).Methods("GET")
	a.Router.HandleFunc("/users", a.V1ListUsers).Methods("GET")
	a.Router.HandleFunc("/users/{id}", a.V1GetUser).Methods("GET")
	a.Router.HandleFunc("/users/{id}", a.V1UpdateSession).Methods("PUT")
	a.Router.HandleFunc("/qids", a.V1ListComposites).Methods("GET")
	a.Router.HandleFunc("/qids/{id}", a.V1GetComposite).Methods("GET")
	a.Router.HandleFunc("/program/{id}", a.V1GetComposite).Methods("GET")
	a.Router.HandleFunc("/qids/{id}", a.V1UpdateComposite).Methods("PUT")
	a.Router.HandleFunc("/event", a.V1HandleEvent).Methods("POST")
	a.Router.HandleFunc("/protocol", a.V1HandleProtocol).Methods("POST")

	// api v2 (next)
	a.Router.HandleFunc("/v2/config", a.V2GetConfig).Methods("GET")

	// admin
	a.Router.HandleFunc("/admin/gateways", a.AdminListGateways).Methods("GET")
	a.Router.HandleFunc("/admin/gateways/{gateway_id}/sessions/{session_id}/handles/{handle_id}/info", a.AdminGatewaysHandleInfo).Methods("GET")

	a.Router.HandleFunc("/admin/rooms", a.AdminListRooms).Methods("GET")
	a.Router.HandleFunc("/admin/rooms", a.AdminCreateRoom).Methods("POST")
	a.Router.HandleFunc("/admin/rooms/{id}", a.AdminGetRoom).Methods("GET")
	a.Router.HandleFunc("/admin/rooms/{id}", a.AdminUpdateRoom).Methods("PUT")
	a.Router.HandleFunc("/admin/rooms/{id}", a.AdminDeleteRoom).Methods("DELETE")
}
