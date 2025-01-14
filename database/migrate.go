package database

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
)

// Migration represents a database migration
type Migration struct {
	Version  int
	Filename string
	SQL      string
}

// LoadMigrations loads all SQL migration files from the migrations directory
func LoadMigrations(migrationsDir string) ([]Migration, error) {
	files, err := ioutil.ReadDir(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %v", err)
	}

	var migrations []Migration
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			content, err := ioutil.ReadFile(filepath.Join(migrationsDir, file.Name()))
			if err != nil {
				return nil, fmt.Errorf("failed to read migration file %s: %v", file.Name(), err)
			}

			var version int
			_, err = fmt.Sscanf(file.Name(), "%d_", &version)
			if err != nil {
				return nil, fmt.Errorf("invalid migration filename %s: %v", file.Name(), err)
			}

			migrations = append(migrations, Migration{
				Version:  version,
				Filename: file.Name(),
				SQL:      string(content),
			})
		}
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// Migrate applies all pending migrations
func (db *DB) Migrate(migrationsDir string) error {
	// Create migrations table if it doesn't exist
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %v", err)
	}

	// Load migrations
	migrations, err := LoadMigrations(migrationsDir)
	if err != nil {
		return err
	}

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Get applied migrations
	rows, err := tx.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %v", err)
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return fmt.Errorf("failed to scan migration version: %v", err)
		}
		applied[version] = true
	}

	// Apply pending migrations
	for _, migration := range migrations {
		if !applied[migration.Version] {
			// Apply migration
			if _, err := tx.Exec(migration.SQL); err != nil {
				return fmt.Errorf("failed to apply migration %s: %v", migration.Filename, err)
			}

			// Record migration
			if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", migration.Version); err != nil {
				return fmt.Errorf("failed to record migration %s: %v", migration.Filename, err)
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}
