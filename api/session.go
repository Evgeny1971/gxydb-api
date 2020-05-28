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

	"github.com/Bnei-Baruch/gxydb-api/common"
	"github.com/Bnei-Baruch/gxydb-api/models"
	"github.com/Bnei-Baruch/gxydb-api/pkg/errs"
	"github.com/Bnei-Baruch/gxydb-api/pkg/sqlutil"
)

type SessionManager interface {
	HandleEvent(context.Context, interface{}) error
	HandleProtocol(context.Context, *janus.TextroomPostMsg) error
	UpsertSession(context.Context, *V1User) error
}

type V1SessionManager struct {
	db    common.DBInterface
	cache *AppCache
}

func NewV1SessionManager(db common.DBInterface, cache *AppCache) SessionManager {
	return &V1SessionManager{
		db:    db,
		cache: cache,
	}
}

func (sm *V1SessionManager) HandleEvent(ctx context.Context, event interface{}) error {
	log.Ctx(ctx).Debug().Interface("event", event).Msg("handle gateway event")

	return sqlutil.InTx(ctx, sm.db, func(tx *sql.Tx) error {
		switch event.(type) {
		case *janus.PluginEvent:
			e := event.(*janus.PluginEvent)
			if e.Event.Plugin == "janus.plugin.videoroom" {
				switch e.Event.Data["event"].(string) {
				case "leaving":
					if err := sm.onVideoroomLeaving(ctx, tx, e); err != nil {
						return pkgerr.Wrap(err, "V1SessionManager.onVideoroomLeaving")
					}
				}
			}
		}

		return nil
	})
}

func (sm *V1SessionManager) HandleProtocol(ctx context.Context, msg *janus.TextroomPostMsg) error {
	logger := log.Ctx(ctx)
	logger.Debug().Interface("msg", msg).Msg("handle protocol message")

	var pMsg V1ProtocolMessageText
	if err := json.Unmarshal([]byte(msg.Text), &pMsg); err != nil {
		return pkgerr.WithStack(WrappingProtocolError(err, fmt.Sprintf("json.Unmarshal: %s", err.Error())))
	}

	return sqlutil.InTx(ctx, sm.db, func(tx *sql.Tx) error {
		switch pMsg.Type {
		case "enter":
			if err := sm.onProtocolEnter(ctx, tx, &pMsg); err != nil {
				return pkgerr.Wrap(err, "V1SessionManager.onProtocolEnter")
			}
		case "question":
			if err := sm.onProtocolQuestion(ctx, tx, &pMsg); err != nil {
				return pkgerr.Wrap(err, "V1SessionManager.onProtocolQuestion")
			}
		case "camera":
			if err := sm.onProtocolCamera(ctx, tx, &pMsg); err != nil {
				return pkgerr.Wrap(err, "V1SessionManager.onProtocolCamera")
			}
		case "sound-test":
			if err := sm.onProtocolSoundTest(ctx, tx, &pMsg); err != nil {
				return pkgerr.Wrap(err, "V1SessionManager.onProtocolSoundTest")
			}
		default:
			logger.Info().
				Interface("pMsg", pMsg).
				Msg("noop")
		}

		return nil
	})
}

func (sm *V1SessionManager) UpsertSession(ctx context.Context, user *V1User) error {
	return sqlutil.InTx(ctx, sm.db, func(tx *sql.Tx) error {
		return sm.upsertSession(ctx, tx, user)
	})
}

func (sm *V1SessionManager) onVideoroomLeaving(ctx context.Context, tx *sql.Tx, event *janus.PluginEvent) error {
	display, ok := event.Event.Data["display"].(string)
	if !ok {
		return pkgerr.New("missing or malformed display")
	}

	var v1User V1User
	if err := json.Unmarshal([]byte(display), &v1User); err != nil {
		return pkgerr.Wrap(err, "json.Unmarshal")
	}

	logger := log.Ctx(ctx)
	logger.Info().Msgf("%s has left room %v", v1User.ID, event.Event.Data["room"])

	userID, err := sm.getInternalUserID(ctx, tx, &v1User)
	if err != nil {
		return pkgerr.Wrap(err, "sm.getInternalUserID")
	}
	if userID == 0 {
		return nil
	}

	return sm.closeSession(ctx, tx, userID)
}

func (sm *V1SessionManager) onProtocolEnter(ctx context.Context, tx *sql.Tx, pMsg *V1ProtocolMessageText) error {
	logger := log.Ctx(ctx)
	logger.Info().Msgf("%s has enter room %d", pMsg.User.ID, pMsg.User.Room)

	userID, err := sm.getInternalUserID(ctx, tx, &pMsg.User)
	if err != nil {
		return pkgerr.Wrap(err, "sm.getInternalUserID")
	}
	if userID > 0 {
		// close existing sessions if any
		if err := sm.closeSession(ctx, tx, userID); err != nil {
			return pkgerr.Wrap(err, "sm.closeSession")
		}
	}

	session, err := sm.makeSession(userID, &pMsg.User)
	if err != nil {
		return pkgerr.Wrap(err, "sm.makeSession")
	}

	err = session.Upsert(tx, true,
		[]string{models.SessionColumns.UserID, models.SessionColumns.GatewayID, models.SessionColumns.GatewaySession},
		boil.Infer(), boil.Infer())
	if err != nil {
		return pkgerr.Wrap(err, "db upsert")
	}

	return nil
}

func (sm *V1SessionManager) onProtocolQuestion(ctx context.Context, tx *sql.Tx, pMsg *V1ProtocolMessageText) error {
	logger := log.Ctx(ctx)
	logger.Info().Msgf("%s set question status to %t", pMsg.User.ID, pMsg.User.Question)
	return sm.upsertSession(ctx, tx, &pMsg.User)
}

func (sm *V1SessionManager) onProtocolCamera(ctx context.Context, tx *sql.Tx, pMsg *V1ProtocolMessageText) error {
	logger := log.Ctx(ctx)
	logger.Info().Msgf("%s set camera status to %t", pMsg.User.ID, pMsg.User.Camera)
	return sm.upsertSession(ctx, tx, &pMsg.User)
}

func (sm *V1SessionManager) onProtocolSoundTest(ctx context.Context, tx *sql.Tx, pMsg *V1ProtocolMessageText) error {
	logger := log.Ctx(ctx)
	logger.Info().Msgf("%s set sound-test status to %t", pMsg.User.ID, pMsg.User.SoundTest)
	return sm.upsertSession(ctx, tx, &pMsg.User)
}

func (sm *V1SessionManager) getInternalUserID(ctx context.Context, tx *sql.Tx, user *V1User) (int64, error) {
	u, ok := sm.cache.users.ByAccountsID(user.ID)
	if ok {
		return u.ID, nil
	}

	u, err := models.Users(
		models.UserWhere.AccountsID.EQ(user.ID)).
		One(tx)
	if err == nil {
		sm.cache.users.Set(u)
		return u.ID, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return 0, pkgerr.Wrap(err, "db fetch userID")
	}

	log.Ctx(ctx).Info().Msgf("Creating new user: %s", user.ID)
	u = &models.User{
		AccountsID: user.ID,
		Email:      null.StringFrom(user.Email),
		Username:   null.StringFrom(user.Username),
	}
	if err := u.Insert(tx, boil.Infer()); err != nil {
		return 0, pkgerr.Wrap(err, "db create user")
	}

	sm.cache.users.Set(u)

	return u.ID, nil
}

func (sm *V1SessionManager) closeSession(ctx context.Context, tx *sql.Tx, userID int64) error {
	rowsAffected, err := models.Sessions(
		models.SessionWhere.UserID.EQ(userID),
		models.SessionWhere.RemovedAt.IsNull()).
		UpdateAll(tx, models.M{models.SessionColumns.RemovedAt: time.Now().UTC()})
	if err != nil {
		return pkgerr.Wrap(err, "db update session")
	}

	log.Ctx(ctx).Info().Msgf("%d sessions were closed", rowsAffected)

	return nil
}

func (sm *V1SessionManager) upsertSession(ctx context.Context, tx *sql.Tx, user *V1User) error {
	userID, err := sm.getInternalUserID(ctx, tx, user)
	if err != nil {
		return pkgerr.Wrap(err, "sm.getInternalUserID")
	}

	session, err := sm.makeSession(userID, user)
	if err != nil {
		return pkgerr.Wrap(err, "sm.makeSession")
	}

	err = session.Upsert(tx, true,
		[]string{models.SessionColumns.UserID, models.SessionColumns.GatewayID, models.SessionColumns.GatewaySession},
		boil.Infer(), boil.Blacklist(models.SessionColumns.UpdatedAt))
	if err != nil {
		return pkgerr.Wrap(err, "db upsert")
	}

	return nil
}

type ProtocolError struct {
	errs.WithMessage
}

func NewProtocolError(msg string) *ProtocolError {
	return &ProtocolError{errs.WithMessage{
		Msg: msg,
	}}
}

func WrappingProtocolError(err error, msg string) *ProtocolError {
	return &ProtocolError{errs.WithMessage{
		Msg: msg,
		Err: err,
	}}
}

func (sm *V1SessionManager) makeSession(userID int64, user *V1User) (*models.Session, error) {
	room, ok := sm.cache.rooms.ByGatewayUID(user.Room)
	if !ok {
		return nil, NewProtocolError(fmt.Sprintf("Unknown room: %d", user.Room))
	}

	gateway, ok := sm.cache.gateways.ByName(user.Janus)
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
