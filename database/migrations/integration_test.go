package migrations

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestIntegration_FullMigrationWorkflow tests the complete migration lifecycle.
func TestIntegration_FullMigrationWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set; skipping integration tests")
	}

	// Connect to test database
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	defer pool.Close()

	// Clean up before test
	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS schema_migrations")
	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS test_table1")
	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS test_table2")
	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS test_table3")

	// Create temp migrations directory
	tmpDir := t.TempDir()

	// Create three migrations
	migrations := map[string][2]string{
		"000001_create_table1": {
			"CREATE TABLE test_table1 (id SERIAL PRIMARY KEY, name TEXT)",
			"DROP TABLE test_table1",
		},
		"000002_create_table2": {
			"CREATE TABLE test_table2 (id SERIAL PRIMARY KEY, value INT)",
			"DROP TABLE test_table2",
		},
		"000003_create_table3": {
			"CREATE TABLE test_table3 (id SERIAL PRIMARY KEY, data TEXT)",
			"DROP TABLE test_table3",
		},
	}

	for name, scripts := range migrations {
		upPath := tmpDir + "/" + name + ".up.sql"
		downPath := tmpDir + "/" + name + ".down.sql"
		_ = os.WriteFile(upPath, []byte(scripts[0]), 0644)
		_ = os.WriteFile(downPath, []byte(scripts[1]), 0644)
	}

	// First, create schema_migrations table (simulating 000000 migration)
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id SERIAL PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		t.Fatalf("create schema_migrations: %v", err)
	}

	runner := New(pool, ctx)

	// Test 1: All migrations pending
	pending, err := runner.GetPendingMigrations(tmpDir)
	if err != nil {
		t.Fatalf("get pending migrations: %v", err)
	}
	if len(pending) != 3 {
		t.Fatalf("expected 3 pending, got %d", len(pending))
	}

	// Test 2: Apply all migrations
	if err := runner.ApplyAll(tmpDir); err != nil {
		t.Fatalf("apply all: %v", err)
	}

	// Test 3: Verify tables exist
	tables := []string{"test_table1", "test_table2", "test_table3"}
	for _, table := range tables {
		var exists bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM information_schema.tables
				WHERE table_name = $1
			)
		`, table).Scan(&exists)
		if err != nil || !exists {
			t.Fatalf("table %s not found", table)
		}
	}

	// Test 4: Verify migrations recorded
	applied, err := runner.GetAppliedMigrations()
	if err != nil {
		t.Fatalf("get applied migrations: %v", err)
	}
	if len(applied) != 3 {
		t.Fatalf("expected 3 applied, got %d", len(applied))
	}

	// Test 5: No pending migrations
	pending, err = runner.GetPendingMigrations(tmpDir)
	if err != nil {
		t.Fatalf("get pending after apply: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("expected 0 pending, got %d", len(pending))
	}

	// Test 6: Revert one migration
	if err := runner.RevertMigration(tmpDir, "000003_create_table3"); err != nil {
		t.Fatalf("revert migration: %v", err)
	}

	// Test 7: Verify table3 gone
	var exists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM information_schema.tables
			WHERE table_name = 'test_table3'
		)
	`).Scan(&exists)
	if err != nil || exists {
		t.Fatalf("test_table3 still exists after revert")
	}

	// Test 8: Verify 2 applied, 1 pending
	applied, err = runner.GetAppliedMigrations()
	if err != nil {
		t.Fatalf("get applied after revert: %v", err)
	}
	if len(applied) != 2 {
		t.Fatalf("expected 2 applied, got %d", len(applied))
	}

	pending, err = runner.GetPendingMigrations(tmpDir)
	if err != nil {
		t.Fatalf("get pending after revert: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending, got %d", len(pending))
	}

	// Test 9: RevertN multiple migrations
	if err := runner.RevertN(tmpDir, 2); err != nil {
		t.Fatalf("revert N: %v", err)
	}

	applied, err = runner.GetAppliedMigrations()
	if err != nil {
		t.Fatalf("get applied after revert N: %v", err)
	}
	if len(applied) != 0 {
		t.Fatalf("expected 0 applied after revert all, got %d", len(applied))
	}

	// Test 10: Reapply all
	if err := runner.ApplyAll(tmpDir); err != nil {
		t.Fatalf("reapply all: %v", err)
	}

	applied, err = runner.GetAppliedMigrations()
	if err != nil {
		t.Fatalf("get applied after reapply: %v", err)
	}
	if len(applied) != 3 {
		t.Fatalf("expected 3 applied after reapply, got %d", len(applied))
	}
}

// TestIntegration_MissingDownFile tests error handling when .down.sql is missing.
func TestIntegration_MissingDownFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create .up.sql without .down.sql
	upPath := tmpDir + "/000001_bad_migration.up.sql"
	_ = os.WriteFile(upPath, []byte("CREATE TABLE test (id INT)"), 0644)

	ctx := context.Background()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	runner := New(pool, ctx)

	// Should error when discovering migrations
	_, err = runner.discoverMigrations(tmpDir)
	if err == nil {
		t.Fatal("expected error for missing .down.sql")
	}
}

// TestIntegration_DuplicateMigration tests error handling for duplicate migrations.
func TestIntegration_DuplicateMigration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS schema_migrations")
	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS test_dup")

	_, _ = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id SERIAL PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)

	tmpDir := t.TempDir()
	_ = os.WriteFile(tmpDir+"/000001_test.up.sql", []byte("CREATE TABLE test_dup (id INT)"), 0644)
	_ = os.WriteFile(tmpDir+"/000001_test.down.sql", []byte("DROP TABLE test_dup"), 0644)

	runner := New(pool, ctx)

	// Apply once
	if err := runner.ApplyMigration(tmpDir, "000001_test"); err != nil {
		t.Fatalf("first apply: %v", err)
	}

	// Try to apply again
	err = runner.ApplyMigration(tmpDir, "000001_test")
	if err == nil {
		t.Fatal("expected error on duplicate migration")
	}
}

// TestIntegration_RevertNonApplied tests error handling when reverting non-applied migration.
func TestIntegration_RevertNonApplied(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS schema_migrations")

	_, _ = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id SERIAL PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)

	tmpDir := t.TempDir()
	_ = os.WriteFile(tmpDir+"/000001_test.up.sql", []byte("CREATE TABLE x (id INT)"), 0644)
	_ = os.WriteFile(tmpDir+"/000001_test.down.sql", []byte("DROP TABLE x"), 0644)

	runner := New(pool, ctx)

	// Try to revert without applying first
	err = runner.RevertMigration(tmpDir, "000001_test")
	if err == nil {
		t.Fatal("expected error reverting non-applied migration")
	}
}

// TestIntegration_ConcurrentMigrations tests that concurrent applies are serialized.
func TestIntegration_ConcurrentMigrations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS schema_migrations")
	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS concurrent_test")

	_, _ = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id SERIAL PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)

	tmpDir := t.TempDir()
	_ = os.WriteFile(tmpDir+"/000001_test.up.sql", []byte("CREATE TABLE concurrent_test (id INT)"), 0644)
	_ = os.WriteFile(tmpDir+"/000001_test.down.sql", []byte("DROP TABLE concurrent_test"), 0644)

	runner := New(pool, ctx)

	// Try to apply concurrently (should be serialized by advisory lock)
	done := make(chan error, 2)

	go func() {
		done <- runner.ApplyMigration(tmpDir, "000001_test")
	}()

	go func() {
		done <- runner.ApplyMigration(tmpDir, "000001_test")
	}()

	// One should succeed, one should fail (duplicate)
	err1 := <-done
	err2 := <-done

	successCount := 0
	if err1 == nil {
		successCount++
	}
	if err2 == nil {
		successCount++
	}

	if successCount != 1 {
		t.Fatalf("expected exactly 1 success in concurrent apply; got %d", successCount)
	}
}
