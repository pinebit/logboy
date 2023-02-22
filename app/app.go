package app

import (
	"context"

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
}

type app struct {
	ctx    context.Context
	logger *zap.SugaredLogger
	config *Config
}

func NewApp() App {
	zapLogger, _ := zap.NewDevelopment()
	logger := zapLogger.Sugar()

	logger.Debug("Loading config from config/config.json...")
	config, err := LoadConfigJSON("config/config.json")
	if err != nil {
		logger.Fatalf("Failed to read config from JSON: %v", err)
	}

	return &app{
		logger: logger,
		config: config,
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
