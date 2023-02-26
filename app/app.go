package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"

	"go.uber.org/zap"

	_ "github.com/joho/godotenv/autoload"
	out "github.com/pinebit/lognite/app/outputs"
	"github.com/pinebit/lognite/app/types"
)

type App interface {
	Start() error
}

type app struct {
	configPath string
	logger     *zap.SugaredLogger
}

func NewApp(configPath string) App {
	zapLogger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	return &app{
		configPath: configPath,
		logger:     zapLogger.Sugar(),
	}
}

func (a *app) Start() error {
	defer a.logger.Sync()

	a.logger.Debugf("Loading config from %s...", a.configPath)
	config, err := LoadConfigJSON(a.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	a.logger.Debug("Configuring contracts...")
	contracts, err := LoadContracts(config, path.Dir(a.configPath))
	if err != nil {
		return fmt.Errorf("failed to configure contracts: %v", err)
	}

	rootCtx, cancel := context.WithCancel(context.Background())
	go shutdownHandler(cancel)

	var outputServices []types.Service
	var outputs types.Outputs
	if config.Outputs.Console == nil || !config.Outputs.Console.Disabled {
		outputs = append(outputs, out.NewLoggerOutput(a.logger))
	}

	if config.Outputs.Postgres != nil {
		pg := out.NewPostgres(a.logger, config.Outputs.Postgres.Retention)
		if err := pg.Connect(rootCtx, config.Outputs.Postgres.URL); err != nil {
			return fmt.Errorf("failed to connect Postgres: url=%s", config.Outputs.Postgres.URL)
		}
		defer pg.Close(rootCtx)

		if err := pg.MigrateSchema(rootCtx, contracts); err != nil {
			return fmt.Errorf("failed to migrate postgres schema: %v", err)
		}

		outputServices = append(outputServices, pg)
		outputs = append(outputs, pg)
	}

	var chainServices []types.Service
	for chainName, chainContracts := range contracts {
		chain := NewChain(chainName, config, chainContracts, a.logger, outputs)
		chainServices = append(chainServices, chain)
	}

	server := NewServer(&config.Server, a.logger)

	// Boot: output -> chain -> server
	a.logger.Debug("Starting services...")
	stopOutputServices := startServices(outputServices)
	stopChainServices := startServices(chainServices)
	stopHttpServer := startServices([]types.Service{server})

	<-rootCtx.Done()

	// Shutdown: server -> chain -> output
	a.logger.Debug("Stopping services...")
	stopHttpServer()
	stopChainServices()
	stopOutputServices()

	return nil
}

func startServices(services []types.Service) (stop func()) {
	var wg sync.WaitGroup
	wg.Add(len(services))

	ctx, cancel := context.WithCancel(context.Background())
	for _, service := range services {
		go service.Run(ctx, wg.Done)
	}

	return func() {
		cancel()
		wg.Wait()
	}
}

func shutdownHandler(cancelFunc func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	<-c
	cancelFunc()
}
