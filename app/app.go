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
	zapLogger, _ := zap.NewProduction()
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

	pg, err := ConnectPostgres(ctx, UnwrapConfigEnvVar(a.config.Postgres.Conn))
	if err != nil {
		a.logger.Fatalw("Failed to connect Postgres", "url", UnwrapConfigEnvVar(a.config.Postgres.Conn))
	}
	defer pg.Close(ctx)

	for _, rpc := range rpcs {
		rpc := rpc

		if err := CreateSchema(ctx, pg, rpc.Name()); err != nil {
			a.logger.Fatalw("Failed to create DB schema", "rpc", rpc.Name(), "err", err)
		}

		for _, contract := range a.contracts {
			if err := CreateEventTable(ctx, pg, rpc.Name(), contract); err != nil {
				a.logger.Fatalw("Failed to create contract table", "err", err)
			}
		}

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
