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
	ABI() ABI
}

type app struct {
	ctx    context.Context
	logger *zap.SugaredLogger
	config *Config
	abi    ABI
}

func NewApp(configPath string) App {
	zapLogger, _ := zap.NewDevelopment()
	logger := zapLogger.Sugar()

	logger.Debugf("Loading config from %s...", configPath)
	config, err := LoadConfigJSON(configPath)
	if err != nil {
		logger.Fatalf("Failed to read config from JSON: %v", err)
	}

	logger.Debug("Loading ABIs...")
	abi, err := LoadABI(config, path.Dir(configPath))
	if err != nil {
		logger.Fatalf("Failed to load ABI: %v", err)
	}

	return &app{
		logger: logger,
		config: config,
		abi:    abi,
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

func (a app) ABI() ABI {
	return a.abi
}

func (a app) Close() {
	a.logger.Sync()
}

func (a *app) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	go ShutdownHandler(cancel)

	g, gctx := errgroup.WithContext(ctx)
	a.ctx = gctx

	s := NewServer(a)
	g.Go(s.Run)

	if err := g.Wait(); err != nil {
		a.logger.Fatalf("Application error: %v", err)
	}
}
