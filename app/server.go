package app

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/pinebit/lognite/app/types"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type Server interface {
	types.Service
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

func (s *server) Run(ctx context.Context, done func()) {
	defer done()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		http.Handle("/metrics", promhttp.Handler())
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("ok"))
		})

		if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
			s.logger.Errorw("HTTP server error", "err", err)
		}
	}()

	go func() {
		defer wg.Done()

		<-ctx.Done()
		if err := s.httpServer.Shutdown(context.Background()); err != nil {
			s.logger.Errorw("HTTP server error", "err", err)
		}
	}()

	s.logger.Debugf("Listening on port %s", s.httpServer.Addr)
	wg.Wait()
}
