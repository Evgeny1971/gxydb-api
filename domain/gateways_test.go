package domain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/Bnei-Baruch/gxydb-api/common"
	"github.com/Bnei-Baruch/gxydb-api/models"
	"github.com/Bnei-Baruch/gxydb-api/pkg/testutil"
)

type GatewaysTestSuite struct {
	ModelsSuite
	testutil.GatewayManager
}

func (s *GatewaysTestSuite) SetupSuite() {
	s.Require().NoError(s.InitTestDB())
	s.GatewayManager.Init()
}

func (s *GatewaysTestSuite) TearDownSuite() {
	s.Require().NoError(s.DestroyTestDB())
	s.Require().NoError(s.GatewayManager.CloseGateway())
}

func (s *GatewaysTestSuite) SetupTest() {
	s.DBCleaner.Acquire(models.TableNames.Gateways)
}

func (s *GatewaysTestSuite) TearDownTest() {
	s.DBCleaner.Clean(models.TableNames.Gateways)
	s.GatewayManager.DestroyGatewaySessions()
}

func (s *GatewaysTestSuite) TestActiveToken() {
	gateway := s.createGateway()
	tm := NewGatewayTokensManager(s.DB, 1)
	token, err := tm.ActiveToken(gateway)
	s.Require().NoError(err, "tm.ActiveToken")
	s.Empty(token, "token")
}

func (s *GatewaysTestSuite) TestSyncAll() {
	gateway := s.createGateway()
	tm := NewGatewayTokensManager(s.DB, 1)
	tm.SyncAll()

	s.Require().NoError(gateway.Reload(s.DB), "gateway.Reload")
	token, err := tm.ActiveToken(gateway)
	s.Require().NoError(err, "tm.ActiveToken")
	s.NotEmpty(token, "token")

	s.Require().NoError(gateway.Reload(s.DB), "gateway.Reload")
	var props map[string]interface{}
	_ = json.Unmarshal(gateway.Properties.JSON, &props)
	tokensProp, _ := props["tokens"]
	s.Len(tokensProp.([]interface{}), 1, "number of tokens")
}

func (s *GatewaysTestSuite) TestRotateTokensWrongAdminPwd() {
	gateway := s.CreateGatewayP(common.GatewayTypeStreaming, s.GatewayManager.Config.AdminURL, "wrong_password")
	tm := NewGatewayTokensManager(s.DB, 1)
	changed, err := tm.syncGatewayTokens(gateway)
	s.False(changed, "changed")
	s.Error(err, "err")
}

func (s *GatewaysTestSuite) createGateway() *models.Gateway {
	return s.CreateGatewayP(common.GatewayTypeRooms, s.GatewayManager.Config.AdminURL, s.GatewayManager.Config.AdminSecret)
}

func TestGatewaysTestSuite(t *testing.T) {
	suite.Run(t, new(GatewaysTestSuite))
}
