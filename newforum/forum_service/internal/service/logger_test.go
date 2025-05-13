package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestInitLogger(t *testing.T) {
	// Reset logger before test
	Logger = nil

	// Test initialization
	InitLogger()

	// Verify logger was created
	assert.NotNil(t, Logger, "Logger should not be nil after initialization")
	assert.IsType(t, &zap.Logger{}, Logger, "Logger should be of type *zap.Logger")
}

func TestGetLogger(t *testing.T) {
	// Reset and initialize logger
	Logger = nil
	InitLogger()

	// Test getting logger
	logger := GetLogger()

	// Verify returned logger
	assert.NotNil(t, logger, "GetLogger should not return nil")
	assert.Equal(t, Logger, logger, "GetLogger should return the initialized logger")
}

func TestLoggerSingleton(t *testing.T) {
	// Reset and initialize logger
	Logger = nil
	InitLogger()

	// Get logger multiple times
	logger1 := GetLogger()
	logger2 := GetLogger()

	// Verify same instance is returned
	assert.Equal(t, logger1, logger2, "Multiple calls to GetLogger should return the same instance")
}
