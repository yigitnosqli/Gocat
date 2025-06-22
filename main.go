package main

import (
	"github.com/ibrahmsql/gocat/cmd"
	"github.com/ibrahmsql/gocat/internal/logger"
)

func main() {
	// Setup logger
	logger.SetupLogger()

	if err := cmd.Execute(); err != nil {
		logger.Fatal("%v", err)
	}
}
