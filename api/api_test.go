package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"

	"github.com/Bnei-Baruch/gxydb-api/models"
	"github.com/Bnei-Baruch/gxydb-api/pkg/stringutil"
	"github.com/Bnei-Baruch/gxydb-api/pkg/testutil"
)

type ApiTestSuite struct {
	suite.Suite
	testutil.TestDBManager
	tx  *sql.Tx
	app *App
}

func (s *ApiTestSuite) SetupSuite() {
	s.Require().Nil(s.InitTestDB())
	s.app = new(App)
	s.app.InitializeWithDB(s.DB, "", true)
}

func (s *ApiTestSuite) TearDownSuite() {
	s.Require().Nil(s.DestroyTestDB())
}

func (s *ApiTestSuite) SetupTest() {
	var err error
	s.tx, err = s.DB.Begin()
	s.Require().Nil(err)
	s.app.DB = s.tx
}

func (s *ApiTestSuite) TearDownTest() {
	err := s.tx.Rollback()
	s.Require().Nil(err)
}

func (s *ApiTestSuite) TestGetRoom() {
	gateway := s.createGateway()
	room := s.createRoom(gateway)
	users := make([]*models.User, 5)
	sessions := make([]*models.Session, len(users))
	for i := range users {
		users[i] = s.createUser()
		sessions[i] = s.createSession(users[i], gateway, room)
	}
	s.Require().NoError(s.app.cache.Reload(s.tx))

	req, _ := http.NewRequest("GET", fmt.Sprintf("/room/%d", room.ID), nil)
	body := s.request200json(req)

	// verify room's attributes
	s.Equal(room.GatewayUID, int(body["room"].(float64)), "room")
	s.Equal(gateway.Name, body["janus"], "Janus")
	s.Equal(room.Name, body["description"], "description")
	s.False(body["questions"].(bool), "questions")
	s.Equal(len(users), int(body["num_users"].(float64)), "num_users")

	// verify room's user sessions
	s.Equal(len(users), len(body["users"].([]interface{})), "users count")
	for i, respUser := range body["users"].([]interface{}) {
		data := respUser.(map[string]interface{})
		found := false
		for j, user := range users {
			if user.AccountsID == data["id"] {
				found = true
				s.assertV1User(sessions[j], data)
				break
			}
		}
		s.True(found, "unknown user [%d] %v", i, data["id"])
	}

	// turn on question mark on some session
	sessions[0].Question = true
	_, err := sessions[0].Update(s.tx, boil.Whitelist(models.SessionColumns.Question))
	s.Require().NoError(err)
	body = s.request200json(req)
	s.True(body["questions"].(bool), "questions true")

	// now turn off question and check
	sessions[0].Question = false
	_, err = sessions[0].Update(s.tx, boil.Whitelist(models.SessionColumns.Question))
	s.Require().NoError(err)
	body = s.request200json(req)
	s.False(body["questions"].(bool), "questions false again")
}

func (s *ApiTestSuite) TestGetRoomNotFound() {
	req, _ := http.NewRequest("GET", "/room/1", nil)
	resp := s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)
}

func (s *ApiTestSuite) TestGetRoomMalformedID() {
	req, _ := http.NewRequest("GET", "/room/id", nil)
	resp := s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)
}

func (s *ApiTestSuite) TestGetRooms() {
	boil.DebugMode = false
	counts := struct {
		gateways        int
		roomPerGateway  int
		sessionsPerRoom int
	}{
		gateways:        3,
		roomPerGateway:  10,
		sessionsPerRoom: 10,
	}
	gateways := make(map[int64]*models.Gateway, counts.gateways)
	rooms := make(map[int]*models.Room, counts.gateways*counts.roomPerGateway)
	sessions := make(map[string]*models.Session, counts.gateways*counts.roomPerGateway*counts.sessionsPerRoom)
	for i := 0; i < counts.gateways; i++ {
		gateway := s.createGateway()
		gateways[gateway.ID] = gateway
		for j := 0; j < counts.roomPerGateway; j++ {
			room := s.createRoom(gateway)
			rooms[room.GatewayUID] = room
			for k := 0; k < counts.sessionsPerRoom; k++ {
				user := s.createUser()
				sessions[user.AccountsID] = s.createSession(user, gateway, room)
			}
		}
	}
	s.Require().NoError(s.app.cache.Reload(s.tx))
	boil.DebugMode = true

	req, _ := http.NewRequest("GET", "/rooms", nil)
	resp := s.request(req)
	s.Require().Equal(http.StatusOK, resp.Code)

	var body []interface{}
	s.Require().NoError(json.Unmarshal(resp.Body.Bytes(), &body))
	s.Equal(counts.gateways*counts.roomPerGateway, len(body), "room count")

	for i, respRoom := range body {
		data := respRoom.(map[string]interface{})
		room, ok := rooms[int(data["room"].(float64))]
		s.Require().True(ok, "unknown room [%d] %v", i, data["room"])

		// verify room's attributes
		s.Equal(gateways[room.DefaultGatewayID].Name, data["janus"], "Janus")
		s.Equal(room.Name, data["description"], "description")
		s.False(data["questions"].(bool), "questions")
		s.Equal(counts.sessionsPerRoom, int(data["num_users"].(float64)), "num_users")

		// verify room's user sessions
		s.Equal(counts.sessionsPerRoom, len(data["users"].([]interface{})), "users count")
		for j, respUser := range data["users"].([]interface{}) {
			data := respUser.(map[string]interface{})
			session, ok := sessions[data["id"].(string)]
			s.Require().True(ok, "unknown session [%d] %v", j, data["id"])
			s.assertV1User(session, data)
		}
	}
}

func (s *ApiTestSuite) request(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	s.app.Handler.ServeHTTP(rr, req)
	return rr
}

func (s *ApiTestSuite) request200json(req *http.Request) map[string]interface{} {
	resp := s.request(req)
	s.Require().Equal(http.StatusOK, resp.Code)
	var body map[string]interface{}
	s.Require().NoError(json.Unmarshal(resp.Body.Bytes(), &body))
	return body
}

func (s *ApiTestSuite) assertV1User(session *models.Session, actual map[string]interface{}) {
	if session.R == nil {
		s.Require().NoError(session.L.LoadUser(s.tx, true, session, nil))
		s.Require().NoError(session.L.LoadRoom(s.tx, true, session, nil))
		s.Require().NoError(session.L.LoadGateway(s.tx, true, session, nil))
	}
	s.Equal(session.R.User.AccountsID, actual["id"], "id")
	s.Equal(session.Display.String, actual["display"], "display")
	s.Equal(session.R.User.Email.String, actual["email"], "email")
	s.Equal(session.R.Room.GatewayUID, int(actual["room"].(float64)), "room")
	s.Equal(session.IPAddress.String, actual["ip"], "ip")
	s.Equal(session.R.Gateway.Name, actual["janus"], "janus")
	s.Equal("user", actual["role"], "role")
	s.Equal(session.UserAgent.String, actual["system"], "system")
	s.Equal("", actual["username"], "username")
	s.Equal(session.CreatedAt.Unix(), int64(actual["timestamp"].(float64)), "timestamp")
	s.Equal(session.GatewaySession.Int64, int64(actual["session"].(float64)), "session")
	s.Equal(session.GatewayHandle.Int64, int64(actual["handle"].(float64)), "handle")
	s.Equal(session.GatewayFeed.Int64, int64(actual["rfid"].(float64)), "rfid")
	s.Equal(session.Camera, actual["camera"], "camera")
	s.Equal(session.Question, actual["question"], "question")
	s.Equal(session.SelfTest, actual["self_test"], "self_test")
	s.Equal(session.SoundTest, actual["sound_test"], "sound_test")
}

func (s *ApiTestSuite) createGateway() *models.Gateway {
	gateway := &models.Gateway{
		Name:          fmt.Sprintf("gateway_%s", stringutil.GenerateName(4)),
		URL:           "url",
		AdminURL:      "admin_url",
		AdminPassword: "admin_password",
	}
	s.Require().NoError(gateway.Insert(s.tx, boil.Infer()))
	return gateway
}

func (s *ApiTestSuite) createUser() *models.User {
	user := &models.User{
		AccountsID: stringutil.GenerateName(36),
		Email:      null.StringFrom("user@example.com"),
		FirstName:  null.StringFrom("first"),
		LastName:   null.StringFrom("last"),
		Username:   null.StringFrom("username"),
	}
	s.Require().NoError(user.Insert(s.tx, boil.Infer()))
	return user
}

func (s *ApiTestSuite) createRoom(gateway *models.Gateway) *models.Room {
	room := &models.Room{
		Name:             fmt.Sprintf("room_%s", stringutil.GenerateName(10)),
		DefaultGatewayID: gateway.ID,
		GatewayUID:       rand.Intn(math.MaxInt32),
	}
	s.Require().NoError(room.Insert(s.tx, boil.Infer()))
	return room
}

func (s *ApiTestSuite) createSession(user *models.User, gateway *models.Gateway, room *models.Room) *models.Session {
	session := &models.Session{
		UserID:         user.ID,
		RoomID:         null.Int64From(room.ID),
		GatewayID:      null.Int64From(gateway.ID),
		GatewaySession: null.Int64From(rand.Int63n(math.MaxInt32)),
		GatewayHandle:  null.Int64From(rand.Int63n(math.MaxInt32)),
		GatewayFeed:    null.Int64From(rand.Int63n(math.MaxInt32)),
		Display:        user.Username,
		Camera:         true,
		Question:       false,
		SelfTest:       true,
		SoundTest:      false,
		UserAgent:      null.StringFrom("user-agent"),
		IPAddress:      null.StringFrom("0.0.0.0"),
	}
	s.Require().NoError(session.Insert(s.tx, boil.Infer()))
	return session
}

func TestApiTestSuite(t *testing.T) {
	suite.Run(t, new(ApiTestSuite))
}
