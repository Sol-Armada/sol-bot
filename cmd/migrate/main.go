package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sol-armada/sol-bot/database/migrations"
)

func main() {
	ctx := context.Background()

	// Parse flags
	pgDSN := flag.String("pg-dsn", envOrDefault("POSTGRES_DSN", ""), "PostgreSQL DSN")
	action := flag.String("action", "status", "Action: up, down, status, reset")
	step := flag.Int("step", 1, "Number of migrations to apply/revert")
	dryRun := flag.Bool("dry-run", false, "Show what would happen without executing")
	force := flag.Bool("force", false, "Force dangerous operations (e.g., reset)")
	flag.Parse()

	// Validate DSN
	if strings.TrimSpace(*pgDSN) == "" {
		log.Fatal("missing postgres DSN, set --pg-dsn or POSTGRES_DSN")
	}

	// Connect to database
	pool, err := pgxpool.New(ctx, *pgDSN)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping postgres: %v", err)
	}

	// Create runner
	runner := migrations.New(pool, ctx)
	migrationsDir := "database/migrations"

	// Handle actions
	switch strings.ToLower(*action) {
	case "up":
		handleUp(ctx, runner, migrationsDir, *step, *dryRun)
	case "down":
		handleDown(ctx, runner, migrationsDir, *step, *dryRun)
	case "status":
		handleStatus(ctx, runner, migrationsDir)
	case "reset":
		if !*force {
			log.Fatal("reset requires --force flag for safety")
		}
		handleReset(ctx, runner, migrationsDir, *dryRun)
	default:
		log.Fatalf("unknown action: %s (valid: up, down, status, reset)", *action)
	}
}

func handleUp(ctx context.Context, runner *migrations.Runner, dir string, steps int, dryRun bool) {
	pending, err := runner.GetPendingMigrations(dir)
	if err != nil {
		log.Fatalf("get pending migrations: %v", err)
	}

	if len(pending) == 0 {
		fmt.Println("✓ Already up-to-date (0 pending migrations)")
		return
	}

	if steps > len(pending) {
		log.Fatalf("cannot apply %d migrations; only %d pending", steps, len(pending))
	}

	toApply := pending[:steps]

	if dryRun {
		fmt.Printf("DRY RUN: Would apply %d migration(s):\n", len(toApply))
		for i, name := range toApply {
			fmt.Printf("  %d. %s\n", i+1, name)
		}
		return
	}

	fmt.Printf("Applying %d migration(s):\n", len(toApply))
	for i, name := range toApply {
		if err := runner.ApplyMigration(dir, name); err != nil {
			log.Fatalf("apply migration %s: %v", name, err)
		}
		fmt.Printf("  [%d/%d] ✓ %s\n", i+1, len(toApply), name)
	}

	fmt.Printf("\n✓ Successfully applied %d migration(s)\n", len(toApply))
}

func handleDown(ctx context.Context, runner *migrations.Runner, dir string, steps int, dryRun bool) {
	applied, err := runner.GetAppliedMigrations()
	if err != nil {
		log.Fatalf("get applied migrations: %v", err)
	}

	if len(applied) == 0 {
		fmt.Println("✓ Nothing to revert (0 applied migrations)")
		return
	}

	if steps > len(applied) {
		log.Fatalf("cannot revert %d migrations; only %d applied", steps, len(applied))
	}

	toRevert := applied[len(applied)-steps:]

	if dryRun {
		fmt.Printf("DRY RUN: Would revert %d migration(s):\n", len(toRevert))
		for i, name := range toRevert {
			fmt.Printf("  %d. %s\n", i+1, name)
		}
		return
	}

	fmt.Printf("Reverting %d migration(s):\n", len(toRevert))

	// Revert in reverse order
	for i := len(toRevert) - 1; i >= 0; i-- {
		name := toRevert[i]
		if err := runner.RevertMigration(dir, name); err != nil {
			log.Fatalf("revert migration %s: %v", name, err)
		}
		fmt.Printf("  [%d/%d] ✓ %s\n", len(toRevert)-i, len(toRevert), name)
	}

	fmt.Printf("\n✓ Successfully reverted %d migration(s)\n", len(toRevert))
}

func handleStatus(ctx context.Context, runner *migrations.Runner, dir string) {
	status, err := runner.GetStatus(dir)
	if err != nil {
		log.Fatalf("get status: %v", err)
	}

	fmt.Println("Migration Status:")
	fmt.Println("================")
	fmt.Printf("Applied:  %d\n", status.AppliedCount)
	fmt.Printf("Pending:  %d\n", status.PendingCount)
	fmt.Println()

	if status.AppliedCount > 0 {
		fmt.Println("Applied migrations:")
		for i, name := range status.Applied {
			fmt.Printf("  %d. %s\n", i+1, name)
		}
		fmt.Println()
	}

	if status.PendingCount > 0 {
		fmt.Println("Pending migrations:")
		for i, name := range status.Pending {
			fmt.Printf("  %d. %s\n", i+1, name)
		}
		fmt.Println()
	}

	if status.PendingCount == 0 {
		fmt.Println("✓ Database is up-to-date")
	}
}

func handleReset(ctx context.Context, runner *migrations.Runner, dir string, dryRun bool) {
	applied, err := runner.GetAppliedMigrations()
	if err != nil {
		log.Fatalf("get applied migrations: %v", err)
	}

	if len(applied) == 0 {
		fmt.Println("✓ Nothing to reset (0 applied migrations)")
		return
	}

	if dryRun {
		fmt.Printf("DRY RUN: Would revert all %d migration(s):\n", len(applied))
		for i, name := range applied {
			fmt.Printf("  %d. %s\n", i+1, name)
		}
		return
	}

	fmt.Printf("⚠️  RESET: Reverting all %d migration(s)...\n", len(applied))

	// Revert all in reverse order
	for i := len(applied) - 1; i >= 0; i-- {
		name := applied[i]
		if err := runner.RevertMigration(dir, name); err != nil {
			log.Fatalf("revert migration %s: %v", name, err)
		}
		fmt.Printf("  [%d/%d] ✓ %s\n", len(applied)-i, len(applied), name)
	}

	fmt.Printf("\n✓ Successfully reverted all %d migration(s)\n", len(applied))
}

func envOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
