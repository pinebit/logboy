package types

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type Contract interface {
	ChainName() string
	Name() string
	ABI() *abi.ABI
	Addresses() []common.Address
	IsEventAllowed(name string) bool
}

type ContractsPerChain map[string][]Contract

type contract struct {
	chainName     string
	name          string
	abi           *abi.ABI
	addresses     []common.Address
	allowedEvents map[string]struct{}
}

func NewContract(chainName, contractName string, abi *abi.ABI, addresses []common.Address, allowedEvents map[string]struct{}) Contract {
	return &contract{
		chainName:     chainName,
		name:          contractName,
		abi:           abi,
		addresses:     addresses,
		allowedEvents: allowedEvents,
	}
}

func (c contract) Addresses() []common.Address {
	return c.addresses
}

func (c contract) IsEventAllowed(name string) bool {
	if len(c.allowedEvents) == 0 {
		return true
	}
	_, exists := c.allowedEvents[name]
	return exists
}

func (c contract) ABI() *abi.ABI {
	return c.abi
}

func (c contract) Name() string {
	return c.name
}

func (c contract) ChainName() string {
	return c.chainName
}
