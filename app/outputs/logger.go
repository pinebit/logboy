package outputs

import (
	"github.com/pinebit/lognite/app/types"
	"go.uber.org/zap"
)

type loggerOutput struct {
	logger *zap.SugaredLogger
}

func NewLoggerOutput(logger *zap.SugaredLogger) types.Output {
	return &loggerOutput{
		logger: logger,
	}
}

func (o loggerOutput) Write(event *types.Event) {
	var kv []interface{}

	kv = append(kv, ".chainName", event.Contract.ChainName())
	kv = append(kv, ".contractName", event.Contract.Name())
	kv = append(kv, ".contractAddress", event.Address)
	kv = append(kv, ".eventName", event.EventName)
	kv = append(kv, ".blockTs", event.BlockTs)
	kv = append(kv, ".blockNumber", event.BlockNumber)
	kv = append(kv, ".blockHash", event.BlockHash)
	kv = append(kv, ".txHash", event.TxHash)
	kv = append(kv, ".txIndex", event.TxIndex)
	kv = append(kv, ".logIndex", event.LogIndex)

	for ak, av := range event.EventArgs {
		kv = append(kv, ak, av)
	}

	o.logger.Infow("Event", kv...)
}
