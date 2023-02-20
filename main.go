package main

import (
	"fmt"

	"github.com/pinebit/smart-contract-monitor/obry"
	"go.uber.org/zap"
)

func main() {
	zapLogger, _ := zap.NewProduction()
	defer zapLogger.Sync()
	logger := zapLogger.Sugar()

	logger.Debug("Reading config...")
	config, err := obry.LoadConfigJSON("config/config.json")
	if err != nil {
		logger.Fatalf("Failed to read config from JSON: %v", err)
	}
	fmt.Println(config)
}
