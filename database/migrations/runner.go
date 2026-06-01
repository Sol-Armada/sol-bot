package migrations

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Runner manages schema migrations.
type Runner struct {
	pool *pgxpool.Pool
	ctx  context.Context
}

// New creates a new migration runner.
func New(pool *pgxpool.Pool, ctx context.Context) *Runner {
	return &Runner{
		pool: pool,
		ctx:  ctx,
	}
}

// Migration represents a single migration file.
type Migration struct {
	Name      string // e.g., "000001_init"
	Version   string // e.g., "000001"
	Applied   bool
	AppliedAt string
}

// GetAppliedMigrations returns a list of applied migration names.
func (r *Runner) GetAppliedMigrations() ([]string, error) {
	// Acquire advisory lock to serialize access
	if _, err := r.pool.Exec(r.ctx, "SELECT pg_advisory_lock(1)"); err != nil {
		return nil, fmt.Errorf("acquire lock: %w", err)
	}
	defer r.pool.Exec(r.ctx, "SELECT pg_advisory_unlock(1)") //nolint:errcheck

	query := "SELECT name FROM schema_migrations ORDER BY name"
	rows, err := r.pool.Query(r.ctx, query)
	if err != nil {
		// Table may not exist yet; return empty list
		if strings.Contains(err.Error(), "does not exist") {
			return []string{}, nil
		}
		return nil, fmt.Errorf("query applied migrations: %w", err)
	}
	defer rows.Close()

	var applied []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan migration name: %w", err)
		}
		applied = append(applied, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return applied, nil
}

// GetPendingMigrations returns a list of migrations that have not been applied yet.
func (r *Runner) GetPendingMigrations(migrationsDir string) ([]string, error) {
	discovered, err := r.discoverMigrations(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("discover migrations: %w", err)
	}

	applied, err := r.GetAppliedMigrations()
	if err != nil {
		return nil, fmt.Errorf("get applied migrations: %w", err)
	}

	appliedMap := make(map[string]bool)
	for _, name := range applied {
		appliedMap[name] = true
	}

	var pending []string
	for _, name := range discovered {
		if !appliedMap[name] {
			pending = append(pending, name)
		}
	}

	sort.Strings(pending)
	return pending, nil
}

// ApplyMigration executes a single migration's up script and records it.
func (r *Runner) ApplyMigration(migrationsDir, name string) error {
	// Acquire advisory lock
	if _, err := r.pool.Exec(r.ctx, "SELECT pg_advisory_lock(1)"); err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}
	defer r.pool.Exec(r.ctx, "SELECT pg_advisory_unlock(1)") //nolint:errcheck

	// Read migration file
	sqlContent, err := readMigrationSQL(migrationsDir, name+".up.sql")
	if err != nil {
		return fmt.Errorf("read migration file %s: %w", name+".up.sql", err)
	}

	sql := string(sqlContent)

	// Start transaction
	tx, err := r.pool.Begin(r.ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(r.ctx) //nolint:errcheck

	// Execute migration
	if _, err := tx.Exec(r.ctx, sql); err != nil {
		return fmt.Errorf("execute migration %s: %w", name, err)
	}

	// Record in schema_migrations
	insertQuery := "INSERT INTO schema_migrations (name, applied_at) VALUES ($1, NOW())"
	if _, err := tx.Exec(r.ctx, insertQuery, name); err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return fmt.Errorf("migration %s already applied", name)
		}
		return fmt.Errorf("record migration %s: %w", name, err)
	}

	// Commit transaction
	if err := tx.Commit(r.ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// RevertMigration executes a single migration's down script and removes the record.
func (r *Runner) RevertMigration(migrationsDir, name string) error {
	// Acquire advisory lock
	if _, err := r.pool.Exec(r.ctx, "SELECT pg_advisory_lock(1)"); err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}
	defer r.pool.Exec(r.ctx, "SELECT pg_advisory_unlock(1)") //nolint:errcheck

	// Check if migration is applied
	var exists bool
	err := r.pool.QueryRow(r.ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name = $1)", name).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check if migration applied: %w", err)
	}
	if !exists {
		return fmt.Errorf("migration %s not applied", name)
	}

	// Read migration file
	sqlContent, err := readMigrationSQL(migrationsDir, name+".down.sql")
	if err != nil {
		return fmt.Errorf("read migration file %s: %w", name+".down.sql", err)
	}

	sql := string(sqlContent)

	// Start transaction
	tx, err := r.pool.Begin(r.ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(r.ctx) //nolint:errcheck

	// Execute migration
	if _, err := tx.Exec(r.ctx, sql); err != nil {
		return fmt.Errorf("execute migration %s: %w", name, err)
	}

	// Remove from schema_migrations
	deleteQuery := "DELETE FROM schema_migrations WHERE name = $1"
	if _, err := tx.Exec(r.ctx, deleteQuery, name); err != nil {
		return fmt.Errorf("remove migration %s: %w", name, err)
	}

	// Commit transaction
	if err := tx.Commit(r.ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// ApplyAll applies all pending migrations in order.
func (r *Runner) ApplyAll(migrationsDir string) error {
	pending, err := r.GetPendingMigrations(migrationsDir)
	if err != nil {
		return fmt.Errorf("get pending migrations: %w", err)
	}

	if len(pending) == 0 {
		return nil // Already up-to-date
	}

	for _, name := range pending {
		if err := r.ApplyMigration(migrationsDir, name); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}

	return nil
}

func (r *Runner) RevertAll(migrationsDir string) error {
	applied, err := r.GetAppliedMigrations()
	if err != nil {
		return fmt.Errorf("get applied migrations: %w", err)
	}

	if len(applied) == 1 {
		return nil // Nothing to revert
	}

	// Revert in reverse order (last applied first)
	for i := len(applied) - 1; i >= 1; i-- {
		name := applied[i]
		if err := r.RevertMigration(migrationsDir, name); err != nil {
			return fmt.Errorf("revert migration %s: %w", name, err)
		}
	}

	return nil
}

// RevertN reverts the last N applied migrations in reverse order.
func (r *Runner) RevertN(migrationsDir string, steps int) error {
	applied, err := r.GetAppliedMigrations()
	if err != nil {
		return fmt.Errorf("get applied migrations: %w", err)
	}

	if len(applied) == 0 {
		return errors.New("no migrations applied")
	}

	if steps > len(applied) {
		return fmt.Errorf("cannot revert %d migrations; only %d applied", steps, len(applied))
	}

	// Revert in reverse order (last applied first)
	for i := range steps {
		idx := len(applied) - 1 - i
		name := applied[idx]
		if err := r.RevertMigration(migrationsDir, name); err != nil {
			return fmt.Errorf("revert migration %s: %w", name, err)
		}
	}

	return nil
}

// discoverMigrations scans the migrations directory and returns all migration names in order.
func (r *Runner) discoverMigrations(dir string) ([]string, error) {
	upRegex := regexp.MustCompile(`^(\d+)_(.+)\.up\.sql$`)
	var migrations []string
	seen := make(map[string]bool)

	files, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read migrations directory: %w", err)
		}

		embeddedEntries, embeddedErr := embeddedMigrations.ReadDir(".")
		if embeddedErr != nil {
			return nil, fmt.Errorf("read migrations directory: %w", err)
		}

		for _, entry := range embeddedEntries {
			fileName := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(fileName, ".up.sql") {
				continue
			}

			match := upRegex.FindStringSubmatch(fileName)
			if match == nil {
				continue
			}

			name := strings.TrimSuffix(fileName, ".up.sql")
			if _, embeddedErr := embeddedMigrations.ReadFile(name + ".down.sql"); embeddedErr != nil {
				return nil, fmt.Errorf("missing .down.sql for %s", name)
			}

			if !seen[name] {
				migrations = append(migrations, name)
				seen[name] = true
			}
		}
	} else {
		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".up.sql") {
				continue
			}

			match := upRegex.FindStringSubmatch(file.Name())
			if match == nil {
				continue
			}

			// Migration name is everything before .up.sql (e.g., "000001_init")
			name := strings.TrimSuffix(file.Name(), ".up.sql")

			// Validate corresponding .down.sql exists
			downPath := filepath.Join(dir, name+".down.sql")
			if _, err := os.ReadFile(downPath); err != nil {
				return nil, fmt.Errorf("missing .down.sql for %s", name)
			}

			if !seen[name] {
				migrations = append(migrations, name)
				seen[name] = true
			}
		}
	}

	// Sort by version prefix
	slices.Sort(migrations)

	return migrations, nil
}

func readMigrationSQL(migrationsDir, fileName string) ([]byte, error) {
	path := filepath.Join(migrationsDir, fileName)
	sqlContent, err := os.ReadFile(path)
	if err == nil {
		return sqlContent, nil
	}

	if !os.IsNotExist(err) {
		return nil, err
	}

	embeddedContent, embeddedErr := embeddedMigrations.ReadFile(fileName)
	if embeddedErr != nil {
		return nil, err
	}

	return embeddedContent, nil
}

// Status returns a summary of applied and pending migrations.
type Status struct {
	Applied      []string
	Pending      []string
	AppliedCount int
	PendingCount int
}

// GetStatus returns migration status.
func (r *Runner) GetStatus(migrationsDir string) (*Status, error) {
	applied, err := r.GetAppliedMigrations()
	if err != nil {
		return nil, fmt.Errorf("get applied migrations: %w", err)
	}

	pending, err := r.GetPendingMigrations(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("get pending migrations: %w", err)
	}

	return &Status{
		Applied:      applied,
		Pending:      pending,
		AppliedCount: len(applied),
		PendingCount: len(pending),
	}, nil
}
