package api

import (
	"fmt"

	"github.com/volatiletech/sqlboiler/boil"

	"github.com/Bnei-Baruch/gxydb-api/models"
)

type AppCache struct {
	gateways map[int64]*models.Gateway
}

func (c *AppCache) Init(db boil.Executor) error {
	return c.Reload(db)
}

func (c *AppCache) Reload(db boil.Executor) error {
	if err := c.loadGateways(db); err != nil {
		return fmt.Errorf("initialize gateways: %w", err)
	}

	return nil
}

func (c *AppCache) loadGateways(db boil.Executor) error {
	gateways, err := models.Gateways().All(db)
	if err != nil {
		return fmt.Errorf("fetch from DB: %w", err)
	}

	c.gateways = make(map[int64]*models.Gateway, len(gateways))
	for i := range gateways {
		c.gateways[gateways[i].ID] = gateways[i]
	}

	return nil
}
