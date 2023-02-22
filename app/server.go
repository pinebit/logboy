package app

import (
	"context"
	"fmt"
	"net/http"

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
	s := &server{
		appContext,
		&http.Server{
			Addr: fmt.Sprintf(":%d", appContext.Config().Server.Port),
		},
	}
	return s
}

func (s *server) Run() error {
	logger := s.appContext.Logger("Server")
	logger.Debug("Starting HTTP server...")

	g, gctx := errgroup.WithContext(s.appContext.Context())
	g.Go(func() error {
		err := s.httpServer.ListenAndServe()
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	})
	g.Go(func() error {
		<-gctx.Done()
		err := s.httpServer.Shutdown(context.Background())
		logger.Debug("Stopped HTTP server")
		return err
	})

	return g.Wait()
}
