package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
)

type EventInputABI struct {
	Indexed      bool   `json:"indexed"`
	InternalType string `json:"internalType"`
	Name         string `json:"name"`
	Type         string `json:"type"`
}

type EventABI struct {
	Name      string          `json:"name"`
	Type      string          `json:"type"`
	Anonymous bool            `json:"anonymous"`
	Inputs    []EventInputABI `json:"inputs"`
}

type ABI interface {
	EventsForContract(name string) []EventABI
}

type abi struct {
	events map[string][]EventABI
}

func LoadABI(config *Config, basePath string) (ABI, error) {
	eventsMap := make(map[string][]EventABI)

	for _, contract := range config.Contracts {
		abiData, err := os.ReadFile(path.Join(basePath, contract.ABI))
		if err != nil {
			return nil, fmt.Errorf("failed to read ABI file: %s, err: %v", contract.ABI, err)
		}

		var events []EventABI
		if err := json.Unmarshal(abiData, &events); err != nil {
			return nil, fmt.Errorf("failed to decode ABI file: %s, err: %v", contract.ABI, err)
		}
		eventsMap[contract.Name] = events
	}

	return &abi{eventsMap}, nil
}

func (a abi) EventsForContract(name string) []EventABI {
	return a.events[name]
}
