package rbac

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
)

// MigrationService handles database migrations for RBAC
type MigrationService struct {
	db            *sqlx.DB
	migrationsDir string
}

// Migration represents a database migration
type Migration struct {
	ID      int
	Name    string
	Content string
	Applied bool
}

// NewMigrationService creates a new migration service
func NewMigrationService(db *sqlx.DB, migrationsDir string) *MigrationService {
	return &MigrationService{
		db:            db,
		migrationsDir: migrationsDir,
	}
}

// InitMigrationTable creates the migrations tracking table
func (ms *MigrationService) InitMigrationTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id INTEGER PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE UNIQUE INDEX IF NOT EXISTS idx_schema_migrations_id ON schema_migrations(id);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_schema_migrations_name ON schema_migrations(name);
	`

	_, err := ms.db.Exec(query)
	return err
}

// GetPendingMigrations returns migrations that haven't been applied
func (ms *MigrationService) GetPendingMigrations() ([]*Migration, error) {
	// Read migration files from directory
	files, err := filepath.Glob(filepath.Join(ms.migrationsDir, "*.sql"))
	if err != nil {
		return nil, fmt.Errorf("failed to read migration files: %w", err)
	}

	// Parse migration files
	var allMigrations []*Migration
	for _, file := range files {
		migration, err := ms.parseMigrationFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to parse migration file %s: %w", file, err)
		}
		allMigrations = append(allMigrations, migration)
	}

	// Sort migrations by ID
	sort.Slice(allMigrations, func(i, j int) bool {
		return allMigrations[i].ID < allMigrations[j].ID
	})

	// Get applied migrations
	appliedMigrations, err := ms.getAppliedMigrations()
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	appliedMap := make(map[int]bool)
	for _, migration := range appliedMigrations {
		appliedMap[migration.ID] = true
	}

	// Filter out applied migrations
	var pending []*Migration
	for _, migration := range allMigrations {
		if !appliedMap[migration.ID] {
			pending = append(pending, migration)
		}
	}

	return pending, nil
}

// ApplyMigrations applies all pending migrations
func (ms *MigrationService) ApplyMigrations() error {
	// Initialize migration table
	if err := ms.InitMigrationTable(); err != nil {
		return fmt.Errorf("failed to initialize migration table: %w", err)
	}

	// Get pending migrations
	migrations, err := ms.GetPendingMigrations()
	if err != nil {
		return fmt.Errorf("failed to get pending migrations: %w", err)
	}

	if len(migrations) == 0 {
		fmt.Println("No pending migrations to apply")
		return nil
	}

	fmt.Printf("Applying %d migrations...\n", len(migrations))

	// Apply each migration in a transaction
	for _, migration := range migrations {
		err := ms.applyMigration(migration)
		if err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migration.Name, err)
		}
		fmt.Printf("Applied migration: %s\n", migration.Name)
	}

	fmt.Println("All migrations applied successfully")
	return nil
}

// applyMigration applies a single migration
func (ms *MigrationService) applyMigration(migration *Migration) error {
	tx, err := ms.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration SQL
	_, err = tx.Exec(migration.Content)
	if err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record migration as applied
	_, err = tx.Exec(
		"INSERT INTO schema_migrations (id, name) VALUES ($1, $2)",
		migration.ID, migration.Name,
	)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}

// parseMigrationFile parses a migration file and extracts ID, name, and content
func (ms *MigrationService) parseMigrationFile(filePath string) (*Migration, error) {
	// Extract filename without extension
	filename := filepath.Base(filePath)
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Parse migration ID from filename (e.g., "001_create_rbac_schema" -> ID: 1)
	parts := strings.SplitN(name, "_", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid migration filename format: %s", filename)
	}

	id, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid migration ID in filename: %s", filename)
	}

	// Read file content
	content, err := os.ReadFile(filePath) // #nosec G304 -- filePath is constructed from admin-configured migrations dir, not user input
	if err != nil {
		return nil, fmt.Errorf("failed to read migration file: %w", err)
	}

	return &Migration{
		ID:      id,
		Name:    name,
		Content: string(content),
	}, nil
}

// getAppliedMigrations returns migrations that have been applied
func (ms *MigrationService) getAppliedMigrations() ([]*Migration, error) {
	var migrations []*Migration

	query := "SELECT id, name FROM schema_migrations ORDER BY id"
	rows, err := ms.db.Query(query)
	if err != nil {
		// If table doesn't exist, return empty list
		if strings.Contains(err.Error(), "does not exist") {
			return migrations, nil
		}
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var migration Migration
		err := rows.Scan(&migration.ID, &migration.Name)
		if err != nil {
			return nil, err
		}
		migration.Applied = true
		migrations = append(migrations, &migration)
	}

	return migrations, rows.Err()
}

// GetMigrationStatus returns the status of all migrations
func (ms *MigrationService) GetMigrationStatus() ([]*Migration, error) {
	// Get all migration files
	files, err := filepath.Glob(filepath.Join(ms.migrationsDir, "*.sql"))
	if err != nil {
		return nil, fmt.Errorf("failed to read migration files: %w", err)
	}

	var allMigrations []*Migration
	for _, file := range files {
		migration, err := ms.parseMigrationFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to parse migration file %s: %w", file, err)
		}
		allMigrations = append(allMigrations, migration)
	}

	// Get applied migrations
	appliedMigrations, err := ms.getAppliedMigrations()
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	appliedMap := make(map[int]bool)
	for _, migration := range appliedMigrations {
		appliedMap[migration.ID] = true
	}

	// Mark applied status
	for _, migration := range allMigrations {
		migration.Applied = appliedMap[migration.ID]
	}

	// Sort by ID
	sort.Slice(allMigrations, func(i, j int) bool {
		return allMigrations[i].ID < allMigrations[j].ID
	})

	return allMigrations, nil
}

// RollbackMigration rolls back the last applied migration
func (ms *MigrationService) RollbackMigration() error {
	// Get the last applied migration
	query := "SELECT id, name FROM schema_migrations ORDER BY id DESC LIMIT 1"
	var migration Migration
	err := ms.db.Get(&migration, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no migrations to rollback")
		}
		return fmt.Errorf("failed to get last migration: %w", err)
	}

	// Check if rollback file exists
	rollbackFile := filepath.Join(ms.migrationsDir, fmt.Sprintf("%03d_%s_rollback.sql", migration.ID, strings.TrimPrefix(migration.Name, fmt.Sprintf("%03d_", migration.ID))))
	content, err := os.ReadFile(rollbackFile) // #nosec G304 -- rollbackFile is constructed from admin-configured migrations dir
	if err != nil {
		return fmt.Errorf("rollback file not found for migration %s: %w", migration.Name, err)
	}

	// Apply rollback in transaction
	tx, err := ms.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute rollback SQL
	_, err = tx.Exec(string(content))
	if err != nil {
		return fmt.Errorf("failed to execute rollback SQL: %w", err)
	}

	// Remove migration record
	_, err = tx.Exec("DELETE FROM schema_migrations WHERE id = $1", migration.ID)
	if err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit rollback: %w", err)
	}

	fmt.Printf("Rolled back migration: %s\n", migration.Name)
	return nil
}
