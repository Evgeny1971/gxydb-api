package domain

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/volatiletech/sqlboiler/boil"
	"golang.org/x/crypto/bcrypt"

	"github.com/Bnei-Baruch/gxydb-api/common"
	"github.com/Bnei-Baruch/gxydb-api/models"
	"github.com/Bnei-Baruch/gxydb-api/pkg/crypt"
	"github.com/Bnei-Baruch/gxydb-api/pkg/stringutil"
	"github.com/Bnei-Baruch/gxydb-api/pkg/testutil"
)

type GatewaysTestSuite struct {
	suite.Suite
	testutil.TestDBManager
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
	gateway := s.createGatewayP(common.GatewayTypeStreaming, "wrong_password")
	tm := NewGatewayTokensManager(s.DB, 1)
	changed, err := tm.syncGatewayTokens(gateway)
	s.False(changed, "changed")
	s.Error(err, "err")
}

func (s *GatewaysTestSuite) createGateway() *models.Gateway {
	return s.createGatewayP(common.GatewayTypeRooms, s.GatewayManager.Config.AdminSecret)
}

func (s *GatewaysTestSuite) createGatewayP(gType string, adminPwd string) *models.Gateway {
	name := fmt.Sprintf("gateway_%s", stringutil.GenerateName(4))
	pwdHash, err := bcrypt.GenerateFromPassword([]byte(name), bcrypt.MinCost)
	s.Require().NoError(err)
	encAdminPwd, err := crypt.Encrypt([]byte(adminPwd), common.Config.Secret)
	s.Require().NoError(err)

	gateway := &models.Gateway{
		Name:           name,
		URL:            "url",
		AdminURL:       s.GatewayManager.Config.AdminURL,
		AdminPassword:  base64.StdEncoding.EncodeToString(encAdminPwd),
		EventsPassword: string(pwdHash),
		Type:           gType,
	}

	s.Require().NoError(gateway.Insert(s.DB, boil.Infer()))

	return gateway
}

func TestGatewaysTestSuite(t *testing.T) {
	suite.Run(t, new(GatewaysTestSuite))
}
