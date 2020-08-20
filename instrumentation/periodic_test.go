package instrumentation

import (
	"encoding/base64"
	"fmt"
	"math"
	"math/rand"
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/suite"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"golang.org/x/crypto/bcrypt"

	"github.com/Bnei-Baruch/gxydb-api/common"
	"github.com/Bnei-Baruch/gxydb-api/models"
	"github.com/Bnei-Baruch/gxydb-api/pkg/crypt"
	"github.com/Bnei-Baruch/gxydb-api/pkg/stringutil"
	"github.com/Bnei-Baruch/gxydb-api/pkg/testutil"
)

type PeriodicCollectorTestSuite struct {
	suite.Suite
	testutil.TestDBManager
}

func (s *PeriodicCollectorTestSuite) SetupSuite() {
	s.Require().NoError(s.InitTestDB())
	Stats.Init()
}

func (s *PeriodicCollectorTestSuite) TearDownSuite() {
	s.Require().NoError(s.DestroyTestDB())
}

func (s *PeriodicCollectorTestSuite) SetupTest() {
	Stats.Reset()
	s.DBCleaner.Acquire(s.AllTables()...)
}

func (s *PeriodicCollectorTestSuite) TearDownTest() {
	s.DBCleaner.Clean(s.AllTables()...)
}

func (s *PeriodicCollectorTestSuite) TestCollectRoomParticipants() {
	gateway := s.createGateway()
	rooms := make([]*models.Room, 10)
	for i := range rooms {
		rooms[i] = s.createRoom(gateway)
		for j := 0; j < i+1; j++ {
			user := s.createUser()
			s.createSession(user, gateway, rooms[i])
		}
	}

	pc := NewPeriodicCollector(s.DB)
	pc.collectRoomParticipants()

	for i := range rooms {
		g, err := Stats.RoomParticipantsGauge.GetMetricWithLabelValues(rooms[i].Name)
		s.NoError(err, "GetMetricWithLabelValues")
		m := new(dto.Metric)
		err = g.Write(m)
		s.NoError(err, "g.Write")
		s.EqualValues(i+1, *m.Gauge.Value, "gauge.value %d", i)
	}
}

func (s *PeriodicCollectorTestSuite) createGateway() *models.Gateway {
	return s.createGatewayP(common.GatewayTypeRooms, "admin_url", "janusoverlord")
}

func (s *PeriodicCollectorTestSuite) createGatewayP(gType string, adminUrl, adminPwd string) *models.Gateway {
	name := fmt.Sprintf("gateway_%s", stringutil.GenerateName(4))
	pwdHash, err := bcrypt.GenerateFromPassword([]byte(name), bcrypt.MinCost)
	s.Require().NoError(err)
	encAdminPwd, err := crypt.Encrypt([]byte(adminPwd), common.Config.Secret)
	s.Require().NoError(err)

	gateway := &models.Gateway{
		Name:           name,
		Description:    null.StringFrom("description"),
		URL:            "url",
		AdminURL:       adminUrl,
		AdminPassword:  base64.StdEncoding.EncodeToString(encAdminPwd),
		EventsPassword: string(pwdHash),
		Type:           gType,
	}

	s.Require().NoError(gateway.Insert(s.DB, boil.Infer()))

	return gateway
}

func (s *PeriodicCollectorTestSuite) createUser() *models.User {
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

func (s *PeriodicCollectorTestSuite) createRoom(gateway *models.Gateway) *models.Room {
	room := &models.Room{
		Name:             fmt.Sprintf("room_%s", stringutil.GenerateName(10)),
		DefaultGatewayID: gateway.ID,
		GatewayUID:       rand.Intn(math.MaxInt32),
	}
	s.Require().NoError(room.Insert(s.DB, boil.Infer()))
	return room
}

func (s *PeriodicCollectorTestSuite) createSession(user *models.User, gateway *models.Gateway, room *models.Room) *models.Session {
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

func TestPeriodicCollectorTestSuite(t *testing.T) {
	suite.Run(t, new(PeriodicCollectorTestSuite))
}
