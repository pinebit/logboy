package types

import (
	"github.com/ethereum/go-ethereum/common"
)

type Event struct {
	EventName string
	EventArgs map[string]interface{}
	Contract  Contract

	Address     common.Address
	BlockNumber uint64
	BlockHash   common.Hash
	TxHash      common.Hash
	TxIndex     uint
	LogIndex    uint
	LogRemoved  bool
}
