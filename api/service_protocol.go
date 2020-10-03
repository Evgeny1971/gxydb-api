package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/edoshor/janus-go"
	pkgerr "github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/Bnei-Baruch/gxydb-api/common"
	"github.com/Bnei-Baruch/gxydb-api/domain"
)

type ServiceProtocolHandler interface {
	HandleMessage(context.Context, *janus.TextroomPostMsg) error
}

type V1ServiceProtocolHandler struct {
	db                     common.DBInterface
	cache                  *AppCache
	roomsStatisticsManager *domain.RoomStatisticsManager
}

func NewV1ServiceProtocolHandler(db common.DBInterface, cache *AppCache, rsm *domain.RoomStatisticsManager) ServiceProtocolHandler {
	return &V1ServiceProtocolHandler{
		db:                     db,
		cache:                  cache,
		roomsStatisticsManager: rsm,
	}
}

func (h *V1ServiceProtocolHandler) HandleMessage(ctx context.Context, msg *janus.TextroomPostMsg) error {
	logger := log.Ctx(ctx)
	logger.Info().Interface("msg", msg).Msg("service protocol message")

	var pMsg V1ServiceProtocolMessageText
	if err := json.Unmarshal([]byte(msg.Text), &pMsg); err != nil {
		return pkgerr.WithStack(WrappingProtocolError(err, fmt.Sprintf("json.Unmarshal: %s", err.Error())))
	}

	switch pMsg.Type {
	case "audio-out":
		if pMsg.Status {
			if pMsg.Room == nil {
				return NewProtocolError("no room specified")
			}

			room, ok := h.cache.rooms.ByGatewayUID(*pMsg.Room)
			if !ok {
				return NewProtocolError(fmt.Sprintf("unknown room %d", *pMsg.Room))
			}

			if err := h.roomsStatisticsManager.OnAir(room.ID); err != nil {
				return pkgerr.Wrap(err, "roomsStatisticsManager.OnAir")
			}
		}
		break
	default:
		break
	}

	return nil
}
