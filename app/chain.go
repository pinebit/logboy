package app

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum"
	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jpillora/backoff"
	"go.uber.org/zap"
)

type Chain interface {
	Service
}

type chain struct {
	name       string
	rpc        string
	contracts  []Contract
	addresses  []common.Address
	addressMap map[common.Address]Contract
	logger     *zap.SugaredLogger
	outputs    Outputs
}

func NewChain(name string, config *Config, contracts []Contract, logger *zap.SugaredLogger, outputs Outputs) Chain {
	var addresses []common.Address
	addressMap := make(map[common.Address]Contract)

	for _, contract := range contracts {
		for _, address := range contract.Addresses() {
			addresses = append(addresses, address)
			addressMap[address] = contract
		}
	}

	return &chain{
		name:       name,
		logger:     logger.Named(name),
		rpc:        config.Chains[name].RPC,
		contracts:  contracts,
		addresses:  addresses,
		addressMap: addressMap,
		outputs:    outputs,
	}
}

func (c chain) Run(ctx context.Context) error {
	backoff := backoff.Backoff{}

	for {
		promReConnections.WithLabelValues(c.name).Inc()

		client, err := ethclient.DialContext(ctx, c.rpc)
		if err != nil {
			c.logger.Errorw("Failed to connect RPC", "url", c.rpc)
		} else {
			c.logger.Debugw("RPC connected", "url", c.rpc)
			promConnections.WithLabelValues(c.name).Inc()
			backoff.Reset()

			func() {
				q := ethereum.FilterQuery{
					Addresses: c.addresses,
				}
				logsCh := make(chan types.Log)

				sub, err := client.SubscribeFilterLogs(ctx, q, logsCh)
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
						promLogsReceived.WithLabelValues(c.name).Inc()
						contract := c.addressMap[log.Address]
						c.handle(log, contract)
					}
				}
			}()
		}

		client.Close()
		promConnections.WithLabelValues(c.name).Dec()

		if errors.Is(ctx.Err(), context.Canceled) {
			return ctx.Err()
		}

		time.Sleep(backoff.Duration())
	}
}

func (c chain) handle(log types.Log, contract Contract) {
	c.logger.Debugw("Log", "chain", c.name, "address", log.Address, "contract", contract.Name(), "tx", log.TxHash)

	abi := contract.ABI()
	event, err := abi.EventByID(log.Topics[0])
	if err == nil {
		args, err := parseArgumentValues(log, abi, event.Inputs, event.Name, c.name, contract.Name())
		if err != nil {
			c.logger.Errorw("Failed to parse event", "name", event.Name, "err", err)
		} else {
			promEvents.WithLabelValues(c.name, contract.Name(), log.Address.Hex(), event.Name).Inc()

			eventData := NewEvent(event.Name, log, contract, args)
			for _, output := range c.outputs.GetAll() {
				output.Write(eventData)
			}
		}
	}
}

func parseArgumentValues(log types.Log, abi *ethabi.ABI, args ethabi.Arguments, logName, chainName, contractName string) (map[string]interface{}, error) {
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
