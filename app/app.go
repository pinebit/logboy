package app

import (
	"context"
	"path"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type App interface {
	Run()
	Close()
}

type AppContext interface {
	Context() context.Context
	Logger(name string) *zap.SugaredLogger
	Config() *Config
	Contracts() []Contract
}

type app struct {
	ctx       context.Context
	logger    *zap.SugaredLogger
	config    *Config
	contracts []Contract
}

func NewApp(configPath string) App {
	zapLogger, _ := zap.NewDevelopment()
	logger := zapLogger.Sugar()

	logger.Debugf("Loading config from %s...", configPath)
	config, err := LoadConfigJSON(configPath)
	if err != nil {
		logger.Fatalf("Failed to read config from JSON: %v", err)
	}

	logger.Debug("Configuring contracts...")
	contracts, err := LoadContracts(config, path.Dir(configPath))
	if err != nil {
		logger.Fatalf("Failed to configure contracts: %v", err)
	}

	return &app{
		logger:    logger,
		config:    config,
		contracts: contracts,
	}
}

func (a app) Context() context.Context {
	return a.ctx
}

func (a app) Logger(name string) *zap.SugaredLogger {
	return a.logger.Named(name)
}

func (a app) Config() *Config {
	return a.config
}

func (a app) Contracts() []Contract {
	return a.contracts
}

func (a app) Close() {
	a.logger.Sync()
}

func (a *app) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	go ShutdownHandler(cancel)

	g, gctx := errgroup.WithContext(ctx)
	a.ctx = gctx

	handler := NewLogHandler(a.logger.Named("handler"))
	rpcs := NewRPCs(a.config, a.logger.Named("rpc"), a.contracts, handler)

	for _, rpc := range rpcs {
		rpc := rpc
		g.Go(func() error {
			rpc.RunLoop(gctx)
			return nil
		})
	}

	s := NewServer(a)
	g.Go(s.Run)

	if err := g.Wait(); err != nil {
		a.logger.Fatalf("Application error: %v", err)
	}
}
