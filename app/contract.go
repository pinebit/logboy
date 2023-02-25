package app

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type Contract interface {
	ChainName() string
	Name() string
	ABI() *abi.ABI
	Addresses() []common.Address
}

type contract struct {
	chainName string
	name      string
	abi       *abi.ABI
	addresses []common.Address
}

func NewContract(chainName, contractName string, abi *abi.ABI, addresses []common.Address) Contract {
	return &contract{
		chainName: chainName,
		name:      contractName,
		abi:       abi,
		addresses: addresses,
	}
}

func (c contract) Addresses() []common.Address {
	return c.addresses
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
