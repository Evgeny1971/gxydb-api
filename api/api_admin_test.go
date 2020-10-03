package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"time"

	janus_plugins "github.com/edoshor/janus-go/plugins"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"

	"github.com/Bnei-Baruch/gxydb-api/common"
	"github.com/Bnei-Baruch/gxydb-api/domain"
	"github.com/Bnei-Baruch/gxydb-api/models"
	"github.com/Bnei-Baruch/gxydb-api/pkg/stringutil"
)

func (s *ApiTestSuite) TestAdmin_ListGatewaysForbidden() {
	req, _ := http.NewRequest("GET", "/admin/gateways", nil)
	resp := s.request(req)
	s.Require().Equal(http.StatusUnauthorized, resp.Code)

	req, _ = http.NewRequest("GET", "/admin/gateways", nil)
	s.apiAuth(req)
	resp = s.request(req)
	s.Require().Equal(http.StatusForbidden, resp.Code)
}

func (s *ApiTestSuite) TestAdmin_ListGatewaysBadRequest() {
	args := [...]string{
		"page_no=0",
		"page_no=-1",
		"page_no=abc",
		"page_size=0",
		"page_size=-1",
		"page_size=abc",
	}
	for i, query := range args {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/admin/gateways?%s", query), nil)
		s.apiAuthP(req, []string{common.RoleRoot})
		resp := s.request(req)
		s.Require().Equal(http.StatusBadRequest, resp.Code, i)
	}
}

func (s *ApiTestSuite) TestAdmin_ListGateways() {
	req, _ := http.NewRequest("GET", "/admin/gateways", nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	body := s.request200json(req)
	s.Equal(0, int(body["total"].(float64)), "total")
	s.Equal(0, len(body["data"].([]interface{})), "len(data)")

	gateways := make([]*models.Gateway, 10)
	for i := range gateways {
		gateways[i] = s.CreateGateway()
	}

	body = s.request200json(req)
	s.Equal(10, int(body["total"].(float64)), "total")
	s.Equal(10, len(body["data"].([]interface{})), "len(data)")

	for i, gateway := range gateways {
		req, _ = http.NewRequest("GET", fmt.Sprintf("/admin/gateways?page_no=%d&page_size=1&order_by=id", i+1), nil)
		s.apiAuthP(req, []string{common.RoleRoot})
		body = s.request200json(req)

		s.Equal(10, int(body["total"].(float64)), "total")
		data := body["data"].([]interface{})
		s.Equal(1, len(data), "len(data)")
		gatewayData := data[0].(map[string]interface{})
		s.Equal(gatewayData["name"], gateway.Name, "name")
		s.Equal(gatewayData["description"], gateway.Description.String, "description")
		s.NotContains(gatewayData, "admin_password", "admin_password")
		s.NotContains(gatewayData, "events_password", "events_password")
	}
}

func (s *ApiTestSuite) TestAdmin_GatewaysHandleInfoForbidden() {
	req, _ := http.NewRequest("GET", "/admin/gateways/1/sessions/1/handles/1/info", nil)
	resp := s.request(req)
	s.Require().Equal(http.StatusUnauthorized, resp.Code)

	req, _ = http.NewRequest("GET", "/admin/gateways/1/sessions/1/handles/1/info", nil)
	s.apiAuth(req)
	resp = s.request(req)
	s.Require().Equal(http.StatusForbidden, resp.Code)
}

func (s *ApiTestSuite) TestAdmin_GatewaysHandleInfoNotFound() {
	req, _ := http.NewRequest("GET", "/admin/gateways/1/sessions/1/handles/1/info", nil)
	s.apiAuthP(req, []string{common.RoleAdmin})
	resp := s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)

	gateway := s.CreateGatewayP(common.GatewayTypeRooms, s.GatewayManager.Config.AdminURL, s.GatewayManager.Config.AdminSecret)
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	req, _ = http.NewRequest("GET", fmt.Sprintf("/admin/gateways/%s/sessions/1/handles/1/info", gateway.Name), nil)
	s.apiAuthP(req, []string{common.RoleAdmin})
	resp = s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)

	session, err := s.NewGatewaySession()
	s.Require().NoError(err, "NewGatewaySession")
	defer session.Destroy()

	req, _ = http.NewRequest("GET", fmt.Sprintf("/admin/gateways/%s/sessions/%d/handles/1/info", gateway.Name, session.ID), nil)
	s.apiAuthP(req, []string{common.RoleAdmin})
	resp = s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)
}

func (s *ApiTestSuite) TestAdmin_GatewaysHandleInfo() {
	gateway := s.CreateGatewayP(common.GatewayTypeRooms, s.GatewayManager.Config.AdminURL, s.GatewayManager.Config.AdminSecret)
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	session, err := s.NewGatewaySession()
	s.Require().NoError(err, "NewGatewaySession")
	defer session.Destroy()

	handle, err := session.Attach("janus.plugin.videoroom")
	s.Require().NoError(err, "session.Attach")
	defer handle.Detach()

	req, _ := http.NewRequest("GET", fmt.Sprintf("/admin/gateways/%s/sessions/%d/handles/%d/info", gateway.Name, session.ID, handle.ID), nil)
	s.apiAuthP(req, []string{common.RoleAdmin})
	body := s.request200json(req)
	s.EqualValues(session.ID, body["session_id"], "session_id")
	s.EqualValues(handle.ID, body["handle_id"], "handle_id")
	s.NotNil(body["info"], "info")
}

func (s *ApiTestSuite) TestAdmin_ListRoomsForbidden() {
	req, _ := http.NewRequest("GET", "/admin/rooms", nil)
	resp := s.request(req)
	s.Require().Equal(http.StatusUnauthorized, resp.Code)

	req, _ = http.NewRequest("GET", "/admin/rooms", nil)
	s.apiAuth(req)
	resp = s.request(req)
	s.Require().Equal(http.StatusForbidden, resp.Code)
}

func (s *ApiTestSuite) TestAdmin_ListRoomsBadRequest() {
	args := [...]string{
		"page_no=0",
		"page_no=-1",
		"page_no=abc",
		"page_size=0",
		"page_size=-1",
		"page_size=abc",
	}
	for i, query := range args {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/admin/rooms?%s", query), nil)
		s.apiAuthP(req, []string{common.RoleRoot})
		resp := s.request(req)
		s.Require().Equal(http.StatusBadRequest, resp.Code, i)
	}
}

func (s *ApiTestSuite) TestAdmin_ListRooms() {
	req, _ := http.NewRequest("GET", "/admin/rooms", nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	body := s.request200json(req)
	s.Equal(0, int(body["total"].(float64)), "total")
	s.Equal(0, len(body["data"].([]interface{})), "len(data)")

	gateway := s.CreateGateway()
	rooms := make([]*models.Room, 10)
	for i := range rooms {
		rooms[i] = s.CreateRoom(gateway)
	}

	body = s.request200json(req)
	s.Equal(10, int(body["total"].(float64)), "total")
	s.Equal(10, len(body["data"].([]interface{})), "len(data)")

	for i, room := range rooms {
		req, _ = http.NewRequest("GET", fmt.Sprintf("/admin/rooms?page_no=%d&page_size=1&order_by=id", i+1), nil)
		s.apiAuthP(req, []string{common.RoleRoot})
		body = s.request200json(req)

		s.Equal(10, int(body["total"].(float64)), "total")
		data := body["data"].([]interface{})
		s.Equal(1, len(data), "len(data)")
		roomData := data[0].(map[string]interface{})
		s.Equal(roomData["name"], room.Name, "name")
		s.EqualValues(roomData["default_gateway_id"], room.DefaultGatewayID, "default_gateway_id")
		s.EqualValues(roomData["gateway_uid"], room.GatewayUID, "gateway_uid")
	}

	// disabled filter
	rooms[0].Disabled = true
	_, err := rooms[0].Update(s.DB, boil.Whitelist(models.RoomColumns.Disabled))
	s.Require().NoError(err)
	req, _ = http.NewRequest("GET", "/admin/rooms?disabled=true", nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	body = s.request200json(req)
	s.Equal(1, int(body["total"].(float64)), "total")
	s.Equal(1, len(body["data"].([]interface{})), "len(data)")
	s.Equal(body["data"].([]interface{})[0].(map[string]interface{})["name"], rooms[0].Name, "name")
	req, _ = http.NewRequest("GET", "/admin/rooms?disabled=false&order_by=id", nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	body = s.request200json(req)
	s.Equal(9, int(body["total"].(float64)), "total")
	s.Equal(9, len(body["data"].([]interface{})), "len(data)")
	data := body["data"].([]interface{})
	for i, item := range data {
		s.Equal(item.(map[string]interface{})["name"], rooms[i+1].Name, "name")
	}

	// removed filter
	rooms[9].RemovedAt = null.TimeFrom(time.Now().UTC())
	_, err = rooms[9].Update(s.DB, boil.Whitelist(models.RoomColumns.RemovedAt))
	s.Require().NoError(err)
	req, _ = http.NewRequest("GET", "/admin/rooms?removed=true", nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	body = s.request200json(req)
	s.Equal(1, int(body["total"].(float64)), "total")
	s.Equal(1, len(body["data"].([]interface{})), "len(data)")
	s.Equal(body["data"].([]interface{})[0].(map[string]interface{})["name"], rooms[9].Name, "name")
	req, _ = http.NewRequest("GET", "/admin/rooms?removed=false&order_by=id", nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	body = s.request200json(req)
	s.Equal(9, int(body["total"].(float64)), "total")
	s.Equal(9, len(body["data"].([]interface{})), "len(data)")
	data = body["data"].([]interface{})
	for i, item := range data {
		s.Equal(item.(map[string]interface{})["name"], rooms[i].Name, "name")
	}

	// gateways filter
	gateway2 := s.CreateGateway()
	rooms[0].DefaultGatewayID = gateway2.ID
	_, err = rooms[0].Update(s.DB, boil.Whitelist(models.RoomColumns.DefaultGatewayID))
	s.Require().NoError(err)
	req, _ = http.NewRequest("GET", fmt.Sprintf("/admin/rooms?gateway_id=%d", gateway2.ID), nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	body = s.request200json(req)
	s.Equal(1, int(body["total"].(float64)), "total")
	s.Equal(1, len(body["data"].([]interface{})), "len(data)")
	s.Equal(body["data"].([]interface{})[0].(map[string]interface{})["name"], rooms[0].Name, "name")
	req, _ = http.NewRequest("GET", fmt.Sprintf("/admin/rooms?gateway_id=%d&order_by=id", gateway.ID), nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	body = s.request200json(req)
	s.Equal(9, int(body["total"].(float64)), "total")
	s.Equal(9, len(body["data"].([]interface{})), "len(data)")
	data = body["data"].([]interface{})
	for i, item := range data {
		s.Equal(item.(map[string]interface{})["name"], rooms[i+1].Name, "name")
	}
	req, _ = http.NewRequest("GET", fmt.Sprintf("/admin/rooms?gateway_id=%d&gateway_id=%d&order_by=id", gateway.ID, gateway2.ID), nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	body = s.request200json(req)
	s.Equal(10, int(body["total"].(float64)), "total")
	s.Equal(10, len(body["data"].([]interface{})), "len(data)")
	data = body["data"].([]interface{})
	for i, item := range data {
		s.Equal(item.(map[string]interface{})["name"], rooms[i].Name, "name")
	}
}

func (s *ApiTestSuite) TestAdmin_GetRoomForbidden() {
	req, _ := http.NewRequest("GET", "/admin/rooms/1", nil)
	resp := s.request(req)
	s.Require().Equal(http.StatusUnauthorized, resp.Code)

	req, _ = http.NewRequest("GET", "/admin/rooms/1", nil)
	s.apiAuth(req)
	resp = s.request(req)
	s.Require().Equal(http.StatusForbidden, resp.Code)
}

func (s *ApiTestSuite) TestAdmin_GetRoomNotFound() {
	req, _ := http.NewRequest("GET", "/admin/rooms/abc", nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	resp := s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)

	req, _ = http.NewRequest("GET", "/admin/rooms/1", nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	resp = s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)
}

func (s *ApiTestSuite) TestAdmin_GetRoom() {
	gateway := s.CreateGateway()
	room := s.CreateRoom(gateway)
	req, _ := http.NewRequest("GET", fmt.Sprintf("/admin/rooms/%d", room.ID), nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	body := s.request200json(req)
	s.Equal(body["name"], room.Name, "name")
	s.EqualValues(body["default_gateway_id"], room.DefaultGatewayID, "default_gateway_id")
	s.EqualValues(body["gateway_uid"], room.GatewayUID, "gateway_uid")
}

func (s *ApiTestSuite) TestAdmin_CreateRoomForbidden() {
	req, _ := http.NewRequest("POST", "/admin/rooms", nil)
	resp := s.request(req)
	s.Require().Equal(http.StatusUnauthorized, resp.Code)

	req, _ = http.NewRequest("POST", "/admin/rooms", nil)
	s.apiAuth(req)
	resp = s.request(req)
	s.Require().Equal(http.StatusForbidden, resp.Code)
}

func (s *ApiTestSuite) TestAdmin_CreateRoomBadRequest() {
	req, _ := http.NewRequest("POST", "/admin/rooms", bytes.NewBuffer([]byte("{\"bad\":\"json")))
	s.apiAuthP(req, []string{common.RoleRoot})
	resp := s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)

	// non existing gateway
	body := models.Room{
		Name:       fmt.Sprintf("room_%s", stringutil.GenerateName(10)),
		GatewayUID: rand.Intn(math.MaxInt32),
	}
	b, _ := json.Marshal(body)
	req, _ = http.NewRequest("POST", "/admin/rooms", bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleRoot})
	resp = s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)

	// invalid gateway uid
	gateway := s.CreateGatewayP(common.GatewayTypeRooms, s.GatewayManager.Config.AdminURL, s.GatewayManager.Config.AdminSecret)
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	body.DefaultGatewayID = gateway.ID
	body.GatewayUID = -8
	b, _ = json.Marshal(body)
	req, _ = http.NewRequest("POST", "/admin/rooms", bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleRoot})
	resp = s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)

	// existing gateway_uid
	room := s.CreateRoom(gateway)
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))
	body.GatewayUID = room.GatewayUID
	b, _ = json.Marshal(body)
	req, _ = http.NewRequest("POST", "/admin/rooms", bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleRoot})
	resp = s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)

	// existing name
	body.Name = room.Name
	body.GatewayUID = room.GatewayUID + 1
	b, _ = json.Marshal(body)
	req, _ = http.NewRequest("POST", "/admin/rooms", bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleRoot})
	resp = s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)

	// invalid name
	body.GatewayUID = room.GatewayUID
	for _, name := range []string{"", "אסור עברית", "123456789012345678901234567890123456789012345678901234567890123456789012345"} {
		body.Name = name
		b, _ = json.Marshal(body)
		req, _ = http.NewRequest("POST", "/admin/rooms", bytes.NewBuffer(b))
		s.apiAuthP(req, []string{common.RoleRoot})
		resp = s.request(req)
		s.Require().Equal(http.StatusBadRequest, resp.Code)
	}
}

func (s *ApiTestSuite) TestAdmin_CreateRoom() {
	gateway := s.CreateGatewayP(common.GatewayTypeRooms, s.GatewayManager.Config.AdminURL, s.GatewayManager.Config.AdminSecret)
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	payload := models.Room{
		Name:             fmt.Sprintf("room_%s", stringutil.GenerateName(10)),
		GatewayUID:       rand.Intn(math.MaxInt32),
		DefaultGatewayID: gateway.ID,
	}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/admin/rooms", bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleRoot})
	body := s.request201json(req)
	s.NotZero(body["id"], "id")
	s.Equal(payload.Name, body["name"], "name")
	s.EqualValues(payload.GatewayUID, body["gateway_uid"], "gateway_uid")
	s.EqualValues(payload.DefaultGatewayID, body["default_gateway_id"], "default_gateway_id")
	s.False(body["disabled"].(bool), "disabled")

	// verify room is created on gateway
	gRoom := s.findRoomInGateway(gateway, int(body["gateway_uid"].(float64)))
	s.Require().NotNil(gRoom, "gateway room")
	s.Equal(gRoom.Description, payload.Name, "gateway room description")

	gChatroom := s.findChatroomInGateway(gateway, int(body["gateway_uid"].(float64)))
	s.Require().NotNil(gRoom, "gateway room")
	s.Equal(gChatroom.Description, payload.Name, "gateway chatroom description")
}

func (s *ApiTestSuite) TestAdmin_UpdateRoomForbidden() {
	req, _ := http.NewRequest("PUT", "/admin/rooms/1", nil)
	resp := s.request(req)
	s.Require().Equal(http.StatusUnauthorized, resp.Code)

	req, _ = http.NewRequest("PUT", "/admin/rooms/1", nil)
	s.apiAuth(req)
	resp = s.request(req)
	s.Require().Equal(http.StatusForbidden, resp.Code)
}

func (s *ApiTestSuite) TestAdmin_UpdateRoomNotFound() {
	req, _ := http.NewRequest("PUT", "/admin/rooms/abc", nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	resp := s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)

	req, _ = http.NewRequest("PUT", "/admin/rooms/1", nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	resp = s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)
}

func (s *ApiTestSuite) TestAdmin_UpdateRoomBadRequest() {
	gateway := s.CreateGatewayP(common.GatewayTypeRooms, s.GatewayManager.Config.AdminURL, s.GatewayManager.Config.AdminSecret)
	room := s.CreateRoom(gateway)
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	req, _ := http.NewRequest("PUT", fmt.Sprintf("/admin/rooms/%d", room.ID), bytes.NewBuffer([]byte("{\"bad\":\"json")))
	s.apiAuthP(req, []string{common.RoleRoot})
	resp := s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)

	// non existing gateway
	body := models.Room{
		Name:       fmt.Sprintf("room_%s", stringutil.GenerateName(10)),
		GatewayUID: rand.Intn(math.MaxInt32),
	}
	b, _ := json.Marshal(body)
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/admin/rooms/%d", room.ID), bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleRoot})
	resp = s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)

	// invalid gateway uid
	body.DefaultGatewayID = gateway.ID
	body.GatewayUID = -8
	b, _ = json.Marshal(body)
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/admin/rooms/%d", room.ID), bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleRoot})
	resp = s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)

	// existing gateway_uid
	room2 := s.CreateRoom(gateway)
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))
	body.GatewayUID = room2.GatewayUID
	b, _ = json.Marshal(body)
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/admin/rooms/%d", room.ID), bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleRoot})
	resp = s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)

	// existing name
	body.GatewayUID = room.GatewayUID
	body.Name = room2.Name
	b, _ = json.Marshal(body)
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/admin/rooms/%d", room.ID), bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleRoot})
	resp = s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)

	// invalid name
	for _, name := range []string{"", "אסור עברית", "123456789012345678901234567890123456789012345678901234567890123456789012345"} {
		body.Name = name
		b, _ = json.Marshal(body)
		req, _ = http.NewRequest("POST", "/admin/rooms", bytes.NewBuffer(b))
		s.apiAuthP(req, []string{common.RoleRoot})
		resp = s.request(req)
		s.Require().Equal(http.StatusBadRequest, resp.Code)
	}
}

func (s *ApiTestSuite) TestAdmin_UpdateRoom() {
	gateway := s.CreateGatewayP(common.GatewayTypeRooms, s.GatewayManager.Config.AdminURL, s.GatewayManager.Config.AdminSecret)
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	payload := models.Room{
		Name:             fmt.Sprintf("room_%s", stringutil.GenerateName(10)),
		GatewayUID:       rand.Intn(math.MaxInt16),
		DefaultGatewayID: gateway.ID,
	}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/admin/rooms", bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleRoot})
	body := s.request201json(req)

	gateway2 := s.CreateGatewayP(common.GatewayTypeRooms, s.GatewayManager.Config.AdminURL, s.GatewayManager.Config.AdminSecret)
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	payload.Name = fmt.Sprintf("%s_edit", payload.Name)
	payload.DefaultGatewayID = gateway2.ID
	payload.Disabled = true
	b, _ = json.Marshal(payload)
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/admin/rooms/%d", int64(body["id"].(float64))), bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleRoot})
	body = s.request200json(req)
	s.Equal(payload.Name, body["name"], "name")
	s.EqualValues(payload.DefaultGatewayID, body["default_gateway_id"], "default_gateway_id")
	s.True(body["disabled"].(bool), "disabled")
	s.Greater(body["updated_at"], body["created_at"], "updated_at > created_at")

	// verify room is updated on gateway
	gRoom := s.findRoomInGateway(gateway, int(body["gateway_uid"].(float64)))
	s.Require().NotNil(gRoom, "gateway room")
	s.Equal(gRoom.Description, payload.Name, "gateway room description")

	gChatroom := s.findChatroomInGateway(gateway, int(body["gateway_uid"].(float64)))
	s.Require().NotNil(gRoom, "gateway room")
	s.Equal(gChatroom.Description, payload.Name, "gateway chatroom description")
}

func (s *ApiTestSuite) TestAdmin_DeleteRoomForbidden() {
	req, _ := http.NewRequest("DELETE", "/admin/rooms/1", nil)
	resp := s.request(req)
	s.Require().Equal(http.StatusUnauthorized, resp.Code)

	req, _ = http.NewRequest("DELETE", "/admin/rooms/1", nil)
	s.apiAuth(req)
	resp = s.request(req)
	s.Require().Equal(http.StatusForbidden, resp.Code)
}

func (s *ApiTestSuite) TestAdmin_DeleteRoomNotFound() {
	req, _ := http.NewRequest("DELETE", "/admin/rooms/abc", nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	resp := s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)

	req, _ = http.NewRequest("DELETE", "/admin/rooms/1", nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	resp = s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)
}

func (s *ApiTestSuite) TestAdmin_DeleteRoom() {
	gateway := s.CreateGatewayP(common.GatewayTypeRooms, s.GatewayManager.Config.AdminURL, s.GatewayManager.Config.AdminSecret)
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	payload := models.Room{
		Name:             fmt.Sprintf("room_%s", stringutil.GenerateName(10)),
		GatewayUID:       rand.Intn(math.MaxInt16),
		DefaultGatewayID: gateway.ID,
	}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/admin/rooms", bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleRoot})
	body := s.request201json(req)

	id := int64(body["id"].(float64))
	req, _ = http.NewRequest("DELETE", fmt.Sprintf("/admin/rooms/%d", id), nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	s.request200json(req)

	// verify room removed_at is set in DB
	room, err := models.FindRoom(s.DB, id)
	s.Require().NoError(err, "models.FindRoom")
	s.True(room.RemovedAt.Valid, "remove_at")

	// verify room does not exists on gateway
	s.Nil(s.findRoomInGateway(gateway, int(body["gateway_uid"].(float64))))
	s.Nil(s.findChatroomInGateway(gateway, int(body["gateway_uid"].(float64))))
}

func (s *ApiTestSuite) TestAdmin_DeleteRoomsStatistics() {
	gateway := s.CreateGateway()
	rooms := make([]*models.Room, 5)
	for i := range rooms {
		rooms[i] = s.CreateRoom(gateway)
		roomStatistic := models.RoomStatistic{
			RoomID: rooms[i].ID,
			OnAir:  i + 1,
		}
		err := roomStatistic.Insert(s.DB, boil.Infer())
		s.Require().NoError(err, "create RoomStatistic")
	}
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	req, _ := http.NewRequest("DELETE", "/admin/rooms_statistics", nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	s.request200json(req)

	count, err := models.RoomStatistics().Count(s.DB)
	s.Require().NoError(err)
	s.EqualValues(0, count)
}

func (s *ApiTestSuite) TestAdmin_ListDynamicConfigsForbidden() {
	req, _ := http.NewRequest("GET", "/admin/dynamic_config", nil)
	resp := s.request(req)
	s.Require().Equal(http.StatusUnauthorized, resp.Code)

	req, _ = http.NewRequest("GET", "/admin/dynamic_config", nil)
	s.apiAuth(req)
	resp = s.request(req)
	s.Require().Equal(http.StatusForbidden, resp.Code)
}

func (s *ApiTestSuite) TestAdmin_ListDynamicConfigsBadRequest() {
	args := [...]string{
		"page_no=0",
		"page_no=-1",
		"page_no=abc",
		"page_size=0",
		"page_size=-1",
		"page_size=abc",
	}
	for i, query := range args {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/admin/dynamic_config?%s", query), nil)
		s.apiAuthP(req, []string{common.RoleRoot})
		resp := s.request(req)
		s.Require().Equal(http.StatusBadRequest, resp.Code, i)
	}
}

func (s *ApiTestSuite) TestAdmin_ListDynamicConfigs() {
	req, _ := http.NewRequest("GET", "/admin/dynamic_config", nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	body := s.request200json(req)
	s.Equal(0, int(body["total"].(float64)), "total")
	s.Equal(0, len(body["data"].([]interface{})), "len(data)")

	kvs := make([]*models.DynamicConfig, 10)
	for i := range kvs {
		kvs[i] = s.createDynamicConfig()
	}

	body = s.request200json(req)
	s.Equal(10, int(body["total"].(float64)), "total")
	s.Equal(10, len(body["data"].([]interface{})), "len(data)")

	for i, kv := range kvs {
		req, _ = http.NewRequest("GET", fmt.Sprintf("/admin/dynamic_config?page_no=%d&page_size=1&order_by=id", i+1), nil)
		s.apiAuthP(req, []string{common.RoleRoot})
		body = s.request200json(req)

		s.Equal(10, int(body["total"].(float64)), "total")
		data := body["data"].([]interface{})
		s.Equal(1, len(data), "len(data)")
		kvData := data[0].(map[string]interface{})
		s.Equal(kvData["key"], kv.Key, "key")
		s.Equal(kvData["value"], kv.Value, "value")
		updatedAt, err := time.Parse(time.RFC3339Nano, kvData["updated_at"].(string))
		s.NoError(err, "time.Parse error")
		s.InEpsilon(updatedAt.UnixNano(), kv.UpdatedAt.UnixNano(), 100, "updated_at")
	}
}

func (s *ApiTestSuite) TestAdmin_GetDynamicConfigForbidden() {
	req, _ := http.NewRequest("GET", "/admin/dynamic_config/1", nil)
	resp := s.request(req)
	s.Require().Equal(http.StatusUnauthorized, resp.Code)

	req, _ = http.NewRequest("GET", "/admin/dynamic_config/1", nil)
	s.apiAuth(req)
	resp = s.request(req)
	s.Require().Equal(http.StatusForbidden, resp.Code)
}

func (s *ApiTestSuite) TestAdmin_GetDynamicConfigNotFound() {
	req, _ := http.NewRequest("GET", "/admin/dynamic_config/abc", nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	resp := s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)

	req, _ = http.NewRequest("GET", "/admin/dynamic_config/1", nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	resp = s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)
}

func (s *ApiTestSuite) TestAdmin_GetDynamicConfig() {
	kv := s.createDynamicConfig()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/admin/dynamic_config/%d", kv.ID), nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	body := s.request200json(req)
	s.Equal(body["key"], kv.Key, "key")
	s.Equal(body["value"], kv.Value, "value")
	updatedAt, err := time.Parse(time.RFC3339Nano, body["updated_at"].(string))
	s.NoError(err, "time.Parse error")
	s.InEpsilon(updatedAt.UnixNano(), kv.UpdatedAt.UnixNano(), 100, "updated_at")
}

func (s *ApiTestSuite) TestAdmin_CreateDynamicConfigForbidden() {
	req, _ := http.NewRequest("POST", "/admin/dynamic_config", nil)
	resp := s.request(req)
	s.Require().Equal(http.StatusUnauthorized, resp.Code)

	req, _ = http.NewRequest("POST", "/admin/dynamic_config", nil)
	s.apiAuth(req)
	resp = s.request(req)
	s.Require().Equal(http.StatusForbidden, resp.Code)
}

func (s *ApiTestSuite) TestAdmin_CreateDynamicConfigBadRequest() {
	req, _ := http.NewRequest("POST", "/admin/dynamic_config", bytes.NewBuffer([]byte("{\"bad\":\"json")))
	s.apiAuthP(req, []string{common.RoleRoot})
	resp := s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)

	// no key
	body := models.DynamicConfig{
		Value: fmt.Sprintf("value_%s", stringutil.GenerateName(10)),
	}
	b, _ := json.Marshal(body)
	req, _ = http.NewRequest("POST", "/admin/dynamic_config", bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleRoot})
	resp = s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)

	// invalid key
	body.Key = stringutil.GenerateName(256)
	b, _ = json.Marshal(body)
	req, _ = http.NewRequest("POST", "/admin/dynamic_config", bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleRoot})
	resp = s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)

	// no value
	body.Key = fmt.Sprintf("key_%s", stringutil.GenerateName(10))
	body.Value = ""
	b, _ = json.Marshal(body)
	req, _ = http.NewRequest("POST", "/admin/dynamic_config", bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleRoot})
	resp = s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)

	// existing key
	kv := s.createDynamicConfig()
	body.Key = kv.Key
	body.Value = fmt.Sprintf("value_%s", stringutil.GenerateName(10))
	b, _ = json.Marshal(body)
	req, _ = http.NewRequest("POST", "/admin/dynamic_config", bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleRoot})
	resp = s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)
}

func (s *ApiTestSuite) TestAdmin_CreateDynamicConfig() {
	payload := models.DynamicConfig{
		Key:   fmt.Sprintf("key_%s", stringutil.GenerateName(10)),
		Value: fmt.Sprintf("value_%s", stringutil.GenerateName(10)),
	}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/admin/dynamic_config", bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleRoot})
	body := s.request201json(req)
	s.NotZero(body["id"], "id")
	s.Equal(payload.Key, body["key"], "key")
	s.Equal(payload.Value, body["value"], "value")
	s.NotZero(body["updated_at"], "updated_at")
	ts, err := time.Parse(time.RFC3339Nano, body["updated_at"].(string))
	s.NoError(err, "parse updated_at")
	s.True(time.Now().After(ts), "now is after updated_at")
}

func (s *ApiTestSuite) TestAdmin_UpdateDynamicConfigForbidden() {
	req, _ := http.NewRequest("PUT", "/admin/dynamic_config/1", nil)
	resp := s.request(req)
	s.Require().Equal(http.StatusUnauthorized, resp.Code)

	req, _ = http.NewRequest("PUT", "/admin/dynamic_config/1", nil)
	s.apiAuth(req)
	resp = s.request(req)
	s.Require().Equal(http.StatusForbidden, resp.Code)
}

func (s *ApiTestSuite) TestAdmin_UpdateDynamicConfigNotFound() {
	req, _ := http.NewRequest("PUT", "/admin/dynamic_config/abc", nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	resp := s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)

	req, _ = http.NewRequest("PUT", "/admin/dynamic_config/1", nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	resp = s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)
}

func (s *ApiTestSuite) TestAdmin_UpdateDynamicConfigBadRequest() {
	kv := s.createDynamicConfig()

	req, _ := http.NewRequest("PUT", fmt.Sprintf("/admin/dynamic_config/%d", kv.ID), bytes.NewBuffer([]byte("{\"bad\":\"json")))
	s.apiAuthP(req, []string{common.RoleRoot})
	resp := s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)

	// no key
	body := models.DynamicConfig{
		Value: fmt.Sprintf("value_%s", stringutil.GenerateName(10)),
	}
	b, _ := json.Marshal(body)
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/admin/dynamic_config/%d", kv.ID), bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleRoot})
	resp = s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)

	// invalid key
	body.Key = stringutil.GenerateName(256)
	b, _ = json.Marshal(body)
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/admin/dynamic_config/%d", kv.ID), bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleRoot})
	resp = s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)

	// no value
	body.Key = fmt.Sprintf("key_%s", stringutil.GenerateName(10))
	body.Value = ""
	b, _ = json.Marshal(body)
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/admin/dynamic_config/%d", kv.ID), bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleRoot})
	resp = s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)
}

func (s *ApiTestSuite) TestAdmin_UpdateDynamicConfig() {
	kv := s.createDynamicConfig()

	payload := models.DynamicConfig{
		Key:   fmt.Sprintf("key_%s", stringutil.GenerateName(10)),
		Value: fmt.Sprintf("value_%s", stringutil.GenerateName(10)),
	}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/admin/dynamic_config/%d", kv.ID), bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleRoot})
	body := s.request200json(req)
	s.Equal(payload.Key, body["key"], "key")
	s.Equal(payload.Value, body["value"], "value")
	ts, err := time.Parse(time.RFC3339Nano, body["updated_at"].(string))
	s.NoError(err, "parse updated_at")
	s.True(time.Now().After(ts), "now is after updated_at")
}

func (s *ApiTestSuite) TestAdmin_SetDynamicConfigNotFound() {
	req, _ := http.NewRequest("POST", "/admin/dynamic_config/abc", nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	resp := s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)

	req, _ = http.NewRequest("POST", "/admin/dynamic_config/1", nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	resp = s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)
}

func (s *ApiTestSuite) TestAdmin_SetDynamicConfigBadRequest() {
	kv := s.createDynamicConfig()
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	req, _ := http.NewRequest("POST", fmt.Sprintf("/admin/dynamic_config/%s", kv.Key), bytes.NewBuffer([]byte("{\"bad\":\"json")))
	s.apiAuthP(req, []string{common.RoleRoot})
	resp := s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)

	// no value
	body := models.DynamicConfig{}
	b, _ := json.Marshal(body)
	req, _ = http.NewRequest("POST", fmt.Sprintf("/admin/dynamic_config/%s", kv.Key), bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleRoot})
	resp = s.request(req)
	s.Require().Equal(http.StatusBadRequest, resp.Code)
}

func (s *ApiTestSuite) TestAdmin_SetDynamicConfig() {
	kv := s.createDynamicConfig()
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	payload := models.DynamicConfig{
		Value: fmt.Sprintf("value_%s", stringutil.GenerateName(10)),
	}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", fmt.Sprintf("/admin/dynamic_config/%s", kv.Key), bytes.NewBuffer(b))
	s.apiAuthP(req, []string{common.RoleRoot})
	resp := s.request(req)
	s.Require().Equal(http.StatusOK, resp.Code)

	req, _ = http.NewRequest("GET", fmt.Sprintf("/admin/dynamic_config/%d", kv.ID), nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	body := s.request200json(req)
	s.Equal(body["key"], kv.Key, "key")
	s.Equal(body["value"], payload.Value, "value")
	updatedAt, err := time.Parse(time.RFC3339Nano, body["updated_at"].(string))
	s.NoError(err, "time.Parse error")
	s.True(updatedAt.After(kv.UpdatedAt), "updated_at is after")
}

func (s *ApiTestSuite) TestAdmin_DeleteDynamicConfigForbidden() {
	req, _ := http.NewRequest("DELETE", "/admin/dynamic_config/1", nil)
	resp := s.request(req)
	s.Require().Equal(http.StatusUnauthorized, resp.Code)

	req, _ = http.NewRequest("DELETE", "/admin/dynamic_config/1", nil)
	s.apiAuth(req)
	resp = s.request(req)
	s.Require().Equal(http.StatusForbidden, resp.Code)
}

func (s *ApiTestSuite) TestAdmin_DeleteDynamicConfigNotFound() {
	req, _ := http.NewRequest("DELETE", "/admin/dynamic_config/abc", nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	resp := s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)

	req, _ = http.NewRequest("DELETE", "/admin/dynamic_config/1", nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	resp = s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)
}

func (s *ApiTestSuite) TestAdmin_DeleteDynamicConfig() {
	kv := s.createDynamicConfig()

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/admin/dynamic_config/%d", kv.ID), nil)
	s.apiAuthP(req, []string{common.RoleRoot})
	s.request200json(req)

	_, err := models.FindDynamicConfig(s.DB, kv.ID)
	s.Equal(err, sql.ErrNoRows, "Row deleted in DB")
}

func (s *ApiTestSuite) findRoomInGateway(gateway *models.Gateway, id int) *janus_plugins.VideoroomRoomFromListResponse {
	api, err := domain.GatewayAdminAPIRegistry.For(gateway)
	s.Require().NoError(err, "Admin API for gateway")

	request := janus_plugins.MakeVideoroomRequestFactory(common.Config.GatewayPluginAdminKey).ListRequest()
	resp, err := api.MessagePlugin(request)
	s.Require().NoError(err, "api.MessagePlugin")

	tResp, _ := resp.(*janus_plugins.VideoroomListResponse)
	for _, x := range tResp.Rooms {
		if x.Room == id {
			return x
		}
	}

	return nil
}

func (s *ApiTestSuite) findChatroomInGateway(gateway *models.Gateway, id int) *janus_plugins.TextroomRoomFromListResponse {
	api, err := domain.GatewayAdminAPIRegistry.For(gateway)
	s.Require().NoError(err, "Admin API for gateway")

	request := janus_plugins.MakeTextroomRequestFactory(common.Config.GatewayPluginAdminKey).ListRequest()
	resp, err := api.MessagePlugin(request)
	s.Require().NoError(err, "api.MessagePlugin")

	tResp, _ := resp.(*janus_plugins.TextroomListResponse)
	for _, x := range tResp.Rooms {
		if x.Room == id {
			return x
		}
	}

	return nil
}

func (s *ApiTestSuite) createDynamicConfig() *models.DynamicConfig {
	kv := &models.DynamicConfig{
		Key:       fmt.Sprintf("key_%s", stringutil.GenerateName(6)),
		Value:     fmt.Sprintf("value_%s", stringutil.GenerateName(6)),
		UpdatedAt: time.Now().UTC(),
	}
	s.Require().NoError(kv.Insert(s.DB, boil.Infer()))
	return kv
}
