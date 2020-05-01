package api

import (
	"net/http"

	pkgerr "github.com/pkg/errors"

	"github.com/Bnei-Baruch/gxydb-api/common"
	"github.com/Bnei-Baruch/gxydb-api/models"
	"github.com/Bnei-Baruch/gxydb-api/pkg/httputil"
)

func (a *App) V2GetConfig(w http.ResponseWriter, r *http.Request) {
	gateways := a.cache.gateways.Values()
	cfg := V2Config{
		Gateways:   make(map[string]map[string]*V2Gateway, len(gateways)),
		IceServers: common.Config.IceServers,
	}

	// TODO: implement gateway tokens
	for _, gateway := range gateways {
		if cfg.Gateways[gateway.Type] == nil {
			cfg.Gateways[gateway.Type] = make(map[string]*V2Gateway)
		}
		cfg.Gateways[gateway.Type][gateway.Name] = &V2Gateway{
			Name:  gateway.Name,
			URL:   gateway.URL,
			Type:  gateway.Type,
			Token: "secret",
		}
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
