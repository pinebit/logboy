package app

import (
	"context"

	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

type LogHandler interface {
	// Must be re-entrant & thread-safe
	Handle(ctx context.Context, rpc RPC, log types.Log, contract Contract)
}

type logHandler struct {
	logger *zap.SugaredLogger
}

var (
	promEvents = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_events",
		Help: "The total number of events per contract, address and event name",
	}, []string{"rpc", "contractName", "contractAddress", "eventName"})
)

func NewLogHandler(logger *zap.SugaredLogger) LogHandler {
	return &logHandler{
		logger: logger,
	}
}

func (h logHandler) Handle(ctx context.Context, rpc RPC, log types.Log, contract Contract) {
	h.logger.Debugw("Log", "connection", rpc.Name(), "address", log.Address, "contract", contract.Name(), "tx", log.TxHash, "topics", len(log.Topics))

	abi := contract.ABI()
	event, err := abi.EventByID(log.Topics[0])
	if err == nil {
		v := make(map[string]interface{})
		if err := abi.UnpackIntoMap(v, event.Name, log.Data); err != nil {
			h.logger.Errorw("Failed to unpack event", "name", event.Name, "err", err)
		} else {
			var logKeysAndValues []interface{}
			logKeysAndValues = append(logKeysAndValues, ".contractName", contract.Name())
			logKeysAndValues = append(logKeysAndValues, ".contractAddress", log.Address)
			logKeysAndValues = append(logKeysAndValues, ".eventName", event.Name)
			logKeysAndValues = append(logKeysAndValues, ".removed", log.Removed)

			for i, input := range event.Inputs {
				if input.Indexed {
					if len(log.Topics) >= i+1 {
						var v interface{}
						topicData := log.Topics[i+1].Bytes()

						switch input.Type.T {
						case ethabi.AddressTy:
							v = common.BytesToAddress(topicData)
						case ethabi.HashTy:
							v = common.BytesToHash(topicData)
						default:
							h.logger.Errorw("Unsupported indexed type", "name", input.Name, "type", input.Type.String())
						}

						logKeysAndValues = append(logKeysAndValues, input.Name, v)
					} else {
						h.logger.Errorw("No topic for indexed input", "name", input.Name)
					}
				}
			}

			for k, v := range v {
				logKeysAndValues = append(logKeysAndValues, k, v)
			}

			h.logger.Infow("Event", logKeysAndValues...)

			promEvents.WithLabelValues(rpc.Name(), contract.Name(), log.Address.Hex(), event.Name).Inc()
		}
	}
}
