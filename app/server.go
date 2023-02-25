package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type Server interface {
	Service
}

type server struct {
	logger     *zap.SugaredLogger
	httpServer *http.Server
}

func NewServer(config *ServerConfig, logger *zap.SugaredLogger) Server {
	return &server{
		logger: logger.Named("server"),
		httpServer: &http.Server{
			Addr: fmt.Sprintf(":%d", config.Port),
		},
	}
}

func (s *server) Run(ctx context.Context) error {
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		s.logger.Debugf("Listening on port %s", s.httpServer.Addr)

		http.Handle("/metrics", promhttp.Handler())
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("ok"))
		})

		err := s.httpServer.ListenAndServe()
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	})

	g.Go(func() error {
		<-gctx.Done()
		err := s.httpServer.Shutdown(context.Background())
		s.logger.Debug("HTTP server stopped")
		return err
	})

	return g.Wait()
}
