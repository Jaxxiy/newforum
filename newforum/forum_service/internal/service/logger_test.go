package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestLoggerInitialization(t *testing.T) {
	logger = nil

	log := InitLogger()
	assert.NotNil(t, log)

	assert.IsType(t, &zap.Logger{}, log)
}

func TestLoggerSingleton(t *testing.T) {
	logger = nil

	log1 := GetLogger()
	log2 := GetLogger()

	assert.Same(t, log1, log2)
}

func TestLoggerMethods(t *testing.T) {
	logger = nil

	log := InitLogger()
	assert.NotNil(t, log)

	assert.NotPanics(t, func() {
		log.Info("test info message")
		log.Error("test error message")
		log.Debug("test debug message")
		log.Warn("test warning message")
	})
}

func TestLoggerWithFields(t *testing.T) {
	logger = nil

	log := InitLogger()
	assert.NotNil(t, log)

	assert.NotPanics(t, func() {
		log.With(
			zap.String("key1", "value1"),
			zap.Int("key2", 123),
		).Info("test message with fields")
	})
}

func TestLoggerDevelopmentMode(t *testing.T) {
	logger = nil

	SetDevelopmentMode(true)
	defer SetDevelopmentMode(false)

	log := InitLogger()
	assert.NotNil(t, log)

	assert.NotPanics(t, func() {
		log.Debug("debug message should be enabled in development mode")
	})
}

func TestLoggerProductionMode(t *testing.T) {
	logger = nil

	SetDevelopmentMode(false)

	log := InitLogger()
	assert.NotNil(t, log)

	assert.NotPanics(t, func() {
		log.Info("info message in production mode")
	})
}

func TestLoggerWithContext(t *testing.T) {
	logger = nil

	log := InitLogger()
	assert.NotNil(t, log)

	assert.NotPanics(t, func() {
		log.With(
			zap.String("request_id", "123"),
			zap.String("user_id", "456"),
		).Info("test message with context")
	})
}

func TestLoggerErrorHandling(t *testing.T) {
	logger = nil

	log := InitLogger()
	assert.NotNil(t, log)

	assert.NotPanics(t, func() {
		log.Error("test error",
			zap.Error(assert.AnError),
			zap.String("additional_info", "test info"),
		)
	})
}
