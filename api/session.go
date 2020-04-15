package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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
	case janus.PluginEvent:
		e := event.(janus.PluginEvent)
		if e.Event.Plugin == "janus.plugin.videoroom" {
			switch e.Event.Data["event"].(string) {
			case "leaving":
				if err := sm.onVideoroomLeaving(ctx, &e); err != nil {
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
		return pkgerr.Wrap(err, "json.Unmarshal")
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

	userID, err := sm.getInternalUserID(ctx, v1User.ID)
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

	userID, err := sm.getInternalUserID(ctx, pMsg.User.ID)
	if err != nil {
		return pkgerr.Wrap(err, "sm.getInternalUserID")
	}
	if userID > 0 {
		// close existing sessions if any
		if err := sm.closeSession(ctx, userID); err != nil {
			return pkgerr.Wrap(err, "sm.closeSession")
		}
	}

	session := sm.makeSession(userID, &pMsg.User)
	err = session.Upsert(sm.db, true,
		[]string{models.SessionColumns.UserID, models.SessionColumns.RemovedAt}, boil.Infer(), boil.Infer())
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

func (sm *V1SessionManager) getInternalUserID(ctx context.Context, key string) (int64, error) {
	var userID int64

	err := models.Users(
		qm.Select(models.UserColumns.ID),
		models.UserWhere.AccountsID.EQ(key)).
		QueryRow(sm.db).
		Scan(&userID)

	if err == nil {
		return userID, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return 0, pkgerr.Wrap(err, "db fetch userID")
	}

	log.Ctx(ctx).Info().Msgf("Creating new user: %s", key)
	user := models.User{
		AccountsID: key,
	}
	if err := user.Insert(sm.db, boil.Infer()); err != nil {
		return 0, pkgerr.Wrap(err, "db create user")
	}

	return user.ID, nil
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
	userID, err := sm.getInternalUserID(ctx, user.ID)
	if err != nil {
		return pkgerr.Wrap(err, "sm.getInternalUserID")
	}

	session := sm.makeSession(userID, user)
	err = session.Upsert(sm.db, true,
		[]string{models.SessionColumns.UserID, models.SessionColumns.RemovedAt}, boil.Infer(), boil.Infer())
	if err != nil {
		return pkgerr.Wrap(err, "db upsert")
	}

	return nil
}

func (sm *V1SessionManager) makeSession(userID int64, user *V1User) *models.Session {
	return &models.Session{
		UserID:         userID,
		RoomID:         null.Int64From(sm.cache.rooms.ByGatewayUID[user.Room].ID),
		GatewayID:      null.Int64From(sm.cache.gateways.ByName[user.Janus].ID),
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
		UpdatedAt:      null.Time{},
	}
}

type V1ProtocolMessageText struct {
	Type   string
	Status bool
	Room   int
	User   V1User
}
