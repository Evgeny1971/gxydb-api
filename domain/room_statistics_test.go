package domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/volatiletech/sqlboiler/boil"

	"github.com/Bnei-Baruch/gxydb-api/models"
)

type RoomStatisticsTestSuite struct {
	ModelsSuite
}

func (s *RoomStatisticsTestSuite) SetupSuite() {
	s.Require().NoError(s.InitTestDB())
}

func (s *RoomStatisticsTestSuite) TearDownSuite() {
	s.Require().NoError(s.DestroyTestDB())
}

func (s *RoomStatisticsTestSuite) SetupTest() {
	s.DBCleaner.Acquire(models.TableNames.Gateways)
}

func (s *RoomStatisticsTestSuite) TearDownTest() {
	s.DBCleaner.Clean(models.TableNames.Gateways)
	//s.GatewayManager.DestroyGatewaySessions()
}

func (s *RoomStatisticsTestSuite) TestGetAll() {
	rms := NewRoomStatisticsManager(s.DB)

	rs, err := rms.GetAll()
	s.Require().NoError(err)
	s.Empty(rs, "empty rooms statistics")

	gateway := s.CreateGateway()
	roomStats := make(map[int64]*models.RoomStatistic)
	for i := 0; i < 5; i++ {
		room := s.CreateRoom(gateway)
		roomStats[room.ID] = &models.RoomStatistic{
			RoomID: room.ID,
			OnAir:  i,
		}
		err = roomStats[room.ID].Insert(s.DB, boil.Infer())
		s.Require().NoError(err)
	}

	rs, err = rms.GetAll()
	s.Require().NoError(err)
	s.Equal(len(rs), len(roomStats), "length")
	for _, roomStat := range rs {
		v, ok := roomStats[roomStat.RoomID]
		s.Require().True(ok, "missing roomID %d", roomStat.RoomID)
		s.Equal(v.OnAir, roomStat.OnAir, "OnAir")
	}
}

func (s *RoomStatisticsTestSuite) TestReset() {
	rms := NewRoomStatisticsManager(s.DB)

	gateway := s.CreateGateway()
	roomStats := make(map[int64]*models.RoomStatistic)
	for i := 0; i < 5; i++ {
		room := s.CreateRoom(gateway)
		roomStats[room.ID] = &models.RoomStatistic{
			RoomID: room.ID,
			OnAir:  i,
		}
		err := roomStats[room.ID].Insert(s.DB, boil.Infer())
		s.Require().NoError(err)
	}

	rs, err := rms.GetAll()
	s.Require().NoError(err)
	s.Equal(len(rs), len(roomStats), "length")

	err = rms.Reset(context.TODO())
	s.Require().NoError(err)

	rs, err = rms.GetAll()
	s.Require().NoError(err)
	s.Empty(rs, "empty rooms statistics")
}

func (s *RoomStatisticsTestSuite) TestOnAir() {
	rms := NewRoomStatisticsManager(s.DB)

	rs, err := rms.GetAll()
	s.Require().NoError(err)
	s.Empty(rs, "empty rooms statistics")

	gateway := s.CreateGateway()
	room := s.CreateRoom(gateway)

	for i := 0; i < 3; i++ {
		err = rms.OnAir(room.ID)
		s.Require().NoError(err)
		rs, err = rms.GetAll()
		s.Require().NoError(err)
		s.Equal(len(rs), 1, "length")
		s.Equal(rs[0].OnAir, i+1, "OnAir")
	}
}

func TestRoomStatisticsTestSuite(t *testing.T) {
	suite.Run(t, new(RoomStatisticsTestSuite))
}
