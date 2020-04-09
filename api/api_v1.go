package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/volatiletech/sqlboiler/queries/qm"

	"github.com/Bnei-Baruch/gxydb-api/models"
	"github.com/Bnei-Baruch/gxydb-api/pkg/httputil"
)

type V1User struct {
	ID        string `json:"id"`
	Display   string `json:"display"`
	Email     string `json:"email"`
	Group     string `json:"group"`
	IP        string `json:"ip"`
	Janus     string `json:"janus"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	System    string `json:"system"`
	Username  string `json:"username"`
	Room      int    `json:"room"`
	Timestamp int64  `json:"timestamp"`
	Session   int64  `json:"session"`
	Handle    int64  `json:"handle"`
	RFID      int64  `json:"rfid"`
	Camera    bool   `json:"camera"`
	Question  bool   `json:"question"`
	SelfTest  bool   `json:"self_test"`
	SoundTest bool   `json:"sound_test"`
}

type V1Room struct {
	Room        int       `json:"room"`
	Janus       string    `json:"janus"`
	Questions   bool      `json:"questions"`
	Description string    `json:"description"`
	NumUsers    int       `json:"num_users"`
	Users       []*V1User `json:"users"`
}

func (a *App) V1GetRooms(w http.ResponseWriter, r *http.Request) {
	files, err := getRooms(a.DB)
	if err != nil {
		httputil.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httputil.RespondWithJSON(w, http.StatusOK, files)
}

func (a *App) V1GetUsers(w http.ResponseWriter, r *http.Request) {
	files, err := getUsers(a.DB)
	if err != nil {
		httputil.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httputil.RespondWithJSON(w, http.StatusOK, files)
}

func (a *App) V1GetRoom(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	room, err := models.Rooms(
		models.RoomWhere.ID.EQ(int64(id)),
		models.RoomWhere.Disabled.EQ(false),
		models.RoomWhere.RemovedAt.IsNull(),
		qm.Load(models.RoomRels.DefaultGateway),
		qm.Load(models.RoomRels.Sessions, models.SessionWhere.RemovedAt.IsNull()),
		qm.Load(qm.Rels(models.RoomRels.Sessions, models.SessionRels.User)),
		qm.Load(qm.Rels(models.RoomRels.Sessions, models.SessionRels.Gateway)),
	).One(a.DB)

	if err != nil {
		switch err {
		case sql.ErrNoRows:
			httputil.RespondWithError(w, http.StatusNotFound, "Not Found")
		default:
			fmt.Printf("Unexpected error %+v", err)
			httputil.RespondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	respRoom := V1Room{
		Room:        room.GatewayUID,
		Janus:       room.R.DefaultGateway.Name,
		Questions:   false,
		Description: room.Name,
		NumUsers:    len(room.R.Sessions),
		Users:       make([]*V1User, len(room.R.Sessions)),
	}

	for i, session := range room.R.Sessions {
		if session.Question {
			respRoom.Questions = true
		}
		respRoom.Users[i] = &V1User{
			ID:        session.R.User.AccountsID,
			Display:   session.Display.String,
			Email:     session.R.User.Email.String,
			Group:     room.Name,
			IP:        session.IPAddress.String,
			Janus:     session.R.Gateway.Name,
			Name:      "",     // Useless. Shouldn't be used on the client side.
			Role:      "user", // fixed. No more "groups" only "users"
			System:    session.UserAgent.String,
			Username:  "", // Useless. Never seen a value here
			Room:      room.GatewayUID,
			Timestamp: session.CreatedAt.Unix(), // Not sure we really need this
			Session:   session.GatewaySession.Int64,
			Handle:    session.GatewayHandle.Int64,
			RFID:      session.GatewayFeed.Int64,
			Camera:    session.Camera,
			Question:  session.Question,
			SelfTest:  session.SelfTest,  // Not sure we really need this
			SoundTest: session.SoundTest, // Not sure we really need this
		}
	}

	httputil.RespondWithJSON(w, http.StatusOK, respRoom)
}

func (a *App) V1GetUser(w http.ResponseWriter, r *http.Request) {
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

func (a *App) V1GetComposite(w http.ResponseWriter, r *http.Request) {
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

func (a *App) V1UpdateComposite(w http.ResponseWriter, r *http.Request) {
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
