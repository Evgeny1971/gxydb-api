package testutil

import (
	"os"

	"github.com/edoshor/janus-go"
)

type GatewayConfig struct {
	GatewayURL  string
	AdminURL    string
	AdminSecret string
}

type GatewayManager struct {
	Config   *GatewayConfig
	client   *janus.Gateway
	adminAPI janus.AdminAPI
}

func (m *GatewayManager) Init() {
	m.Config = &GatewayConfig{
		GatewayURL:  "ws://localhost:8188/",
		AdminURL:    "http://localhost:7088/admin",
		AdminSecret: "janusoverlord",
	}

	if val := os.Getenv("TEST_GATEWAY_URL"); val != "" {
		m.Config.GatewayURL = val
	}
	if val := os.Getenv("TEST_GATEWAY_ADMIN_URL"); val != "" {
		m.Config.AdminURL = val
	}
	if val := os.Getenv("TEST_GATEWAY_ADMIN_SECRET"); val != "" {
		m.Config.AdminSecret = val
	}
}

func (m *GatewayManager) Gateway() (*janus.Gateway, error) {
	if m.client != nil {
		return m.client, nil
	}

	var err error
	m.client, err = janus.Connect(m.Config.GatewayURL)
	if err != nil {
		return nil, err
	}

	adminAPI, err := m.GatewayAdminAPI()
	if err != nil {
		return nil, err
	}

	_, err = adminAPI.AddToken("default-test-token", []string{})
	if err != nil {
		return nil, err
	}
	m.client.Token = "default-test-token"

	return m.client, nil
}

func (m *GatewayManager) GatewayAdminAPI() (janus.AdminAPI, error) {
	if m.adminAPI != nil {
		return m.adminAPI, nil
	}

	var err error
	m.adminAPI, err = janus.NewAdminAPI(m.Config.AdminURL, m.Config.AdminSecret)
	return m.adminAPI, err
}

func (m *GatewayManager) NewGatewaySession() (*janus.Session, error) {
	g, err := m.Gateway()
	if err != nil {
		return nil, err
	}
	return g.Create()
}

func (m *GatewayManager) DestroyGatewaySessions() {
	if m.client == nil {
		return
	}

	for _, session := range m.client.Sessions {
		for _, handle := range session.Handles {
			handle.Detach()
		}
		session.Destroy()
	}
}

func (m *GatewayManager) CloseGateway() error {
	if m.client == nil {
		return nil
	}

	m.DestroyGatewaySessions()

	return m.client.Close()
}
