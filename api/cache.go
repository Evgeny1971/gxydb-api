package api

import (
	"sync"

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
	sm sync.Map
}

func (c *GatewayCache) Reload(db boil.Executor) error {
	c.sm = sync.Map{}
	gateways, err := models.Gateways().All(db)
	if err != nil {
		return pkgerr.WithStack(err)
	}

	for i := range gateways {
		c.Set(gateways[i])
	}

	return nil
}

func (c *GatewayCache) Get(key interface{}) *models.Gateway {
	if x, ok := c.sm.Load(key); ok {
		return x.(*models.Gateway)
	}
	return nil
}

func (c *GatewayCache) Set(gateway *models.Gateway) {
	c.sm.Store(gateway.ID, gateway)
	c.sm.Store(gateway.Name, gateway)
}

func (c *GatewayCache) Values() []*models.Gateway {
	m := make(map[*models.Gateway]struct{}, 0)
	c.sm.Range(func(key, value interface{}) bool {
		m[value.(*models.Gateway)] = struct{}{}
		return true
	})

	values := make([]*models.Gateway, len(m))
	i := 0
	for k, _ := range m {
		values[i] = k
		i++
	}

	return values
}

type RoomCache struct {
	sm sync.Map
}

func (c *RoomCache) Reload(db boil.Executor) error {
	c.sm = sync.Map{}
	rooms, err := models.Rooms(
		models.RoomWhere.Disabled.EQ(false),
		models.RoomWhere.RemovedAt.IsNull(),
	).All(db)
	if err != nil {
		return pkgerr.WithStack(err)
	}

	for i := range rooms {
		c.Set(rooms[i])
	}

	return nil
}

func (c *RoomCache) Get(key interface{}) *models.Room {
	if x, ok := c.sm.Load(key); ok {
		return x.(*models.Room)
	}
	return nil
}

func (c *RoomCache) Set(room *models.Room) {
	c.sm.Store(room.ID, room)
	c.sm.Store(room.GatewayUID, room)
}

func (c *RoomCache) Values() []*models.Room {
	m := make(map[*models.Room]struct{}, 0)
	c.sm.Range(func(key, value interface{}) bool {
		m[value.(*models.Room)] = struct{}{}
		return true
	})

	values := make([]*models.Room, len(m))
	i := 0
	for k, _ := range m {
		values[i] = k
		i++
	}

	return values
}
