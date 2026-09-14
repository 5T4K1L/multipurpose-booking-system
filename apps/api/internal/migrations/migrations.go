package migrations

import (
	"fmt"
	"path/filepath"

	// Migration engine.
	"github.com/golang-migrate/migrate/v4"

	// PostgreSQL migration driver.
	_ "github.com/golang-migrate/migrate/v4/database/postgres"

	// Loads migrations from local files.
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Run applies all pending database migrations.
func Run(databaseURL string, migrationsPath string) error {
	// Convert the migration path to a clean relative path.
	normalizedPath := filepath.ToSlash(migrationsPath)

	// Load migrations from the local filesystem.
	sourceURL := "file://" + normalizedPath

	// Create the migration runner using PostgreSQL.
	migrator, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		return fmt.Errorf("create migration runner: %w", err)
	}
	defer migrator.Close()

	// Apply every migration that has not been applied yet.
	if err := migrator.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
