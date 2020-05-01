package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/subosito/gotenv"

	"github.com/Bnei-Baruch/gxydb-api/common"
)

var rootCmd = &cobra.Command{
	Use:   "gxydb-api",
	Short: "BB Galaxy ",
	Long:  `Backend API for the BB galaxy system`,
}

func init() {
	cobra.OnInitialize(initConfig)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	gotenv.Load()
	common.Init()
}
