package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/Bnei-Baruch/gxydb-api/api"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Serve the backend API",
	Run:   serverFn,
}

func init() {
	rootCmd.AddCommand(serverCmd)
}

func serverFn(cmd *cobra.Command, args []string) {
	listenAddress := os.Getenv("LISTEN_ADDRESS")
	dbUrl := os.Getenv("DB_URL")
	accountsUrl := os.Getenv("ACC_URL")
	skipAuth := os.Getenv("SKIP_AUTH") == "true"
	skipEventsAuth := os.Getenv("SKIP_EVENTS_AUTH") == "true"

	a := api.App{}
	a.Initialize(dbUrl, accountsUrl, skipAuth, skipEventsAuth)
	a.Run(listenAddress)
}
