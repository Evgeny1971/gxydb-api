package instrumentation

import (
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/suite"

	"github.com/Bnei-Baruch/gxydb-api/domain"
	"github.com/Bnei-Baruch/gxydb-api/models"
)

type PeriodicCollectorTestSuite struct {
	domain.ModelsSuite
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
	gateway := s.CreateGateway()
	rooms := make([]*models.Room, 10)
	for i := range rooms {
		rooms[i] = s.CreateRoom(gateway)
		for j := 0; j < i+1; j++ {
			user := s.CreateUser()
			s.CreateSession(user, gateway, rooms[i])
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

func TestPeriodicCollectorTestSuite(t *testing.T) {
	suite.Run(t, new(PeriodicCollectorTestSuite))
}
