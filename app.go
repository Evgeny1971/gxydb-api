// app.go

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/coreos/go-oidc"
	"log"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type App struct {
	tokenVerifier *oidc.IDTokenVerifier
	Router        *mux.Router
	DB            *sql.DB
}

func (a *App) initOidc(acc string) {
	var oidcIDTokenVerifier *oidc.IDTokenVerifier
	oidcProvider, err := oidc.NewProvider(context.TODO(), acc)
	if err != nil {
		panic("Login failed:" + err.Error())
	}
	oidcIDTokenVerifier = oidcProvider.Verifier(&oidc.Config{
		SkipClientIDCheck: true,
	})
	a.tokenVerifier = oidcIDTokenVerifier
}

func (a *App) Initialize(user string, password string, dbname string) {
	connectionString := fmt.Sprintf("postgres://%s:%s@localhost/%s?sslmode=disable", user, password, dbname)

	var err error
	a.DB, err = sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatal(err)
	}

	a.Router = mux.NewRouter()
	a.initializeRoutes()
}

func (a *App) Run(addr string) {
	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Content-Length", "Accept-Encoding", "Content-Range", "Content-Disposition", "Authorization"})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "DELETE", "POST", "PUT", "OPTIONS"})

	log.Fatal(http.ListenAndServe(":8080", handlers.CORS(originsOk, headersOk, methodsOk)(a.Router)))
}

func (a *App) initializeRoutes() {
	//a.Router.Use(a.loggingMiddleware)
	a.Router.HandleFunc("/rooms", a.getRooms).Methods("GET")
	a.Router.HandleFunc("/users", a.getUsers).Methods("GET")
	a.Router.HandleFunc("/room/{id}", a.getRoom).Methods("GET")
	a.Router.HandleFunc("/user/{id}", a.getUser).Methods("GET")
	a.Router.HandleFunc("/room/{id}", a.postRoom).Methods("PUT")
	a.Router.HandleFunc("/user", a.postUser).Methods("PUT")
	a.Router.HandleFunc("/room/{id}", a.deleteRoom).Methods("DELETE")
	a.Router.HandleFunc("/user/{id}", a.deleteUser).Methods("DELETE")
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(code)
	w.Write(response)
}
