package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

func main() {
	zapLogger, _ := zap.NewProduction()
	defer zapLogger.Sync()
	logger := zapLogger.Sugar()

	logger.Info("Starting service...")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

		sig := <-ch
		logger.Infof("Shutting down due to %s", sig)
		cancel()
	}()

	logger.Info("Connecting to RPC...")
	client, err := ethclient.DialContext(ctx, os.Getenv("RPC_URL"))
	if err != nil {
		logger.Fatalw("Failed to connect to RPC", "err", err)
	}

	logger.Info("Subscribing for heads...")
	headsCh := make(chan *types.Header)
	sub, err := client.SubscribeNewHead(ctx, headsCh)
	if err != nil {
		logger.Fatalw("Failed to SubscribeNewHead", "err", err)
	}
	defer sub.Unsubscribe()

mainLoop:
	for {
		select {
		case <-ctx.Done():
			break mainLoop
		case err := <-sub.Err():
			logger.Errorf("Head subscription error: %v", err)
			break mainLoop
		case header := <-headsCh:
			logger.Infof("Received header: %d", header.Number)
		}
	}

	logger.Info("Service stopped.")
}
