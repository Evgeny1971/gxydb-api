package api

import (
	"encoding/json"
	"time"

	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"

	"github.com/Bnei-Baruch/gxydb-api/common"
	"github.com/Bnei-Baruch/gxydb-api/models"
)

func (s *ApiTestSuite) TestSessions_Clean() {
	gateway := s.CreateGateway()
	room := s.CreateRoom(gateway)
	users := make([]*models.User, 5)
	sessions := make([]*models.Session, len(users))
	for i := range users {
		users[i] = s.CreateUser()
		sessions[i] = s.CreateSession(users[i], gateway, room)
	}
	s.Require().NoError(s.app.cache.ReloadAll(s.DB))

	// make 2 as dead
	affected, err := models.Sessions(models.SessionWhere.ID.IN([]int64{sessions[0].ID, sessions[1].ID})).
		UpdateAll(s.DB, models.M{
			"updated_at": time.Now().UTC().Add(-(common.Config.DeadSessionPeriod + time.Minute)),
		})
	s.Require().NoError(err)
	s.Require().EqualValues(2, affected)

	// make one of them as revived
	s.Require().NoError(sessions[1].Reload(s.DB))
	b, err := json.Marshal(map[string]interface{}{
		"close_session": time.Now().UTC(),
	})
	s.Require().NoError(err)
	sessions[1].Properties = null.JSONFrom(b)
	_, err = sessions[1].Update(s.DB, boil.Infer())
	s.Require().NoError(err)

	psc := NewPeriodicSessionCleaner(s.DB)
	psc.clean()

	cleanedSessions, err := models.Sessions(models.SessionWhere.RemovedAt.IsNotNull()).All(s.DB)
	s.Require().NoError(err, "fetch cleaned sessions")
	s.Equal(2, len(cleanedSessions), "len(cleanedSessions)")

	var props map[string]interface{}
	s.Require().NoError(sessions[0].Reload(s.DB))
	s.Require().NoError(sessions[0].Properties.Unmarshal(&props))
	s.NotNil(props["clean_session"], "session[0] clean_session property")
	s.Require().NoError(sessions[1].Reload(s.DB))
	s.Require().NoError(sessions[1].Properties.Unmarshal(&props))
	s.NotNil(props["clean_session"], "session[1] clean_session property")
	s.NotNil(props["close_session"], "session[1] should keep close_session property")
}
