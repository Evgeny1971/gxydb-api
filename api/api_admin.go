package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/edoshor/janus-go"
	"github.com/gorilla/mux"
	pkgerr "github.com/pkg/errors"

	"github.com/Bnei-Baruch/gxydb-api/common"
	"github.com/Bnei-Baruch/gxydb-api/domain"
	"github.com/Bnei-Baruch/gxydb-api/middleware"
	"github.com/Bnei-Baruch/gxydb-api/pkg/httputil"
)

func (a *App) AdminGatewaysHandleInfo(w http.ResponseWriter, r *http.Request) {
	if !common.Config.SkipPermissions && !middleware.RequestHasRole(r, common.RoleAdmin, common.RoleRoot) {
		httputil.NewForbiddenError().Abort(w, r)
		return
	}

	vars := mux.Vars(r)
	gatewayID := vars["gateway_id"]
	gateway, ok := a.cache.gateways.ByName(gatewayID)
	if !ok {
		httputil.NewNotFoundError().Abort(w, r)
		return
	}

	sessionIDStr := vars["session_id"]
	sessionID, err := strconv.ParseUint(sessionIDStr, 10, 64)
	if err != nil {
		httputil.NewNotFoundError().Abort(w, r)
		return
	}

	handleIDStr := vars["handle_id"]
	handleID, err := strconv.ParseUint(handleIDStr, 10, 64)
	if err != nil {
		httputil.NewNotFoundError().Abort(w, r)
		return
	}

	api, err := domain.GatewayAdminAPIRegistry.For(gateway)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithMessage(err, "init admin api")).Abort(w, r)
		return
	}

	info, err := api.HandleInfo(sessionID, handleID)
	if err != nil {
		var tErr *janus.ErrorAMResponse
		if errors.As(err, &tErr) {
			if tErr.Err.Code == 458 || tErr.Err.Code == 459 { // no such session or no such handle
				httputil.NewNotFoundError().Abort(w, r)
				return
			}
		}
		httputil.NewInternalError(pkgerr.Wrap(err, "api.HandleInfo")).Abort(w, r)
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, info)
}
