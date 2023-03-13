package config

import (
	"fmt"
	"io/ioutil"

	"github.com/pelletier/go-toml"
)

var (
	DefaultConfigPath = "./config.toml"
)

// Config defines all necessary configuration parameters.
type tomlConfig struct {
	Conf        MainConfig                  `toml:"config"`
	MonConf     MonitorConfig               `toml:"monitor"`
	Chains      map[string]ChainConfig      `toml:"chains"`
	Connections map[string]ConnectionConfig `toml:"connections"`
}

type MainConfig struct {
	Target         string   `toml:"target"` //mainnet, testnet, devnet for check invalid config.
	Mode           string   `toml:"mode"`
	UniqueName     string   `toml:"unique_name"`
	DSN            string   `toml:"db_dsn"`
	RecoverModules []string `toml:"recover_modules"`
	LogFile        string   `toml:"log_file"`
	LogLevel       int      `toml:"log_level"`
	FlushForce     int      `toml:"flush_force"`
	HACounterIP    string   `toml:"ha_counter_ip"`
	HAPort         int      `toml:"ha_port"`
	TraceFile      string   `toml:"trace_file"`
	//LocalGrpcEndpoint string   `toml:"local_grpc_endpoint"`
	//LocalRestEndpoint string   `toml:"local_rest_endpoint"`

}

type ChainConfig struct {
	ChainId        string `toml:"chain_id"`
	RPCEndpoint    string `toml:"rpc_endpoint"`
	GRPCEndpoint   string `toml:"grpc_endpoint"`
	TraceStoreFile string `toml:"tracestore"` //only for main chain
	AccPrefix      string `toml:"acc_prefix"`
	MainSide       int    `toml:"main"`
	SubDisabled    int    `toml:"sub_disabled"` //subscription disable (disable rpc connection. go routine PROC not executed)
}

type MonitorConfig struct {
	QueueName       string `toml:"queue_name"`
	Region          string `toml:"aws_region"`
	AwsKey          string `toml:"aws_key"`
	AwsSecret       string `toml:"aws_secret"`
	BotAPI          string `toml:"bot_api"`
	BotMasterChatId int64  `toml:"bot_master_chat_id"`
}

type ConnectionConfig struct {
	Channel        string `toml:"channel"`
	CounterChainId string `toml:"counter_chain_id"`
	CounterChannel string `toml:"counter_channel"`
}

func Load(configPath string) (*tomlConfig, error) {
	if configPath == "" {
		return nil, fmt.Errorf("empty configuration path")
	}

	// Read file
	var cfg tomlConfig
	configData, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %s", err)
	}
	err = toml.Unmarshal(configData, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config: %s", err)
	}

	return &cfg, nil
}
