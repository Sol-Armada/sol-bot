package database

//go:generate sh -c "cd .. && sqlc generate"

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresClient wraps the pgx pool used by the PostgreSQL backend.
type PostgresClient struct {
	Pool *pgxpool.Pool
}

func NewPostgres(ctx context.Context, cfg PostgresConfig) (*PostgresClient, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	if cfg.MaxConns > 0 {
		poolCfg.MaxConns = cfg.MaxConns
	}
	if cfg.MinConns > 0 {
		poolCfg.MinConns = cfg.MinConns
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &PostgresClient{Pool: pool}, nil
}

func (c *PostgresClient) Close() {
	if c == nil || c.Pool == nil {
		return
	}
	c.Pool.Close()
}
