package app

import (
	"os"
	"os/signal"
	"syscall"
)

func ShutdownHandler(cancelFunc func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	<-c
	cancelFunc()
}
