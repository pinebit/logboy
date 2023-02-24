package app

import (
	"context"

	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

type LogHandler interface {
	// Must be re-entrant & thread-safe
	Handle(ctx context.Context, chain Chain, log types.Log, contract Contract)
}

type logHandler struct {
	logger  *zap.SugaredLogger
	outputs Outputs
}

var (
	promEvents = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_events",
		Help: "The total number of events per contract, address and event name",
	}, []string{"chainName", "contractName", "contractAddress", "eventName"})
)

func NewLogHandler(logger *zap.SugaredLogger, outputs Outputs) LogHandler {
	return &logHandler{
		logger:  logger,
		outputs: outputs,
	}
}

func (h logHandler) Handle(ctx context.Context, chain Chain, log types.Log, contract Contract) {
	h.logger.Debugw("Log", "chain", chain.Name(), "address", log.Address, "contract", contract.Name(), "tx", log.TxHash)

	abi := contract.ABI()
	event, err := abi.EventByID(log.Topics[0])
	if err == nil {
		args, err := parseArgumentValues(log, abi, event.Inputs, event.Name, chain.Name(), contract.Name())
		if err != nil {
			h.logger.Errorw("Failed to parse event", "name", event.Name, "err", err)
		} else {
			promEvents.WithLabelValues(chain.Name(), contract.Name(), log.Address.Hex(), event.Name).Inc()

			for _, output := range h.outputs.GetAll() {
				output.Write(ctx, log, contract, event.Name, args)
			}
		}
	}
}

func parseArgumentValues(log types.Log, abi ethabi.ABI, args ethabi.Arguments, logName, chainName, contractName string) (map[string]interface{}, error) {
	dataValues := make(map[string]interface{})
	if err := abi.UnpackIntoMap(dataValues, logName, log.Data); err != nil {
		return nil, err
	}

	topicValues := make(map[string]interface{})
	indexedArgs := indexedArguments(args)
	if err := ethabi.ParseTopicsIntoMap(topicValues, indexedArgs, log.Topics[1:]); err != nil {
		return nil, err
	}

	for k, v := range dataValues {
		topicValues[k] = v
	}

	return topicValues, nil
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
