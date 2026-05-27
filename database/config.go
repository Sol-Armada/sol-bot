package database

import (
	"fmt"
	"net/url"
	"time"
)

// Config is the top-level database configuration shared by all backends.
type Config struct {
	Postgres PostgresConfig
}

// PostgresConfig contains PostgreSQL connection settings for the new backend.
type PostgresConfig struct {
	Host           string
	Port           int
	Username       string
	Password       string
	Database       string
	SSLMode        string
	MaxConns       int32
	MinConns       int32
	ConnectTimeout time.Duration
}

func (c PostgresConfig) DSN() string {
	dsn := &url.URL{
		Scheme: "postgres",
		Host:   fmt.Sprintf("%s:%d", c.Host, c.Port),
		Path:   c.Database,
		User:   url.UserPassword(c.Username, c.Password),
	}

	query := dsn.Query()
	if c.SSLMode != "" {
		query.Set("sslmode", c.SSLMode)
	}
	if c.ConnectTimeout > 0 {
		query.Set("connect_timeout", fmt.Sprintf("%d", int(c.ConnectTimeout.Seconds())))
	}
	dsn.RawQuery = query.Encode()

	return dsn.String()
}
