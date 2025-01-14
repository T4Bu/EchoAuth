package main

import (
	"EchoAuth/config"
	"EchoAuth/database"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setupTestDB(t *testing.T) *database.DB {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("Skipping test, DATABASE_URL not set")
		return nil
	}

	db, err := database.InitDB(dsn)
	if err != nil {
		t.Skipf("Skipping test, could not connect to database: %v", err)
		return nil
	}
	return db
}

func TestNewDependencies(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	cfg := &config.Config{
		JWTSecret:   "test-secret",
		JWTExpiry:   3600,
		Environment: "test",
	}

	deps := NewDependencies(db, nil, cfg)

	assert.NotNil(t, deps)
	assert.NotNil(t, deps.UserRepo)
	assert.NotNil(t, deps.TokenRepo)
	assert.NotNil(t, deps.AuthService)
	assert.Nil(t, deps.RedisClient) // Redis client is nil as expected
}
