package api

import (
	"encoding/json"
	"fmt"

	pkgerr "github.com/pkg/errors"

	"github.com/Bnei-Baruch/gxydb-api/domain"
)

type ServiceProtocolHandler interface {
	HandleMessage(string) error
}

type V1ServiceProtocolHandler struct {
	cache                  *AppCache
	roomsStatisticsManager *domain.RoomStatisticsManager
}

func NewV1ServiceProtocolHandler(cache *AppCache, rsm *domain.RoomStatisticsManager) ServiceProtocolHandler {
	return &V1ServiceProtocolHandler{
		cache:                  cache,
		roomsStatisticsManager: rsm,
	}
}

func (h *V1ServiceProtocolHandler) HandleMessage(payload string) error {
	var pMsg V1ServiceProtocolMessageText
	if err := json.Unmarshal([]byte(payload), &pMsg); err != nil {
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
