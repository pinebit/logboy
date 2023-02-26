package types

import "github.com/ethereum/go-ethereum/core/types"

type Event struct {
	Name     string
	Log      types.Log
	Contract Contract
	Args     map[string]interface{}
}

func NewEvent(name string, log types.Log, contract Contract, args map[string]interface{}) *Event {
	return &Event{
		Name:     name,
		Log:      log,
		Contract: contract,
		Args:     args,
	}
}
