package app

import (
	"context"
	"errors"
	"fmt"
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
	name            string
	rpc             string
	contracts       []types.Contract
	addresses       []ethcommon.Address
	addressMap      map[ethcommon.Address]types.Contract
	logger          *zap.SugaredLogger
	outputs         types.Outputs
	confirmations   uint
	lastBlockNumber uint64
	lastBlockHash   ethcommon.Hash
}

var (
	zeroHash = ethcommon.HexToHash("0x0")
)

func NewChain(chainName string, config ChainConfig, contracts []types.Contract, logger *zap.SugaredLogger, outputs types.Outputs) Chain {
	var addresses []ethcommon.Address
	addressMap := make(map[ethcommon.Address]types.Contract)

	for _, contract := range contracts {
		for _, address := range contract.Addresses() {
			addresses = append(addresses, address)
			addressMap[address] = contract
		}
	}

	return &chain{
		name:          chainName,
		logger:        logger.Named(chainName),
		rpc:           config.RPC,
		contracts:     contracts,
		addresses:     addresses,
		addressMap:    addressMap,
		outputs:       outputs,
		confirmations: config.Confirmations,
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

func (c *chain) receiveLoop(ctx context.Context, client *ethclient.Client) {
	headersCh := make(chan *ethtypes.Header)
	sub, err := client.SubscribeNewHead(ctx, headersCh)
	if err != nil {
		c.logger.Errorw("Call to SubscribeNewHead failed, will reconnect", "err", err)
		return
	}
	defer sub.Unsubscribe()

	var stopAtBlockNumber uint64
	timer := time.NewTimer(common.DefaultBackfillInterval)

	for {
		select {
		case <-ctx.Done():
			return
		case subErr := <-sub.Err():
			c.logger.Errorw("Logs subscription error, will reconnect", "err", subErr)
			return
		case header := <-headersCh:
			stopAtBlockNumber = header.Number.Uint64() - uint64(c.confirmations)
			if c.lastBlockNumber == 0 {
				c.lastBlockNumber = stopAtBlockNumber - 1
			}
		case <-timer.C:
			if stopAtBlockNumber > c.lastBlockNumber {
				nextBlockNumber := c.lastBlockNumber + 1
				if err := c.getBlockLogs(ctx, client, nextBlockNumber); err != nil {
					c.logger.Errorw("Failed pulling logs for block, will reconnect", "blockNumber", nextBlockNumber, "err", err)
					return
				}
			}
		}
		if stopAtBlockNumber > c.lastBlockNumber {
			timer.Reset(common.DefaultBackfillInterval)
		}
	}
}

func (c *chain) getBlockLogs(ctx context.Context, client *ethclient.Client, blockNumber uint64) error {
	bigBlockNumber := big.NewInt(int64(blockNumber))
	header, err := client.HeaderByNumber(ctx, bigBlockNumber)
	if err != nil {
		return fmt.Errorf("call to HeaderByNumber failed: %v", err)
	}
	if c.lastBlockHash != zeroHash && header.ParentHash != c.lastBlockHash {
		common.PromReorgErrors.WithLabelValues(c.name).Inc()
		return fmt.Errorf("block parent hash mismatch, likely due to large reorg")
	}
	if c.lastBlockNumber != header.Number.Uint64()-1 {
		common.PromReorgErrors.WithLabelValues(c.name).Inc()
		return fmt.Errorf("block number is not in-order, likely due to large reorg")
	}
	logs, err := client.FilterLogs(ctx, ethereum.FilterQuery{
		Addresses: c.addresses,
		FromBlock: bigBlockNumber,
		ToBlock:   bigBlockNumber,
	})
	if err != nil {
		return fmt.Errorf("call to FilterLogs failed: %v", err)
	}
	for _, log := range logs {
		if log.Removed {
			common.PromReorgErrors.WithLabelValues(c.name).Inc()
			c.logger.Errorw("Ignoring unexpected removed log", "tx_hash", log.TxHash, "tx_index", log.TxIndex)
		} else {
			c.decodeAndOutputLog(&log, header.Time)
		}
	}
	c.lastBlockNumber = blockNumber
	c.lastBlockHash = header.Hash()
	return nil
}

func (c chain) decodeAndOutputLog(log *ethtypes.Log, timestamp uint64) {
	contract := c.addressMap[log.Address]
	common.PromLogsReceived.WithLabelValues(c.name, contract.Name()).Inc()
	blockTs := time.Unix(int64(timestamp), 0)
	event, err := decodeEvent(blockTs, log, contract)
	if err != nil {
		common.PromEventsMalformed.WithLabelValues(c.name, contract.Name()).Inc()
		c.logger.Errorw("Failed to decode event", "err", err)
	} else if event != nil {
		common.PromEvents.WithLabelValues(c.name, contract.Name(), event.EventName).Inc()
		c.outputs.Write(event)
	}
}
