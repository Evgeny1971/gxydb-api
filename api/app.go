package api

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/coreos/go-oidc"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
	"github.com/volatiletech/sqlboiler/boil"

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
	tokenVerifier  *oidc.IDTokenVerifier
	cache          *AppCache
	sessionManager SessionManager
}

func (a *App) initOidc(issuer string) {
	oidcProvider, err := oidc.NewProvider(context.TODO(), issuer)
	if err != nil {
		log.Fatal().Err(err).Msg("oidc.NewProvider")
	}

	a.tokenVerifier = oidcProvider.Verifier(&oidc.Config{
		SkipClientIDCheck: true,
	})
}

func (a *App) Initialize(dbUrl, accountsUrl string, skipAuth bool) {
	log.Info().Msg("initializing app")
	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatal().Err(err).Msg("sql.Open")
	}

	a.InitializeWithDB(db, accountsUrl, skipAuth)
}

func (a *App) InitializeWithDB(db DBInterface, accountsUrl string, skipAuth bool) {
	a.DB = db

	a.Router = mux.NewRouter()
	a.initializeRoutes()

	if !skipAuth {
		a.initOidc(accountsUrl)
	}

	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Content-Length", "Accept-Encoding", "Content-Range", "Content-Disposition", "Authorization"})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "DELETE", "POST", "PUT", "OPTIONS"})
	cors := handlers.CORS(originsOk, headersOk, methodsOk)

	a.Handler = middleware.LoggingMiddleware(
		middleware.RecoveryMiddleware(
			middleware.RealIPMiddleware(
				middleware.AuthenticationMiddleware(a.tokenVerifier, skipAuth)(
					cors(a.Router)))))

	a.cache = new(AppCache)
	if err := a.cache.Init(db); err != nil {
		log.Fatal().Err(err).Msg("initialize app cache")
	}

	a.sessionManager = NewV1SessionManager(a.DB, a.cache)
}

func (a *App) Run(listenAddr string) {
	addr := listenAddr
	if addr == "" {
		addr = ":8080"
	}

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
	a.Router.HandleFunc("/qids", a.V1ListComposites).Methods("GET")
	a.Router.HandleFunc("/qids/{id}", a.V1GetComposite).Methods("GET")
	a.Router.HandleFunc("/qids/{id}", a.V1UpdateComposite).Methods("PUT")
	a.Router.HandleFunc("/event", a.V1HandleEvent).Methods("POST")
	a.Router.HandleFunc("/protocol", a.V1HandleProtocol).Methods("POST")

	// api v2
	//a.Router.HandleFunc("/groups", a.getGroups).Methods("GET")
	//a.Router.HandleFunc("/rooms", a.getRooms).Methods("GET") 	 			// Current
	//a.Router.HandleFunc("/room/{id}", a.getRoom).Methods("GET")			// Current
	//a.Router.HandleFunc("/room/{id}", a.postRoom).Methods("PUT")
	//a.Router.HandleFunc("/room/{id}", a.deleteRoom).Methods("DELETE")
	//a.Router.HandleFunc("/users", a.getUsers).Methods("GET")				// Current
	//a.Router.HandleFunc("/user", a.postUser).Methods("PUT")
	//a.Router.HandleFunc("/user/{id}", a.getUser).Methods("GET")
	//a.Router.HandleFunc("/user/{id}", a.deleteUser).Methods("DELETE")
	//a.Router.HandleFunc("/qids/{id}", a.getQuad).Methods("GET")							// Current
	//a.Router.HandleFunc("/qids/{id}", a.putQuad).Methods("PUT")							// Current
}
