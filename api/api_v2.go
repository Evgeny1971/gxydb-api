package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	pkgerr "github.com/pkg/errors"

	"github.com/Bnei-Baruch/gxydb-api/common"
	"github.com/Bnei-Baruch/gxydb-api/middleware"
	"github.com/Bnei-Baruch/gxydb-api/models"
	"github.com/Bnei-Baruch/gxydb-api/pkg/httputil"
)

func (a *App) V2GetConfig(w http.ResponseWriter, r *http.Request) {
	gateways := a.cache.gateways.Values()
	cfg := V2Config{
		Gateways:   make(map[string]map[string]*V2Gateway, len(gateways)),
		IceServers: common.Config.IceServers,
	}

	withAdmin := middleware.RequestHasRole(r, common.RoleRoot, common.RoleAdmin)

	// TODO: implement gateway tokens
	for _, gateway := range gateways {
		respGateway := &V2Gateway{
			Name:  gateway.Name,
			URL:   gateway.URL,
			Type:  gateway.Type,
			Token: "secret",
		}

		if withAdmin {
			respGateway.AdminURL = gateway.AdminURL
			respGateway.AdminPassword = gateway.AdminPassword // TODO: probably will not be stored plain text in DB
		}

		if cfg.Gateways[gateway.Type] == nil {
			cfg.Gateways[gateway.Type] = make(map[string]*V2Gateway)
		}
		cfg.Gateways[gateway.Type][gateway.Name] = respGateway
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
