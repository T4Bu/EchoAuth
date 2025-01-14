package database

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestInitDB_Unit(t *testing.T) {
	tests := []struct {
		name        string
		dbURL       string
		wantErr     bool
		errContains string
	}{
		{
			name:        "Empty URL",
			dbURL:       "",
			wantErr:     true,
			errContains: "database URL cannot be empty",
		},
		{
			name:        "Invalid URL",
			dbURL:       "invalid-url",
			wantErr:     true,
			errContains: "failed to connect to database",
		},
		{
			name:        "Valid URL format but no server",
			dbURL:       "host=localhost port=5432 dbname=test",
			wantErr:     true,
			errContains: "failed to connect to database",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := InitDB(tt.dbURL)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, db)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, db)
		})
	}
}

func TestMigrate_Unit(t *testing.T) {
	// Create mock database
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()

	db := &DB{mockDB}

	// Expect migrations table check
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Expect query for applied migrations
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT version FROM schema_migrations").
		WillReturnRows(sqlmock.NewRows([]string{"version"}))

	// Since we can't actually read files in unit tests, we'll verify that
	// the function handles empty migrations directory gracefully
	err = db.Migrate("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read migrations directory")

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInitDB_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Use environment variables or configuration for these values in a real application
	testDBURL := "host=localhost user=test_user password=test_pass dbname=test_db sslmode=disable"

	db, err := InitDB(testDBURL)
	if err != nil {
		t.Skipf("Skipping integration test, could not connect to database: %v", err)
		return
	}

	assert.NotNil(t, db)

	// Test the connection
	assert.NoError(t, db.Ping())

	// Clean up
	db.Close()
}
