package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/volatiletech/sqlboiler/boil"

	"github.com/Bnei-Baruch/gxydb-api/models"
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
	gateway := models.Gateway{
		Name:          "gxy",
		URL:           "url",
		AdminURL:      "admin_url",
		AdminPassword: "admin_password",
	}
	s.Require().NoError(gateway.Insert(s.tx, boil.Infer()))
	room := models.Room{
		Name:             "Test Room",
		DefaultGatewayID: gateway.ID,
		GatewayUID:       1234,
	}
	s.Require().NoError(room.Insert(s.tx, boil.Infer()))

	req, _ := http.NewRequest("GET", fmt.Sprintf("/room/%d", room.ID), nil)
	resp := s.request(req)
	s.Require().Equal(http.StatusOK, resp.Code)

	var body map[string]interface{}
	s.Require().NoError(json.Unmarshal(resp.Body.Bytes(), &body))
	s.EqualValues(room.GatewayUID, int64(body["room"].(float64)), "room")
	s.Equal(gateway.Name, body["janus"], "Janus")
	s.Equal(room.Name, body["description"], "description")
	s.False(body["questions"].(bool), "questions")
	s.Zero(len(body["users"].([]interface{})), "users")
}

func (s *ApiTestSuite) TestGetRoomNotFound() {
	req, _ := http.NewRequest("GET", "/room/1", nil)
	resp := s.request(req)
	s.Require().Equal(http.StatusNotFound, resp.Code)
}

func (s *ApiTestSuite) request(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	s.app.Handler.ServeHTTP(rr, req)
	return rr
}

func (s *ApiTestSuite) checkResponseCode(expected, actual int) {
	s.Require().Equal(expected, actual)

}

func TestApiTestSuite(t *testing.T) {
	suite.Run(t, new(ApiTestSuite))
}
