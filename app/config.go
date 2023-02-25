package app

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"gopkg.in/yaml.v3"
)

type ConsoleConfig struct {
	Disabled bool `yaml:"disabled"`
}

type PostgresConfig struct {
	URL       string         `yaml:"url"`
	Retention *time.Duration `yaml:"retention"`
}

type ServerConfig struct {
	Port uint16 `yaml:"port"`
}

type ContractConfig struct {
	ABI       string           `yaml:"abi"`
	Address   common.Address   `yaml:"address"`
	Addresses []common.Address `yaml:"addresses"`
}

type ChainConfig struct {
	RPC       string                    `yaml:"rpc"`
	Contracts map[string]ContractConfig `yaml:"contracts"`
}

type OutputsConfig struct {
	Console  *ConsoleConfig  `yaml:"console"`
	Postgres *PostgresConfig `yaml:"postgres"`
}

type Config struct {
	Chains  map[string]ChainConfig `yaml:"chains"`
	Server  ServerConfig           `yaml:"server"`
	Outputs OutputsConfig          `yaml:"outputs"`
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
		config.Server.Port = defaultServerPort
	}

	if config.Outputs.Postgres != nil && config.Outputs.Postgres.Retention == nil {
		retention := defaultPostgresRetention
		config.Outputs.Postgres.Retention = &retention
	}

	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return config, err
}

func validateConfig(config *Config) error {
	zeroAddress := common.HexToAddress("0x00")
	validIdentifier := regexp.MustCompile(`^[a-zA-Z]+(\_[a-zA-Z0-9]+)*$`)

	if config.Server.Port == 0 {
		return errors.New("server 'port' is not specified or cannot be zero")
	}

	if len(config.Chains) == 0 {
		return errors.New("configuration has no chains")
	}

	for chainName, chain := range config.Chains {
		if !validIdentifier.MatchString(chainName) {
			return fmt.Errorf("chain name '%s' is not a valid identifier", chainName)
		}
		if !strings.HasPrefix(chain.RPC, "wss://") {
			return fmt.Errorf("chain '%s' 'rpc' has not wss scheme: %s", chainName, chain.RPC)
		}
		if len(chain.Contracts) == 0 {
			return fmt.Errorf("chain '%s' has no contracts configured", chainName)
		}

		for contractName, contract := range chain.Contracts {
			if !validIdentifier.MatchString(contractName) {
				return fmt.Errorf("chain '%s' contract name '%s' is not a valid identifier", chainName, contractName)
			}
			if len(contract.ABI) == 0 {
				return fmt.Errorf("chain '%s' contract '%s' has no 'abi' specified", chainName, contractName)
			}
			if contract.Address != zeroAddress && len(contract.Addresses) != 0 {
				return fmt.Errorf("chain '%s' contract '%s' has both 'address' and 'addresses' specified", chainName, contractName)
			}
			if contract.Address == zeroAddress && len(contract.Addresses) == 0 {
				return fmt.Errorf("chain '%s' contract '%s' has neither 'address' nor 'addresses' specified", chainName, contractName)
			}
		}
	}

	if config.Outputs.Postgres != nil {
		if len(config.Outputs.Postgres.URL) == 0 {
			return errors.New("'outputs.postgres' has no 'url' specified")
		}
		if *config.Outputs.Postgres.Retention < time.Hour {
			return errors.New("'outputs.postgres.retention' must be longer than 1h")
		}
	}

	return nil
}
