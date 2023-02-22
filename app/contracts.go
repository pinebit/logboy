package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

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
}

type contracts struct {
	events map[string][]EventABI
}

var (
	promConfiguredEvents = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "obry_configured_events",
		Help: "The total number of events loaded for configured contracts",
	}, []string{"contractName"})
)

func LoadContracts(config *Config, basePath string) (Contracts, error) {
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
		promConfiguredEvents.WithLabelValues(contract.Name).Add(float64(len(events)))
	}

	return &contracts{eventsMap}, nil
}

func (c contracts) EventsForContract(name string) []EventABI {
	return c.events[name]
}
