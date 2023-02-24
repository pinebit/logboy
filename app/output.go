package app

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/core/types"
	"go.uber.org/zap"
)

type Output interface {
	Write(ctx context.Context, log types.Log, contract Contract, event string, args map[string]interface{})
}

type Outputs interface {
	Add(output Output)
	GetAll() []Output
}

type outputs struct {
	queue []Output
	lock  sync.RWMutex
}

func NewOutputs() Outputs {
	return &outputs{}
}

func (o *outputs) Add(output Output) {
	o.lock.Lock()
	defer o.lock.Unlock()
	o.queue = append(o.queue, output)
}

func (o *outputs) GetAll() []Output {
	o.lock.RLock()
	defer o.lock.RUnlock()
	return o.queue
}

type loggerOutput struct {
	logger *zap.SugaredLogger
}

func NewLoggerOutput(logger *zap.SugaredLogger) Output {
	return &loggerOutput{
		logger: logger,
	}
}

func (o loggerOutput) Write(ctx context.Context, log types.Log, contract Contract, event string, args map[string]interface{}) {
	var logKeysAndValues []interface{}
	logKeysAndValues = append(logKeysAndValues, ".chainName", contract.Chain())
	logKeysAndValues = append(logKeysAndValues, ".contractName", contract.Name())
	logKeysAndValues = append(logKeysAndValues, ".contractAddress", log.Address)
	logKeysAndValues = append(logKeysAndValues, ".name", event)
	logKeysAndValues = append(logKeysAndValues, ".removed", log.Removed)

	for ak, av := range args {
		logKeysAndValues = append(logKeysAndValues, ak, av)
	}

	o.logger.Infow("Event", logKeysAndValues...)
}
