package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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

type Contracts interface {
	EventsForContract(name string) []EventABI
	AddressesForContract(name string) []common.Address
}

type contracts struct {
	events    map[string][]EventABI
	addresses map[string][]common.Address
}

var (
	promConfiguredEvents = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "obry_configured_events",
		Help: "The total number of events configured per contract",
	}, []string{"contractName"})

	promConfiguredAddresses = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "obry_configured_addresses",
		Help: "The total number of addresses configured per contract",
	}, []string{"contractName"})
)

func LoadContracts(config *Config, basePath string) (Contracts, error) {
	eventsMap := make(map[string][]EventABI)
	addressesMap := make(map[string][]common.Address)

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

		var addresses []common.Address
		for _, address := range contract.Addresses {
			if !common.IsHexAddress(address) {
				return nil, fmt.Errorf("failed to parse %s address: %s", contract.Name, address)
			}
			addresses = append(addresses, common.HexToAddress(address))
		}
		addressesMap[contract.Name] = addresses

		promConfiguredEvents.WithLabelValues(contract.Name).Add(float64(len(events)))
		promConfiguredAddresses.WithLabelValues(contract.Name).Add(float64(len(contract.Addresses)))
	}

	return &contracts{eventsMap, addressesMap}, nil
}

func (c contracts) EventsForContract(name string) []EventABI {
	return c.events[name]
}

func (c contracts) AddressesForContract(name string) []common.Address {
	return c.addresses[name]
}
