package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/edoshor/janus-go"
	pkgerr "github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"

	"github.com/Bnei-Baruch/gxydb-api/models"
)

type SessionManager interface {
	HandleEvent(context.Context, interface{}) error
	HandleProtocol(context.Context, *janus.TextroomPostMsg) error
}

type V1SessionManager struct {
	db    DBInterface
	cache *AppCache
}

func NewV1SessionManager(db DBInterface, cache *AppCache) SessionManager {
	return &V1SessionManager{
		db:    db,
		cache: cache,
	}
}

func (sm *V1SessionManager) HandleEvent(ctx context.Context, event interface{}) error {
	log.Ctx(ctx).Debug().Interface("event", event).Msg("handle gateway event")

	switch event.(type) {
	case *janus.PluginEvent:
		e := event.(*janus.PluginEvent)
		if e.Event.Plugin == "janus.plugin.videoroom" {
			switch e.Event.Data["event"].(string) {
			case "leaving":
				if err := sm.onVideoroomLeaving(ctx, e); err != nil {
					return pkgerr.Wrap(err, "V1SessionManager.onVideoroomLeaving")
				}
			}
		}
	}

	return nil
}

func (sm *V1SessionManager) HandleProtocol(ctx context.Context, msg *janus.TextroomPostMsg) error {
	logger := log.Ctx(ctx)
	logger.Debug().Interface("msg", msg).Msg("handle protocol message")

	var pMsg V1ProtocolMessageText
	if err := json.Unmarshal([]byte(msg.Text), &pMsg); err != nil {
		return pkgerr.WithStack(WrappingProtocolError(err, fmt.Sprintf("json.Unmarshal: %s", err.Error())))
	}

	switch pMsg.Type {
	case "enter":
		if err := sm.onProtocolEnter(ctx, &pMsg); err != nil {
			return pkgerr.Wrap(err, "V1SessionManager.onProtocolEnter")
		}
	case "question":
		if err := sm.onProtocolQuestion(ctx, &pMsg); err != nil {
			return pkgerr.Wrap(err, "V1SessionManager.onProtocolQuestion")
		}
	case "camera":
		if err := sm.onProtocolCamera(ctx, &pMsg); err != nil {
			return pkgerr.Wrap(err, "V1SessionManager.onProtocolCamera")
		}
	case "sound-test":
		if err := sm.onProtocolSoundTest(ctx, &pMsg); err != nil {
			return pkgerr.Wrap(err, "V1SessionManager.onProtocolSoundTest")
		}
	default:
		logger.Info().
			Interface("pMsg", pMsg).
			Msg("noop")
	}

	return nil
}

func (sm *V1SessionManager) onVideoroomLeaving(ctx context.Context, event *janus.PluginEvent) error {
	display, ok := event.Event.Data["display"].(string)
	if !ok {
		return pkgerr.New("missing or malformed display")
	}

	var v1User V1User
	if err := json.Unmarshal([]byte(display), &v1User); err != nil {
		return pkgerr.Wrap(err, "json.Unmarshal")
	}

	logger := log.Ctx(ctx)
	logger.Info().Msgf("%s has left room %d", v1User.ID, v1User.Room)

	userID, err := sm.getInternalUserID(ctx, &v1User)
	if err != nil {
		return pkgerr.Wrap(err, "sm.getInternalUserID")
	}
	if userID == 0 {
		return nil
	}

	return sm.closeSession(ctx, userID)
}

func (sm *V1SessionManager) onProtocolEnter(ctx context.Context, pMsg *V1ProtocolMessageText) error {
	logger := log.Ctx(ctx)
	logger.Info().Msgf("%s has enter room %d", pMsg.User.ID, pMsg.User.Room)

	userID, err := sm.getInternalUserID(ctx, &pMsg.User)
	if err != nil {
		return pkgerr.Wrap(err, "sm.getInternalUserID")
	}
	if userID > 0 {
		// close existing sessions if any
		if err := sm.closeSession(ctx, userID); err != nil {
			return pkgerr.Wrap(err, "sm.closeSession")
		}
	}
	session, err := sm.makeSession(userID, &pMsg.User)
	if err != nil {
		return pkgerr.Wrap(err, "sm.makeSession")
	}

	err = session.Upsert(sm.db, true,
		[]string{models.SessionColumns.UserID, models.SessionColumns.GatewayID, models.SessionColumns.GatewaySession},
		boil.Infer(), boil.Infer())
	if err != nil {
		return pkgerr.Wrap(err, "db upsert")
	}

	return nil
}

func (sm *V1SessionManager) onProtocolQuestion(ctx context.Context, pMsg *V1ProtocolMessageText) error {
	logger := log.Ctx(ctx)
	logger.Info().Msgf("%s set question status to %t", pMsg.User.ID, pMsg.User.Question)
	return sm.upsertSession(ctx, &pMsg.User)
}

func (sm *V1SessionManager) onProtocolCamera(ctx context.Context, pMsg *V1ProtocolMessageText) error {
	logger := log.Ctx(ctx)
	logger.Info().Msgf("%s set camera status to %t", pMsg.User.ID, pMsg.User.Camera)
	return sm.upsertSession(ctx, &pMsg.User)
}

func (sm *V1SessionManager) onProtocolSoundTest(ctx context.Context, pMsg *V1ProtocolMessageText) error {
	logger := log.Ctx(ctx)
	logger.Info().Msgf("%s set sound-test status to %t", pMsg.User.ID, pMsg.User.SoundTest)
	return sm.upsertSession(ctx, &pMsg.User)
}

func (sm *V1SessionManager) getInternalUserID(ctx context.Context, user *V1User) (int64, error) {
	var userID int64

	err := models.Users(
		qm.Select(models.UserColumns.ID),
		models.UserWhere.AccountsID.EQ(user.ID)).
		QueryRow(sm.db).
		Scan(&userID)

	if err == nil {
		return userID, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return 0, pkgerr.Wrap(err, "db fetch userID")
	}

	log.Ctx(ctx).Info().Msgf("Creating new user: %s", user.ID)
	newUser := models.User{
		AccountsID: user.ID,
		Email:      null.StringFrom(user.Email),
		Username:   null.StringFrom(user.Username),
	}
	if err := newUser.Insert(sm.db, boil.Infer()); err != nil {
		return 0, pkgerr.Wrap(err, "db create user")
	}

	return newUser.ID, nil
}

func (sm *V1SessionManager) closeSession(ctx context.Context, userID int64) error {
	rowsAffected, err := models.Sessions(
		models.SessionWhere.UserID.EQ(userID),
		models.SessionWhere.RemovedAt.IsNull()).
		UpdateAll(sm.db, models.M{models.SessionColumns.RemovedAt: time.Now().UTC()})
	if err != nil {
		return pkgerr.Wrap(err, "db update session")
	}

	log.Ctx(ctx).Info().Msgf("%d sessions were closed", rowsAffected)

	return nil
}

func (sm *V1SessionManager) upsertSession(ctx context.Context, user *V1User) error {
	userID, err := sm.getInternalUserID(ctx, user)
	if err != nil {
		return pkgerr.Wrap(err, "sm.getInternalUserID")
	}

	session, err := sm.makeSession(userID, user)
	if err != nil {
		return pkgerr.Wrap(err, "sm.makeSession")
	}

	err = session.Upsert(sm.db, true,
		[]string{models.SessionColumns.UserID, models.SessionColumns.GatewayID, models.SessionColumns.GatewaySession},
		boil.Infer(), boil.Blacklist(models.SessionColumns.UpdatedAt))
	if err != nil {
		return pkgerr.Wrap(err, "db upsert")
	}

	return nil
}

type ProtocolError struct {
	msg   string
	cause error
}

func (pe *ProtocolError) Error() string {
	return pe.msg
}

func (pe *ProtocolError) Cause() error {
	return pe.cause
}

func NewProtocolError(msg string) *ProtocolError {
	return &ProtocolError{msg: msg}
}

func WrappingProtocolError(err error, msg string) *ProtocolError {
	return &ProtocolError{cause: err, msg: msg}
}

func (sm *V1SessionManager) makeSession(userID int64, user *V1User) (*models.Session, error) {
	room, ok := sm.cache.rooms.ByGatewayUID[user.Room]
	if !ok {
		return nil, NewProtocolError(fmt.Sprintf("Unknown room: %d", user.Room))
	}

	gateway, ok := sm.cache.gateways.ByName[user.Janus]
	if !ok {
		return nil, NewProtocolError(fmt.Sprintf("Unknown gateway: %s", user.Janus))
	}

	return &models.Session{
		UserID:         userID,
		RoomID:         null.Int64From(room.ID),
		GatewayID:      null.Int64From(gateway.ID),
		GatewaySession: null.Int64From(user.Session),
		GatewayHandle:  null.Int64From(user.Handle),
		GatewayFeed:    null.Int64From(user.RFID),
		Display:        null.StringFrom(user.Display),
		Camera:         user.Camera,
		Question:       user.Question,
		SelfTest:       user.SelfTest,
		SoundTest:      user.SoundTest,
		UserAgent:      null.StringFrom(user.System),
		IPAddress:      null.StringFrom(user.IP),
		UpdatedAt:      null.TimeFrom(time.Now().UTC()),
	}, nil
}
