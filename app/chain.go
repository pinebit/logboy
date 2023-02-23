package app

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

type Chain interface {
	Name() string
	RunLoop(ctx context.Context)
}

type shared struct {
	logger     *zap.SugaredLogger
	handler    LogHandler
	addresses  []common.Address
	addressMap map[common.Address]Contract
}

type chain struct {
	name   string
	rpc    string
	shared *shared
}

var (
	promConnections = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lognite_rpc_alive_connections",
		Help: "The current number of alive RPC connections per chain",
	}, []string{"chainName"})

	promReConnections = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_rpc_reconnections",
		Help: "The total number of RPC reconnections per chain",
	}, []string{"chainName"})

	promLogsReceived = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_logs_received",
		Help: "The total number of received logs per chain",
	}, []string{"chainName"})
)

func NewChains(config *Config, logger *zap.SugaredLogger, contracts []Contract, handler LogHandler) []Chain {
	var addresses []common.Address
	addressMap := make(map[common.Address]Contract)

	for _, contract := range contracts {
		for _, address := range contract.Addresses() {
			addresses = append(addresses, address)
			addressMap[address] = contract
		}
	}

	shared := &shared{
		logger:     logger,
		handler:    handler,
		addresses:  addresses,
		addressMap: addressMap,
	}

	var chains []Chain
	for chainName, chainConfig := range config.Chains {
		chain := &chain{
			name:   chainName,
			rpc:    chainConfig.RPC,
			shared: shared,
		}
		chains = append(chains, chain)
	}
	return chains
}

func (c chain) Name() string {
	return c.name
}

func (c chain) URL() string {
	return c.rpc
}

func (c chain) RunLoop(ctx context.Context) {
	logger := c.shared.logger.Named(c.name)

	for {
		promReConnections.WithLabelValues(c.name).Inc()

		client, err := ethclient.DialContext(ctx, c.rpc)
		if err != nil {
			logger.Errorw("Failed to connect RPC", "url", c.rpc)
		} else {
			logger.Debugw("RPC connected", "url", c.rpc)
			promConnections.WithLabelValues(c.name).Inc()

			func() {
				q := ethereum.FilterQuery{
					Addresses: c.shared.addresses,
				}
				logsCh := make(chan types.Log)

				sub, err := client.SubscribeFilterLogs(ctx, q, logsCh)
				if err != nil {
					logger.Errorw("Failed to subscribe to logs", "err", err)
					return
				}
				defer sub.Unsubscribe()

				for {
					select {
					case <-ctx.Done():
						return
					case subErr := <-sub.Err():
						logger.Errorw("Subscription error", "err", subErr)
						return
					case log := <-logsCh:
						promLogsReceived.WithLabelValues(c.name).Inc()
						contract := c.shared.addressMap[log.Address]
						c.shared.handler.Handle(ctx, c, log, contract)
					}
				}
			}()
		}

		client.Close()
		promConnections.WithLabelValues(c.name).Dec()

		if errors.Is(ctx.Err(), context.Canceled) {
			return
		}

		time.Sleep(time.Second)
	}
}
