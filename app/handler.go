package app

import (
	"context"

	"github.com/ethereum/go-ethereum/core/types"
	"go.uber.org/zap"
)

type LogHandler interface {
	// Must be re-entrant & thread-safe
	Handle(ctx context.Context, rpc RPC, log types.Log, contract Contract)
}

type logHandler struct {
	logger *zap.SugaredLogger
}

func NewLogHandler(logger *zap.SugaredLogger) LogHandler {
	return &logHandler{
		logger: logger,
	}
}

func (h logHandler) Handle(ctx context.Context, rpc RPC, log types.Log, contract Contract) {
	h.logger.Debugw("Log", "connection", rpc.Name(), "address", log.Address, "contract", contract.Name(), "blockNumber", log.BlockNumber)
}
