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
	var logKeysAndValues []interface{}
	logKeysAndValues = append(logKeysAndValues, ".chainName", event.Contract.ChainName())
	logKeysAndValues = append(logKeysAndValues, ".contractName", event.Contract.Name())
	logKeysAndValues = append(logKeysAndValues, ".contractAddress", event.Log.Address)
	logKeysAndValues = append(logKeysAndValues, ".name", event.Name)
	logKeysAndValues = append(logKeysAndValues, ".removed", event.Log.Removed)

	for ak, av := range event.Args {
		logKeysAndValues = append(logKeysAndValues, ak, av)
	}

	o.logger.Infow("Event", logKeysAndValues...)
}
