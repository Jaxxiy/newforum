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
