package app

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

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

func LoadContracts(config *Config, basePath string) ([]Contract, error) {
	var contracts []Contract

	for _, contractConfig := range config.Contracts {
		abiData, err := os.ReadFile(path.Join(basePath, contractConfig.ABI))
		if err != nil {
			return nil, fmt.Errorf("failed to read ABI file: %s, err: %v", contractConfig.ABI, err)
		}

		abi, err := abi.JSON(strings.NewReader(string(abiData)))
		if err != nil {
			return nil, fmt.Errorf("failed to decode ABI: %s, err: %v", contractConfig.ABI, err)
		}

		var addresses []common.Address
		for _, address := range contractConfig.Addresses {
			if !common.IsHexAddress(address) {
				return nil, fmt.Errorf("failed to parse %s address: %s", contractConfig.Name, address)
			}
			addresses = append(addresses, common.HexToAddress(address))
		}

		promConfiguredEvents.WithLabelValues(contractConfig.Name).Add(float64(len(abi.Events)))
		promConfiguredAddresses.WithLabelValues(contractConfig.Name).Add(float64(len(contractConfig.Addresses)))

		contracts = append(contracts, NewContract(contractConfig.Name, abi, addresses))
	}

	return contracts, nil
}
