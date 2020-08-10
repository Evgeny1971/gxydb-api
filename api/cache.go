package api

import (
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru"
	pkgerr "github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/volatiletech/sqlboiler/queries/qm"

	"github.com/Bnei-Baruch/gxydb-api/common"
	"github.com/Bnei-Baruch/gxydb-api/domain"
	"github.com/Bnei-Baruch/gxydb-api/models"
)

type AppCache struct {
	db            common.DBInterface
	gateways      *GatewayCache
	gatewayTokens *GatewayTokenCache
	rooms         *RoomCache
	users         *UserCache
	dynamicConfig *DynamicConfigCache
	ticker        *time.Ticker
	ticks         int64
}

func (c *AppCache) Init(db common.DBInterface) error {
	c.db = db
	c.gateways = new(GatewayCache)
	c.gatewayTokens = new(GatewayTokenCache)
	c.rooms = new(RoomCache)
	c.users = new(UserCache)
	c.dynamicConfig = new(DynamicConfigCache)

	c.ticker = time.NewTicker(time.Second)
	go func() {
		for range c.ticker.C {
			c.ticks++
			if c.ticks%3600 == 0 {
				if err := c.gatewayTokens.Reload(c.db); err != nil {
					log.Error().Err(err).Msg("gatewayTokens.Reload")
				}
			}
			if c.ticks%60 == 0 {
				if err := c.dynamicConfig.Reload(c.db); err != nil {
					log.Error().Err(err).Msg("dynamicConfig.Reload")
				}
			}
		}
	}()

	return c.ReloadAll(db)
}

func (c *AppCache) Close() {
	c.ticker.Stop()
}

func (c *AppCache) ReloadAll(db common.DBInterface) error {
	if err := c.gateways.Reload(db); err != nil {
		return pkgerr.Wrap(err, "reload gateways")
	}

	if err := c.gatewayTokens.Reload(db); err != nil {
		return pkgerr.Wrap(err, "reload gateway_tokens")
	}

	if err := c.rooms.Reload(db); err != nil {
		return pkgerr.Wrap(err, "reload rooms")
	}

	if err := c.users.Reload(db); err != nil {
		return pkgerr.Wrap(err, "reload users")
	}

	if err := c.dynamicConfig.Reload(db); err != nil {
		return pkgerr.Wrap(err, "reload dynamicConfig")
	}

	return nil
}

type GatewayCache struct {
	byID   map[int64]*models.Gateway
	byName map[string]*models.Gateway
	lock   sync.RWMutex
}

func (c *GatewayCache) Reload(db common.DBInterface) error {
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

type GatewayTokenCache struct {
	byID map[int64]string
	lock sync.RWMutex
}

func (c *GatewayTokenCache) Reload(db common.DBInterface) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	tm := domain.NewGatewayTokensManager(db, 1)

	gateways, err := models.Gateways().All(db)
	if err != nil {
		return pkgerr.WithStack(err)
	}

	c.byID = make(map[int64]string, len(gateways))
	for _, gateway := range gateways {
		c.byID[gateway.ID], err = tm.ActiveToken(gateway)
		if err != nil {
			return pkgerr.WithMessagef(err, "tm.ActiveToken %s", gateway.Name)
		}
	}

	return nil
}

func (c *GatewayTokenCache) ByID(id int64) (string, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	token, ok := c.byID[id]
	return token, ok
}

type RoomCache struct {
	m    map[int]*models.Room
	lock sync.RWMutex
}

func (c *RoomCache) Reload(db common.DBInterface) error {
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

type UserCache struct {
	cache *lru.ARCCache
}

func (c *UserCache) Reload(db common.DBInterface) error {
	users, err := models.Users(
		models.UserWhere.Disabled.EQ(false),
		models.UserWhere.RemovedAt.IsNull(),
		qm.OrderBy(fmt.Sprintf("%s desc", models.UserColumns.CreatedAt)),
		qm.Limit(500),
	).All(db)
	if err != nil {
		return pkgerr.WithStack(err)
	}

	if c.cache != nil {
		c.cache.Purge()
	}
	c.cache, _ = lru.NewARC(5_000)
	for _, user := range users {
		c.cache.Add(user.AccountsID, user)
	}

	return nil
}

func (c *UserCache) ByAccountsID(id string) (*models.User, bool) {
	u, ok := c.cache.Get(id)
	if ok {
		return u.(*models.User), true
	}

	return nil, false
}

func (c *UserCache) Set(user *models.User) {
	c.cache.Add(user.AccountsID, user)
}

type DynamicConfigCache struct {
	m            map[string]*models.DynamicConfig
	lock         sync.RWMutex
	lastModified time.Time
}

func (c *DynamicConfigCache) Reload(db common.DBInterface) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	kvs, err := models.DynamicConfigs().All(db)
	if err != nil {
		return pkgerr.WithStack(err)
	}

	c.m = make(map[string]*models.DynamicConfig, len(kvs))
	for _, kv := range kvs {
		c.m[kv.Key] = kv
		if c.lastModified.Before(kv.UpdatedAt) {
			c.lastModified = kv.UpdatedAt
		}
	}

	return nil
}

func (c *DynamicConfigCache) ByKey(key string) (*models.DynamicConfig, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	r, ok := c.m[key]
	return r, ok
}

func (c *DynamicConfigCache) Set(kv *models.DynamicConfig) {
	c.lock.Lock()
	c.m[kv.Key] = kv
	if c.lastModified.Before(kv.UpdatedAt) {
		c.lastModified = kv.UpdatedAt
	}
	c.lock.Unlock()
}

func (c *DynamicConfigCache) Values() []*models.DynamicConfig {
	c.lock.RLock()
	defer c.lock.RUnlock()

	values := make([]*models.DynamicConfig, len(c.m))
	i := 0
	for _, v := range c.m {
		values[i] = v
		i++
	}

	return values
}

func (c *DynamicConfigCache) LastModified() time.Time {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.lastModified
}
