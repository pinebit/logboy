package app

import (
	"encoding/json"
	"os"
)

type PostgresConfig struct {
	Conn string `json:"conn"`
}

type ServerConfig struct {
	Port uint16 `json:"port"`
}

type ContractConfig struct {
	Name      string   `json:"name"`
	ABI       string   `json:"abi"`
	Addresses []string `json:"addresses"`
}

type Config struct {
	RPC       map[string]string `json:"rpc"`
	Postgres  PostgresConfig    `json:"postgres"`
	Server    ServerConfig      `json:"server"`
	Contracts []ContractConfig  `json:"contracts"`
}

func LoadConfigJSON(jsonPath string) (*Config, error) {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, err
	}

	config := &Config{}
	if err := json.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, err
}
