package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

var (
	promHeadsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "scm_heads_count",
		Help: "Number of Heads received",
	})

	RPC_URL      = os.Getenv("RPC_URL")
	METRICS_PORT = os.Getenv("METRICS_PORT")
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

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(":"+METRICS_PORT, nil)
		if err != nil {
			logger.Fatalf("Failed to start metrics http server", "err", err)
		}
	}()

	logger.Info("Connecting to RPC...")
	client, err := ethclient.DialContext(ctx, RPC_URL)
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
			promHeadsCount.Inc()
		}
	}

	logger.Info("Service stopped.")
}
