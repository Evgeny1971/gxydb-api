package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/volatiletech/sqlboiler/boil"
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

type V1Composite struct {
	VQuad []*V1CompositeRoom `json:"vquad"`
}

type V1CompositeRoom struct {
	V1Room
	Position int `json:"queue"`
}

func (a *App) V1ListRooms(w http.ResponseWriter, r *http.Request) {
	rooms, err := models.Rooms(
		models.RoomWhere.Disabled.EQ(false),
		models.RoomWhere.RemovedAt.IsNull(),
		qm.Load(models.RoomRels.Sessions, models.SessionWhere.RemovedAt.IsNull()),
		qm.Load(qm.Rels(models.RoomRels.Sessions, models.SessionRels.User)),
	).All(a.DB)

	if err != nil {
		httputil.NewInternalError(err).Abort(w)
		return
	}

	respRooms := make([]*V1Room, len(rooms))
	for i := range rooms {
		room := rooms[i]

		respRoom := &V1Room{
			Room:        room.GatewayUID,
			Janus:       a.cache.gateways.ByID[room.DefaultGatewayID].Name,
			Description: room.Name,
			NumUsers:    len(room.R.Sessions),
			Users:       make([]*V1User, len(room.R.Sessions)),
		}

		for i, session := range room.R.Sessions {
			if session.Question {
				respRoom.Questions = true
			}
			respRoom.Users[i] = a.makeV1User(room, session)
		}

		respRooms[i] = respRoom
	}

	httputil.RespondWithJSON(w, http.StatusOK, respRooms)
}

func (a *App) V1GetRoom(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		httputil.NewBadRequestError(err, "malformed id").Abort(w)
		return
	}

	room, err := models.Rooms(
		models.RoomWhere.ID.EQ(int64(id)),
		models.RoomWhere.Disabled.EQ(false),
		models.RoomWhere.RemovedAt.IsNull(),
		qm.Load(models.RoomRels.DefaultGateway),
		qm.Load(models.RoomRels.Sessions, models.SessionWhere.RemovedAt.IsNull(), qm.OrderBy(models.SessionColumns.CreatedAt)),
		qm.Load(qm.Rels(models.RoomRels.Sessions, models.SessionRels.User)),
		qm.Load(qm.Rels(models.RoomRels.Sessions, models.SessionRels.Gateway)),
	).One(a.DB)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.NewNotFoundError().Abort(w)
		} else {
			httputil.NewInternalError(err).Abort(w)
		}
		return
	}

	respRoom := &V1Room{
		Room:        room.GatewayUID,
		Janus:       room.R.DefaultGateway.Name,
		Description: room.Name,
		NumUsers:    len(room.R.Sessions),
		Users:       make([]*V1User, len(room.R.Sessions)),
	}

	for i, session := range room.R.Sessions {
		if session.Question {
			respRoom.Questions = true
		}
		respRoom.Users[i] = a.makeV1User(room, session)
	}

	httputil.RespondWithJSON(w, http.StatusOK, respRoom)
}

func (a *App) V1ListUsers(w http.ResponseWriter, r *http.Request) {
	sessions, err := models.Sessions(
		models.SessionWhere.RemovedAt.IsNull(),
		qm.Load(models.SessionRels.User),
		qm.Load(models.SessionRels.Room),
	).All(a.DB)

	if err != nil {
		httputil.NewInternalError(err).Abort(w)
		return
	}

	respSessions := make(map[string]*V1User, len(sessions))
	for i := range sessions {
		session := sessions[i]
		respSessions[session.R.User.AccountsID] = a.makeV1User(session.R.Room, session)
	}

	httputil.RespondWithJSON(w, http.StatusOK, respSessions)
}

func (a *App) V1GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if len(id) > 36 {
		httputil.NewBadRequestError(nil, "malformed id").Abort(w)
		return
	}

	var userID int64
	err := models.Users(
		qm.Select(models.UserColumns.ID),
		models.UserWhere.AccountsID.EQ(id),
		models.UserWhere.Disabled.EQ(false),
		models.UserWhere.RemovedAt.IsNull(),
	).QueryRow(a.DB).Scan(&userID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.NewNotFoundError().Abort(w)
		} else {
			httputil.NewInternalError(err).Abort(w)
		}
		return
	}

	session, err := models.Sessions(
		models.SessionWhere.UserID.EQ(userID),
		models.SessionWhere.RemovedAt.IsNull(),
		qm.Load(models.SessionRels.User),
		qm.Load(models.SessionRels.Room),
	).One(a.DB)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.NewNotFoundError().Abort(w)
		} else {
			httputil.NewInternalError(err).Abort(w)
		}
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, a.makeV1User(session.R.Room, session))
}

func (a *App) V1ListComposites(w http.ResponseWriter, r *http.Request) {
	composites, err := models.Composites(
		qm.Load(models.CompositeRels.CompositesRooms, qm.OrderBy(models.CompositesRoomColumns.Position)),
		qm.Load(qm.Rels(models.CompositeRels.CompositesRooms, models.CompositesRoomRels.Room),
			models.RoomWhere.Disabled.EQ(false),
			models.RoomWhere.RemovedAt.IsNull()),
		qm.Load(qm.Rels(models.CompositeRels.CompositesRooms, models.CompositesRoomRels.Room, models.RoomRels.Sessions),
			models.SessionWhere.RemovedAt.IsNull()),
	).All(a.DB)

	if err != nil {
		httputil.NewInternalError(err).Abort(w)
		return
	}

	respComposites := make(map[string]*V1Composite, len(composites))
	for _, composite := range composites {
		respComposites[composite.Name] = a.makeV1Composite(composite)
	}

	httputil.RespondWithJSON(w, http.StatusOK, respComposites)
}

func (a *App) V1GetComposite(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if len(id) > 16 {
		httputil.NewBadRequestError(nil, "malformed id").Abort(w)
		return
	}

	composite, err := models.Composites(
		models.CompositeWhere.Name.EQ(id),
		qm.Load(models.CompositeRels.CompositesRooms),
		qm.Load(qm.Rels(models.CompositeRels.CompositesRooms, models.CompositesRoomRels.Room),
			models.RoomWhere.Disabled.EQ(false),
			models.RoomWhere.RemovedAt.IsNull()),
		qm.Load(qm.Rels(models.CompositeRels.CompositesRooms, models.CompositesRoomRels.Room, models.RoomRels.Sessions),
			models.SessionWhere.RemovedAt.IsNull(), qm.OrderBy(models.SessionColumns.CreatedAt)),
	).One(a.DB)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.NewNotFoundError().Abort(w)
		} else {
			httputil.NewInternalError(err).Abort(w)
		}
		return
	}

	respComposite := a.makeV1Composite(composite)
	httputil.RespondWithJSON(w, http.StatusOK, respComposite)
}

func (a *App) V1UpdateComposite(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if len(id) > 16 {
		httputil.NewBadRequestError(nil, "malformed id").Abort(w)
		return
	}

	var data V1Composite
	if err := httputil.DecodeJSONBody(w, r, &data); err != nil {
		err.Abort(w)
		return
	}

	composite, err := models.Composites(
		models.CompositeWhere.Name.EQ(id),
	).One(a.DB)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.NewNotFoundError().Abort(w)
		} else {
			httputil.NewInternalError(err).Abort(w)
		}
		return
	}

	hErr := a.inTx(func(tx boil.Transactor) *httputil.HttpError {
		cRooms := make(models.CompositesRoomSlice, len(data.VQuad))
		for i, item := range data.VQuad {
			gateway, ok := a.cache.gateways.ByName[item.Janus]
			if !ok {
				return httputil.NewBadRequestError(nil, fmt.Sprintf("unknown gateway %s", item.Janus))
			}
			room, ok := a.cache.rooms.ByGatewayUID[item.Room]
			if !ok {
				return httputil.NewBadRequestError(nil, fmt.Sprintf("unknown room %d", item.Room))
			}

			cRooms[i] = &models.CompositesRoom{
				RoomID:    room.ID,
				GatewayID: gateway.ID,
				Position:  item.Position,
			}
		}

		if _, err := composite.CompositesRooms().DeleteAll(tx); err != nil {
			return httputil.NewInternalError(err)
		}
		if err := composite.AddCompositesRooms(tx, true, cRooms...); err != nil {
			return httputil.NewInternalError(err)
		}

		httputil.RespondWithJSON(w, http.StatusOK, map[string]string{"result": "success"})

		return nil
	})
	if hErr != nil {
		hErr.Abort(w)
		return
	}
}

func (a *App) inTx(f func(boil.Transactor) *httputil.HttpError) *httputil.HttpError {
	tx, err := a.DB.Begin()
	if err != nil {
		return httputil.NewInternalError(err)
	}

	// rollback on panics
	defer func() {
		if p := recover(); p != nil {
			if ex := tx.Rollback(); ex != nil {
				fmt.Printf("rollback error %+v\n", err)
			}
			panic(p) // re-throw panic after Rollback
		}
	}()

	// invoke logic and rollback on errors
	if err := f(tx); err != nil {
		if ex := tx.Rollback(); ex != nil {
			return httputil.NewInternalError(err)
		}
		return err
	}

	// commit transaction
	if err := tx.Commit(); err != nil {
		return httputil.NewInternalError(err)
	}

	return nil
}

func (a *App) makeV1User(room *models.Room, session *models.Session) *V1User {
	user := &V1User{
		ID:        session.R.User.AccountsID,
		Display:   session.Display.String,
		Email:     session.R.User.Email.String,
		Group:     room.Name,
		IP:        session.IPAddress.String,
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

	if session.GatewayID.Valid {
		user.Janus = a.cache.gateways.ByID[session.GatewayID.Int64].Name
	}

	return user
}

func (a *App) makeV1Composite(composite *models.Composite) *V1Composite {
	respComposite := &V1Composite{
		VQuad: make([]*V1CompositeRoom, len(composite.R.CompositesRooms)),
	}

	for j, cRoom := range composite.R.CompositesRooms {
		room := cRoom.R.Room
		respRoom := &V1CompositeRoom{
			V1Room: V1Room{
				Room:        room.GatewayUID,
				Janus:       a.cache.gateways.ByID[cRoom.GatewayID].Name,
				Description: room.Name,
				NumUsers:    len(room.R.Sessions),
			},
			Position: cRoom.Position,
		}

		for _, session := range room.R.Sessions {
			if session.Question {
				respRoom.Questions = true
				break
			}
		}

		respComposite.VQuad[j] = respRoom
	}

	return respComposite
}
