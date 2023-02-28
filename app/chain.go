package app

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jpillora/backoff"
	"github.com/pinebit/lognite/app/common"
	"github.com/pinebit/lognite/app/types"
	"go.uber.org/zap"
)

type Chain interface {
	types.Service
}

type chain struct {
	name       string
	rpc        string
	contracts  []types.Contract
	addresses  []ethcommon.Address
	addressMap map[ethcommon.Address]types.Contract
	logger     *zap.SugaredLogger
	outputs    types.Outputs
}

func NewChain(name, rpc string, contracts []types.Contract, logger *zap.SugaredLogger, outputs types.Outputs) Chain {
	var addresses []ethcommon.Address
	addressMap := make(map[ethcommon.Address]types.Contract)

	for _, contract := range contracts {
		for _, address := range contract.Addresses() {
			addresses = append(addresses, address)
			addressMap[address] = contract
		}
	}

	return &chain{
		name:       name,
		logger:     logger.Named(name),
		rpc:        rpc,
		contracts:  contracts,
		addresses:  addresses,
		addressMap: addressMap,
		outputs:    outputs,
	}
}

func (c chain) Run(ctx context.Context, done func()) {
	defer done()

	backoff := backoff.Backoff{}

	for {
		common.PromReConnections.WithLabelValues(c.name).Inc()

		client, err := ethclient.DialContext(ctx, c.rpc)
		if err != nil {
			c.logger.Errorw("Failed to connect RPC", "url", c.rpc)
		} else {
			c.logger.Debugw("RPC connected", "url", c.rpc)
			common.PromConnections.WithLabelValues(c.name).Inc()
			backoff.Reset()

			c.receiveLoop(ctx, client)
		}

		client.Close()
		common.PromConnections.WithLabelValues(c.name).Dec()

		if errors.Is(ctx.Err(), context.Canceled) {
			return
		}

		time.Sleep(backoff.Duration())
	}
}

func (c chain) receiveLoop(ctx context.Context, client *ethclient.Client) {
	headers := NewHeaders()
	filterQuery := ethereum.FilterQuery{
		Addresses: c.addresses,
	}
	logsCh := make(chan ethtypes.Log)
	sub, err := client.SubscribeFilterLogs(ctx, filterQuery, logsCh)
	if err != nil {
		c.logger.Errorw("Failed to subscribe to logs", "err", err)
		return
	}
	defer sub.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return
		case subErr := <-sub.Err():
			c.logger.Errorw("Subscription error", "err", subErr)
			return
		case log := <-logsCh:
			lastHeader := headers.FindHeaderByNumber(log.BlockNumber)
			if lastHeader == nil {
				header, err := client.HeaderByNumber(ctx, big.NewInt(int64(log.BlockNumber)))
				if err != nil {
					c.logger.Errorw("Failed to get HeaderByNumber", "err", err)
					return
				}
				headers.AddHeader(header)
				lastHeader = header
			}
			contract := c.addressMap[log.Address]
			common.PromLogsReceived.WithLabelValues(c.name, contract.Name()).Inc()
			blockTs := time.Unix(lastHeader.Number.Int64(), 0)
			event, err := decodeEvent(blockTs, &log, contract)
			if err != nil {
				c.logger.Errorw("Failed to parse event", "err", err)
			} else if event != nil {
				common.PromEvents.WithLabelValues(c.name, contract.Name(), event.EventName).Inc()

				for _, output := range c.outputs {
					output.Write(event)
				}
			}
		}
	}
}
