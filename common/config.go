package common

import (
	"os"
	"strings"
)

type config struct {
	ListenAddress    string
	DBUrl            string
	AccountsUrl      string
	SkipAuth         bool
	SkipEventsAuth   bool
	SkipPermissions  bool
	IceServers       map[string][]string
	ServicePasswords []string
}

func newConfig() *config {
	return &config{
		ListenAddress:    ":8081",
		DBUrl:            "postgres://user:password@localhost/galaxy?sslmode=disable",
		AccountsUrl:      "https://accounts.kbb1.com/auth/realms/main",
		SkipAuth:         false,
		SkipEventsAuth:   false,
		SkipPermissions:  false,
		IceServers:       make(map[string][]string),
		ServicePasswords: make([]string, 0),
	}
}

var Config *config

func Init() {
	Config = newConfig()

	if val := os.Getenv("LISTEN_ADDRESS"); val != "" {
		Config.ListenAddress = val
	}
	if val := os.Getenv("DB_URL"); val != "" {
		Config.DBUrl = val
	}
	if val := os.Getenv("ACCOUNTS_URL"); val != "" {
		Config.AccountsUrl = val
	}
	if val := os.Getenv("SKIP_AUTH"); val != "" {
		Config.SkipAuth = val == "true"
	}
	if val := os.Getenv("SKIP_EVENTS_AUTH"); val != "" {
		Config.SkipEventsAuth = val == "true"
	}
	if val := os.Getenv("SKIP_PERMISSIONS"); val != "" {
		Config.SkipPermissions = val == "true"
	}
	if val := os.Getenv("ICE_SERVERS_ROOMS"); val != "" {
		Config.IceServers["rooms"] = strings.Split(val, ",")
	}
	if val := os.Getenv("ICE_SERVERS_STREAMING"); val != "" {
		Config.IceServers["streaming"] = strings.Split(val, ",")
	}
	if val := os.Getenv("SERVICE_PASSWORDS"); val != "" {
		Config.ServicePasswords = strings.Split(val, ",")
	}
}
