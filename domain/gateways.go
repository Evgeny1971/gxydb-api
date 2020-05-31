package domain

import (
	"encoding/base64"
	"encoding/json"
	"sync"
	"time"

	"github.com/edoshor/janus-go"
	pkgerr "github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"

	"github.com/Bnei-Baruch/gxydb-api/common"
	"github.com/Bnei-Baruch/gxydb-api/models"
	"github.com/Bnei-Baruch/gxydb-api/pkg/crypt"
	"github.com/Bnei-Baruch/gxydb-api/pkg/stringutil"
)

var GatewayAdminAPIRegistry = NewGatewayAdminAPIRegistry()

type GatewayToken struct {
	Token     string    `json:"token"`
	Plugins   []string  `json:"plugins"`
	CreatedAt time.Time `json:"created_at"`
}

type GatewayTokensManager struct {
	db     common.DBInterface
	maxAge time.Duration
}

func NewGatewayTokensManager(db common.DBInterface, maxAge time.Duration) *GatewayTokensManager {
	return &GatewayTokensManager{
		db:     db,
		maxAge: maxAge,
	}
}

func (tm *GatewayTokensManager) ActiveToken(gateway *models.Gateway) (string, error) {
	if !gateway.Properties.Valid {
		return "", nil
	}

	var props map[string]interface{}
	if err := json.Unmarshal(gateway.Properties.JSON, &props); err != nil {
		return "", pkgerr.Wrap(err, "json.Unmarshal gateway.Properties")
	}

	tokensProp, ok := props["tokens"]
	if !ok {
		return "", nil
	}

	b, err := json.Marshal(tokensProp)
	if err != nil {
		return "", pkgerr.Wrap(err, "json.Marshal tokens property")
	}

	var tokens []*GatewayToken
	if err := json.Unmarshal(b, &tokens); err != nil {
		return "", pkgerr.Wrap(err, "json.Unmarshal tokens")
	}

	if len(tokens) == 0 {
		return "", nil
	}

	token := tokens[len(tokens)-1]
	decTokenB, err := base64.StdEncoding.DecodeString(token.Token)
	if err != nil {
		return "", pkgerr.Wrap(err, "base64 decode token")
	}
	decToken, err := crypt.Decrypt(decTokenB, common.Config.Secret)
	if err != nil {
		return "", pkgerr.Wrap(err, "crypt.Decrypt")
	}

	return decToken, nil
}

func (tm *GatewayTokensManager) RotateAll() error {
	gateways, err := models.Gateways(
		models.GatewayWhere.Disabled.EQ(false),
		models.GatewayWhere.RemovedAt.IsNull()).
		All(tm.db)
	if err != nil {
		return pkgerr.WithStack(err)
	}
	log.Info().Msgf("got %d gateways from DB", len(gateways))

	for _, gateway := range gateways {
		if err := tm.rotateGatewayTokens(gateway); err != nil {
			log.Error().Err(err).Msgf("error rotating gateway tokens %s", gateway.Name)
		}
	}

	return nil
}

func (tm *GatewayTokensManager) rotateGatewayTokens(gateway *models.Gateway) error {
	log.Info().Msgf("Rotating tokens of gateway %s", gateway.Name)

	var props map[string]interface{}
	if gateway.Properties.Valid {
		if err := json.Unmarshal(gateway.Properties.JSON, &props); err != nil {
			return pkgerr.Wrap(err, "json.Unmarshal gateway.Properties")
		}
	} else {
		props = make(map[string]interface{})
	}

	var tokens []*GatewayToken
	tokensProp, ok := props["tokens"]
	if ok {
		b, err := json.Marshal(tokensProp)
		if err != nil {
			return pkgerr.Wrap(err, "json.Marshal tokens property")
		}
		if err := json.Unmarshal(b, &tokens); err != nil {
			return pkgerr.Wrap(err, "json.Unmarshal tokens")
		}

		log.Info().Msgf("gateway %s has %d tokens", gateway.Name, len(tokens))
	} else {
		tokens = make([]*GatewayToken, 0)
	}

	// remove expired tokens
	maxAgeTS := time.Now().UTC().Add(-tm.maxAge)
	tokensToSave := make([]*GatewayToken, 0)
	for _, token := range tokens {
		if token.CreatedAt.Before(maxAgeTS) {
			if err := tm.removeToken(gateway, token); err != nil {
				log.Error().Err(err).Msgf("error removing token")
			}
		} else {
			tokensToSave = append(tokensToSave, token)
		}
	}

	// create new token
	token, err := tm.createToken(gateway)
	if err != nil {
		return pkgerr.WithMessage(err, "create token")
	}
	tokensToSave = append(tokensToSave, token)

	// update props in DB
	props["tokens"] = tokensToSave
	b, err := json.Marshal(props)
	if err != nil {
		return pkgerr.Wrap(err, "json.Marshal props")
	}
	gateway.Properties = null.JSONFrom(b)
	if _, err := gateway.Update(tm.db, boil.Whitelist(models.GatewayColumns.Properties)); err != nil {
		return pkgerr.WithMessage(err, "gateway.Update")
	}

	return nil
}

func (tm *GatewayTokensManager) createToken(gateway *models.Gateway) (*GatewayToken, error) {
	log.Info().Msgf("creating new token on gateway %s", gateway.Name)

	token := GatewayToken{
		Token:     stringutil.GenerateUID(16),
		CreatedAt: time.Now().UTC(),
	}
	switch gateway.Type {
	case common.GatewayTypeRooms:
		token.Plugins = []string{"janus.plugin.videoroom", "janus.plugin.textroom"}
	case common.GatewayTypeStreaming:
		token.Plugins = []string{"janus.plugin.streaming"}
	default:
		break
	}

	api, err := GatewayAdminAPIRegistry.For(gateway)
	if err != nil {
		return nil, pkgerr.WithMessage(err, "Admin API for gateway")
	}

	if _, err := api.AddToken(token.Token, token.Plugins); err != nil {
		return nil, pkgerr.Wrap(err, "Admin API add token")
	}

	encToken, err := crypt.Encrypt([]byte(token.Token), common.Config.Secret)
	if err != nil {
		return nil, pkgerr.Wrap(err, "crypt.Encrypt new token")
	}
	token.Token = base64.StdEncoding.EncodeToString(encToken)

	return &token, nil
}

func (tm *GatewayTokensManager) removeToken(gateway *models.Gateway, token *GatewayToken) error {
	log.Info().Msgf("remove token %s from gateway %s", token.Token, gateway.Name)

	decTokenB, err := base64.StdEncoding.DecodeString(token.Token)
	if err != nil {
		return pkgerr.Wrap(err, "base64 decode token")
	}
	decToken, err := crypt.Decrypt(decTokenB, common.Config.Secret)
	if err != nil {
		return pkgerr.Wrap(err, "crypt.Decrypt")
	}

	api, err := GatewayAdminAPIRegistry.For(gateway)
	if err != nil {
		return pkgerr.WithMessage(err, "Admin API for gateway")
	}

	if _, err := api.RemoveToken(decToken); err != nil {
		return pkgerr.Wrap(err, "Admin API remove token")
	}

	return nil
}

type gatewayAdminAPIRegistry struct {
	lock     sync.RWMutex
	registry map[int64]janus.AdminAPI
}

func NewGatewayAdminAPIRegistry() *gatewayAdminAPIRegistry {
	return &gatewayAdminAPIRegistry{
		registry: make(map[int64]janus.AdminAPI),
	}
}

func (r *gatewayAdminAPIRegistry) For(gateway *models.Gateway) (janus.AdminAPI, error) {
	if api, ok := r.Get(gateway); ok {
		return api, nil
	}

	aPwdB, err := base64.StdEncoding.DecodeString(gateway.AdminPassword)
	if err != nil {
		return nil, pkgerr.Wrap(err, "base64 decode admin password")
	}
	adminPwd, err := crypt.Decrypt(aPwdB, common.Config.Secret)
	if err != nil {
		return nil, pkgerr.Wrap(err, "decrypt admin password")
	}

	api, err := janus.NewAdminAPI(gateway.AdminURL, adminPwd)
	if err != nil {
		return nil, pkgerr.Wrap(err, "janus.NewAdminAPI")
	}

	r.Set(gateway, api)
	return api, nil
}

func (r *gatewayAdminAPIRegistry) Get(gateway *models.Gateway) (janus.AdminAPI, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	api, ok := r.registry[gateway.ID]
	return api, ok
}

func (r *gatewayAdminAPIRegistry) Set(gateway *models.Gateway, api janus.AdminAPI) {
	r.lock.Lock()
	r.registry[gateway.ID] = api
	r.lock.Unlock()
}
