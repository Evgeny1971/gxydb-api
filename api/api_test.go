package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/coreos/go-oidc"
	"github.com/edoshor/janus-go"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"golang.org/x/crypto/bcrypt"

	"github.com/Bnei-Baruch/gxydb-api/common"
	"github.com/Bnei-Baruch/gxydb-api/middleware"
	"github.com/Bnei-Baruch/gxydb-api/models"
	"github.com/Bnei-Baruch/gxydb-api/pkg/stringutil"
	"github.com/Bnei-Baruch/gxydb-api/pkg/testutil"
)

type ApiTestSuite struct {
	suite.Suite
	testutil.TestDBManager
	app           *App
	tokenVerifier *testutil.OIDCTokenVerifier
}

func (s *ApiTestSuite) SetupSuite() {
	s.Require().NoError(s.InitTestDB())
	s.tokenVerifier = new(testutil.OIDCTokenVerifier)
	s.app = new(App)
	s.app.InitializeWithDeps(s.DB, s.tokenVerifier)
}

func (s *ApiTestSuite) TearDownSuite() {
	s.Require().NoError(s.DestroyTestDB())
}

func (s *ApiTestSuite) SetupTest() {
	s.DBCleaner.Acquire(s.AllTables()...)
}

func (s *ApiTestSuite) TearDownTest() {
	s.assertTokenVerifier()
	s.DBCleaner.Clean(s.AllTables()...)
}

func (s *ApiTestSuite) TestListGroups() {
	counts := struct {
		gateways       int
		roomPerGateway int
	}{
		gateways:       3,
		roomPerGateway: 5,
	}
	gateways := make(map[int64]*models.Gateway, counts.gateways)
	rooms := make(map[int]*models.Room, counts.gateways*counts.roomPerGateway)
	for i := 0; i < counts.gateways; i++ {
		gateway := s.createGateway()
		gateways[gateway.ID] = gateway
		for j := 0; j < counts.roomPerGateway; j++ {
			room := s.createRoom(gateway)
			rooms[room.GatewayUID] = room
		}
	}

	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	req, _ := http.NewRequest("GET", "/groups", nil)
	s.apiAuth(req)
	body := s.request200json(req)
	respRooms, ok := body["rooms"].([]interface{})
	s.Require().True(ok, "rooms is array")
	s.Equal(counts.gateways*counts.roomPerGateway, len(respRooms), "group count")

	lastDescription := ""
	for i, respRoom := range respRooms {
		data := respRoom.(map[string]interface{})
		room, ok := rooms[int(data["room"].(float64))]
		s.Require().True(ok, "unknown room [%d] %v", i, data["room"])
		s.Equal(gateways[room.DefaultGatewayID].Name, data["janus"], "Janus")
		s.Equal(room.Name, data["description"], "description")
		s.GreaterOrEqual(data["description"], lastDescription, "order by")
		lastDescription = data["description"].(string)
	}
}

func (s *ApiTestSuite) TestCreateGroupMalformedID() {
	req, _ := http.NewRequest("PUT", "/group/id", nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	resp := s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)
}

func (s *ApiTestSuite) TestCreateGroupBadJSON() {
	req, _ := http.NewRequest("PUT", "/group/1234", bytes.NewBuffer([]byte("{\"bad\":\"json")))
	s.apiAuthP(req, []string{common.RoleRoot})
	resp := s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)
}

func (s *ApiTestSuite) TestCreateGroupUnknownGateway() {
	roomInfo := V1RoomInfo{
		Room:        1234,
		Janus:       "unknown",
		Description: "description",
	}

	b, _ := json.Marshal(roomInfo)
	req, _ := http.NewRequest("PUT", "/group/1234", bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleRoot})
	resp := s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)
}

func (s *ApiTestSuite) TestCreateGroup() {
	gateway := s.createGateway()
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	roomInfo := V1RoomInfo{
		Room:        1234,
		Janus:       gateway.Name,
		Description: "description",
	}

	b, _ := json.Marshal(roomInfo)
	req, _ := http.NewRequest("PUT", "/group/1234", bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleRoot})
	resp := s.request(req)
	s.Require().Equal(http.StatusOK, resp.Code)

	req, _ = http.NewRequest("GET", "/groups", nil)
	s.apiAuth(req)
	body := s.request200json(req)
	respRooms, ok := body["rooms"].([]interface{})
	s.Require().True(ok, "rooms is array")
	s.Equal(1, len(respRooms), "group count")
	data := respRooms[0].(map[string]interface{})
	s.Equal(roomInfo.Room, int(data["room"].(float64)), "Janus")
	s.Equal(roomInfo.Janus, data["janus"], "Janus")
	s.Equal(roomInfo.Description, data["description"], "description")
}

func (s *ApiTestSuite) TestCreateGroupExiting() {
	gateway := s.createGateway()
	room := s.createRoom(gateway)
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	roomInfo := V1RoomInfo{
		Room:        room.GatewayUID,
		Janus:       gateway.Name,
		Description: "updated name",
	}

	b, _ := json.Marshal(roomInfo)
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/group/%d", roomInfo.Room), bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleRoot})
	resp := s.request(req)
	s.Require().Equal(http.StatusOK, resp.Code)

	req, _ = http.NewRequest("GET", "/groups", nil)
	s.apiAuth(req)
	body := s.request200json(req)
	respRooms, ok := body["rooms"].([]interface{})
	s.Require().True(ok, "rooms is array")
	s.Equal(1, len(respRooms), "group count")
	data := respRooms[0].(map[string]interface{})
	s.Equal(roomInfo.Room, int(data["room"].(float64)), "Janus")
	s.Equal(roomInfo.Janus, data["janus"], "Janus")
	s.Equal(roomInfo.Description, data["description"], "description")
}

func (s *ApiTestSuite) TestGetRoomMalformedID() {
	req, _ := http.NewRequest("GET", "/room/id", nil)
	s.apiAuth(req)
	resp := s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)
}

func (s *ApiTestSuite) TestGetRoomNotFound() {
	req, _ := http.NewRequest("GET", "/room/1", nil)
	s.apiAuth(req)
	resp := s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)

	// disabled room
	gateway := s.createGateway()
	room := s.createRoom(gateway)
	room.Disabled = true
	_, err := room.Update(s.DB, boil.Whitelist(models.RoomColumns.Disabled))
	s.Require().NoError(err)
	req, _ = http.NewRequest("GET", fmt.Sprintf("/room/%d", room.GatewayUID), nil)
	s.apiAuth(req)
	resp = s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)

	// removed room
	room.Disabled = false
	room.RemovedAt = null.TimeFrom(time.Now().UTC())
	_, err = room.Update(s.DB, boil.Whitelist(models.RoomColumns.Disabled, models.RoomColumns.RemovedAt))
	s.Require().NoError(err)
	req, _ = http.NewRequest("GET", fmt.Sprintf("/room/%d", room.GatewayUID), nil)
	s.apiAuth(req)
	resp = s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)
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
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	req, _ := http.NewRequest("GET", fmt.Sprintf("/room/%d", room.GatewayUID), nil)
	s.apiAuth(req)
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
				s.assertV1Session(sessions[j], data)
				break
			}
		}
		s.True(found, "unknown user [%d] %v", i, data["id"])
	}

	// turn on question mark on some session
	sessions[0].Question = true
	_, err := sessions[0].Update(s.DB, boil.Whitelist(models.SessionColumns.Question))
	s.Require().NoError(err)
	body = s.request200json(req)
	s.True(body["questions"].(bool), "questions true")

	// now turn off question and check
	sessions[0].Question = false
	_, err = sessions[0].Update(s.DB, boil.Whitelist(models.SessionColumns.Question))
	s.Require().NoError(err)
	body = s.request200json(req)
	s.False(body["questions"].(bool), "questions false again")
}

func (s *ApiTestSuite) TestListRooms() {
	counts := struct {
		gateways        int
		roomPerGateway  int
		sessionsPerRoom int
	}{
		gateways:        3,
		roomPerGateway:  3,
		sessionsPerRoom: 5,
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

	// create some inactive rooms
	for _, v := range gateways {
		s.createRoom(v)
	}

	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	req, _ := http.NewRequest("GET", "/rooms", nil)
	s.apiAuth(req)
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
			s.assertV1Session(session, data)
		}
	}
}

func (s *ApiTestSuite) TestListRoomsIsSorted() {
	gateway := s.createGateway()
	rooms := make([]*models.Room, 10)
	sessions := make([]*models.Session, len(rooms))
	for i := range rooms {
		rooms[i] = s.createRoom(gateway)
		user := s.createUser()
		sessions[i] = s.createSession(user, gateway, rooms[i])
	}
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	req, _ := http.NewRequest("GET", "/rooms", nil)
	s.apiAuth(req)
	resp := s.request(req)
	s.Require().Equal(http.StatusOK, resp.Code)

	var body []interface{}
	s.Require().NoError(json.Unmarshal(resp.Body.Bytes(), &body))
	s.Equal(len(rooms), len(body), "room count")

	for i, respRoom := range body {
		data := respRoom.(map[string]interface{})
		room := rooms[i]

		// verify room's attributes
		s.Equal(gateway.Name, data["janus"], "Janus")
		s.Equal(room.Name, data["description"], "description")
		s.False(data["questions"].(bool), "questions")
		s.Equal(1, int(data["num_users"].(float64)), "num_users")
	}

	// reorder sessions created_at
	for i, session := range sessions {
		session.CreatedAt = session.CreatedAt.Add(time.Duration(len(sessions)-i) * time.Second)
		_, err := session.Update(s.DB, boil.Whitelist(models.SessionColumns.CreatedAt))
		s.Require().NoError(err)
	}

	resp = s.request(req)
	s.Require().Equal(http.StatusOK, resp.Code)

	var body2 []interface{}
	s.Require().NoError(json.Unmarshal(resp.Body.Bytes(), &body2))
	s.Equal(len(rooms), len(body2), "room count")

	for i, respRoom := range body2 {
		data := respRoom.(map[string]interface{})
		room := rooms[len(rooms)-1-i]

		// verify room's attributes
		s.Equal(gateway.Name, data["janus"], "Janus")
		s.Equal(room.Name, data["description"], "description")
		s.False(data["questions"].(bool), "questions")
		s.Equal(1, int(data["num_users"].(float64)), "num_users")
	}
}

func (s *ApiTestSuite) TestGetUserMalformedID() {
	req, _ := http.NewRequest("GET", "/users/1234567890123456789012345678901234567890", nil)
	s.apiAuth(req)
	resp := s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)
}

func (s *ApiTestSuite) TestGetUserNotFound() {
	req, _ := http.NewRequest("GET", "/users/1", nil)
	s.apiAuth(req)
	resp := s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)

	// existing user without active session
	user := s.createUser()
	req, _ = http.NewRequest("GET", fmt.Sprintf("/users/%s", user.AccountsID), nil)
	s.apiAuth(req)
	resp = s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)

	// disabled user
	user.Disabled = true
	_, err := user.Update(s.DB, boil.Whitelist(models.UserColumns.Disabled))
	s.Require().NoError(err)
	resp = s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)

	// removed user
	user.Disabled = false
	user.RemovedAt = null.TimeFrom(time.Now().UTC())
	_, err = user.Update(s.DB, boil.Whitelist(models.UserColumns.Disabled, models.UserColumns.RemovedAt))
	s.Require().NoError(err)
	resp = s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)
}

func (s *ApiTestSuite) TestGetUser() {
	gateway := s.createGateway()
	room := s.createRoom(gateway)
	user := s.createUser()
	session := s.createSession(user, gateway, room)
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	req, _ := http.NewRequest("GET", fmt.Sprintf("/users/%s", user.AccountsID), nil)
	s.apiAuth(req)
	body := s.request200json(req)
	s.assertV1Session(session, body)

	// turn camera off
	session.Camera = false
	_, err := session.Update(s.DB, boil.Whitelist(models.SessionColumns.Camera))
	s.Require().NoError(err)
	body = s.request200json(req)
	s.False(body["camera"].(bool), "camera false")

	// turn camera on again
	session.Camera = true
	_, err = session.Update(s.DB, boil.Whitelist(models.SessionColumns.Camera))
	s.Require().NoError(err)
	body = s.request200json(req)
	s.True(body["camera"].(bool), "camera true")
}

func (s *ApiTestSuite) TestListUsers() {
	counts := struct {
		gateways        int
		roomPerGateway  int
		sessionsPerRoom int
	}{
		gateways:        2,
		roomPerGateway:  3,
		sessionsPerRoom: 5,
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
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	// create some inactive users
	for i := 0; i < counts.sessionsPerRoom; i++ {
		s.createUser()
	}

	req, _ := http.NewRequest("GET", "/users", nil)
	s.apiAuth(req)
	body := s.request200json(req)

	s.Equal(counts.gateways*counts.roomPerGateway*counts.sessionsPerRoom, len(body), "user count")

	for id, respSession := range body {
		data := respSession.(map[string]interface{})
		session, ok := sessions[data["id"].(string)]
		s.Require().True(ok, "unknown session [%s] %v", id, data["room"])
		s.assertV1Session(session, data)
	}
}

func (s *ApiTestSuite) TestUpdateSessionMalformedID() {
	req, _ := http.NewRequest("PUT", "/users/1234567890123456789012345678901234567890", nil)
	s.apiAuth(req)
	resp := s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)
}

func (s *ApiTestSuite) TestUpdateSessionBadJSON() {
	req, _ := http.NewRequest("PUT", "/users/12345678901234567890", bytes.NewBuffer([]byte("{\"bad\":\"json")))
	s.apiAuth(req)
	resp := s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)
}

func (s *ApiTestSuite) TestUpdateSessionUnknownGateway() {
	user := s.createUser()
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	v1User := s.makeV1user(nil, nil, user)
	payloadJson, _ := json.Marshal(v1User)

	req, _ := http.NewRequest("PUT", fmt.Sprintf("/users/%s", user.AccountsID), bytes.NewBuffer(payloadJson))
	s.apiAuth(req)
	resp := s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)
}

func (s *ApiTestSuite) TestUpdateSessionUnknownRoom() {
	user := s.createUser()
	gateway := s.createGateway()
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	v1User := s.makeV1user(gateway, nil, user)
	payloadJson, _ := json.Marshal(v1User)

	req, _ := http.NewRequest("PUT", fmt.Sprintf("/users/%s", user.AccountsID), bytes.NewBuffer(payloadJson))
	s.apiAuth(req)
	resp := s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)
}

func (s *ApiTestSuite) TestUpdateSession() {
	gateway := s.createGateway()
	room := s.createRoom(gateway)
	user := s.createUser()
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	v1User := s.makeV1user(gateway, room, user)
	payloadJson, _ := json.Marshal(v1User)
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/users/%s", user.AccountsID), bytes.NewBuffer(payloadJson))
	s.apiAuth(req)
	s.request200json(req)
	req, _ = http.NewRequest("GET", fmt.Sprintf("/users/%s", user.AccountsID), nil)
	s.apiAuth(req)
	body := s.request200json(req)
	s.assertV1User(v1User, body)

	v1User.Question = true
	v1User.Camera = true
	payloadJson, _ = json.Marshal(v1User)
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/users/%s", user.AccountsID), bytes.NewBuffer(payloadJson))
	s.apiAuth(req)
	s.request200json(req)
	req, _ = http.NewRequest("GET", fmt.Sprintf("/users/%s", user.AccountsID), nil)
	s.apiAuth(req)
	body = s.request200json(req)
	s.assertV1User(v1User, body)
}

func (s *ApiTestSuite) TestGetCompositeMalformedID() {
	req, _ := http.NewRequest("GET", "/qids/12345678901234567890", nil)
	s.apiAuth(req)
	resp := s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)
}

func (s *ApiTestSuite) TestGetCompositeNotFound() {
	req, _ := http.NewRequest("GET", "/qids/q1", nil)
	s.apiAuth(req)
	resp := s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)
}

func (s *ApiTestSuite) TestGetComposite() {
	gateway := s.createGateway()
	rooms := make([]*models.Room, 4)
	sessions := make([][]*models.Session, len(rooms))
	sessionsByID := make(map[string]*models.Session)
	for i := 0; i < 4; i++ {
		rooms[i] = s.createRoom(gateway)
		sessions[i] = make([]*models.Session, i+1)
		for j := 0; j < i+1; j++ {
			user := s.createUser()
			sessions[i][j] = s.createSession(user, gateway, rooms[i])
			sessionsByID[user.AccountsID] = sessions[i][j]
		}
	}
	composite := s.createComposite(rooms)
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	req, _ := http.NewRequest("GET", fmt.Sprintf("/qids/%s", composite.Name), nil)
	s.apiAuth(req)
	body := s.request200json(req)
	vquad, ok := body["vquad"]
	s.Require().True(ok, "vquad")
	vquadArr, ok := vquad.([]interface{})
	s.Require().True(ok, "vquad array")

	for i, respCRoom := range vquadArr {
		croom, ok := respCRoom.(map[string]interface{})
		s.Require().True(ok, "vquad array item")
		s.Equal(rooms[i].GatewayUID, int(croom["room"].(float64)), "room")
		s.Equal(gateway.Name, croom["janus"], "Janus")
		s.Equal(rooms[i].Name, croom["description"], "description")
		s.False(croom["questions"].(bool), "questions")
		s.Equal(i+1, int(croom["num_users"].(float64)), "num_users")

		// verify room's user sessions
		s.Equal(i+1, len(croom["users"].([]interface{})), "users count")
		for j, respUser := range croom["users"].([]interface{}) {
			data := respUser.(map[string]interface{})
			session, ok := sessionsByID[data["id"].(string)]
			s.Require().True(ok, "unknown session [%d][%d] %v", i, j, data["id"])
			s.assertV1Session(session, data)
		}
	}

	// turn on question mark on some session
	sessions[1][0].Question = true
	_, err := sessions[1][0].Update(s.DB, boil.Whitelist(models.SessionColumns.Question))
	s.Require().NoError(err)
	body = s.request200json(req)
	s.True(body["vquad"].([]interface{})[1].(map[string]interface{})["questions"].(bool), "questions true")

	// now turn off question and check
	sessions[1][0].Question = false
	_, err = sessions[1][0].Update(s.DB, boil.Whitelist(models.SessionColumns.Question))
	s.Require().NoError(err)
	body = s.request200json(req)
	s.False(body["vquad"].([]interface{})[1].(map[string]interface{})["questions"].(bool), "questions false again")
}

func (s *ApiTestSuite) TestListComposites() {
	counts := struct {
		gateways       int
		roomPerGateway int
	}{
		gateways:       2,
		roomPerGateway: 8,
	}
	gateways := make(map[int64]*models.Gateway, counts.gateways)
	rooms := make([][]*models.Room, counts.gateways)
	sessions := make(map[string]*models.Session)
	for i := 0; i < counts.gateways; i++ {
		gateway := s.createGateway()
		gateways[gateway.ID] = gateway
		rooms[i] = make([]*models.Room, counts.roomPerGateway)
		for j := 0; j < counts.roomPerGateway; j++ {
			room := s.createRoom(gateway)
			rooms[i][j] = room
			for k := 0; k < j%4+1; k++ {
				user := s.createUser()
				sessions[user.AccountsID] = s.createSession(user, gateway, room)
			}
		}
	}

	composites := make(map[string]*models.Composite, 4)
	composite := s.createComposite(rooms[0][0:4])
	composites[composite.Name] = composite
	composite = s.createComposite(rooms[0][4:])
	composites[composite.Name] = composite
	composite = s.createComposite(rooms[1][0:4])
	composites[composite.Name] = composite
	composite = s.createComposite(rooms[1][4:])
	composites[composite.Name] = composite

	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	req, _ := http.NewRequest("GET", "/qids", nil)
	s.apiAuth(req)
	body := s.request200json(req)

	s.Equal(4, len(body), "composites count")

	for name, respComposite := range body {
		composite, ok := composites[name]
		s.Require().True(ok, "unknown composite [%s] %v", name, respComposite)

		data, ok := respComposite.(map[string]interface{})
		s.Require().True(ok, "composite structure [%s] %v", name, respComposite)
		vquad, ok := data["vquad"]
		s.Require().True(ok, "vquad")
		vquadArr, ok := vquad.([]interface{})
		s.Require().True(ok, "vquad array")

		for i, respCRoom := range vquadArr {
			croom, ok := respCRoom.(map[string]interface{})
			s.Require().True(ok, "vquad array item")

			s.Require().NoError(composite.R.CompositesRooms[i].L.LoadRoom(s.DB, true, composite.R.CompositesRooms[i], nil))
			s.Require().NoError(composite.R.CompositesRooms[i].L.LoadGateway(s.DB, true, composite.R.CompositesRooms[i], nil))
			room := composite.R.CompositesRooms[i].R.Room
			s.Equal(room.GatewayUID, int(croom["room"].(float64)), "room")
			s.Equal(composite.R.CompositesRooms[i].R.Gateway.Name, croom["janus"], "Janus")
			s.Equal(room.Name, croom["description"], "description")
			s.False(croom["questions"].(bool), "questions")
			s.Equal(i+1, int(croom["num_users"].(float64)), "num_users")

			// verify room's user sessions
			s.Equal(i+1, len(croom["users"].([]interface{})), "users count")
			for j, respUser := range croom["users"].([]interface{}) {
				data := respUser.(map[string]interface{})
				session, ok := sessions[data["id"].(string)]
				s.Require().True(ok, "unknown session [%s][%d][%d] %v", name, i, j, data["id"])
				s.assertV1Session(session, data)
			}
		}
	}
}

func (s *ApiTestSuite) TestUpdateCompositeMalformedID() {
	req, _ := http.NewRequest("PUT", "/qids/12345678901234567890", nil)
	s.apiAuthP(req, []string{common.RoleShidur})
	resp := s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)
}

func (s *ApiTestSuite) TestUpdateCompositeNotFound() {
	b, _ := json.Marshal(V1Composite{})
	req, _ := http.NewRequest("PUT", "/qids/q1", bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleShidur})
	resp := s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)
}

func (s *ApiTestSuite) TestUpdateCompositeBadJSON() {
	req, _ := http.NewRequest("PUT", "/qids/q1", bytes.NewBuffer([]byte("{\"bad\":\"json")))
	s.apiAuthP(req, []string{common.RoleShidur})
	resp := s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)
}

func (s *ApiTestSuite) TestUpdateComposite() {
	gateway := s.createGateway()
	rooms := make([]*models.Room, 4)
	for i := 0; i < 4; i++ {
		rooms[i] = s.createRoom(gateway)
	}
	composite := s.createComposite(rooms)
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	req, _ := http.NewRequest("GET", fmt.Sprintf("/qids/%s", composite.Name), nil)
	s.apiAuth(req)
	body := s.request200json(req)
	s.assertTokenVerifier()

	rooms[0] = s.createRoom(gateway)
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	body["vquad"].([]interface{})[0].(map[string]interface{})["room"] = rooms[0].GatewayUID
	body["vquad"].([]interface{})[0].(map[string]interface{})["queue"] = 5
	b, _ := json.Marshal(body)
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/qids/%s", composite.Name), bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleShidur})
	body = s.request200json(req)
	s.Equal("success", body["result"], "PUT result")

	req, _ = http.NewRequest("GET", fmt.Sprintf("/qids/%s", composite.Name), nil)
	s.apiAuth(req)
	body = s.request200json(req)
	vquad, ok := body["vquad"]
	s.Require().True(ok, "vquad")
	vquadArr, ok := vquad.([]interface{})
	s.Require().True(ok, "vquad array")

	for i, respCRoom := range vquadArr {
		croom, ok := respCRoom.(map[string]interface{})
		s.Require().True(ok, "vquad array item")
		s.Equal(rooms[i].GatewayUID, int(croom["room"].(float64)), "room")
		s.Equal(gateway.Name, croom["janus"], "Janus")
		s.Equal(rooms[i].Name, croom["description"], "description")
		s.False(croom["questions"].(bool), "questions")
		s.Equal(0, int(croom["num_users"].(float64)), "num_users")
	}
}

func (s *ApiTestSuite) TestUpdateCompositeClear() {
	gateway := s.createGateway()
	rooms := make([]*models.Room, 4)
	for i := 0; i < 4; i++ {
		rooms[i] = s.createRoom(gateway)
	}
	composite := s.createComposite(rooms)
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/qids/%s", composite.Name), bytes.NewBuffer([]byte("{\"vquad\":[]}")))
	s.apiAuthP(req, []string{common.RoleShidur})
	body := s.request200json(req)
	s.Equal("success", body["result"], "PUT result")

	req, _ = http.NewRequest("GET", fmt.Sprintf("/qids/%s", composite.Name), nil)
	s.apiAuth(req)
	body = s.request200json(req)
	vquad, ok := body["vquad"]
	s.Require().True(ok, "vquad")
	vquadArr, ok := vquad.([]interface{})
	s.Require().True(ok, "vquad array")
	s.Empty(vquadArr, 0, "vquad len")
}

func (s *ApiTestSuite) TestUpdateCompositeDuplicateRoom() {
	gateway := s.createGateway()
	rooms := make([]*models.Room, 4)
	for i := 0; i < 4; i++ {
		rooms[i] = s.createRoom(gateway)
	}
	composite := s.createComposite(rooms)
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	req, _ := http.NewRequest("GET", fmt.Sprintf("/qids/%s", composite.Name), nil)
	s.apiAuth(req)
	body := s.request200json(req)
	s.assertTokenVerifier()
	bodyArr := body["vquad"].([]interface{})
	for i := 0; i < 4; i++ {
		bodyArr[i].(map[string]interface{})["room"] = rooms[0].GatewayUID
		bodyArr[i].(map[string]interface{})["queue"] = 5 + i
	}
	b, _ := json.Marshal(body)
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/qids/%s", composite.Name), bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleShidur})
	body = s.request200json(req)
	s.Equal("success", body["result"], "PUT result")

	req, _ = http.NewRequest("GET", fmt.Sprintf("/qids/%s", composite.Name), nil)
	s.apiAuth(req)
	body = s.request200json(req)
	vquad, ok := body["vquad"]
	s.Require().True(ok, "vquad")
	vquadArr, ok := vquad.([]interface{})
	s.Require().True(ok, "vquad array")

	for _, respCRoom := range vquadArr {
		croom, ok := respCRoom.(map[string]interface{})
		s.Require().True(ok, "vquad array item")
		s.Equal(rooms[0].GatewayUID, int(croom["room"].(float64)), "room")
		s.Equal(gateway.Name, croom["janus"], "Janus")
		s.Equal(rooms[0].Name, croom["description"], "description")
		s.False(croom["questions"].(bool), "questions")
		s.Equal(0, int(croom["num_users"].(float64)), "num_users")
	}
}

func (s *ApiTestSuite) TestHandleApiUnauthorized() {
	req, _ := http.NewRequest("GET", "/v2/config", nil)
	resp := s.request(req)
	s.Equal(http.StatusUnauthorized, resp.Code, "no header")

	req.SetBasicAuth("", "")
	resp = s.request(req)
	s.Equal(http.StatusUnauthorized, resp.Code, "no username password")

	req.SetBasicAuth("username", "")
	resp = s.request(req)
	s.Equal(http.StatusUnauthorized, resp.Code, "no password")

	req.SetBasicAuth("", "password")
	resp = s.request(req)
	s.Equal(http.StatusUnauthorized, resp.Code, "no username")

	req.SetBasicAuth("username", "password")
	resp = s.request(req)
	s.Equal(http.StatusUnauthorized, resp.Code, "wrong username password")
}

func (s *ApiTestSuite) TestHandleApiForbidden() {
	for _, role := range common.AllRoles {
		req, _ := http.NewRequest("GET", "/v2/config", nil)
		s.apiAuthP(req, []string{role})
		s.request200json(req)
		s.assertTokenVerifier()
	}

	req, _ := http.NewRequest("GET", "/v2/config", nil)
	s.apiAuthP(req, []string{"some_role"})
	resp := s.request(req)
	s.Equal(http.StatusForbidden, resp.Code, "unknown role")
	s.assertTokenVerifier()
}

func (s *ApiTestSuite) TestHandleGatewayUnauthorized() {
	for _, endpoint := range [...]string{"/event", "/protocol"} {
		req, _ := http.NewRequest("POST", endpoint, bytes.NewBuffer([]byte("{}")))
		resp := s.request(req)
		s.Equal(http.StatusUnauthorized, resp.Code, "%s no header", endpoint)

		req.SetBasicAuth("", "")
		resp = s.request(req)
		s.Equal(http.StatusUnauthorized, resp.Code, "%s no username password", endpoint)

		req.SetBasicAuth("username", "")
		resp = s.request(req)
		s.Equal(http.StatusUnauthorized, resp.Code, "%s no password", endpoint)

		req.SetBasicAuth("", "password")
		resp = s.request(req)
		s.Equal(http.StatusUnauthorized, resp.Code, "%s no username", endpoint)

		req.SetBasicAuth("username", "password")
		resp = s.request(req)
		s.Equal(http.StatusUnauthorized, resp.Code, "%s wrong username password", endpoint)
	}
}

func (s *ApiTestSuite) TestHandleEventBadJSON() {
	gateway := s.createGateway()
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	req, _ := http.NewRequest("POST", "/event", bytes.NewBuffer([]byte("{\"bad\":\"json")))
	req.SetBasicAuth(gateway.Name, gateway.Name)
	resp := s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)
}

func (s *ApiTestSuite) TestHandleEventUnknownType() {
	gateway := s.createGateway()
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	req, _ := http.NewRequest("POST", "/event", bytes.NewBuffer([]byte("{\"type\":7}")))
	req.SetBasicAuth(gateway.Name, gateway.Name)
	resp := s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)
}

func (s *ApiTestSuite) TestHandleEventVideoroomLeaving() {
	gateway := s.createGateway()
	room := s.createRoom(gateway)
	user := s.createUser()
	session := s.createSession(user, gateway, room)
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	s.Require().NoError(session.L.LoadUser(s.DB, true, session, nil))
	v1User := s.app.makeV1User(room, session)
	v1UserJson, _ := json.Marshal(v1User)
	event := janus.PluginEvent{
		BaseEvent: janus.BaseEvent{
			Emitter:   gateway.Name,
			Type:      64,
			Timestamp: time.Now().UTC().Unix(),
			Session:   uint64(session.GatewaySession.Int64),
			Handle:    uint64(session.GatewayHandle.Int64),
		},
		Event: janus.PluginEventBody{
			Plugin: "janus.plugin.videoroom",
			Data: map[string]interface{}{
				"event":   "leaving",
				"display": string(v1UserJson),
			},
		},
	}
	b, _ := json.Marshal(event)

	req, _ := http.NewRequest("POST", "/event", bytes.NewBuffer(b))
	req.SetBasicAuth(gateway.Name, gateway.Name)
	s.request200json(req)

	s.Require().NoError(session.Reload(s.DB))
	s.True(session.RemovedAt.Valid, "removed_at")
}

func (s *ApiTestSuite) TestHandleEventVideoroomLeavingUnknownUser() {
	gateway := s.createGateway()
	room := s.createRoom(gateway)
	user := s.createUser()
	session := s.createSession(user, gateway, room)
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	s.Require().NoError(session.L.LoadUser(s.DB, true, session, nil))
	v1User := s.app.makeV1User(room, session)
	v1User.ID = "some_new_user_id"
	v1UserJson, _ := json.Marshal(v1User)
	event := janus.PluginEvent{
		BaseEvent: janus.BaseEvent{
			Emitter:   gateway.Name,
			Type:      64,
			Timestamp: time.Now().UTC().Unix(),
			Session:   uint64(session.GatewaySession.Int64),
			Handle:    uint64(session.GatewayHandle.Int64),
		},
		Event: janus.PluginEventBody{
			Plugin: "janus.plugin.videoroom",
			Data: map[string]interface{}{
				"event":   "leaving",
				"display": string(v1UserJson),
			},
		},
	}
	b, _ := json.Marshal(event)

	req, _ := http.NewRequest("POST", "/event", bytes.NewBuffer(b))
	req.SetBasicAuth(gateway.Name, gateway.Name)
	s.request200json(req)

	// no changes to existing sessions as we expect a noop
	count, err := models.Sessions().Count(s.DB)
	s.Require().NoError(err)
	s.EqualValues(1, count, "existing sessions count")
	s.Require().NoError(session.Reload(s.DB))
	s.False(session.RemovedAt.Valid, "removed_at")

	// new user record
	ok, err := models.Users(models.UserWhere.AccountsID.EQ(v1User.ID)).Exists(s.DB)
	s.Require().NoError(err)
	s.True(ok, "new user exists")
}

func (s *ApiTestSuite) TestHandleProtocolBadJSON() {
	gateway := s.createGateway()
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	req, _ := http.NewRequest("POST", "/protocol", bytes.NewBuffer([]byte("{\"bad\":\"json")))
	req.SetBasicAuth(gateway.Name, gateway.Name)
	resp := s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)
}

func (s *ApiTestSuite) TestHandleProtocolUnknownType() {
	gateway := s.createGateway()
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	req, _ := http.NewRequest("POST", "/protocol", bytes.NewBuffer([]byte("{\"type\":7}")))
	req.SetBasicAuth(gateway.Name, gateway.Name)
	resp := s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)
}

func (s *ApiTestSuite) TestHandleProtocolBadTextJSON() {
	gateway := s.createGateway()
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	event := janus.TextroomPostMsg{
		Textroom: "message",
		Room:     1000,
		From:     "someone",
		Date:     janus.DateTime{Time: time.Now()},
		Text:     "{\"bad\":\"json",
		Whisper:  false,
	}
	b, _ := json.Marshal(event)

	req, _ := http.NewRequest("POST", "/protocol", bytes.NewBuffer(b))
	req.SetBasicAuth(gateway.Name, gateway.Name)
	resp := s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)
}

func (s *ApiTestSuite) TestHandleProtocolUnknownGateway() {
	user := s.createUser()
	gateway := s.createGateway()
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	v1User := s.makeV1user(nil, nil, user)
	payload := map[string]interface{}{
		"type":   "enter",
		"status": true,
		"user":   v1User,
	}
	payloadJson, _ := json.Marshal(payload)

	event := janus.TextroomPostMsg{
		Textroom: "message",
		Room:     1000,
		From:     v1User.ID,
		Date:     janus.DateTime{Time: time.Now()},
		Text:     string(payloadJson),
		Whisper:  false,
	}
	b, _ := json.Marshal(event)

	req, _ := http.NewRequest("POST", "/protocol", bytes.NewBuffer(b))
	req.SetBasicAuth(gateway.Name, gateway.Name)
	resp := s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)
}

func (s *ApiTestSuite) TestHandleProtocolUnknownRoom() {
	gateway := s.createGateway()
	user := s.createUser()
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	v1User := s.makeV1user(gateway, nil, user)
	payload := map[string]interface{}{
		"type":   "enter",
		"status": true,
		"user":   v1User,
	}
	payloadJson, _ := json.Marshal(payload)

	event := janus.TextroomPostMsg{
		Textroom: "message",
		Room:     1000,
		From:     v1User.ID,
		Date:     janus.DateTime{Time: time.Now()},
		Text:     string(payloadJson),
		Whisper:  false,
	}
	b, _ := json.Marshal(event)

	req, _ := http.NewRequest("POST", "/protocol", bytes.NewBuffer(b))
	req.SetBasicAuth(gateway.Name, gateway.Name)
	resp := s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)
}

func (s *ApiTestSuite) TestHandleProtocolEnter() {
	gateway := s.createGateway()
	room := s.createRoom(gateway)
	user := s.createUser()
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	v1User := s.makeV1user(gateway, room, user)

	payload := map[string]interface{}{
		"type":   "enter",
		"status": true,
		"room":   room.GatewayUID,
		"user":   v1User,
	}
	payloadJson, _ := json.Marshal(payload)

	event := janus.TextroomPostMsg{
		Textroom: "message",
		Room:     1000,
		From:     v1User.ID,
		Date:     janus.DateTime{Time: time.Now()},
		Text:     string(payloadJson),
		Whisper:  false,
	}
	b, _ := json.Marshal(event)

	req, _ := http.NewRequest("POST", "/protocol", bytes.NewBuffer(b))
	req.SetBasicAuth(gateway.Name, gateway.Name)
	s.request200json(req)

	req, _ = http.NewRequest("GET", fmt.Sprintf("/users/%s", user.AccountsID), nil)
	s.apiAuth(req)
	body := s.request200json(req)
	s.assertV1User(v1User, body)
}

func (s *ApiTestSuite) TestHandleProtocolEnterUnknownUser() {
	gateway := s.createGateway()
	room := s.createRoom(gateway)
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	v1User := s.makeV1user(gateway, room, nil)
	v1User.ID = "some_new_user_id"
	v1User.Display = "test user"
	v1User.Email = "user@example.com"
	v1User.Username = "username"

	payload := map[string]interface{}{
		"type":   "enter",
		"status": true,
		"room":   room.GatewayUID,
		"user":   v1User,
	}
	payloadJson, _ := json.Marshal(payload)

	event := janus.TextroomPostMsg{
		Textroom: "message",
		Room:     1000,
		From:     v1User.ID,
		Date:     janus.DateTime{Time: time.Now()},
		Text:     string(payloadJson),
		Whisper:  false,
	}
	b, _ := json.Marshal(event)

	req, _ := http.NewRequest("POST", "/protocol", bytes.NewBuffer(b))
	req.SetBasicAuth(gateway.Name, gateway.Name)
	s.request200json(req)

	req, _ = http.NewRequest("GET", "/users/some_new_user_id", nil)
	s.apiAuth(req)
	body := s.request200json(req)
	s.assertV1User(v1User, body)
}

func (s *ApiTestSuite) TestHandleProtocolEnterExistingSession() {
	gateway := s.createGateway()
	room := s.createRoom(gateway)
	user := s.createUser()
	session := s.createSession(user, gateway, room)
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	v1User := s.makeV1user(gateway, room, user)
	v1User.Session = session.GatewaySession.Int64
	v1User.Handle = session.GatewayHandle.Int64

	payload := map[string]interface{}{
		"type":   "enter",
		"status": true,
		"room":   room.GatewayUID,
		"user":   v1User,
	}
	payloadJson, _ := json.Marshal(payload)

	event := janus.TextroomPostMsg{
		Textroom: "message",
		Room:     1000,
		From:     v1User.ID,
		Date:     janus.DateTime{Time: time.Now()},
		Text:     string(payloadJson),
		Whisper:  false,
	}
	b, _ := json.Marshal(event)

	req, _ := http.NewRequest("POST", "/protocol", bytes.NewBuffer(b))
	req.SetBasicAuth(gateway.Name, gateway.Name)
	s.request200json(req)

	req, _ = http.NewRequest("GET", fmt.Sprintf("/users/%s", user.AccountsID), nil)
	s.apiAuth(req)
	body := s.request200json(req)
	s.assertV1User(v1User, body)

	s.Require().NoError(session.Reload(s.DB))
	s.False(session.RemovedAt.Valid, "removed_at")
}

func (s *ApiTestSuite) TestHandleProtocolQuestion() {
	gateway := s.createGateway()
	room := s.createRoom(gateway)
	user := s.createUser()
	session := s.createSession(user, gateway, room)
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	v1User := s.makeV1user(gateway, room, user)
	v1User.Session = session.GatewaySession.Int64
	v1User.Handle = session.GatewayHandle.Int64
	v1User.Question = true
	payload := map[string]interface{}{
		"type":   "question",
		"status": true,
		"room":   room.GatewayUID,
		"user":   v1User,
	}
	payloadJson, _ := json.Marshal(payload)

	event := janus.TextroomPostMsg{
		Textroom: "message",
		Room:     1000,
		From:     v1User.ID,
		Date:     janus.DateTime{Time: time.Now()},
		Text:     string(payloadJson),
		Whisper:  false,
	}
	b, _ := json.Marshal(event)

	req, _ := http.NewRequest("POST", "/protocol", bytes.NewBuffer(b))
	req.SetBasicAuth(gateway.Name, gateway.Name)
	s.request200json(req)

	req, _ = http.NewRequest("GET", fmt.Sprintf("/users/%s", user.AccountsID), nil)
	s.apiAuth(req)
	body := s.request200json(req)
	s.assertV1User(v1User, body)

	s.Require().NoError(session.Reload(s.DB))
	s.True(session.Question, "question")

	// now turn it off
	v1User.Question = false
	payloadJson, _ = json.Marshal(payload)
	event.Text = string(payloadJson)
	b, _ = json.Marshal(event)

	req, _ = http.NewRequest("POST", "/protocol", bytes.NewBuffer(b))
	req.SetBasicAuth(gateway.Name, gateway.Name)
	s.request200json(req)

	req, _ = http.NewRequest("GET", fmt.Sprintf("/users/%s", user.AccountsID), nil)
	s.apiAuth(req)
	body = s.request200json(req)
	s.assertV1User(v1User, body)

	s.Require().NoError(session.Reload(s.DB))
	s.False(session.Question, "question false")
}

func (s *ApiTestSuite) TestHandleProtocolCamera() {
	gateway := s.createGateway()
	room := s.createRoom(gateway)
	user := s.createUser()
	session := s.createSession(user, gateway, room)
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	v1User := s.makeV1user(gateway, room, user)
	v1User.Session = session.GatewaySession.Int64
	v1User.Handle = session.GatewayHandle.Int64
	v1User.Camera = true
	payload := map[string]interface{}{
		"type":   "camera",
		"status": true,
		"room":   room.GatewayUID,
		"user":   v1User,
	}
	payloadJson, _ := json.Marshal(payload)

	event := janus.TextroomPostMsg{
		Textroom: "message",
		Room:     1000,
		From:     v1User.ID,
		Date:     janus.DateTime{Time: time.Now()},
		Text:     string(payloadJson),
		Whisper:  false,
	}
	b, _ := json.Marshal(event)

	req, _ := http.NewRequest("POST", "/protocol", bytes.NewBuffer(b))
	req.SetBasicAuth(gateway.Name, gateway.Name)
	s.request200json(req)

	req, _ = http.NewRequest("GET", fmt.Sprintf("/users/%s", user.AccountsID), nil)
	s.apiAuth(req)
	body := s.request200json(req)
	s.assertV1User(v1User, body)

	s.Require().NoError(session.Reload(s.DB))
	s.True(session.Camera, "camera")

	// now turn it off
	v1User.Camera = false
	payloadJson, _ = json.Marshal(payload)
	event.Text = string(payloadJson)
	b, _ = json.Marshal(event)

	req, _ = http.NewRequest("POST", "/protocol", bytes.NewBuffer(b))
	req.SetBasicAuth(gateway.Name, gateway.Name)
	s.request200json(req)

	req, _ = http.NewRequest("GET", fmt.Sprintf("/users/%s", user.AccountsID), nil)
	s.apiAuth(req)
	body = s.request200json(req)
	s.assertV1User(v1User, body)

	s.Require().NoError(session.Reload(s.DB))
	s.False(session.Camera, "camera false")
}

func (s *ApiTestSuite) TestHandleProtocolSoundTest() {
	gateway := s.createGateway()
	room := s.createRoom(gateway)
	user := s.createUser()
	session := s.createSession(user, gateway, room)
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	v1User := s.makeV1user(gateway, room, user)
	v1User.Session = session.GatewaySession.Int64
	v1User.Handle = session.GatewayHandle.Int64
	v1User.SoundTest = true
	payload := map[string]interface{}{
		"type":   "sound-test",
		"status": true,
		"room":   room.GatewayUID,
		"user":   v1User,
	}
	payloadJson, _ := json.Marshal(payload)

	event := janus.TextroomPostMsg{
		Textroom: "message",
		Room:     1000,
		From:     v1User.ID,
		Date:     janus.DateTime{Time: time.Now()},
		Text:     string(payloadJson),
		Whisper:  false,
	}
	b, _ := json.Marshal(event)

	req, _ := http.NewRequest("POST", "/protocol", bytes.NewBuffer(b))
	req.SetBasicAuth(gateway.Name, gateway.Name)
	s.request200json(req)

	req, _ = http.NewRequest("GET", fmt.Sprintf("/users/%s", user.AccountsID), nil)
	s.apiAuth(req)
	body := s.request200json(req)
	s.assertV1User(v1User, body)

	s.Require().NoError(session.Reload(s.DB))
	s.True(session.SoundTest, "sound-test")

	// now turn it off
	v1User.SoundTest = false
	payloadJson, _ = json.Marshal(payload)
	event.Text = string(payloadJson)
	b, _ = json.Marshal(event)

	req, _ = http.NewRequest("POST", "/protocol", bytes.NewBuffer(b))
	req.SetBasicAuth(gateway.Name, gateway.Name)
	s.request200json(req)

	req, _ = http.NewRequest("GET", fmt.Sprintf("/users/%s", user.AccountsID), nil)
	s.apiAuth(req)
	body = s.request200json(req)
	s.assertV1User(v1User, body)

	s.Require().NoError(session.Reload(s.DB))
	s.False(session.SoundTest, "sound-test false")
}

func (s *ApiTestSuite) TestV2GetConfig() {
	roomsGateways := make(map[string]*models.Gateway)
	streamingGateways := make(map[string]*models.Gateway)
	for i := 0; i < 3; i++ {
		gateway := s.createGateway()
		roomsGateways[gateway.Name] = gateway
		gateway = s.createGatewayP(common.GatewayTypeStreaming)
		streamingGateways[gateway.Name] = gateway
	}
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	req, _ := http.NewRequest("GET", "/v2/config", nil)
	s.apiAuth(req)
	body := s.request200json(req)

	gateways := body["gateways"].(map[string]interface{})
	for name, respGateway := range gateways[common.GatewayTypeRooms].(map[string]interface{}) {
		gateway, ok := roomsGateways[name]
		s.Require().True(ok, "unknown rooms gateway %s", name)
		data := respGateway.(map[string]interface{})
		s.Equal(gateway.Name, data["name"], "name")
		s.Equal(gateway.URL, data["url"], "url")
		s.Equal(gateway.Type, data["type"], "type")
		s.Empty(data["admin_url"], "admin_url")
		s.Empty(data["admin_password"], "admin_password")
	}
	for name, respGateway := range gateways[common.GatewayTypeStreaming].(map[string]interface{}) {
		gateway, ok := streamingGateways[name]
		s.Require().True(ok, "unknown rooms gateway %s", name)
		data := respGateway.(map[string]interface{})
		s.Equal(gateway.Name, data["name"], "name")
		s.Equal(gateway.URL, data["url"], "url")
		s.Equal(gateway.Type, data["type"], "type")
		s.Empty(data["admin_url"], "admin_url")
		s.Empty(data["admin_password"], "admin_password")
	}

	iceServers := body["ice_servers"].(map[string]interface{})

	s.ElementsMatch(common.Config.IceServers[common.GatewayTypeRooms], iceServers[common.GatewayTypeRooms], "rooms ice servers")
	s.ElementsMatch(common.Config.IceServers[common.GatewayTypeStreaming], iceServers[common.GatewayTypeStreaming], "streaming ice servers")
}

func (s *ApiTestSuite) TestV2GetConfigWithAdmin() {
	gateway := s.createGateway()
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	for _, role := range []string{common.RoleAdmin, common.RoleRoot} {
		req, _ := http.NewRequest("GET", "/v2/config", nil)
		s.apiAuthP(req, []string{role})
		body := s.request200json(req)
		gateways := body["gateways"].(map[string]interface{})
		for _, respGateway := range gateways[common.GatewayTypeRooms].(map[string]interface{}) {
			data := respGateway.(map[string]interface{})
			s.Equal(gateway.Name, data["name"], "name")
			s.Equal(gateway.URL, data["url"], "url")
			s.Equal(gateway.Type, data["type"], "type")
			s.Equal(gateway.AdminURL, data["admin_url"], "admin_url")
			s.Equal(gateway.AdminPassword, data["admin_password"], "admin_password")
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

func (s *ApiTestSuite) apiAuth(req *http.Request) {
	s.apiAuthP(req, []string{common.RoleUser})
}

func (s *ApiTestSuite) apiAuthP(req *http.Request, roles []string) {
	req.Header.Set("Authorization", "Bearer token")

	oidcIDToken := &oidc.IDToken{
		Issuer:          "https://test.issuer",
		Audience:        []string{"Audience"},
		Subject:         "Subject",
		Expiry:          time.Now().Add(10 * time.Minute),
		IssuedAt:        time.Now(),
		Nonce:           "nonce",
		AccessTokenHash: "access_token_hash",
	}

	claims := middleware.IDTokenClaims{
		Aud: oidcIDToken.Audience,
		Exp: int(oidcIDToken.Expiry.Unix()),
		Iat: int(oidcIDToken.IssuedAt.Unix()),
		Iss: oidcIDToken.Issuer,
		RealmAccess: middleware.Roles{
			Roles: roles,
		},
		Sub: oidcIDToken.Subject,
	}

	b, err := json.Marshal(claims)
	s.Require().NoError(err, "json.Marshal(claims)")

	pointerVal := reflect.ValueOf(oidcIDToken)
	val := reflect.Indirect(pointerVal)
	member := val.FieldByName("claims")
	ptrToY := unsafe.Pointer(member.UnsafeAddr())
	realPtrToY := (*[]byte)(ptrToY)
	*realPtrToY = b

	s.tokenVerifier.On("Verify", mock.Anything, "token").Return(oidcIDToken, nil)
}

func (s *ApiTestSuite) assertV1Session(session *models.Session, actual map[string]interface{}) {
	if session.R == nil {
		s.Require().NoError(session.L.LoadUser(s.DB, true, session, nil))
		s.Require().NoError(session.L.LoadRoom(s.DB, true, session, nil))
		s.Require().NoError(session.L.LoadGateway(s.DB, true, session, nil))
	}
	s.Equal(session.R.User.AccountsID, actual["id"], "id")
	s.Equal(session.Display.String, actual["display"], "display")
	s.Equal(session.R.User.Email.String, actual["email"], "email")
	s.Equal(session.R.Room.GatewayUID, int(actual["room"].(float64)), "room")
	s.Equal(session.IPAddress.String, actual["ip"], "ip")
	s.Equal(session.R.Gateway.Name, actual["janus"], "janus")
	s.Equal("user", actual["role"], "role")
	s.Equal(session.UserAgent.String, actual["system"], "system")
	s.Equal(session.R.User.Username.String, actual["username"], "username")
	s.Equal(session.CreatedAt.Unix(), int64(actual["timestamp"].(float64)), "timestamp")
	s.Equal(session.GatewaySession.Int64, int64(actual["session"].(float64)), "session")
	s.Equal(session.GatewayHandle.Int64, int64(actual["handle"].(float64)), "handle")
	s.Equal(session.GatewayFeed.Int64, int64(actual["rfid"].(float64)), "rfid")
	s.Equal(session.Camera, actual["camera"], "camera")
	s.Equal(session.Question, actual["question"], "question")
	s.Equal(session.SelfTest, actual["self_test"], "self_test")
	s.Equal(session.SoundTest, actual["sound_test"], "sound_test")
}

func (s *ApiTestSuite) assertV1User(v1User *V1User, body map[string]interface{}) {
	s.Equal(v1User.ID, body["id"], "id")
	s.Equal(v1User.Display, body["display"], "display")
	s.Equal(v1User.Email, body["email"], "email")
	s.Equal(v1User.Room, int(body["room"].(float64)), "room")
	s.Equal(v1User.IP, body["ip"], "ip")
	s.Equal(v1User.Janus, body["janus"], "janus")
	s.Equal(v1User.Role, body["role"], "role")
	s.Equal(v1User.System, body["system"], "system")
	s.Equal(v1User.Username, body["username"], "username")
	s.InEpsilon(v1User.Timestamp, int64(body["timestamp"].(float64)), 1, "timestamp")
	s.Equal(v1User.Session, int64(body["session"].(float64)), "session")
	s.Equal(v1User.Handle, int64(body["handle"].(float64)), "handle")
	s.Equal(v1User.RFID, int64(body["rfid"].(float64)), "rfid")
	s.Equal(v1User.Camera, body["camera"], "camera")
	s.Equal(v1User.Question, body["question"], "question")
	s.Equal(v1User.SelfTest, body["self_test"], "self_test")
	s.Equal(v1User.SoundTest, body["sound_test"], "sound_test")
}

func (s *ApiTestSuite) createGateway() *models.Gateway {
	return s.createGatewayP(common.GatewayTypeRooms)
}

func (s *ApiTestSuite) createGatewayP(gType string) *models.Gateway {
	name := fmt.Sprintf("gateway_%s", stringutil.GenerateName(4))
	pwdHash, err := bcrypt.GenerateFromPassword([]byte(name), bcrypt.MinCost)
	s.Require().NoError(err)

	gateway := &models.Gateway{
		Name:           name,
		URL:            "url",
		AdminURL:       "admin_url",
		AdminPassword:  "admin_password",
		EventsPassword: string(pwdHash),
		Type:           gType,
	}

	s.Require().NoError(gateway.Insert(s.DB, boil.Infer()))

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
	s.Require().NoError(user.Insert(s.DB, boil.Infer()))
	return user
}

func (s *ApiTestSuite) createRoom(gateway *models.Gateway) *models.Room {
	room := &models.Room{
		Name:             fmt.Sprintf("room_%s", stringutil.GenerateName(10)),
		DefaultGatewayID: gateway.ID,
		GatewayUID:       rand.Intn(math.MaxInt32),
	}
	s.Require().NoError(room.Insert(s.DB, boil.Infer()))
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
	s.Require().NoError(session.Insert(s.DB, boil.Infer()))
	return session
}

func (s *ApiTestSuite) createComposite(rooms []*models.Room) *models.Composite {
	composite := &models.Composite{
		Name: fmt.Sprintf("q%d", rand.Intn(math.MaxInt16)),
	}
	s.Require().NoError(composite.Insert(s.DB, boil.Infer()))

	cRooms := make([]*models.CompositesRoom, len(rooms))
	for i, room := range rooms {
		cRooms[i] = &models.CompositesRoom{
			RoomID:    room.ID,
			GatewayID: room.DefaultGatewayID,
			Position:  i + 1,
		}
	}
	s.Require().NoError(composite.AddCompositesRooms(s.DB, true, cRooms...))

	return composite
}

func (s *ApiTestSuite) makeV1user(gateway *models.Gateway, room *models.Room, user *models.User) *V1User {
	v1User := &V1User{
		Group:     "Test Room",
		IP:        "127.0.0.1",
		Name:      fmt.Sprintf("user-%s", stringutil.GenerateName(4)),
		Role:      "user",
		System:    "user_agent",
		Timestamp: time.Now().Unix(),
		Session:   rand.Int63n(math.MaxInt32),
		Handle:    rand.Int63n(math.MaxInt32),
		RFID:      rand.Int63n(math.MaxInt32),
		Camera:    false,
		Question:  false,
		SelfTest:  false,
		SoundTest: false,
	}
	if user != nil {
		v1User.ID = user.AccountsID
		v1User.Display = fmt.Sprintf("%s %s", user.FirstName.String, user.LastName.String)
		v1User.Email = user.Email.String
		v1User.Username = user.Username.String
	}

	if gateway != nil {
		v1User.Janus = gateway.Name
	}
	if room != nil {
		v1User.Room = room.GatewayUID
	}

	return v1User
}

func (s *ApiTestSuite) assertTokenVerifier() {
	s.tokenVerifier.AssertExpectations(s.T())
	s.tokenVerifier.ExpectedCalls = nil
	s.tokenVerifier.Calls = nil
}

func TestApiTestSuite(t *testing.T) {
	suite.Run(t, new(ApiTestSuite))
}
