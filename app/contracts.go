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
				abiData, err := os.ReadFile(abiFilePath)
				if err != nil {
					return nil, fmt.Errorf("failed to read ABI file: %s, err: %v", contractConfig.ABI, err)
				}

				abi, err := ethabi.JSON(strings.NewReader(string(abiData)))
				if err != nil {
					return nil, fmt.Errorf("failed to decode ABI: %s, err: %v", contractConfig.ABI, err)
				}

				abiCache[abiFilePath] = &abi
			}

			abi := abiCache[abiFilePath]
			addresses := contractConfig.Addresses
			if contractConfig.Address != ethcommon.HexToAddress("0x00") {
				addresses = append(addresses, contractConfig.Address)
			}

			common.PromConfiguredEvents.WithLabelValues(chainName, contractName).Add(float64(len(abi.Events)))
			common.PromConfiguredAddresses.WithLabelValues(chainName, contractName).Add(float64(len(addresses)))

			newContract := types.NewContract(chainName, contractName, abi, addresses)
			contracts[chainName] = append(contracts[chainName], newContract)
		}
	}

	return contracts, nil
}
