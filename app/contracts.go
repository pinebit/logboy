package app

import (
	"fmt"
	"os"
	"path"
	"strings"

	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/pinebit/lognite/app/common"
	"github.com/pinebit/lognite/app/types"
)

func LoadContracts(config *Config, basePath string) (types.ContractsPerChain, error) {
	contracts := make(types.ContractsPerChain)
	abiCache := make(map[string]*ethabi.ABI)

	for chainName, chainConfig := range config.Chains {
		for contractName, contractConfig := range chainConfig.Contracts {
			abiFilePath := path.Join(basePath, contractConfig.ABI)
			if _, exists := abiCache[abiFilePath]; !exists {
				abi, err := readABI(abiFilePath)
				if err != nil {
					return nil, err
				}
				abiCache[abiFilePath] = abi
			}

			abi := abiCache[abiFilePath]
			addresses := contractConfig.Addresses
			if contractConfig.Address != ethcommon.HexToAddress("0x00") {
				addresses = append(addresses, contractConfig.Address)
			}

			common.PromConfiguredEvents.WithLabelValues(chainName, contractName).Add(float64(len(abi.Events)))
			common.PromConfiguredAddresses.WithLabelValues(chainName, contractName).Add(float64(len(addresses)))

			allowedEvents := make(map[string]struct{})
			for _, eventName := range contractConfig.Events {
				allowedEvents[eventName] = struct{}{}
			}

			newContract := types.NewContract(chainName, contractName, abi, addresses, allowedEvents)
			contracts[chainName] = append(contracts[chainName], newContract)
		}
	}

	return contracts, nil
}

func readABI(filepath string) (*ethabi.ABI, error) {
	abiData, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read ABI file: %s, err: %v", filepath, err)
	}

	abi, err := ethabi.JSON(strings.NewReader(string(abiData)))
	if err != nil {
		return nil, fmt.Errorf("failed to decode ABI: %s, err: %v", filepath, err)
	}

	return &abi, nil
}
