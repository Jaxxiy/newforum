package service

import (
	"sync"

	"go.uber.org/zap"
)

var (
	logger *zap.Logger
	once   sync.Once
	isDev  bool
)

// SetDevelopmentMode sets whether the logger should be in development mode
func SetDevelopmentMode(dev bool) {
	isDev = dev
	// Reset logger so it will be recreated with new mode
	logger = nil
}

// InitLogger initializes the logger
func InitLogger() *zap.Logger {
	var err error
	var log *zap.Logger

	if isDev {
		log, err = zap.NewDevelopment()
	} else {
		log, err = zap.NewProduction()
	}

	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}

	return log
}

// GetLogger returns the singleton logger instance
func GetLogger() *zap.Logger {
	once.Do(func() {
		logger = InitLogger()
	})
	return logger
}
