package domain

import (
	"encoding/base64"
	"fmt"
	"math"
	"math/rand"

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

type ModelsSuite struct {
	suite.Suite
	testutil.TestDBManager
}

func (s *ModelsSuite) CreateGateway() *models.Gateway {
	return s.CreateGatewayP(common.GatewayTypeRooms, "admin_url", "janusoverlord")
}

func (s *ModelsSuite) CreateGatewayP(gType string, adminUrl, adminPwd string) *models.Gateway {
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

func (s *ModelsSuite) CreateUser() *models.User {
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

func (s *ModelsSuite) CreateRoom(gateway *models.Gateway) *models.Room {
	room := &models.Room{
		Name:             fmt.Sprintf("room_%s", stringutil.GenerateName(10)),
		DefaultGatewayID: gateway.ID,
		GatewayUID:       rand.Intn(math.MaxInt32),
	}
	s.Require().NoError(room.Insert(s.DB, boil.Infer()))
	return room
}

func (s *ModelsSuite) CreateSession(user *models.User, gateway *models.Gateway, room *models.Room) *models.Session {
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

func (s *ModelsSuite) CreateComposite(rooms []*models.Room) *models.Composite {
	composite := &models.Composite{
		Name: fmt.Sprintf("q%d", rand.Intn(math.MaxInt16)),
	}
	s.Require().NoError(composite.Insert(s.DB, boil.Infer()))

	cRooms := make([]*models.CompositesRoom, len(rooms))
	for i, room := range rooms {
		cRooms[i] = &models.CompositesRoom{
			RoomID:    room.ID,
			GatewayID: room.DefaultGatewayID,
			Position:  i + 1,
		}
	}
	s.Require().NoError(composite.AddCompositesRooms(s.DB, true, cRooms...))

	return composite
}
