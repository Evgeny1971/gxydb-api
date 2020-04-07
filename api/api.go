package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/Bnei-Baruch/gxydb-api/pkg/httputil"
)

func (a *App) getGroups(w http.ResponseWriter, r *http.Request) {
	files, err := getGroups(a.DB)
	if err != nil {
		httputil.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httputil.RespondWithJSON(w, http.StatusOK, files)
}

func (a *App) getRooms(w http.ResponseWriter, r *http.Request) {
	files, err := getRooms(a.DB)
	if err != nil {
		httputil.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httputil.RespondWithJSON(w, http.StatusOK, files)
}

func (a *App) getUsers(w http.ResponseWriter, r *http.Request) {
	files, err := getUsers(a.DB)
	if err != nil {
		httputil.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httputil.RespondWithJSON(w, http.StatusOK, files)
}

func (a *App) getRoom(w http.ResponseWriter, r *http.Request) {
	var i rooms
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		httputil.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	i.Room = id

	if err := i.getRoom(a.DB); err != nil {
		switch err {
		case sql.ErrNoRows:
			httputil.RespondWithError(w, http.StatusNotFound, "Not Found")
		default:
			httputil.RespondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	httputil.RespondWithJSON(w, http.StatusOK, i)
}

func (a *App) getUser(w http.ResponseWriter, r *http.Request) {
	var i users
	vars := mux.Vars(r)
	i.ID = vars["id"]

	if err := i.getUser(a.DB); err != nil {
		switch err {
		case sql.ErrNoRows:
			httputil.RespondWithError(w, http.StatusNotFound, "Not Found")
		default:
			httputil.RespondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	httputil.RespondWithJSON(w, http.StatusOK, i)
}

func (a *App) postRoom(w http.ResponseWriter, r *http.Request) {
	var i rooms
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		httputil.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	i.Room = id

	d := json.NewDecoder(r.Body)
	if err := d.Decode(&i); err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	defer r.Body.Close()

	if err := i.postRoom(a.DB); err != nil {
		httputil.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httputil.RespondWithJSON(w, http.StatusOK, map[string]string{"result": "success"})
}

func (a *App) postUser(w http.ResponseWriter, r *http.Request) {
	var i users

	d := json.NewDecoder(r.Body)
	if err := d.Decode(&i); err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	defer r.Body.Close()

	if err := i.postUser(a.DB); err != nil {
		httputil.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httputil.RespondWithJSON(w, http.StatusOK, map[string]string{"result": "success"})
}

func (a *App) deleteRoom(w http.ResponseWriter, r *http.Request) {
	var i rooms
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		httputil.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	i.Room = id

	if err := i.deleteRoom(a.DB); err != nil {
		httputil.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httputil.RespondWithJSON(w, http.StatusOK, map[string]string{"result": "success"})
}

func (a *App) deleteUser(w http.ResponseWriter, r *http.Request) {
	var i users
	vars := mux.Vars(r)
	i.ID = vars["id"]

	if err := i.deleteUser(a.DB); err != nil {
		httputil.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httputil.RespondWithJSON(w, http.StatusOK, map[string]string{"result": "success"})
}
