package migrations

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestRunner provides helpers for migration tests.
type TestRunner struct {
	pool          *pgxpool.Pool
	ctx           context.Context
	migrationsDir string
	t             *testing.T
}

// NewTestRunner creates a test runner with a test database.
func NewTestRunner(t *testing.T) *TestRunner {
	ctx := context.Background()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set; skipping migration tests")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping test database: %v", err)
	}

	// Create temp migrations directory
	tmpDir, err := os.MkdirTemp("", "migrations_test_")
	if err != nil {
		pool.Close()
		t.Fatalf("create temp directory: %v", err)
	}

	return &TestRunner{
		pool:          pool,
		ctx:           ctx,
		migrationsDir: tmpDir,
		t:             t,
	}
}

// Close cleans up test resources.
func (tr *TestRunner) Close() {
	// Clean up schema_migrations table
	_, _ = tr.pool.Exec(tr.ctx, "DROP TABLE IF EXISTS schema_migrations")
	tr.pool.Close()
	os.RemoveAll(tr.migrationsDir)
}

// CreateMigration creates a test migration file.
func (tr *TestRunner) CreateMigration(name, upSQL, downSQL string) {
	upPath := filepath.Join(tr.migrationsDir, name+".up.sql")
	if err := os.WriteFile(upPath, []byte(upSQL), 0644); err != nil {
		tr.t.Fatalf("write %s: %v", upPath, err)
	}

	downPath := filepath.Join(tr.migrationsDir, name+".down.sql")
	if err := os.WriteFile(downPath, []byte(downSQL), 0644); err != nil {
		tr.t.Fatalf("write %s: %v", downPath, err)
	}
}

// TestDiscoverMigrations verifies that discovery finds all migration files.
func TestDiscoverMigrations(t *testing.T) {
	tr := NewTestRunner(t)
	defer tr.Close()

	// Create migrations table first
	if _, err := tr.pool.Exec(tr.ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id SERIAL PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		t.Fatalf("create schema_migrations: %v", err)
	}

	// Create test migrations
	tr.CreateMigration(
		"000001_test",
		"CREATE TABLE test1 (id INT)",
		"DROP TABLE test1",
	)
	tr.CreateMigration(
		"000002_test",
		"CREATE TABLE test2 (id INT)",
		"DROP TABLE test2",
	)

	runner := New(tr.pool, tr.ctx)
	discovered, err := runner.discoverMigrations(tr.migrationsDir)
	if err != nil {
		t.Fatalf("discover migrations: %v", err)
	}

	if len(discovered) != 2 {
		t.Fatalf("expected 2 migrations, got %d", len(discovered))
	}

	if discovered[0] != "000001_test" || discovered[1] != "000002_test" {
		t.Fatalf("unexpected migration order: %v", discovered)
	}
}

// TestApplyMigration verifies that a migration is applied and recorded.
func TestApplyMigration(t *testing.T) {
	tr := NewTestRunner(t)
	defer tr.Close()

	// Create schema_migrations table
	if _, err := tr.pool.Exec(tr.ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id SERIAL PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		t.Fatalf("create schema_migrations: %v", err)
	}

	// Create test migration
	tr.CreateMigration(
		"000001_test",
		"CREATE TABLE test_table (id INT PRIMARY KEY)",
		"DROP TABLE test_table",
	)

	runner := New(tr.pool, tr.ctx)

	// Apply migration
	if err := runner.ApplyMigration(tr.migrationsDir, "000001_test"); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	// Verify table was created
	var exists bool
	err := tr.pool.QueryRow(tr.ctx, `
		SELECT EXISTS(
			SELECT 1 FROM information_schema.tables
			WHERE table_name = 'test_table'
		)
	`).Scan(&exists)
	if err != nil {
		t.Fatalf("check table exists: %v", err)
	}
	if !exists {
		t.Fatal("test_table was not created")
	}

	// Verify migration was recorded
	applied, err := runner.GetAppliedMigrations()
	if err != nil {
		t.Fatalf("get applied migrations: %v", err)
	}
	if len(applied) != 1 || applied[0] != "000001_test" {
		t.Fatalf("expected 1 applied migration, got %v", applied)
	}
}

// TestRevertMigration verifies that a migration can be reverted.
func TestRevertMigration(t *testing.T) {
	tr := NewTestRunner(t)
	defer tr.Close()

	// Create schema_migrations table
	if _, err := tr.pool.Exec(tr.ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id SERIAL PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		t.Fatalf("create schema_migrations: %v", err)
	}

	// Create test migration
	tr.CreateMigration(
		"000001_test",
		"CREATE TABLE test_table (id INT PRIMARY KEY)",
		"DROP TABLE test_table",
	)

	runner := New(tr.pool, tr.ctx)

	// Apply migration
	if err := runner.ApplyMigration(tr.migrationsDir, "000001_test"); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	// Revert migration
	if err := runner.RevertMigration(tr.migrationsDir, "000001_test"); err != nil {
		t.Fatalf("revert migration: %v", err)
	}

	// Verify table was dropped
	var exists bool
	err := tr.pool.QueryRow(tr.ctx, `
		SELECT EXISTS(
			SELECT 1 FROM information_schema.tables
			WHERE table_name = 'test_table'
		)
	`).Scan(&exists)
	if err != nil {
		t.Fatalf("check table exists: %v", err)
	}
	if exists {
		t.Fatal("test_table still exists after revert")
	}

	// Verify migration was removed
	applied, err := runner.GetAppliedMigrations()
	if err != nil {
		t.Fatalf("get applied migrations: %v", err)
	}
	if len(applied) != 0 {
		t.Fatalf("expected 0 applied migrations, got %v", applied)
	}
}

// TestApplyAll verifies that all pending migrations are applied in order.
func TestApplyAll(t *testing.T) {
	tr := NewTestRunner(t)
	defer tr.Close()

	// Create schema_migrations table
	if _, err := tr.pool.Exec(tr.ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id SERIAL PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		t.Fatalf("create schema_migrations: %v", err)
	}

	// Create test migrations
	tr.CreateMigration(
		"000001_test",
		"CREATE TABLE test1 (id INT PRIMARY KEY)",
		"DROP TABLE test1",
	)
	tr.CreateMigration(
		"000002_test",
		"CREATE TABLE test2 (id INT PRIMARY KEY)",
		"DROP TABLE test2",
	)

	runner := New(tr.pool, tr.ctx)

	// Apply all
	if err := runner.ApplyAll(tr.migrationsDir); err != nil {
		t.Fatalf("apply all: %v", err)
	}

	// Verify both tables exist
	tables := []string{"test1", "test2"}
	for _, table := range tables {
		var exists bool
		err := tr.pool.QueryRow(tr.ctx, fmt.Sprintf(`
			SELECT EXISTS(
				SELECT 1 FROM information_schema.tables
				WHERE table_name = '%s'
			)
		`, table)).Scan(&exists)
		if err != nil {
			t.Fatalf("check table %s exists: %v", table, err)
		}
		if !exists {
			t.Fatalf("table %s was not created", table)
		}
	}

	// Verify both migrations were recorded
	applied, err := runner.GetAppliedMigrations()
	if err != nil {
		t.Fatalf("get applied migrations: %v", err)
	}
	if len(applied) != 2 {
		t.Fatalf("expected 2 applied migrations, got %v", applied)
	}
}

// TestGetPending verifies that pending migrations are correctly identified.
func TestGetPending(t *testing.T) {
	tr := NewTestRunner(t)
	defer tr.Close()

	// Create schema_migrations table
	if _, err := tr.pool.Exec(tr.ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id SERIAL PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		t.Fatalf("create schema_migrations: %v", err)
	}

	// Create test migrations
	tr.CreateMigration("000001_test", "CREATE TABLE test1 (id INT)", "DROP TABLE test1")
	tr.CreateMigration("000002_test", "CREATE TABLE test2 (id INT)", "DROP TABLE test2")
	tr.CreateMigration("000003_test", "CREATE TABLE test3 (id INT)", "DROP TABLE test3")

	runner := New(tr.pool, tr.ctx)

	// Apply first two
	if err := runner.ApplyMigration(tr.migrationsDir, "000001_test"); err != nil {
		t.Fatalf("apply 000001: %v", err)
	}
	if err := runner.ApplyMigration(tr.migrationsDir, "000002_test"); err != nil {
		t.Fatalf("apply 000002: %v", err)
	}

	// Check pending
	pending, err := runner.GetPendingMigrations(tr.migrationsDir)
	if err != nil {
		t.Fatalf("get pending: %v", err)
	}

	if len(pending) != 1 || pending[0] != "000003_test" {
		t.Fatalf("expected 1 pending migration (000003_test), got %v", pending)
	}
}
