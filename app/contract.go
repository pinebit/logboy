package app

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type Contract interface {
	Chain() string
	Name() string
	ABI() abi.ABI
	Addresses() []common.Address
}

type contract struct {
	chain     string
	name      string
	abi       abi.ABI
	addresses []common.Address
}

func NewContract(chain, name string, abi abi.ABI, addresses []common.Address) Contract {
	return &contract{
		chain:     chain,
		name:      name,
		abi:       abi,
		addresses: addresses,
	}
}

func (c contract) Addresses() []common.Address {
	return c.addresses
}

func (c contract) ABI() abi.ABI {
	return c.abi
}

func (c contract) Name() string {
	return c.name
}

func (c contract) Chain() string {
	return c.chain
}
