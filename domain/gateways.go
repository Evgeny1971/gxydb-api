package domain

import (
	"encoding/base64"
	"encoding/json"
	"sync"
	"time"

	janus_admin "github.com/edoshor/janus-go/admin"
	pkgerr "github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"

	"github.com/Bnei-Baruch/gxydb-api/common"
	"github.com/Bnei-Baruch/gxydb-api/models"
	"github.com/Bnei-Baruch/gxydb-api/pkg/crypt"
	"github.com/Bnei-Baruch/gxydb-api/pkg/patterns"
	"github.com/Bnei-Baruch/gxydb-api/pkg/stringutil"
)

var GatewayAdminAPIRegistry = NewGatewayAdminAPIRegistry()

type GatewayToken struct {
	Token     string    `json:"token"`
	Plugins   []string  `json:"plugins"`
	CreatedAt time.Time `json:"created_at"`
}

func (t *GatewayToken) Decrypt() (string, error) {
	decTokenB, err := base64.StdEncoding.DecodeString(t.Token)
	if err != nil {
		return "", pkgerr.Wrap(err, "base64 decode token")
	}

	decToken, err := crypt.Decrypt(decTokenB, common.Config.Secret)
	if err != nil {
		return "", pkgerr.Wrap(err, "crypt.Decrypt")
	}

	return decToken, nil
}

type GatewayTokensManager struct {
	*patterns.SimpleObservable
	db     common.DBInterface
	maxAge time.Duration
	ticker *time.Ticker
	wg     sync.WaitGroup
	wip    bool
}

func NewGatewayTokensManager(db common.DBInterface, maxAge time.Duration) *GatewayTokensManager {
	return &GatewayTokensManager{
		SimpleObservable: patterns.NewSimpleObservable(),
		db:               db,
		maxAge:           maxAge,
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

	return tokens[len(tokens)-1].Decrypt()
}

func (tm *GatewayTokensManager) Monitor() {
	if tm.ticker != nil {
		return
	}

	log.Info().Msg("monitoring gateways tokens")
	tm.ticker = time.NewTicker(10 * time.Second)
	go func() {
		for range tm.ticker.C {
			tm.SyncAll()
		}
		tm.wg.Done()
	}()
	tm.wg.Add(1)
}

func (tm *GatewayTokensManager) Close() {
	log.Info().Msg("GatewayTokensManager.Close")
	tm.ticker.Stop()
	log.Info().Msg("GatewayTokensManager.Close Waiting for worker goroutine to finish")
	tm.wg.Wait()
}

func (tm *GatewayTokensManager) SyncAll() {
	if tm.wip {
		log.Info().Msg("GatewayTokensManager.SyncAll WIP, skipping.")
		return
	}
	tm.wip = true
	defer func() { tm.wip = false }()

	gateways, err := models.Gateways(
		models.GatewayWhere.Disabled.EQ(false),
		models.GatewayWhere.RemovedAt.IsNull()).
		All(tm.db)
	if err != nil {
		log.Error().Err(err).Msg("GatewayTokensManager.SyncAll fetch gateways from DB")
		return
	}

	notify := false
	for _, gateway := range gateways {
		changed, err := tm.syncGatewayTokens(gateway)
		if err != nil {
			log.Error().Err(err).Msgf("GatewayTokensManager.SyncAll synchronizing gateway tokens %s", gateway.Name)
		}
		if changed {
			notify = true
		}
	}
	if notify {
		tm.NotifyAll(common.EventGatewayTokensChanged)
	}
}

func (tm *GatewayTokensManager) syncGatewayTokens(gateway *models.Gateway) (bool, error) {
	var props map[string]interface{}
	if gateway.Properties.Valid {
		if err := json.Unmarshal(gateway.Properties.JSON, &props); err != nil {
			return false, pkgerr.Wrap(err, "json.Unmarshal gateway.Properties")
		}
	} else {
		props = make(map[string]interface{})
	}

	var tokens []*GatewayToken
	tokensProp, ok := props["tokens"]
	if ok {
		b, err := json.Marshal(tokensProp)
		if err != nil {
			return false, pkgerr.Wrap(err, "json.Marshal tokens property")
		}
		if err := json.Unmarshal(b, &tokens); err != nil {
			return false, pkgerr.Wrap(err, "json.Unmarshal tokens")
		}
	} else {
		tokens = make([]*GatewayToken, 0)
	}

	support := true
	tokensResp, err := tm.listTokens(gateway)
	if err != nil {
		var e *janus_admin.ErrorAMResponse
		if pkgerr.As(err, &e) && e.Err.Code == 490 { // Stored-Token based authentication disabled
			support = false
		} else {
			return false, pkgerr.Wrap(err, "gateway AdminAPI listTokens")
		}
	}

	// gateway doesn't support tokens
	if !support {
		log.Warn().Msgf("GatewayTokensManager.syncGatewayTokens %s does not support tokens", gateway.Name)

		if len(tokens) == 0 { // nothing to do
			return false, nil
		}

		// delete all our tokens if any
		props["tokens"] = nil
		b, err := json.Marshal(props)
		if err != nil {
			return false, pkgerr.Wrap(err, "json.Marshal props")
		}
		gateway.Properties = null.JSONFrom(b)
		if _, err := gateway.Update(tm.db, boil.Whitelist(models.GatewayColumns.Properties)); err != nil {
			return false, pkgerr.WithMessage(err, "gateway.Update")
		}

		return true, nil
	}

	// gateway support tokens
	// delete any token the gateway have that we don't from the gateway
	// expired tokens in our db should be deleted from both db and gateway
	// ensure all non expired tokens exist on gateway
	// generate new token if no next token in DB or previous one was created more than a day ago

	gatewayTokens := tokensResp.(*janus_admin.ListTokensResponse)
	gatewayTokensMap := make(map[string]*janus_admin.StoredToken)
	for _, token := range gatewayTokens.Data["tokens"] {
		gatewayTokensMap[token.Token] = token
	}

	removeOnGateway := make([]string, 0)
	addOnGateway := make([]string, 0)
	nextDBTokens := make([]*GatewayToken, 0)
	changed := false

	dbTokensMap := make(map[string]*GatewayToken)
	maxAgeTS := time.Now().UTC().Add(-tm.maxAge)
	for _, token := range tokens {
		decToken, err := token.Decrypt()
		if err != nil {
			return false, pkgerr.WithMessage(err, "decrypt token")
		}
		dbTokensMap[decToken] = token

		if token.CreatedAt.Before(maxAgeTS) { // token has expired
			changed = true
			if _, ok := gatewayTokensMap[decToken]; ok {
				removeOnGateway = append(removeOnGateway, decToken) // remove it from gateway if it's there
			}
		} else { // token is valid
			nextDBTokens = append(nextDBTokens, token) // keep it in DB
			if _, ok := gatewayTokensMap[decToken]; !ok {
				addOnGateway = append(addOnGateway, decToken) // add it to gateway if it's not there already
			}
		}
	}

	// maybe something on gateway that DB is not aware of ?
	for token := range gatewayTokensMap {
		if _, ok := dbTokensMap[token]; !ok {
			removeOnGateway = append(removeOnGateway, token)
		}
	}

	// generate new token if no next token in DB or previous one was created more than a day ago
	if len(nextDBTokens) == 0 ||
		tokens[len(tokens)-1].CreatedAt.Before(time.Now().UTC().Add(-24*time.Hour)) {
		token, err := tm.createToken(gateway, stringutil.GenerateUID(16))
		if err != nil {
			return false, pkgerr.WithMessage(err, "create token")
		}
		nextDBTokens = append(nextDBTokens, token)
		changed = true
	}

	// sync to gateway
	for _, token := range addOnGateway {
		_, err := tm.createToken(gateway, token)
		if err != nil {
			return false, pkgerr.WithMessage(err, "create token [existing]")
		}
	}
	for _, token := range removeOnGateway {
		if err := tm.removeToken(gateway, token); err != nil {
			log.Error().Err(err).Msgf("GatewayTokensManager.syncGatewayTokens remove token on gateway %s", gateway.Name)
		}
	}

	// save changes in DB
	if changed {
		props["tokens"] = nextDBTokens
		b, err := json.Marshal(props)
		if err != nil {
			return false, pkgerr.Wrap(err, "json.Marshal props")
		}
		gateway.Properties = null.JSONFrom(b)
		if _, err := gateway.Update(tm.db, boil.Whitelist(models.GatewayColumns.Properties)); err != nil {
			return false, pkgerr.WithMessage(err, "gateway.Update")
		}
	}

	return changed, nil
}

func (tm *GatewayTokensManager) createToken(gateway *models.Gateway, tokenStr string) (*GatewayToken, error) {
	log.Info().Msgf("GatewayTokensManager.createToken %s", gateway.Name)

	token := GatewayToken{
		Token:     tokenStr,
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

func (tm *GatewayTokensManager) removeToken(gateway *models.Gateway, tokenStr string) error {
	log.Info().Msgf("GatewayTokensManager.removeToken %s from gateway %s", tokenStr, gateway.Name)
	api, err := GatewayAdminAPIRegistry.For(gateway)
	if err != nil {
		return pkgerr.WithMessage(err, "Admin API for gateway")
	}

	if _, err := api.RemoveToken(tokenStr); err != nil {
		return pkgerr.Wrap(err, "Admin API remove token")
	}

	return nil
}

func (tm *GatewayTokensManager) listTokens(gateway *models.Gateway) (interface{}, error) {
	api, err := GatewayAdminAPIRegistry.For(gateway)
	if err != nil {
		return nil, pkgerr.WithMessage(err, "Admin API for gateway")
	}

	resp, err := api.ListTokens()
	if err != nil {
		return nil, pkgerr.Wrap(err, "Admin API list token")
	}

	return resp, nil
}

type gatewayAdminAPIRegistry struct {
	lock     sync.RWMutex
	registry map[int64]janus_admin.AdminAPI
}

func NewGatewayAdminAPIRegistry() *gatewayAdminAPIRegistry {
	return &gatewayAdminAPIRegistry{
		registry: make(map[int64]janus_admin.AdminAPI),
	}
}

func (r *gatewayAdminAPIRegistry) For(gateway *models.Gateway) (janus_admin.AdminAPI, error) {
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

	api, err := janus_admin.NewAdminAPI(gateway.AdminURL, adminPwd)
	if err != nil {
		return nil, pkgerr.Wrap(err, "janus.NewAdminAPI")
	}

	r.Set(gateway, api)
	return api, nil
}

func (r *gatewayAdminAPIRegistry) Get(gateway *models.Gateway) (janus_admin.AdminAPI, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	api, ok := r.registry[gateway.ID]
	return api, ok
}

func (r *gatewayAdminAPIRegistry) Set(gateway *models.Gateway, api janus_admin.AdminAPI) {
	r.lock.Lock()
	r.registry[gateway.ID] = api
	r.lock.Unlock()
}
