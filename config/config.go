package config

import (
	"cremon-liquidity/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/crescent-network/crescent/v4/app"
	"github.com/rs/zerolog/log"
)

// config parameters updated from DB
type ConfigParam struct {
	Key             string `db:"key"`
	Value           string `db:"value"`
	UpdateTimestamp int64  `db:"update_ts"`
	Desc            string `db:"desc"`
}

// for Monitoring process (MonProc)
type AwsSQSConfig struct {
	QueueName string
	Region    string
	AwsKey    string
	AwsSecret string
}

// Telegram Bot config
type TGConfig struct {
	BotAPI    string
	BotMaster int64
}

type Config struct {
	IdName      string // id for distinguish mutiple cremons
	Version     string // cremon version
	Chains      []*types.Chain
	MainIdx     int // crescent chain index in Chains struct. It's used to set sdktypes.Config(Bech32) at main.go
	MainChainId string
	DatabaseURI string
	NodeIP      string
	LogFile     string
	LogLevel    int
	HACounterIP string      // cremon local IP for HA(master -> slave, slave->master) should be in same VPC.
	Cdc         codec.Codec // crescent app codec for marshal & unmarshal
	ParamMap    map[string]ConfigParam
	MonConf     AwsSQSConfig
	TgConf      TGConfig
	HANotReady  bool // if true, can't run a HA process
	HAPort      int
	HAStatus    int    // disconnected:0, connected:1
	Mode        string // working mode
	Target      string // Target network: mainnet, testnet, devnet
	TraceFile   string
}

var singletonConfig *Config

func NewConfig() *Config {
	log.Info().Msg("Config 시작")

	// 이건 왜 하는지 모르겠다.
	if singletonConfig != nil {
		log.Info().Msg("이게 nil 이 아닐 수도 있나?")
		return singletonConfig
	}

	// Load toml file
	cfgFile, err := Load(DefaultConfigPath)
	if err != nil {
		log.Panic().Str("error", err.Error()).Msg("invalid config file. can not load")
	}

	// 체인 정보
	i := 0
	chains := make([]*types.Chain, 0, len(cfgFile.Chains))
	//var
	//HAProcDisabled := false
	mainIdx := 0
	mainChainId := ""
	for _, chainCfg := range cfgFile.Chains {
		if chainCfg.MainSide == 1 {
			mainChainId = chainCfg.ChainId
			mainIdx = i // idx in Chains array. not cfgFileChains
		}
		if chainCfg.ChainId == "" || chainCfg.AccPrefix == "" || chainCfg.RPCEndpoint == "" {
			log.Panic().Msg("Invalid Chain Config")
		}
		c := types.Chain{
			ChainId:              chainCfg.ChainId,
			AccPrefix:            chainCfg.AccPrefix,
			StoreFile:            chainCfg.TraceStoreFile,
			WsRpcEndpoint:        chainCfg.RPCEndpoint,
			GRPCEndpoint:         chainCfg.GRPCEndpoint,
			MainSide:             chainCfg.MainSide,
			SubscriptionDisabled: chainCfg.SubDisabled,
		}
		c.CounterChannel = make(map[string]types.CounterChannelInfo)
		for _, cInfo := range cfgFile.Connections {
			if c.ChainId == mainChainId {
				v := types.CounterChannelInfo{
					ChainId:     cInfo.CounterChainId,
					ChannelName: cInfo.CounterChannel,
				}
				c.CounterChannel[cInfo.Channel] = v
			} else {
				if cInfo.CounterChainId == c.ChainId {
					v := types.CounterChannelInfo{
						ChainId:     mainChainId,
						ChannelName: cInfo.Channel,
					}
					c.CounterChannel[cInfo.CounterChannel] = v
				}
			}
		}
		if chainCfg.MainSide != 0 && len(c.CounterChannel) == 0 {
			log.Panic().Msg("Invalid Chain Config. check IBC channel Info")
		}
		chains = append(chains, &c)

		log.Info().Str("chainId", c.ChainId).Str("prefix", c.AccPrefix).Str("rpcEP", c.WsRpcEndpoint).Str("gprcEP", c.GRPCEndpoint).Int("idx", i).Msg("Load Config")
		i++
	}
	log.Info().Msg("Channel Loaded")

	// set marshaler for decoding trace-store
	// TODO: check Marshaler is thread-safe
	cdc := app.MakeEncodingConfig().Marshaler

	conf := &Config{
		//IdName:      cfgFile.Conf.UniqueName,
		//Version:     Version,
		Chains: chains,
		//DatabaseURI: cfgFile.Conf.DSN,
		//NodeIP:      "0.0.0.0", // listening ip for HA. not used.
		// LogFile:     "",
		// LogLevel:    cfgFile.Conf.LogLevel,
		// HACounterIP: cfgFile.Conf.HACounterIP,
		MainIdx:     mainIdx,
		MainChainId: mainChainId,
		Cdc:         cdc,
		TraceFile:   cfgFile.Conf.TraceFile,
		// MonConf:     monConf,
		// TgConf:      tgConf,
		// HAPort:      cfgFile.Conf.HAPort,
		// HANotReady:  HAProcDisabled,
		// Mode:        cfgFile.Conf.Mode,
		// ParamMap:    cfgMap,
		//LocalRestEndpoint: cfgFile.Conf.LocalRestEndpoint,
		//LocalGrpcEndpoint: cfgFile.Conf.LocalGrpcEndpoint,
	}

	singletonConfig = conf

	conf.Version = "18"

	log.Info().Msg("--------- Default Config Loaded successfully ----------")
	return conf
}
