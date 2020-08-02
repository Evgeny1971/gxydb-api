package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Bnei-Baruch/gxydb-api/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of gxydb-api",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Backend API for Galaxy version %s\n", version.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
