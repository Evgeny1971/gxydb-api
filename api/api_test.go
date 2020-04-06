package api

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"

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
}

func (s *ApiTestSuite) TearDownTest() {
	err := s.tx.Rollback()
	s.Require().Nil(err)
}

func (s *ApiTestSuite) TestGetRoom() {
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
