package app

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
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
	var lastBlockNumber uint64
	var lastBlockTs time.Time
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
			var blockTs time.Time
			if log.BlockNumber != lastBlockNumber {
				header, err := client.HeaderByNumber(ctx, big.NewInt(int64(log.BlockNumber)))
				if err != nil {
					c.logger.Errorw("Failed to get HeaderByNumber", "err", err)
					return
				}
				lastBlockNumber = log.BlockNumber
				lastBlockTs = time.Unix(int64(header.Time), 0)
			}
			blockTs = lastBlockTs

			common.PromLogsReceived.WithLabelValues(c.name).Inc()
			contract := c.addressMap[log.Address]
			c.handle(blockTs, &log, contract)
		}
	}
}

func (c chain) handle(blockTs time.Time, log *ethtypes.Log, contract types.Contract) {
	c.logger.Debugw("Log", "chain", c.name, "address", log.Address, "contract", contract.Name(), "tx", log.TxHash)

	abi := contract.ABI()
	event, err := abi.EventByID(log.Topics[0])
	if err == nil && contract.IsEventAllowed(event.Name) {
		args, err := parseArgumentValues(log, abi, event)
		if err != nil {
			c.logger.Errorw("Failed to parse event", "name", event.Name, "err", err)
		} else {
			common.PromEvents.WithLabelValues(c.name, contract.Name(), log.Address.Hex(), event.Name).Inc()

			eventData := &types.Event{
				EventName:   event.Name,
				EventArgs:   args,
				Contract:    contract,
				Address:     log.Address,
				BlockTs:     blockTs,
				BlockNumber: log.BlockNumber,
				BlockHash:   log.BlockHash,
				TxHash:      log.TxHash,
				TxIndex:     log.TxIndex,
				LogIndex:    log.Index,
				LogRemoved:  log.Removed,
			}

			for _, output := range c.outputs {
				output.Write(eventData)
			}
		}
	}
}

func parseArgumentValues(log *ethtypes.Log, abi *ethabi.ABI, event *ethabi.Event) (map[string]interface{}, error) {
	dataValues := make(map[string]interface{})
	if err := abi.UnpackIntoMap(dataValues, event.Name, log.Data); err != nil {
		return nil, err
	}

	allValues := make(map[string]interface{})
	indexedArgs := indexedArguments(event.Inputs)
	if err := ethabi.ParseTopicsIntoMap(allValues, indexedArgs, log.Topics[1:]); err != nil {
		return nil, err
	}

	for k, v := range dataValues {
		allValues[k] = v
	}

	return allValues, nil
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
