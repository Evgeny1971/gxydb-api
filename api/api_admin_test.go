package api

import (
	"fmt"
	"net/http"

	"github.com/Bnei-Baruch/gxydb-api/common"
)

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

	gateway := s.createGatewayP(common.GatewayTypeRooms, s.GatewayManager.Config.AdminURL, s.GatewayManager.Config.AdminSecret)
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	req, _ = http.NewRequest("GET", fmt.Sprintf("/admin/gateways/%s/sessions/1/handles/1/info", gateway.Name), nil)
	s.apiAuthP(req, []string{common.RoleAdmin})
	resp = s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)

	session, err := s.NewGatewaySession()
	s.Require().NoError(err, "NewGatewaySession")
	defer session.Destroy()

	req, _ = http.NewRequest("GET", fmt.Sprintf("/admin/gateways/%s/sessions/%d/handles/1/info", gateway.Name, session.Id), nil)
	s.apiAuthP(req, []string{common.RoleAdmin})
	resp = s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)
}

func (s *ApiTestSuite) TestAdmin_GatewaysHandleInfo() {
	gateway := s.createGatewayP(common.GatewayTypeRooms, s.GatewayManager.Config.AdminURL, s.GatewayManager.Config.AdminSecret)
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	session, err := s.NewGatewaySession()
	s.Require().NoError(err, "NewGatewaySession")
	defer session.Destroy()

	handle, err := session.Attach("janus.plugin.videoroom")
	s.Require().NoError(err, "session.Attach")
	defer handle.Detach()

	req, _ := http.NewRequest("GET", fmt.Sprintf("/admin/gateways/%s/sessions/%d/handles/%d/info", gateway.Name, session.Id, handle.Id), nil)
	s.apiAuthP(req, []string{common.RoleAdmin})
	body := s.request200json(req)
	s.EqualValues(session.Id, body["session_id"], "session_id")
	s.EqualValues(handle.Id, body["handle_id"], "handle_id")
	s.NotNil(body["info"], "info")
}
