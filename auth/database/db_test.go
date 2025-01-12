package database

import (
	"os"
	"strings"
	"testing"
)

func TestInitDB_Unit(t *testing.T) {
	// Test with invalid URL - this is a unit test that doesn't need a real DB
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected InitDB to panic with invalid URL")
		} else {
			// Verify the panic message contains expected error
			panicMsg, ok := r.(string)
			if !ok || !strings.Contains(panicMsg, "Failed to connect to database") {
				t.Errorf("Expected panic message to contain 'Failed to connect to database', got %v", r)
			}
		}
	}()

	InitDB("invalid-url")
	t.Error("Expected InitDB to panic, but it didn't")
}

func TestInitDB_Integration(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	// Use test database for integration tests
	db, err := InitDB("host=localhost user=postgres password=postgres dbname=auth_test_db port=5433 sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying *sql.DB: %v", err)
	}
	defer sqlDB.Close()

	// Verify connection
	err = sqlDB.Ping()
	if err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}
}
