package cmd

import (
	"cremon-liquidity/global"
	"cremon-liquidity/proc"
	"encoding/json"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cobraCmd *cobra.Command, args []string) {
		cfg := global.Config
		msdCfg, err := json.Marshal(cfg)
		if err != nil {
			panic(err)
		}
		log.Info().Msg(string(msdCfg))

		// Start Process Manager
		pm, err := proc.NewProcManager(cobraCmd.Context(), cfg)
		if err != nil {
			log.Panic().Str("err", err.Error()).Msg("Init ProcManager Failed")
		}

		// entry
		// never return
		pm.Start("start")
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
