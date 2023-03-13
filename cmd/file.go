package cmd

import (
	"cremon-liquidity/global"
	"cremon-liquidity/proc"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cobraCmd *cobra.Command, args []string) {
		log.Info().Msg("======= start trace file command ========")
		cfg := global.Config
		log.Info().Msg(cfg.TraceFile)

		// Start Process Manager
		pm, err := proc.NewProcManager(cobraCmd.Context(), cfg)
		if err != nil {
			log.Panic().Str("err", err.Error()).Msg("Init ProcManager Failed")
		}

		// entry
		// never return
		pm.Start("file")
	},
}

func init() {
	rootCmd.AddCommand(fileCmd)
}
