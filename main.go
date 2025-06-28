package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ibrahmsql/gocat/cmd"
	"github.com/ibrahmsql/gocat/internal/logger"
)

func main() {
	// Setup logger
	logger.SetupLogger()
	logger.SetLevel(logger.LevelInfo)

	// Create context that cancels on interrupt signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Received interrupt signal, shutting down gracefully...")
		cancel()
	}()

	// Execute the command
	if err := cmd.ExecuteContext(ctx); err != nil {
		logger.Fatal("Application error: %v", err)
		os.Exit(1)
	}
}
