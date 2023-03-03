package app

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/pinebit/lognite/app/common"
	"gopkg.in/yaml.v3"
)

type ConsoleConfig struct {
	Disabled bool `yaml:"disabled"`
}

type PostgresConfig struct {
	URL       string        `yaml:"url"`
	Retention time.Duration `yaml:"retention"`
}

type ServerConfig struct {
	Port uint16 `yaml:"port"`
}

type ContractConfig struct {
	ABI       string              `yaml:"abi"`
	Address   ethcommon.Address   `yaml:"address"`
	Addresses []ethcommon.Address `yaml:"addresses"`
	Events    []string            `yaml:"events"`
}

type ChainConfig struct {
	RPC           string                    `yaml:"rpc"`
	Confirmations uint                      `yaml:"confirmations"`
	Contracts     map[string]ContractConfig `yaml:"contracts"`
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

func LoadConfig(filepath string) (*Config, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	data = []byte(os.ExpandEnv(string(data)))

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	adjustDefaultValues(config)

	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return config, err
}

func adjustDefaultValues(config *Config) {
	if config.Server.Port == 0 {
		config.Server.Port = common.DefaultServerPort
	}

	if config.Outputs.Postgres != nil && config.Outputs.Postgres.Retention.Nanoseconds() == 0 {
		config.Outputs.Postgres.Retention = common.DefaultPostgresRetention
	}

	for chainName, chain := range config.Chains {
		if chain.Confirmations == 0 {
			chain.Confirmations = common.DefaultConfirmations
			config.Chains[chainName] = chain
		}
	}
}

func validateConfig(config *Config) error {
	zeroAddress := ethcommon.HexToAddress("0x00")
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
		if chain.Confirmations > 10000 {
			return fmt.Errorf("chain '%s' 'confirmations' is too large", chainName)
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
			for _, eventName := range contract.Events {
				if !validIdentifier.MatchString(eventName) {
					return fmt.Errorf("chain '%s' contract '%s' has invalid 'events' value: '%s'", chainName, contractName, eventName)
				}
			}
		}
	}

	if config.Outputs.Postgres != nil {
		if len(config.Outputs.Postgres.URL) == 0 {
			return errors.New("'outputs.postgres' has no 'url' specified")
		}
		if config.Outputs.Postgres.Retention < time.Hour {
			return errors.New("'outputs.postgres.retention' must be longer than 1h")
		}
	}

	return nil
}
