package instrumentation

import (
	"time"

	janus_admin "github.com/edoshor/janus-go/admin"
	pkgerr "github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/volatiletech/sqlboiler/queries"

	"github.com/Bnei-Baruch/gxydb-api/common"
	"github.com/Bnei-Baruch/gxydb-api/domain"
	"github.com/Bnei-Baruch/gxydb-api/models"
)

type PeriodicCollector struct {
	ticker *time.Ticker
	ticks  int64
	db     common.DBInterface
}

func NewPeriodicCollector(db common.DBInterface) *PeriodicCollector {
	return &PeriodicCollector{
		ticker: time.NewTicker(time.Second),
		db:     db,
	}
}

func (pc *PeriodicCollector) Start() {
	if pc.ticker != nil {
		pc.ticker.Stop()
	}

	log.Info().Msg("periodically collecting stats")
	pc.ticker = time.NewTicker(time.Second)
	go pc.run()
}

func (pc *PeriodicCollector) Close() {
	if pc.ticker != nil {
		pc.ticker.Stop()
	}
}

func (pc *PeriodicCollector) run() {
	for range pc.ticker.C {
		pc.ticks++
		pc.collectRoomParticipants()
		pc.collectGatewaySessions()
	}
}

func (pc *PeriodicCollector) collectRoomParticipants() {
	rows, err := queries.Raw(`select r.name, count(distinct s.user_id)
										from sessions s inner join rooms r on s.room_id = r.id
										where s.removed_at is null
										group by r.id;`).Query(pc.db)
	if err != nil {
		log.Error().Err(err).Msg("PeriodicCollector.collectRoomParticipants queries.Raw")
		return
	}

	Stats.RoomParticipantsGauge.Reset()

	for rows.Next() {
		var name string
		var count int64
		if err = rows.Scan(&name, &count); err != nil {
			log.Error().Err(err).Msg("PeriodicCollector.collectRoomParticipants rows.Scan")
		} else {
			Stats.RoomParticipantsGauge.WithLabelValues(name).Set(float64(count))
		}
	}

	if err := rows.Err(); err != nil {
		log.Error().Err(err).Msg("PeriodicCollector.collectRoomParticipants rows.Err")
		return
	}
}

type gatewayCallRes struct {
	gateway  *models.Gateway
	sessions int
	duration time.Duration
	err      error
}

func (pc *PeriodicCollector) collectGatewaySessions() {
	gateways, err := models.Gateways(
		models.GatewayWhere.Disabled.EQ(false),
		models.GatewayWhere.RemovedAt.IsNull()).
		All(pc.db)
	if err != nil {
		log.Error().Err(err).Msg("PeriodicCollector.collectGatewaySessions models.Gateways().All")
		return
	}

	Stats.GatewaySessionsGauge.Reset()

	c := make(chan *gatewayCallRes)

	for _, gateway := range gateways {
		go func(g *models.Gateway, c chan *gatewayCallRes) {
			res := &gatewayCallRes{gateway: g}
			start := time.Now()
			defer func() {
				res.duration = time.Now().Sub(start)
				c <- res
			}()

			api, err := domain.GatewayAdminAPIRegistry.For(g)
			if err != nil {
				res.err = pkgerr.Wrap(err, "domain.GatewayAdminAPIRegistry.For")
				return
			}

			apiRes, err := api.ListSessions()
			if err != nil {
				res.err = pkgerr.Wrap(err, "api.ListSessions")
				return
			}

			tApiRes, ok := apiRes.(*janus_admin.ListSessionsResponse)
			if !ok {
				res.err = pkgerr.Errorf("unexpected api.ListSessions response: %+v", apiRes)
				return
			}

			res.sessions = len(tApiRes.Sessions)
		}(gateway, c)
	}

	timeout := time.After(900 * time.Millisecond)
	for i := range gateways {
		select {
		case res := <-c:
			if res.err != nil {
				log.Error().
					Err(res.err).
					Dur("duration", res.duration).
					Str("gateway", res.gateway.Name).
					Msg("PeriodicCollector.collectGatewaySessions error")
			}
			Stats.GatewaySessionsGauge.WithLabelValues(res.gateway.Name, res.gateway.Type).Set(float64(res.sessions))
		case <-timeout:
			log.Error().Msgf("PeriodicCollector.collectGatewaySessions timeout (i, len)=(%d,%d)", i, len(gateways))
			break
		}
	}
}
