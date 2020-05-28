package cmd

import (
	"database/sql"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/Bnei-Baruch/gxydb-api/common"
	"github.com/Bnei-Baruch/gxydb-api/domain"
)

var rotateTokensCmd = &cobra.Command{
	Use:   "rotate_tokens",
	Short: "Rotate gateways stored tokens",
	Run:   rotateTokensFn,
}

func init() {
	rotateTokensCmd.Flags().IntVarP(&maxAge, "max-age", "a", 7, "Token maximum age in days")
	rootCmd.AddCommand(rotateTokensCmd)
}

var db *sql.DB
var maxAge int

func rotateTokensFn(cmd *cobra.Command, args []string) {
	var err error

	log.Info().Msg("starting gateways token rotation")
	log.Info().Msgf("max-age is %d days", maxAge)

	db, err = sql.Open("postgres", common.Config.DBUrl)
	if err != nil {
		log.Fatal().Err(err).Msg("sql.Open")
	}

	tokenManager := domain.NewGatewayTokensManager(db, time.Duration(maxAge)*24*time.Hour)
	if err := tokenManager.RotateAll(); err != nil {
		log.Fatal().Err(err).Msg("error rotating tokens")
	}

	log.Info().Msg("finish")
}
