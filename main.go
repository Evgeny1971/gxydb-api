package main

import (
	"os"

	"github.com/Bnei-Baruch/gxydb-api/api"
)

func main() {
	listenAddress := os.Getenv("LISTEN_ADDRESS")
	dbUrl := os.Getenv("DB_URL")
	accountsUrl := os.Getenv("ACC_URL")
	skipAuth := os.Getenv("SKIP_AUTH") == "true"

	a := api.App{}
	a.Initialize(dbUrl, accountsUrl, skipAuth)
	a.Run(listenAddress)
}
