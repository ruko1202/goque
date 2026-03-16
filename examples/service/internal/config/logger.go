package config

import (
	"log"

	"github.com/ruko1202/xlog"
	"go.uber.org/zap"
)

// InitLogger .
//
//nolint:gocritic
func InitLogger() xlog.Logger {
	logger, err := zap.NewProduction(
		zap.AddCaller(),
		zap.AddCallerSkip(2),
	)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	return xlog.NewZapAdapter(logger)
}
