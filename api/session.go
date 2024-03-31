package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/edoshor/janus-go"
	"github.com/lib/pq"
	pkgerr "github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries"
	"github.com/volatiletech/sqlboiler/queries/qm"

	"github.com/Bnei-Baruch/gxydb-api/common"
	"github.com/Bnei-Baruch/gxydb-api/models"
	"github.com/Bnei-Baruch/gxydb-api/pkg/errs"
	"github.com/Bnei-Baruch/gxydb-api/pkg/sqlutil"
)

type SessionManager interface {
	HandleEvent(context.Context, interface{}) error
	HandleProtocol(context.Context, *janus.TextroomPostMsg) error
	UpsertSession(context.Context, *V1User) error
	Start()
	Close()
}

type V1SessionManager struct {
	db      common.DBInterface
	cache   *AppCache
	cleaner *PeriodicSessionCleaner
}

func NewV1SessionManager(db common.DBInterface, cache *AppCache) SessionManager {
	return &V1SessionManager{
		db:      db,
		cache:   cache,
		cleaner: NewPeriodicSessionCleaner(db),
	}
}

func (sm *V1SessionManager) HandleEvent(ctx context.Context, event interface{}) error {
	log.Ctx(ctx).Debug().Interface("event", event).Msg("handle gateway event")

	return sqlutil.InTx(ctx, sm.db, func(tx *sql.Tx) error {
		switch event.(type) {
		case *janus.PluginEvent:
			e := event.(*janus.PluginEvent)
			if e.Event.Plugin == "janus.plugin.videoroom" {
				eventType, ok := e.Event.Data["event"]
				if !ok {
					eventType, ok = e.Event.Data["videoroom"] // audio level change events have a little different structure.
				}
				if eventType == nil {
					log.Ctx(ctx).Warn().Interface("event", event).Msg("type less gateway event")
					return nil
				}
				eventTypeStr, ok := eventType.(string)
				if !ok {
					log.Ctx(ctx).Warn().Interface("event", event).Msg("event type is expected to be a string")
					return nil
				}

				switch eventTypeStr {
				case "leaving", "kicked", "unpublished":
					if err := sm.onVideoroomLeaving(ctx, tx, e, eventTypeStr); err != nil {
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

func (sm *V1SessionManager) Start() {
	sm.cleaner.Start()
}

func (sm *V1SessionManager) Close() {
	sm.cleaner.Close()
}

func (sm *V1SessionManager) onVideoroomLeaving(ctx context.Context, tx *sql.Tx, event *janus.PluginEvent, eventType string) error {
	display, ok := event.Event.Data["display"].(string)
	if !ok {
		return nil // some service users don't set their display. ignore this event.
	}

	var v1User V1User
	if err := json.Unmarshal([]byte(display), &v1User); err != nil {
		return pkgerr.Wrap(err, "json.Unmarshal")
	}

	logger := log.Ctx(ctx)
	logger.Info().Msgf("%s has left room %v [%s]", v1User.ID, event.Event.Data["room"], eventType)

	userID, err := sm.getInternalUserID(ctx, tx, &v1User)
	if err != nil {
		// we ignore ProtocolError here so that we could close sessions for disabled users
		var pErr *ProtocolError
		if !errors.As(err, &pErr) {
			return pkgerr.Wrap(err, "sm.getInternalUserID")
		}
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
		if u.Disabled {
			return u.ID, NewProtocolError(fmt.Sprintf("Disabled user: %s", user.ID))
		} else {
			sm.cache.users.Set(u)
			return u.ID, nil
		}
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
	b, err := json.Marshal(map[string]interface{}{
		"close_session": time.Now().UTC(),
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("SessionManager.closeSession json.Marshal")
	}

	res, err := queries.Raw("update sessions set properties = coalesce(properties, '{}'::jsonb) || $1, removed_at = $2 where user_id = $3 and removed_at is null",
		string(b), time.Now().UTC(), userID,
	).Exec(tx)
	if err != nil {
		return pkgerr.Wrap(err, "db update session")
	}

	rowsAffected, err := res.RowsAffected()
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
		boil.Blacklist(models.SessionColumns.CreatedAt, models.SessionColumns.Properties), boil.Infer())
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

	s := models.Session{
		UserID:                userID,
		RoomID:                null.Int64From(room.ID),
		GatewayID:             null.Int64From(gateway.ID),
		GatewaySession:        null.Int64From(user.Session),
		GatewayHandle:         null.Int64From(user.Handle),
		GatewayFeed:           null.StringFrom(user.RFID),
		GatewayHandleTextroom: null.Int64From(user.TextroomHandle),
		Display:               null.StringFrom(user.Display),
		Camera:                user.Camera,
		Question:              user.Question,
		SelfTest:              user.SelfTest,
		SoundTest:             user.SoundTest,
		UserAgent:             null.StringFrom(user.System),
		UpdatedAt:             null.TimeFrom(time.Now().UTC()),
	}

	if ip := net.ParseIP(user.IP); ip != nil {
		s.IPAddress = null.StringFrom(ip.String())
	}

	if extraB, err := json.Marshal(user.Extra); err == nil {
		s.Extra = null.JSONFrom(extraB)
	}

	return &s, nil
}

type PeriodicSessionCleaner struct {
	ticker *time.Ticker
	db     common.DBInterface
}

func NewPeriodicSessionCleaner(db common.DBInterface) *PeriodicSessionCleaner {
	return &PeriodicSessionCleaner{db: db}
}

func (psc *PeriodicSessionCleaner) Start() {
	if common.Config.CleanSessionsInterval <= 0 {
		return
	}

	log.Info().Msg("periodically cleaning sessions")
	psc.ticker = time.NewTicker(common.Config.CleanSessionsInterval)
	go psc.run()
}

func (psc *PeriodicSessionCleaner) Close() {
	if psc.ticker != nil {
		psc.ticker.Stop()
	}
}

func (psc *PeriodicSessionCleaner) run() {
	for range psc.ticker.C {
		psc.clean()
	}
}

func (psc *PeriodicSessionCleaner) clean() {
	// Note that we might be missing a DB index here.
	// However, at a < 10k rows table my tests showed it was irrelevant. So maybe add it in the future.
	mods := []qm.QueryMod{
		models.SessionWhere.RemovedAt.IsNull(),
		models.SessionWhere.UpdatedAt.LT(null.TimeFrom(time.Now().Add(-common.Config.DeadSessionPeriod))),
	}

	// fetch for visibility
	sessions, err := models.Sessions(mods...).All(psc.db)
	if err != nil {
		log.Error().Err(err).Msg("PeriodicSessionCleaner fetch sessions")
		return
	}
	log.Info().Msgf("PeriodicSessionCleaner found %d sessions to be cleaned", len(sessions))

	if len(sessions) == 0 {
		return
	}

	// clean
	b, err := json.Marshal(map[string]interface{}{
		"clean_session": time.Now().UTC(),
	})
	if err != nil {
		log.Error().Err(err).Msg("PeriodicSessionCleaner json.Marshal")
	}
	err = sqlutil.InTx(context.TODO(), psc.db, func(tx *sql.Tx) error {
		ids := make([]int64, len(sessions))
		for i := range sessions {
			ids[i] = sessions[i].ID
		}
		res, err := queries.Raw("update sessions set properties = coalesce(properties, '{}'::jsonb) || $1, removed_at = $2 where id = ANY($3)",
			string(b), time.Now().UTC(), pq.Array(ids),
		).Exec(tx)
		if err != nil {
			return pkgerr.WithStack(err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return pkgerr.WithStack(err)
		}

		if int(affected) != len(sessions) {
			log.Warn().
				Int("sessions", len(sessions)).
				Int64("affected", affected).
				Msg("PeriodicSessionCleaner clean sessions rows affected mismatch")
			return nil
		}

		return nil
	})

	if err != nil {
		log.Error().Err(err).Msg("PeriodicSessionCleaner clean sessions")
		return
	}

	// report
	dead := make([]*models.Session, 0)
	revived := make([]*models.Session, 0)
	for _, session := range sessions {
		var props map[string]interface{}
		if session.Properties.Valid {
			if err := json.Unmarshal(session.Properties.JSON, &props); err != nil {
				log.Warn().Err(err).Bytes("json", session.Properties.JSON).Msg("json.Unmarshal session.Properties")
				continue
			}
			if _, ok := props["close_session"]; ok {
				revived = append(revived, session)
				continue
			}
		}
		dead = append(dead, session)
	}
	log.Info().
		Int("total", len(sessions)).
		Int("dead", len(dead)).
		Int("revived", len(revived)).
		Msg("PeriodicSessionCleaner summary")
	for _, s := range dead {
		log.Info().
			Int64("session", s.ID).
			Int64("user", s.UserID).
			Int64("gateway", s.GatewayID.Int64).
			Msg("PeriodicSessionCleaner dead")
	}
	for _, s := range revived {
		log.Info().
			Int64("session", s.ID).
			Int64("user", s.UserID).
			Int64("gateway", s.GatewayID.Int64).
			Str("properties", string(s.Properties.JSON)).
			Msg("PeriodicSessionCleaner revived")
	}
}
