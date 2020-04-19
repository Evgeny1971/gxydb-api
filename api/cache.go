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
	byID   map[int64]*models.Gateway
	byName map[string]*models.Gateway
	lock   sync.RWMutex
}

func (c *GatewayCache) Reload(db boil.Executor) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	gateways, err := models.Gateways().All(db)
	if err != nil {
		return pkgerr.WithStack(err)
	}

	c.byID = make(map[int64]*models.Gateway, len(gateways))
	c.byName = make(map[string]*models.Gateway, len(gateways))
	for _, gateway := range gateways {
		c.byID[gateway.ID] = gateway
		c.byName[gateway.Name] = gateway
	}

	return nil
}

func (c *GatewayCache) ByID(id int64) (*models.Gateway, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	gateway, ok := c.byID[id]
	return gateway, ok
}

func (c *GatewayCache) ByName(name string) (*models.Gateway, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	gateway, ok := c.byName[name]
	return gateway, ok
}

func (c *GatewayCache) Set(gateway *models.Gateway) {
	c.lock.Lock()
	c.byID[gateway.ID] = gateway
	c.byName[gateway.Name] = gateway
	c.lock.Unlock()
}

func (c *GatewayCache) Values() []*models.Gateway {
	c.lock.RLock()
	defer c.lock.RUnlock()

	values := make([]*models.Gateway, len(c.byID))
	i := 0
	for _, v := range c.byID {
		values[i] = v
		i++
	}

	return values
}

type RoomCache struct {
	m    map[int]*models.Room
	lock sync.RWMutex
}

func (c *RoomCache) Reload(db boil.Executor) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	rooms, err := models.Rooms(
		models.RoomWhere.Disabled.EQ(false),
		models.RoomWhere.RemovedAt.IsNull(),
	).All(db)
	if err != nil {
		return pkgerr.WithStack(err)
	}

	c.m = make(map[int]*models.Room, len(rooms))
	for _, room := range rooms {
		c.m[room.GatewayUID] = room
	}

	return nil
}

func (c *RoomCache) ByGatewayUID(uid int) (*models.Room, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	r, ok := c.m[uid]
	return r, ok
}

func (c *RoomCache) Set(room *models.Room) {
	c.lock.Lock()
	c.m[room.GatewayUID] = room
	c.lock.Unlock()
}

func (c *RoomCache) Values() []*models.Room {
	c.lock.RLock()
	defer c.lock.RUnlock()

	values := make([]*models.Room, len(c.m))
	i := 0
	for _, v := range c.m {
		values[i] = v
		i++
	}

	return values
}
