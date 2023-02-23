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

type RPC interface {
	Name() string
	URL() string
	RunLoop(ctx context.Context)
}

type shared struct {
	logger     *zap.SugaredLogger
	handler    LogHandler
	addresses  []common.Address
	addressMap map[common.Address]Contract
}

type rpc struct {
	name   string
	url    string
	shared *shared
}

var (
	promConnections = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lognite_rpc_alive_connections",
		Help: "The current number of alive RPC connections",
	}, []string{"connection"})

	promReConnections = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_rpc_reconnections",
		Help: "The total number of RPC reconnections",
	}, []string{"connection"})

	promLogsReceived = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_logs_received",
		Help: "The total number of received logs per connection",
	}, []string{"connection"})
)

func NewRPCs(config *Config, logger *zap.SugaredLogger, contracts []Contract, handler LogHandler) []RPC {
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

	var rpcs []RPC
	for name, url := range config.RPC {
		rpc := &rpc{
			name:   name,
			url:    url,
			shared: shared,
		}
		rpcs = append(rpcs, rpc)
	}
	return rpcs
}

func (r rpc) Name() string {
	return r.name
}

func (r rpc) URL() string {
	return r.url
}

func (r rpc) RunLoop(ctx context.Context) {
	logger := r.shared.logger.Named(r.name)

	for {
		promReConnections.WithLabelValues(r.name).Inc()

		client, err := ethclient.DialContext(ctx, r.url)
		if err != nil {
			logger.Errorw("Failed to connect RPC", "url", r.url)
		} else {
			promConnections.WithLabelValues(r.name).Inc()

			func() {
				q := ethereum.FilterQuery{
					Addresses: r.shared.addresses,
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
						promLogsReceived.WithLabelValues(r.name).Inc()
						contract := r.shared.addressMap[log.Address]
						r.shared.handler.Handle(ctx, r, log, contract)
					}
				}
			}()
		}

		client.Close()
		promConnections.WithLabelValues(r.name).Dec()

		if errors.Is(ctx.Err(), context.Canceled) {
			return
		}

		time.Sleep(time.Second)
	}
}
