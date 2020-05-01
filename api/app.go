package api

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/coreos/go-oidc"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
	"github.com/rs/zerolog/log"
	"github.com/volatiletech/sqlboiler/boil"

	"github.com/Bnei-Baruch/gxydb-api/common"
	"github.com/Bnei-Baruch/gxydb-api/pkg/middleware"
)

type DBInterface interface {
	boil.Executor
	boil.Beginner
}

type App struct {
	Router         *mux.Router
	Handler        http.Handler
	DB             DBInterface
	cache          *AppCache
	sessionManager SessionManager
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

func (a *App) InitializeWithDeps(db DBInterface, tokenVerifier middleware.OIDCTokenVerifier) {
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
							a.Router))))))
}

func (a *App) Run() {
	addr := common.Config.ListenAddress
	log.Info().Msgf("app run %s", addr)
	if err := http.ListenAndServe(addr, a.Handler); err != nil {
		log.Fatal().Err(err).Msg("http.ListenAndServe")
	}
}

func (a *App) initializeRoutes() {
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
}
