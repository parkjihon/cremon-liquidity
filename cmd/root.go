package cmd

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cremon",
	Short: "dex monitoring tool for crescent",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	RunE: func(cmd *cobra.Command, args []string) error {

		cmd.Help()
		//-- stop infinity
		return nil

	},
}

func Execute() {
	log.Info().Msg("======= cmd started =======")
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	log.Info().Msg("cmd init function triggered")
	rootCmd.Flags().BoolP("toggle", "t", false, "Help Message for toggle")

	// for debugging
	if len(os.Args) == 1 {
		args := append([]string{startCmd.Use}, os.Args[1:]...)
		rootCmd.SetArgs(args)
	}
}
