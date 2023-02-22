package app

import "github.com/ethereum/go-ethereum/common"

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

type Contract interface {
	Name() string
	Events() []EventABI
	Addresses() []common.Address
}

type contract struct {
	name      string
	events    []EventABI
	addresses []common.Address
}

func NewContract(name string, events []EventABI, addresses []common.Address) Contract {
	return &contract{
		name:      name,
		events:    events,
		addresses: addresses,
	}
}

func (c contract) Addresses() []common.Address {
	return c.addresses
}

func (c contract) Events() []EventABI {
	return c.events
}

func (c contract) Name() string {
	return c.name
}
