package api

import (
	pkgerr "github.com/pkg/errors"
	"github.com/volatiletech/sqlboiler/boil"

	"github.com/Bnei-Baruch/gxydb-api/models"
)

type AppCache struct {
	gateways *GatewayCache
	rooms    *RoomCache
}

func (c *AppCache) Init(db boil.Executor) error {
	c.gateways = new(GatewayCache)
	c.rooms = new(RoomCache)
	return c.Reload(db)
}

func (c *AppCache) Reload(db boil.Executor) error {
	if err := c.gateways.Reload(db); err != nil {
		return pkgerr.Wrap(err, "reload gateways")
	}
	if err := c.rooms.Reload(db); err != nil {
		return pkgerr.Wrap(err, "reload rooms")
	}

	return nil
}

type DBCache interface {
	Reload(db boil.Executor) error
}

type GatewayCache struct {
	ByID   map[int64]*models.Gateway
	ByName map[string]*models.Gateway
}

func (c *GatewayCache) Reload(db boil.Executor) error {
	gateways, err := models.Gateways().All(db)
	if err != nil {
		return pkgerr.Wrap(err, "db fetch")
	}

	c.ByID = make(map[int64]*models.Gateway, len(gateways))
	c.ByName = make(map[string]*models.Gateway, len(gateways))
	for i := range gateways {
		c.ByID[gateways[i].ID] = gateways[i]
		c.ByName[gateways[i].Name] = gateways[i]
	}

	return nil
}

type RoomCache struct {
	ByID         map[int64]*models.Room
	ByGatewayUID map[int]*models.Room
}

func (c *RoomCache) Reload(db boil.Executor) error {
	rooms, err := models.Rooms(
		models.RoomWhere.Disabled.EQ(false),
		models.RoomWhere.RemovedAt.IsNull(),
	).All(db)
	if err != nil {
		return pkgerr.Wrap(err, "db fetch")
	}

	c.ByID = make(map[int64]*models.Room, len(rooms))
	c.ByGatewayUID = make(map[int]*models.Room, len(rooms))
	for i := range rooms {
		c.ByID[rooms[i].ID] = rooms[i]
		c.ByGatewayUID[rooms[i].GatewayUID] = rooms[i]
	}

	return nil
}
