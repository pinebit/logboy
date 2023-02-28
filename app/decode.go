package app

import (
	"time"

	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/pinebit/lognite/app/types"
)

func decodeEvent(blockTs time.Time, log *ethtypes.Log, contract types.Contract) (*types.Event, error) {
	abi := contract.ABI()
	event, err := abi.EventByID(log.Topics[0])
	if err != nil {
		return nil, err
	}
	if !contract.IsEventAllowed(event.Name) {
		return nil, nil
	}

	args, err := parseArgumentValues(log, abi, event)
	if err != nil {
		return nil, err
	} else {
		eventData := &types.Event{
			EventName:   event.Name,
			EventArgs:   args,
			Contract:    contract,
			Address:     log.Address,
			BlockTs:     blockTs,
			BlockNumber: log.BlockNumber,
			BlockHash:   log.BlockHash,
			TxHash:      log.TxHash,
			TxIndex:     log.TxIndex,
			LogIndex:    log.Index,
			LogRemoved:  log.Removed,
		}
		return eventData, nil
	}
}

func parseArgumentValues(log *ethtypes.Log, abi *ethabi.ABI, event *ethabi.Event) (map[string]interface{}, error) {
	dataValues := make(map[string]interface{})
	if err := abi.UnpackIntoMap(dataValues, event.Name, log.Data); err != nil {
		return nil, err
	}

	allValues := make(map[string]interface{})
	indexedArgs := indexedArguments(event.Inputs)
	if err := ethabi.ParseTopicsIntoMap(allValues, indexedArgs, log.Topics[1:]); err != nil {
		return nil, err
	}

	for k, v := range dataValues {
		allValues[k] = v
	}

	hexifyRawBytes(allValues)

	return allValues, nil
}

func indexedArguments(args ethabi.Arguments) ethabi.Arguments {
	var indexed ethabi.Arguments
	for _, arg := range args {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	return indexed
}

func hexifyRawBytes(kv map[string]interface{}) {
	for k, v := range kv {
		switch val := v.(type) {
		case []byte:
			kv[k] = hexutil.Encode(val)
		case [32]byte:
			kv[k] = hexutil.Encode(val[:])
		}
	}
}
