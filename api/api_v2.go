package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	pkgerr "github.com/pkg/errors"

	"github.com/Bnei-Baruch/gxydb-api/common"
	"github.com/Bnei-Baruch/gxydb-api/models"
	"github.com/Bnei-Baruch/gxydb-api/pkg/httputil"
)

func (a *App) V2GetConfig(w http.ResponseWriter, r *http.Request) {
	cfg := V2Config{
		Gateways:      make(map[string]map[string]*V2Gateway),
		IceServers:    common.Config.IceServers,
		DynamicConfig: make(map[string]string),
	}

	gateways := a.cache.gateways.Values()
	for _, gateway := range gateways {
		if gateway.Disabled || gateway.RemovedAt.Valid {
			continue
		}

		token, _ := a.cache.gatewayTokens.ByID(gateway.ID)
		respGateway := &V2Gateway{
			Name:  gateway.Name,
			URL:   gateway.URL,
			Type:  gateway.Type,
			Token: token,
		}

		if cfg.Gateways[gateway.Type] == nil {
			cfg.Gateways[gateway.Type] = make(map[string]*V2Gateway)
		}
		cfg.Gateways[gateway.Type][gateway.Name] = respGateway
	}

	kvs := a.cache.dynamicConfig.Values()
	for _, kv := range kvs {
		cfg.DynamicConfig[kv.Key] = kv.Value
	}

	httputil.RespondWithJSON(w, http.StatusOK, cfg)
}

func (a *App) ListRooms(w http.ResponseWriter, r *http.Request) {
	rooms, err := models.Rooms(
		models.RoomWhere.Disabled.EQ(false),
		models.RoomWhere.RemovedAt.IsNull(),
	).All(a.DB)

	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, rooms)
}

func (a *App) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	err := a.DB.(*sql.DB).PingContext(ctx)
	if err != nil {
		httputil.RespondWithError(w, http.StatusFailedDependency, fmt.Sprintf("DB ping: %s", err.Error()))
		return
	}

	if ctx.Err() == context.DeadlineExceeded {
		httputil.RespondWithError(w, http.StatusServiceUnavailable, "timeout")
		return
	}

	httputil.RespondSuccess(w)
}
