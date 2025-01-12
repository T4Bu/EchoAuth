package logger

import (
	"os"
	"testing"
)

func TestInit(t *testing.T) {
	// Test default initialization
	Init()

	// Test with environment variable
	os.Setenv("LOG_LEVEL", "debug")
	Init()

	// Cleanup
	os.Unsetenv("LOG_LEVEL")
}

func TestGetLogger(t *testing.T) {
	Init()
	logger := GetLogger("test")
	// Just verify we can write to the logger without panicking
	logger.Info().Msg("Test log message")
}
