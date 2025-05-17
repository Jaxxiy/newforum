package logger

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

var (
	logger *zap.Logger
	once   sync.Once
	isDev  bool
)

func SetDevelopmentMode(dev bool) {
	isDev = dev
	logger = nil
}

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

func GetLogger() *zap.Logger {
	once.Do(func() {
		logger = InitLogger()
	})
	return logger
}

// Helper functions for creating zap fields
func String(key string, value string) zap.Field {
	return zap.String(key, value)
}

func Int(key string, value int) zap.Field {
	return zap.Int(key, value)
}

func Error(err error) zap.Field {
	return zap.Error(err)
}

func Bool(key string, value bool) zap.Field {
	return zap.Bool(key, value)
}

func Duration(key string, value time.Duration) zap.Field {
	return zap.Duration(key, value)
}

func Any(key string, value interface{}) zap.Field {
	return zap.Any(key, value)
}
