package signals

import (
	"os"
	"os/signal"
	"syscall"
)

// BlockExitSignals blocks SIGINT and SIGTERM signals
func BlockExitSignals() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		// Block the signal, do nothing
	}()
}

// SetupSignalHandler sets up a signal handler that calls the provided function
func SetupSignalHandler(handler func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		handler()
	}()
}
