package api

import (
	"context"
	"database/sql"
	"log"
	"net/http"

	"github.com/coreos/go-oidc"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"

	"github.com/Bnei-Baruch/gxydb-api/pkg/auth"
)

type App struct {
	tokenVerifier *oidc.IDTokenVerifier
	Router        *mux.Router
	DB            *sql.DB
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
	var err error
	a.DB, err = sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatal(err)
	}

	a.Router = mux.NewRouter()
	a.initializeRoutes()
	if !skipAuth {
		a.initOidc(accountsUrl)
		a.Router.Use(auth.Middleware(a.tokenVerifier))
	}
}

func (a *App) Run(listenAddr string) {
	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Content-Length", "Accept-Encoding", "Content-Range", "Content-Disposition", "Authorization"})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "DELETE", "POST", "PUT", "OPTIONS"})
	cors := handlers.CORS(originsOk, headersOk, methodsOk)

	addr := listenAddr
	if addr == "" {
		addr = ":8080"
	}

	log.Fatal(http.ListenAndServe(addr, cors(a.Router)))
}

func (a *App) initializeRoutes() {
	a.Router.HandleFunc("/groups", a.getGroups).Methods("GET")
	a.Router.HandleFunc("/rooms", a.getRooms).Methods("GET")
	a.Router.HandleFunc("/users", a.getUsers).Methods("GET")
	a.Router.HandleFunc("/room/{id}", a.getRoom).Methods("GET")
	a.Router.HandleFunc("/user/{id}", a.getUser).Methods("GET")
	a.Router.HandleFunc("/room/{id}", a.postRoom).Methods("PUT")
	a.Router.HandleFunc("/user", a.postUser).Methods("PUT")
	a.Router.HandleFunc("/room/{id}", a.deleteRoom).Methods("DELETE")
	a.Router.HandleFunc("/user/{id}", a.deleteUser).Methods("DELETE")
}
