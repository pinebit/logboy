package app

import (
	"os"

	"github.com/ethereum/go-ethereum/common"
	"gopkg.in/yaml.v3"
)

type PostgresConfig struct {
	URL string `yaml:"url"`
}

type ServerConfig struct {
	Port uint16 `yaml:"port"`
}

type ContractConfig struct {
	ABI       string           `yaml:"abi"`
	Addresses []common.Address `yaml:"addresses"`
}

type ChainConfig struct {
	RPC       string                    `yaml:"rpc"`
	Contracts map[string]ContractConfig `yaml:"contracts"`
}

type Config struct {
	Postgres PostgresConfig         `yaml:"postgres"`
	Chains   map[string]ChainConfig `yaml:"chains"`
	Server   ServerConfig           `yaml:"server"`
}

func LoadConfigJSON(jsonPath string) (*Config, error) {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, err
	}

	data = []byte(os.ExpandEnv(string(data)))

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	if config.Server.Port == 0 {
		config.Server.Port = 3000
	}

	return config, err
}
