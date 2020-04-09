package api

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/coreos/go-oidc"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/volatiletech/sqlboiler/boil"

	"github.com/Bnei-Baruch/gxydb-api/pkg/auth"
)

type App struct {
	tokenVerifier *oidc.IDTokenVerifier
	DB            boil.Executor
	Router        *mux.Router
	Handler       http.Handler
}

func (a *App) initOidc(issuer string) {
	oidcProvider, err := oidc.NewProvider(context.TODO(), issuer)
	if err != nil {
		log.Fatalf("Error initializing auth %v", err)
	}

	a.tokenVerifier = oidcProvider.Verifier(&oidc.Config{
		SkipClientIDCheck: true,
	})
}

func (a *App) Initialize(dbUrl, accountsUrl string, skipAuth bool) {
	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatal(err)
	}

	a.InitializeWithDB(db, accountsUrl, skipAuth)
}

func (a *App) InitializeWithDB(db boil.Executor, accountsUrl string, skipAuth bool) {
	a.DB = db

	a.Router = mux.NewRouter()
	a.initializeRoutes()

	if !skipAuth {
		a.initOidc(accountsUrl)
		a.Router.Use(auth.Middleware(a.tokenVerifier))
	}

	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Content-Length", "Accept-Encoding", "Content-Range", "Content-Disposition", "Authorization"})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "DELETE", "POST", "PUT", "OPTIONS"})
	cors := handlers.CORS(originsOk, headersOk, methodsOk)

	recovery := handlers.RecoveryHandler(handlers.PrintRecoveryStack(true))

	a.Handler = handlers.LoggingHandler(os.Stdout, recovery(cors(a.Router)))
}

func (a *App) Run(listenAddr string) {
	addr := listenAddr
	if addr == "" {
		addr = ":8080"
	}

	log.Fatal(http.ListenAndServe(addr, a.Handler))
}

func (a *App) initializeRoutes() {
	// api v1 (current)
	a.Router.HandleFunc("/rooms", a.V1GetRooms).Methods("GET")     // Current
	a.Router.HandleFunc("/room/{id}", a.V1GetRoom).Methods("GET")  // Current
	a.Router.HandleFunc("/users", a.V1GetUsers).Methods("GET")     // Current
	a.Router.HandleFunc("/users/{id}", a.V1GetUser).Methods("GET") // Current
	//a.Router.HandleFunc("/qids/{id}", a.getQuad).Methods("GET")							// Current
	//a.Router.HandleFunc("/qids/{id}", a.putQuad).Methods("PUT")							// Current

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
