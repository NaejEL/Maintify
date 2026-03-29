package rbac

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func TestMigrationService_InitMigrationTable(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	ms := NewMigrationService(sqlxDB, "/tmp")

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = ms.InitMigrationTable()
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMigrationService_GetPendingMigrations(t *testing.T) {
	// Create temp directory for migrations
	tmpDir, err := os.MkdirTemp("", "migrations")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create dummy migration files
	files := map[string]string{
		"001_init.sql":   "CREATE TABLE test (id int);",
		"002_update.sql": "ALTER TABLE test ADD COLUMN name varchar(255);",
	}

	for name, content := range files {
		err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	ms := NewMigrationService(sqlxDB, tmpDir)

	// Mock getAppliedMigrations
	rows := sqlmock.NewRows([]string{"id", "name"}).
		AddRow(1, "001_init")
	mock.ExpectQuery("SELECT id, name FROM schema_migrations ORDER BY id").
		WillReturnRows(rows)

	pending, err := ms.GetPendingMigrations()
	assert.NoError(t, err)
	assert.Len(t, pending, 1)
	assert.Equal(t, 2, pending[0].ID)
	assert.Equal(t, "002_update", pending[0].Name)
	assert.Equal(t, "ALTER TABLE test ADD COLUMN name varchar(255);", pending[0].Content)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMigrationService_ApplyMigrations(t *testing.T) {
	// Create temp directory for migrations
	tmpDir, err := os.MkdirTemp("", "migrations")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create dummy migration file
	err = os.WriteFile(filepath.Join(tmpDir, "001_init.sql"), []byte("CREATE TABLE test (id int);"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	ms := NewMigrationService(sqlxDB, tmpDir)

	// Expect InitMigrationTable
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Expect GetPendingMigrations -> getAppliedMigrations
	// Return empty rows (no applied migrations)
	mock.ExpectQuery("SELECT id, name FROM schema_migrations ORDER BY id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}))

	// Expect applyMigration transaction
	mock.ExpectBegin()
	mock.ExpectExec("CREATE TABLE test \\(id int\\);").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO schema_migrations").
		WithArgs(1, "001_init").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = ms.ApplyMigrations()
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMigrationService_ApplyMigrations_NoPending(t *testing.T) {
	// Create temp directory for migrations
	tmpDir, err := os.MkdirTemp("", "migrations")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	ms := NewMigrationService(sqlxDB, tmpDir)

	// Expect InitMigrationTable
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Expect GetPendingMigrations -> getAppliedMigrations
	mock.ExpectQuery("SELECT id, name FROM schema_migrations ORDER BY id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}))

	err = ms.ApplyMigrations()
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMigrationService_ParseMigrationFile_InvalidFilename(t *testing.T) {
	ms := NewMigrationService(nil, "")

	// Create a temp file with invalid name
	tmpFile, err := os.CreateTemp("", "invalid.sql")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	_, err = ms.parseMigrationFile(tmpFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid migration filename format")
}

func TestMigrationService_ParseMigrationFile_InvalidID(t *testing.T) {
	ms := NewMigrationService(nil, "")

	// Create a temp file with invalid ID
	tmpDir, err := os.MkdirTemp("", "migrations")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	filename := filepath.Join(tmpDir, "abc_test.sql")
	err = os.WriteFile(filename, []byte("test"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ms.parseMigrationFile(filename)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid migration ID")
}

func TestMigrationService_GetAppliedMigrations_TableDoesNotExist(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	ms := NewMigrationService(sqlxDB, "")

	mock.ExpectQuery("SELECT id, name FROM schema_migrations ORDER BY id").
		WillReturnError(fmt.Errorf("relation \"schema_migrations\" does not exist"))

	migrations, err := ms.getAppliedMigrations()
	assert.NoError(t, err)
	assert.Empty(t, migrations)
	assert.NoError(t, mock.ExpectationsWereMet())
}
