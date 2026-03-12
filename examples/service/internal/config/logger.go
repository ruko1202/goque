package config

import (
	"log"
)

func InitLogger() *xfield.Logger {
	logger, err := xfield.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	return logger
}
