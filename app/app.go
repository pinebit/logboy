package app

import (
	"context"
	"errors"
	"path"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	_ "github.com/joho/godotenv/autoload"
)

type App interface {
	Start()
	Close()
}

type app struct {
	logger    *zap.SugaredLogger
	config    *Config
	contracts map[string][]Contract
}

func NewApp(configPath string) App {
	zapLogger, _ := zap.NewProduction()
	logger := zapLogger.Sugar()

	logger.Debugf("Loading config from %s...", configPath)
	config, err := LoadConfigJSON(configPath)
	if err != nil {
		logger.Fatalf("Failed to read config file: %v", err)
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

func (a app) Close() {
	a.logger.Sync()
}

func (a *app) Start() {
	var services []Service

	ctx, cancel := context.WithCancel(context.Background())
	go ShutdownHandler(cancel)

	g, gctx := errgroup.WithContext(ctx)

	outputs := NewOutputs()
	if a.config.Outputs.Console == nil || !a.config.Outputs.Console.Disabled {
		outputs.Add(NewLoggerOutput(a.logger))
	}

	if a.config.Outputs.Postgres != nil {
		db := NewDatabase(a.logger, a.config.Outputs.Postgres)
		if err := db.Connect(ctx, a.config.Outputs.Postgres.URL); err != nil {
			a.logger.Fatalw("Failed to connect Postgres", "url", a.config.Outputs.Postgres.URL)
		}
		defer db.Close(ctx)

		if err := db.MigrateSchema(ctx, a.contracts); err != nil {
			a.logger.Fatalw("Database.CreateSchemas failed", "err", err)
		}

		services = append(services, db)
		outputs.Add(db)
	}

	for chainName, chainContracts := range a.contracts {
		chain := NewChain(chainName, a.config, chainContracts, a.logger, outputs)
		services = append(services, chain)
	}

	server := NewServer(&a.config.Server, a.logger)
	services = append(services, server)

	for _, service := range services {
		service := service

		g.Go(func() error {
			return service.Run(gctx)
		})
	}

	if err := g.Wait(); err != nil {
		if !errors.Is(err, context.Canceled) {
			a.logger.Fatalw("Application error", "err", err)
		} else {
			a.logger.Debug("Application is stopped gracefully.")
		}
	}
}
