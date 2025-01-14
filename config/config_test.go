package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Test default values
	cfg := LoadConfig()
	if cfg.Port != "8080" {
		t.Errorf("Expected default port 8080, got %s", cfg.Port)
	}

	// Test environment variable override
	os.Setenv("PORT", "3000")
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("DATABASE_URL", "test-db-url")

	cfg = LoadConfig()
	if cfg.Port != "3000" {
		t.Errorf("Expected port 3000, got %s", cfg.Port)
	}
	if cfg.JWTSecret != "test-secret" {
		t.Errorf("Expected JWT secret test-secret, got %s", cfg.JWTSecret)
	}
	if cfg.DatabaseURL != "test-db-url" {
		t.Errorf("Expected database URL test-db-url, got %s", cfg.DatabaseURL)
	}

	// Cleanup
	os.Unsetenv("PORT")
	os.Unsetenv("JWT_SECRET")
	os.Unsetenv("DATABASE_URL")
}
