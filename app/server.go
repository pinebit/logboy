package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
)

type Server interface {
	Run() error
}

type server struct {
	appContext AppContext
	httpServer *http.Server
}

func NewServer(appContext AppContext) Server {
	return &server{
		appContext,
		&http.Server{
			Addr: fmt.Sprintf(":%d", appContext.Config().Server.Port),
		},
	}
}

func (s *server) Run() error {
	logger := s.appContext.Logger("server")
	g, gctx := errgroup.WithContext(s.appContext.Context())

	g.Go(func() error {
		logger.Debugf("Listening on port %s", s.httpServer.Addr)

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
		logger.Debug("HTTP server stopped")
		return err
	})

	return g.Wait()
}
