package app

import (
	"sync"

	"go.uber.org/zap"
)

type Output interface {
	Write(event *Event)
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

func (o loggerOutput) Write(event *Event) {
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
