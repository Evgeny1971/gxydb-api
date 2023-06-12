package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	janus_admin "github.com/edoshor/janus-go/admin"
	janus_plugins "github.com/edoshor/janus-go/plugins"
	"github.com/gorilla/mux"
	pkgerr "github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"

	"github.com/Bnei-Baruch/gxydb-api/common"
	"github.com/Bnei-Baruch/gxydb-api/domain"
	"github.com/Bnei-Baruch/gxydb-api/middleware"
	"github.com/Bnei-Baruch/gxydb-api/models"
	"github.com/Bnei-Baruch/gxydb-api/pkg/httputil"
	"github.com/Bnei-Baruch/gxydb-api/pkg/mathutil"
	"github.com/Bnei-Baruch/gxydb-api/pkg/sqlutil"
)

func (a *App) AdminListGateways(w http.ResponseWriter, r *http.Request) {
	if !common.Config.SkipPermissions && !middleware.RequestHasRole(r, common.RoleRoot) {
		httputil.NewForbiddenError().Abort(w, r)
		return
	}

	query := r.URL.Query()
	listParams, err := ParseListParams(query)
	if err != nil {
		httputil.NewBadRequestError(err, "malformed list parameters").Abort(w, r)
		return
	}

	mods := make([]qm.QueryMod, 0)

	// count query
	var total int64
	countMods := append([]qm.QueryMod{qm.Select("count(DISTINCT id)")}, mods...)
	err = models.Gateways(countMods...).QueryRow(a.DB).Scan(&total)
	if err != nil {
		httputil.NewInternalError(err).Abort(w, r)
		return
	} else if total == 0 {
		httputil.RespondWithJSON(w, http.StatusOK, GatewaysResponse{Gateways: make([]*GatewayDTO, 0)})
		return
	}

	// order, limit, offset
	_, offset := listParams.appendListMods(&mods)
	if int64(offset) >= total {
		httputil.RespondWithJSON(w, http.StatusOK, GatewaysResponse{Gateways: make([]*GatewayDTO, 0)})
		return
	}

	// data query
	gateways, err := models.Gateways(mods...).All(a.DB)
	if err != nil {
		httputil.NewInternalError(err).Abort(w, r)
		return
	}

	dtos := make([]*GatewayDTO, len(gateways))
	for i := range gateways {
		dtos[i] = NewGatewayDTO(gateways[i])
	}

	httputil.RespondWithJSON(w, http.StatusOK, GatewaysResponse{
		ListResponse: ListResponse{
			Total: total,
		},
		Gateways: dtos,
	})
}

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
		var tErr *janus_admin.ErrorAMResponse
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

func (a *App) AdminListRooms(w http.ResponseWriter, r *http.Request) {
	if !common.Config.SkipPermissions && !middleware.RequestHasRole(r, common.RoleRoot) {
		httputil.NewForbiddenError().Abort(w, r)
		return
	}

	query := r.URL.Query()
	listParams, err := ParseListParams(query)
	if err != nil {
		httputil.NewBadRequestError(err, "malformed list parameters").Abort(w, r)
		return
	}

	roomsRequest, err := ParseRoomsRequest(query)
	if err != nil {
		httputil.NewBadRequestError(err, "malformed rooms request parameters").Abort(w, r)
		return
	}

	mods := make([]qm.QueryMod, 0)

	// filters
	if roomsRequest.Disabled.Valid {
		mods = append(mods, models.RoomWhere.Disabled.EQ(roomsRequest.Disabled.Bool))
	}
	if roomsRequest.Removed.Valid {
		if roomsRequest.Removed.Bool {
			mods = append(mods, models.RoomWhere.RemovedAt.IsNotNull())
		} else {
			mods = append(mods, models.RoomWhere.RemovedAt.IsNull())
		}
	}
	if len(roomsRequest.Gateways) > 0 {
		mods = append(mods, models.RoomWhere.DefaultGatewayID.IN(roomsRequest.Gateways))
	}
	if len(roomsRequest.Term) > 0 {
		var clauses []string
		var args []interface{}

		// numeric value ?
		if numVal, err := strconv.ParseUint(roomsRequest.Term, 10, 64); err == nil {
			clauses = append(clauses,
				fmt.Sprintf("%s = ?", models.RoomColumns.ID),
				fmt.Sprintf("%s = ?", models.RoomColumns.GatewayUID))
			args = append(args, numVal, numVal)
		}

		clauses = append(clauses, "name ~* ?")
		args = append(args, roomsRequest.Term)

		mods = append(mods, qm.Where(strings.Join(clauses, " OR "), args...))
	}

	// count query
	var total int64
	countMods := append([]qm.QueryMod{qm.Select("count(DISTINCT id)")}, mods...)
	err = models.Rooms(countMods...).QueryRow(a.DB).Scan(&total)
	if err != nil {
		httputil.NewInternalError(err).Abort(w, r)
		return
	} else if total == 0 {
		httputil.RespondWithJSON(w, http.StatusOK, RoomsResponse{Rooms: make([]*models.Room, 0)})
		return
	}

	// order, limit, offset
	_, offset := listParams.appendListMods(&mods)
	if int64(offset) >= total {
		httputil.RespondWithJSON(w, http.StatusOK, RoomsResponse{Rooms: make([]*models.Room, 0)})
		return
	}

	// data query
	rooms, err := models.Rooms(mods...).All(a.DB)
	if err != nil {
		httputil.NewInternalError(err).Abort(w, r)
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, RoomsResponse{
		ListResponse: ListResponse{
			Total: total,
		},
		Rooms: rooms,
	})
}

func (a *App) AdminCreateRoom(w http.ResponseWriter, r *http.Request) {
	if !common.Config.SkipPermissions && !middleware.RequestHasRole(r, common.RoleRoot) {
		httputil.NewForbiddenError().Abort(w, r)
		return
	}

	var data models.Room
	if err := httputil.DecodeJSONBody(w, r, &data); err != nil {
		err.Abort(w, r)
		return
	}
	a.requestContext(r).Params = data

	if _, ok := a.cache.gateways.ByID(data.DefaultGatewayID); !ok {
		httputil.NewBadRequestError(nil, "gateway doesn't exists").Abort(w, r)
		return
	}

	if exists, _ := models.Rooms(models.RoomWhere.Name.EQ(data.Name)).Exists(a.DB); exists {
		httputil.NewBadRequestError(nil, "room already exists [name]").Abort(w, r)
		return
	}

	if len(data.Name) == 0 || len(data.Name) > 64 {
		httputil.NewBadRequestError(nil, "name is missing or longer than 64 characters").Abort(w, r)
		return
	}

	if data.GatewayUID == 0 {
		var maxUID int
		if err := models.NewQuery(qm.Select("coalesce(max(gateway_uid), 0) + 1"), qm.From(models.TableNames.Rooms)).
			QueryRow(a.DB).Scan(&maxUID); err != nil {
			httputil.NewInternalError(pkgerr.Wrap(err, "fetch max gateway_uid")).Abort(w, r)
			return
		}
		data.GatewayUID = maxUID
	} else if data.GatewayUID < 0 {
		httputil.NewBadRequestError(nil, "gateway_uid must be a positive integer").Abort(w, r)
		return
	} else if _, ok := a.cache.rooms.ByGatewayUID(data.GatewayUID); ok {
		httputil.NewBadRequestError(nil, "room already exists [gateway_uid]").Abort(w, r)
		return
	}

	err := sqlutil.InTx(r.Context(), a.DB, func(tx *sql.Tx) error {
		// create room in DB
		if err := data.Insert(tx, boil.Whitelist("name", "default_gateway_id", "gateway_uid", "disabled", "region")); err != nil {
			return pkgerr.WithStack(err)
		}

		// create room in gateways
		room := &janus_plugins.VideoroomRoom{
			Room:               data.GatewayUID,
			Description:        data.Name,
			Secret:             common.Config.GatewayRoomsSecret,
			Publishers:         100,
			Bitrate:            64000,
			FirFreq:            10,
			AudioCodec:         "opus",
			VideoCodec:         "h264",
			H264Profile:        "42e01f",
			AudioLevelExt:      true,
			AudioLevelEvent:    true,
			AudioActivePackets: 25,
			AudioLevelAverage:  100,
			VideoOrientExt:     true,
			PlayoutDelayExt:    true,
			TransportWideCCExt: true,
		}
		request := janus_plugins.MakeVideoroomRequestFactory(common.Config.GatewayPluginAdminKey).
			CreateRequest(room, true, nil)

		for _, gateway := range a.cache.gateways.Values() {
			if gateway.Disabled || gateway.RemovedAt.Valid || gateway.Type != common.GatewayTypeRooms {
				continue
			}

			api, err := domain.GatewayAdminAPIRegistry.For(gateway)
			if err != nil {
				return pkgerr.WithMessage(err, "Admin API for gateway")
			}

			if _, err = api.MessagePlugin(request); err != nil {
				return pkgerr.Wrap(err, "api.MessagePlugin [videoroom]")
			}
		}

		return nil
	})

	if err != nil {
		var hErr *httputil.HttpError
		if errors.As(err, &hErr) {
			hErr.Abort(w, r)
		} else {
			httputil.NewInternalError(err).Abort(w, r)
		}
		return
	}

	if err := a.cache.rooms.Reload(a.DB); err != nil {
		log.Error().Err(err).Msg("Reload cache")
	}

	httputil.RespondWithJSON(w, http.StatusCreated, data)
}

func (a *App) AdminGetRoom(w http.ResponseWriter, r *http.Request) {
	if !common.Config.SkipPermissions && !middleware.RequestHasRole(r, common.RoleRoot) {
		httputil.NewForbiddenError().Abort(w, r)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		httputil.NewNotFoundError().Abort(w, r)
		return
	}

	room, err := models.FindRoom(a.DB, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.NewNotFoundError().Abort(w, r)
		} else {
			httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		}
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, room)
}

func (a *App) AdminUpdateRoom(w http.ResponseWriter, r *http.Request) {
	if !common.Config.SkipPermissions && !middleware.RequestHasRole(r, common.RoleRoot) {
		httputil.NewForbiddenError().Abort(w, r)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		httputil.NewNotFoundError().Abort(w, r)
		return
	}

	room, err := models.FindRoom(a.DB, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.NewNotFoundError().Abort(w, r)
		} else {
			httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		}
		return
	}

	var data models.Room
	if err := httputil.DecodeJSONBody(w, r, &data); err != nil {
		err.Abort(w, r)
		return
	}
	a.requestContext(r).Params = data

	if data.GatewayUID <= 0 {
		httputil.NewBadRequestError(nil, "gateway_uid must be a positive integer").Abort(w, r)
		return
	}

	if len(data.Name) == 0 || len(data.Name) > 64 {
		httputil.NewBadRequestError(nil, "name is missing or longer than 64 characters").Abort(w, r)
		return
	}

	if _, ok := a.cache.gateways.ByID(data.DefaultGatewayID); !ok {
		httputil.NewBadRequestError(nil, "gateway doesn't exists").Abort(w, r)
		return
	}

	if exists, _ := models.Rooms(models.RoomWhere.GatewayUID.EQ(data.GatewayUID), models.RoomWhere.ID.NEQ(room.ID)).Exists(a.DB); exists {
		httputil.NewBadRequestError(nil, "room already exists [gateway_uid]").Abort(w, r)
		return
	}

	if exists, _ := models.Rooms(models.RoomWhere.Name.EQ(data.Name), models.RoomWhere.ID.NEQ(room.ID)).Exists(a.DB); exists {
		httputil.NewBadRequestError(nil, "room already exists [name]").Abort(w, r)
		return
	}

	err = sqlutil.InTx(r.Context(), a.DB, func(tx *sql.Tx) error {
		shouldUpdateGateways := room.Name != data.Name &&
			!room.RemovedAt.Valid

		// update room in DB
		room.Name = data.Name
		room.DefaultGatewayID = data.DefaultGatewayID
		room.Disabled = data.Disabled
		room.Region = data.Region
		room.UpdatedAt = null.TimeFrom(time.Now().UTC())
		if _, err := room.Update(tx, boil.Whitelist("name", "default_gateway_id", "disabled", "region", "updated_at")); err != nil {
			return pkgerr.WithStack(err)
		}

		if !shouldUpdateGateways {
			return nil
		}

		// update room in gateways
		room := &janus_plugins.VideoroomRoomForEdit{
			Room:        data.GatewayUID,
			Description: data.Name,
			Publishers:  100,   // same as create
			Bitrate:     64000, // same as create
			FirFreq:     10,    // same as create
		}
		request := janus_plugins.MakeVideoroomRequestFactory(common.Config.GatewayPluginAdminKey).
			EditRequest(room, true, common.Config.GatewayRoomsSecret)

		for _, gateway := range a.cache.gateways.Values() {
			if gateway.Disabled || gateway.RemovedAt.Valid || gateway.Type != common.GatewayTypeRooms {
				continue
			}

			api, err := domain.GatewayAdminAPIRegistry.For(gateway)
			if err != nil {
				return pkgerr.WithMessage(err, "Admin API for gateway")
			}

			if _, err = api.MessagePlugin(request); err != nil {
				return pkgerr.Wrap(err, "api.MessagePlugin [videoroom]")
			}
		}

		return nil
	})

	if err != nil {
		var hErr *httputil.HttpError
		if errors.As(err, &hErr) {
			hErr.Abort(w, r)
		} else {
			httputil.NewInternalError(err).Abort(w, r)
		}
		return
	}

	if err := a.cache.rooms.Reload(a.DB); err != nil {
		log.Error().Err(err).Msg("Reload cache")
	}

	httputil.RespondWithJSON(w, http.StatusOK, room)
}

func (a *App) AdminDeleteRoom(w http.ResponseWriter, r *http.Request) {
	if !common.Config.SkipPermissions && !middleware.RequestHasRole(r, common.RoleRoot) {
		httputil.NewForbiddenError().Abort(w, r)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		httputil.NewNotFoundError().Abort(w, r)
		return
	}

	room, err := models.FindRoom(a.DB, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.NewNotFoundError().Abort(w, r)
		} else {
			httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		}
		return
	}

	err = sqlutil.InTx(r.Context(), a.DB, func(tx *sql.Tx) error {
		room.RemovedAt = null.TimeFrom(time.Now().UTC())
		if _, err := room.Update(tx, boil.Whitelist(models.RoomColumns.RemovedAt)); err != nil {
			return httputil.NewInternalError(pkgerr.WithStack(err))
		}

		request := janus_plugins.MakeVideoroomRequestFactory(common.Config.GatewayPluginAdminKey).
			DestroyRequest(room.GatewayUID, true, common.Config.GatewayRoomsSecret)

		for _, gateway := range a.cache.gateways.Values() {
			if gateway.Disabled || gateway.RemovedAt.Valid || gateway.Type != common.GatewayTypeRooms {
				continue
			}

			api, err := domain.GatewayAdminAPIRegistry.For(gateway)
			if err != nil {
				return pkgerr.WithMessage(err, "Admin API for gateway")
			}

			if _, err = api.MessagePlugin(request); err != nil {
				return pkgerr.Wrap(err, "api.MessagePlugin [videoroom]")
			}
		}

		return nil
	})

	if err != nil {
		var hErr *httputil.HttpError
		if errors.As(err, &hErr) {
			hErr.Abort(w, r)
		} else {
			httputil.NewInternalError(err).Abort(w, r)
		}
		return
	}

	if err := a.cache.rooms.Reload(a.DB); err != nil {
		log.Error().Err(err).Msg("Reload cache")
	}

	httputil.RespondSuccess(w)
}

func (a *App) AdminDeleteRoomsStatistics(w http.ResponseWriter, r *http.Request) {
	if !common.Config.SkipPermissions && !middleware.RequestHasRole(r, common.RoleShidur, common.RoleRoot) {
		httputil.NewForbiddenError().Abort(w, r)
		return
	}

	if err := a.roomsStatisticsManager.Reset(r.Context()); err != nil {
		httputil.NewInternalError(err).Abort(w, r)
		return
	}

	httputil.RespondSuccess(w)
}

func (a *App) AdminListDynamicConfigs(w http.ResponseWriter, r *http.Request) {
	if !common.Config.SkipPermissions && !middleware.RequestHasRole(r, common.RoleRoot) {
		httputil.NewForbiddenError().Abort(w, r)
		return
	}

	query := r.URL.Query()
	listParams, err := ParseListParams(query)
	if err != nil {
		httputil.NewBadRequestError(err, "malformed list parameters").Abort(w, r)
		return
	}

	mods := make([]qm.QueryMod, 0)

	// count query
	var total int64
	countMods := append([]qm.QueryMod{qm.Select("count(DISTINCT id)")}, mods...)
	err = models.DynamicConfigs(countMods...).QueryRow(a.DB).Scan(&total)
	if err != nil {
		httputil.NewInternalError(err).Abort(w, r)
		return
	} else if total == 0 {
		httputil.RespondWithJSON(w, http.StatusOK, DynamicConfigsResponse{Items: make([]*models.DynamicConfig, 0)})
		return
	}

	// order, limit, offset
	if listParams.OrderBy == "" {
		listParams.OrderBy = "key asc"
	}
	_, offset := listParams.appendListMods(&mods)
	if int64(offset) >= total {
		httputil.RespondWithJSON(w, http.StatusOK, DynamicConfigsResponse{Items: make([]*models.DynamicConfig, 0)})
		return
	}

	// data query
	kvs, err := models.DynamicConfigs(mods...).All(a.DB)
	if err != nil {
		httputil.NewInternalError(err).Abort(w, r)
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, DynamicConfigsResponse{
		ListResponse: ListResponse{
			Total: total,
		},
		Items: kvs,
	})
}

func (a *App) AdminCreateDynamicConfig(w http.ResponseWriter, r *http.Request) {
	if !common.Config.SkipPermissions && !middleware.RequestHasRole(r, common.RoleRoot) {
		httputil.NewForbiddenError().Abort(w, r)
		return
	}

	var data models.DynamicConfig
	if err := httputil.DecodeJSONBody(w, r, &data); err != nil {
		err.Abort(w, r)
		return
	}
	a.requestContext(r).Params = data

	if len(data.Key) == 0 || len(data.Key) > 255 {
		httputil.NewBadRequestError(nil, "key is missing or longer than 255 characters").Abort(w, r)
		return
	}

	if len(data.Value) == 0 {
		httputil.NewBadRequestError(nil, "value is missing").Abort(w, r)
		return
	}

	if exists, _ := models.DynamicConfigs(models.DynamicConfigWhere.Key.EQ(data.Key)).Exists(a.DB); exists {
		httputil.NewBadRequestError(nil, "key already exists").Abort(w, r)
		return
	}

	err := sqlutil.InTx(r.Context(), a.DB, func(tx *sql.Tx) error {
		data.UpdatedAt = time.Now().UTC()
		if err := data.Insert(tx, boil.Whitelist("key", "value", "updated_at")); err != nil {
			return pkgerr.WithStack(err)
		}
		return nil
	})

	if err != nil {
		var hErr *httputil.HttpError
		if errors.As(err, &hErr) {
			hErr.Abort(w, r)
		} else {
			httputil.NewInternalError(err).Abort(w, r)
		}
		return
	}

	if err := a.cache.dynamicConfig.Reload(a.DB); err != nil {
		log.Error().Err(err).Msg("Reload cache")
	}

	httputil.RespondWithJSON(w, http.StatusCreated, data)
}

func (a *App) AdminGetDynamicConfig(w http.ResponseWriter, r *http.Request) {
	if !common.Config.SkipPermissions && !middleware.RequestHasRole(r, common.RoleRoot) {
		httputil.NewForbiddenError().Abort(w, r)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		httputil.NewNotFoundError().Abort(w, r)
		return
	}

	kv, err := models.FindDynamicConfig(a.DB, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.NewNotFoundError().Abort(w, r)
		} else {
			httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		}
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, kv)
}

func (a *App) AdminUpdateDynamicConfig(w http.ResponseWriter, r *http.Request) {
	if !common.Config.SkipPermissions && !middleware.RequestHasRole(r, common.RoleRoot) {
		httputil.NewForbiddenError().Abort(w, r)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		httputil.NewNotFoundError().Abort(w, r)
		return
	}

	kv, err := models.FindDynamicConfig(a.DB, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.NewNotFoundError().Abort(w, r)
		} else {
			httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		}
		return
	}

	var data models.DynamicConfig
	if err := httputil.DecodeJSONBody(w, r, &data); err != nil {
		err.Abort(w, r)
		return
	}
	a.requestContext(r).Params = data

	if len(data.Key) == 0 || len(data.Key) > 255 {
		httputil.NewBadRequestError(nil, "key is missing or longer than 255 characters").Abort(w, r)
		return
	}

	if len(data.Value) == 0 {
		httputil.NewBadRequestError(nil, "value is missing").Abort(w, r)
		return
	}

	if exists, _ := models.DynamicConfigs(models.DynamicConfigWhere.Key.EQ(data.Key)).Exists(a.DB); exists {
		httputil.NewBadRequestError(nil, "key already exists").Abort(w, r)
		return
	}

	err = sqlutil.InTx(r.Context(), a.DB, func(tx *sql.Tx) error {
		kv.Key = data.Key
		kv.Value = data.Value
		kv.UpdatedAt = time.Now().UTC()
		if _, err := kv.Update(tx, boil.Whitelist("key", "value", "updated_at")); err != nil {
			return pkgerr.WithStack(err)
		}

		return nil
	})

	if err != nil {
		var hErr *httputil.HttpError
		if errors.As(err, &hErr) {
			hErr.Abort(w, r)
		} else {
			httputil.NewInternalError(err).Abort(w, r)
		}
		return
	}

	if err := a.cache.dynamicConfig.Reload(a.DB); err != nil {
		log.Error().Err(err).Msg("Reload cache")
	}

	httputil.RespondWithJSON(w, http.StatusOK, kv)
}

func (a *App) AdminSetDynamicConfig(w http.ResponseWriter, r *http.Request) {
	if !common.Config.SkipPermissions && !middleware.RequestHasRole(r, common.RoleRoot) {
		httputil.NewForbiddenError().Abort(w, r)
		return
	}

	vars := mux.Vars(r)
	key := vars["key"]
	kv, ok := a.cache.dynamicConfig.ByKey(key)
	if !ok {
		httputil.NewNotFoundError().Abort(w, r)
		return
	}

	var data models.DynamicConfig
	if err := httputil.DecodeJSONBody(w, r, &data); err != nil {
		err.Abort(w, r)
		return
	}
	a.requestContext(r).Params = data

	if len(data.Value) == 0 {
		httputil.NewBadRequestError(nil, "value is missing").Abort(w, r)
		return
	}

	err := sqlutil.InTx(r.Context(), a.DB, func(tx *sql.Tx) error {
		kv.Value = data.Value
		kv.UpdatedAt = time.Now().UTC()
		if _, err := kv.Update(tx, boil.Whitelist("value", "updated_at")); err != nil {
			return pkgerr.WithStack(err)
		}

		return nil
	})

	if err != nil {
		var hErr *httputil.HttpError
		if errors.As(err, &hErr) {
			hErr.Abort(w, r)
		} else {
			httputil.NewInternalError(err).Abort(w, r)
		}
		return
	}

	if err := a.cache.dynamicConfig.Reload(a.DB); err != nil {
		log.Error().Err(err).Msg("Reload cache")
	}

	httputil.RespondSuccess(w)
}

func (a *App) AdminDeleteDynamicConfig(w http.ResponseWriter, r *http.Request) {
	if !common.Config.SkipPermissions && !middleware.RequestHasRole(r, common.RoleRoot) {
		httputil.NewForbiddenError().Abort(w, r)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		httputil.NewNotFoundError().Abort(w, r)
		return
	}

	kv, err := models.FindDynamicConfig(a.DB, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.NewNotFoundError().Abort(w, r)
		} else {
			httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		}
		return
	}

	err = sqlutil.InTx(r.Context(), a.DB, func(tx *sql.Tx) error {
		if _, err := kv.Delete(tx); err != nil {
			return httputil.NewInternalError(pkgerr.WithStack(err))
		}

		return nil
	})

	if err != nil {
		var hErr *httputil.HttpError
		if errors.As(err, &hErr) {
			hErr.Abort(w, r)
		} else {
			httputil.NewInternalError(err).Abort(w, r)
		}
		return
	}

	if err := a.cache.dynamicConfig.Reload(a.DB); err != nil {
		log.Error().Err(err).Msg("Reload cache")
	}

	httputil.RespondSuccess(w)
}

type ListParams struct {
	PageNumber int    `json:"page_no"`
	PageSize   int    `json:"page_size"`
	OrderBy    string `json:"order_by"`
	GroupBy    string `json:"-"`
}

func (p *ListParams) appendListMods(mods *[]qm.QueryMod) (int, int) {
	// group to remove duplicates
	if p.GroupBy == "" {
		*mods = append(*mods, qm.GroupBy("id"))
	} else {
		*mods = append(*mods, qm.GroupBy(p.GroupBy))
	}

	if p.OrderBy == "" {
		*mods = append(*mods, qm.OrderBy("created_at desc"))
	} else {
		*mods = append(*mods, qm.OrderBy(p.OrderBy))
	}

	var limit, offset int
	if p.PageSize == 0 {
		limit = common.APIDefaultPageSize
	} else {
		limit = mathutil.Min(p.PageSize, common.APIMaxPageSize)
	}
	if p.PageNumber > 1 {
		offset = (p.PageNumber - 1) * limit
	}

	*mods = append(*mods, qm.Limit(limit))
	if offset != 0 {
		*mods = append(*mods, qm.Offset(offset))
	}

	return limit, offset
}

func ParseListParams(query url.Values) (*ListParams, error) {
	params := &ListParams{
		PageNumber: 1,
		PageSize:   50,
	}

	strVal := query.Get("page_no")
	if strVal != "" {
		if val, err := strconv.Atoi(strVal); err != nil {
			return nil, fmt.Errorf("page_no is not an integer: %w", err)
		} else if val < 1 {
			return nil, fmt.Errorf("page_no must be at least 1")
		} else {
			params.PageNumber = val
		}
	}

	strVal = query.Get("page_size")
	if strVal != "" {
		if val, err := strconv.Atoi(strVal); err != nil {
			return nil, fmt.Errorf("page_size is not an integer: %w", err)
		} else if val < 1 {
			return nil, fmt.Errorf("page_size must be at least 1")
		} else {
			params.PageSize = val
		}
	}

	strVal = query.Get("order_by")
	if strVal != "" {
		params.OrderBy = strVal
	}

	return params, nil
}

type ListResponse struct {
	Total int64 `json:"total"`
}

type RoomsRequest struct {
	Gateways []int64
	Disabled null.Bool
	Removed  null.Bool
	Term     string
}

type RoomsResponse struct {
	ListResponse
	Rooms []*models.Room `json:"data"`
}

type DynamicConfigsResponse struct {
	ListResponse
	Items []*models.DynamicConfig `json:"data"`
}

func ParseRoomsRequest(query url.Values) (*RoomsRequest, error) {
	req := &RoomsRequest{}

	strVals := query["gateway_id"]
	if len(strVals) > 0 {
		req.Gateways = make([]int64, len(strVals))
		for i, sVal := range strVals {
			if val, err := strconv.ParseInt(sVal, 10, 64); err != nil {
				return nil, fmt.Errorf("gateway is not an integer: %w", err)
			} else if val < 1 {
				return nil, fmt.Errorf("gateway ID must be at least 1")
			} else {
				req.Gateways[i] = val
			}
		}
	}

	strVal := query.Get("disabled")
	if strVal != "" {
		switch strVal {
		case "true":
			req.Disabled = null.BoolFrom(true)
		case "false":
			req.Disabled = null.BoolFrom(false)
		default:
			return nil, fmt.Errorf("disabled must be either `true` or `false`")
		}
	}

	strVal = query.Get("removed")
	if strVal != "" {
		switch strVal {
		case "true":
			req.Removed = null.BoolFrom(true)
		case "false":
			req.Removed = null.BoolFrom(false)
		default:
			return nil, fmt.Errorf("removed must be either `true` or `false`")
		}
	}

	req.Term = query.Get("term")

	return req, nil
}

type GatewayDTO struct {
	ID          int64       `json:"id"`
	Name        string      `json:"name"`
	Description null.String `json:"description,omitempty"`
	URL         string      `json:"url"`
	Disabled    bool        `json:"disabled"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   null.Time   `json:"updated_at,omitempty"`
	RemovedAt   null.Time   `json:"removed_at,omitempty"`
	Type        string      `json:"type"`
}

func NewGatewayDTO(g *models.Gateway) *GatewayDTO {
	return &GatewayDTO{
		ID:          g.ID,
		Name:        g.Name,
		Description: g.Description,
		URL:         g.URL,
		Disabled:    g.Disabled,
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
		RemovedAt:   g.RemovedAt,
		Type:        g.Type,
	}
}

type GatewaysResponse struct {
	ListResponse
	Gateways []*GatewayDTO `json:"data"`
}
