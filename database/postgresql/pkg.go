package postgresql

// type Client struct {
// 	Pool    *pgxpool.Pool
// 	Queries *dbgen.Queries
// }

// var client *Client

// func New(ctx context.Context, cfg database.PostgresConfig) (*Client, error) {
// 	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
// 	if err != nil {
// 		return nil, fmt.Errorf("parse postgres config: %w", err)
// 	}

// 	if cfg.MaxConns > 0 {
// 		poolCfg.MaxConns = cfg.MaxConns
// 	}
// 	if cfg.MinConns > 0 {
// 		poolCfg.MinConns = cfg.MinConns
// 	}

// 	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
// 	if err != nil {
// 		return nil, fmt.Errorf("create postgres pool: %w", err)
// 	}

// 	if err := pool.Ping(ctx); err != nil {
// 		pool.Close()
// 		return nil, fmt.Errorf("ping postgres: %w", err)
// 	}

// 	newClient := &Client{
// 		Pool:    pool,
// 		Queries: dbgen.New(pool),
// 	}
// 	client = newClient
// 	return newClient, nil
// }

// func Get() *Client {
// 	return client
// }

// func (c *Client) Close() {
// 	if c == nil || c.Pool == nil {
// 		return
// 	}
// 	c.Pool.Close()
// }
