package database

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestLoadMigrations(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "migrations")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test migration files
	files := map[string]string{
		"001_create_users.sql":      "CREATE TABLE users (id SERIAL PRIMARY KEY);",
		"002_create_tokens.sql":     "CREATE TABLE tokens (id UUID PRIMARY KEY);",
		"003_add_user_columns.sql":  "ALTER TABLE users ADD COLUMN email VARCHAR(255);",
		"invalid_migration.sql":     "INVALID SQL",
		"not_a_migration.txt":       "NOT A MIGRATION",
		"004_add_token_columns.sql": "ALTER TABLE tokens ADD COLUMN user_id INTEGER;",
		"005_add_foreign_keys.sql":  "ALTER TABLE tokens ADD FOREIGN KEY (user_id) REFERENCES users(id);",
	}

	for filename, content := range files {
		err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
		assert.NoError(t, err)
	}

	// Test loading migrations
	migrations, err := LoadMigrations(tempDir)
	assert.NoError(t, err)

	// Verify migrations are loaded and sorted correctly
	assert.Len(t, migrations, 5)
	assert.Equal(t, 1, migrations[0].Version)
	assert.Equal(t, 2, migrations[1].Version)
	assert.Equal(t, 3, migrations[2].Version)
	assert.Equal(t, 4, migrations[3].Version)
	assert.Equal(t, 5, migrations[4].Version)

	// Verify migration content
	assert.Equal(t, "CREATE TABLE users (id SERIAL PRIMARY KEY);", migrations[0].SQL)
}

func TestMigrate(t *testing.T) {
	// Create mock database
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()

	db := &DB{mockDB}

	// Create temporary directory with test migrations
	tempDir, err := os.MkdirTemp("", "migrations")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test migration files
	files := map[string]string{
		"001_create_users.sql":  "CREATE TABLE users (id SERIAL PRIMARY KEY);",
		"002_create_tokens.sql": "CREATE TABLE tokens (id UUID PRIMARY KEY);",
	}

	for filename, content := range files {
		err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
		assert.NoError(t, err)
	}

	// Expect migrations table creation
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Expect transaction for migrations
	mock.ExpectBegin()

	// Expect query for applied migrations
	mock.ExpectQuery("SELECT version FROM schema_migrations").
		WillReturnRows(sqlmock.NewRows([]string{"version"}))

	// Expect first migration
	mock.ExpectExec("CREATE TABLE users").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO schema_migrations").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Expect second migration
	mock.ExpectExec("CREATE TABLE tokens").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO schema_migrations").
		WithArgs(2).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Expect transaction commit
	mock.ExpectCommit()

	// Run migrations
	err = db.Migrate(tempDir)
	assert.NoError(t, err)

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMigrate_WithExistingMigrations(t *testing.T) {
	// Create mock database
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()

	db := &DB{mockDB}

	// Create temporary directory with test migrations
	tempDir, err := os.MkdirTemp("", "migrations")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test migration files
	files := map[string]string{
		"001_create_users.sql":  "CREATE TABLE users (id SERIAL PRIMARY KEY);",
		"002_create_tokens.sql": "CREATE TABLE tokens (id UUID PRIMARY KEY);",
		"003_add_columns.sql":   "ALTER TABLE users ADD COLUMN email VARCHAR(255);",
	}

	for filename, content := range files {
		err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
		assert.NoError(t, err)
	}

	// Expect migrations table creation
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Expect transaction for migrations
	mock.ExpectBegin()

	// Expect query for applied migrations (return some existing migrations)
	mock.ExpectQuery("SELECT version FROM schema_migrations").
		WillReturnRows(sqlmock.NewRows([]string{"version"}).
			AddRow(1).
			AddRow(2))

	// Expect only the third migration (others are already applied)
	mock.ExpectExec("ALTER TABLE users").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO schema_migrations").
		WithArgs(3).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Expect transaction commit
	mock.ExpectCommit()

	// Run migrations
	err = db.Migrate(tempDir)
	assert.NoError(t, err)

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}
