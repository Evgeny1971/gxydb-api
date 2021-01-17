package api

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"net"

	"github.com/eclipse/paho.golang/paho"
	pkgerr "github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/Bnei-Baruch/gxydb-api/common"
)

type MQTTListener struct {
	client                 *paho.Client
	cache                  *AppCache
	serviceProtocolHandler ServiceProtocolHandler
}

func NewMQTTListener(cache *AppCache, sph ServiceProtocolHandler) *MQTTListener {
	return &MQTTListener{
		cache:                  cache,
		serviceProtocolHandler: sph,
	}
}

func (l *MQTTListener) Start() error {
	l.client = paho.NewClient(paho.ClientConfig{
		ClientID: "gxydb-api",
		OnConnectionLost: func(err error) {
			log.Warn().Msgf("MQTT OnConnectionLost: %+v", err)
			if err := l.init(); err != nil {
				log.Error().Err(err).Msg("error initializing mqtt on connection lost")
			}
		},
	})
	l.client.SetErrorLogger(NewPahoLogAdapter(zerolog.ErrorLevel))
	debugLog := NewPahoLogAdapter(zerolog.InfoLevel)
	l.client.SetDebugLogger(debugLog)
	l.client.PingHandler.SetDebug(debugLog)
	l.client.Router.SetDebug(debugLog)
	return l.init()
}

func (l *MQTTListener) init() error {
	log.Info().Msg("Initializing MQTT Listener")

	var conn net.Conn
	var err error
	if common.Config.MQTTSecure {
		conn, err = tls.Dial("tcp", common.Config.MQTTBrokerUrl, nil)
	} else {
		conn, err = net.Dial("tcp", common.Config.MQTTBrokerUrl)
	}
	if err != nil {
		return pkgerr.Wrap(err, "conn.Dial")
	}

	l.client.Conn = conn

	var sessionExpiryInterval = uint32(math.MaxUint32)

	cp := &paho.Connect{
		ClientID:   common.Config.MQTTClientID,
		KeepAlive:  30,
		CleanStart: true,
		Properties: &paho.ConnectProperties{
			SessionExpiryInterval: &sessionExpiryInterval,
		},
	}

	var pwd string
	if dc, ok := l.cache.dynamicConfig.ByKey(common.DynamicConfigMQTTAuth); ok {
		pwd = dc.Value
	} else {
		pwd = common.Config.MQTTPassword
	}
	if pwd != "" {
		cp.Username = "gxydb-api"
		cp.Password = []byte(pwd)
		cp.UsernameFlag = true
		cp.PasswordFlag = true
	}

	ca, err := l.client.Connect(context.Background(), cp)
	if err != nil {
		return pkgerr.Wrap(err, "client.Connect")
	}
	if ca.ReasonCode != 0 {
		return pkgerr.Errorf("MQTT connect error: %d - %s", ca.ReasonCode, ca.Properties.ReasonString)
	}

	l.client.Router.RegisterHandler("galaxy/service/#", l.HandleServiceProtocol)

	sa, err := l.client.Subscribe(context.Background(), &paho.Subscribe{
		Subscriptions: map[string]paho.SubscribeOptions{
			"galaxy/service/#": {QoS: byte(2)},
		},
	})
	if err != nil {
		return pkgerr.Wrap(err, "client.Subscribe")
	}
	if sa.Reasons[0] != byte(2) {
		return pkgerr.Errorf("MQTT subscribe error: %d ", sa.Reasons[0])
	}

	return nil
}

func (l *MQTTListener) Close() error {
	if err := l.client.Disconnect(&paho.Disconnect{ReasonCode: 0}); err != nil {
		return pkgerr.Wrap(err, "client.Disconnect")
	}
	return nil
}

func (l *MQTTListener) HandleServiceProtocol(p *paho.Publish) {
	log.Info().Str("payload", p.String()).Msg("MQTT handle service protocol")
	if err := l.serviceProtocolHandler.HandleMessage(string(p.Payload)); err != nil {
		log.Error().Err(err).Msg("service protocol error")
	}
}

type PahoLogAdapter struct {
	level zerolog.Level
}

func NewPahoLogAdapter(level zerolog.Level) *PahoLogAdapter {
	return &PahoLogAdapter{level: level}
}

func (a *PahoLogAdapter) Println(v ...interface{}) {
	log.WithLevel(a.level).Msgf("mqtt: %s", fmt.Sprint(v...))
}

func (a *PahoLogAdapter) Printf(format string, v ...interface{}) {
	log.WithLevel(a.level).Msgf("mqtt: %s", fmt.Sprintf(format, v...))
}
