package main

import (
	"cremon-liquidity/cmd"
	"cremon-liquidity/config"
	"cremon-liquidity/global"
	"os"
	"strconv"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Logger Start
	output := zerolog.ConsoleWriter{Out: os.Stderr}
	log.Logger = zerolog.New(output).With().Timestamp().Logger()
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMicro
	//log.TimeFieldFormat
	log.Info().Msg("----------- Program Start --------------")

	// Load Config
	global.Config = config.NewConfig()

	// Logger setting
	lv := int8(global.Config.LogLevel)
	if lv <= 5 {
		zerolog.SetGlobalLevel(zerolog.Level(lv)) // if lv==0	level="debug"
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// Print config
	log.Info().Str("zerolog Global Level", zerolog.GlobalLevel().String()).Str("Config.LogLevel", strconv.Itoa(int(lv))).Msg("Log Level check")
	log.Info().Str("version", global.Config.Version).Str("id", global.Config.IdName).Msg("Config Loaded")

	// sdk config
	if len(global.Config.Chains) > 0 {
		sdkConfig := sdktypes.GetConfig()
		prefixAcc := global.Config.Chains[global.Config.MainIdx].AccPrefix
		sdkConfig.SetBech32PrefixForAccount(prefixAcc, prefixAcc+sdktypes.PrefixPublic)
	}

	cmd.Execute()
}
