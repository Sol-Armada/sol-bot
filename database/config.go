package database

import (
	"fmt"
	"net/url"
	"time"
)

type Driver string

const (
	DriverMongo    Driver = "mongo"
	DriverPostgres Driver = "postgres"
)

// Config is the top-level database configuration shared by all backends.
type Config struct {
	Driver   Driver
	Mongo    MongoConfig
	Postgres PostgresConfig
}

func (c Config) SelectedDriver() Driver {
	if c.Driver == "" {
		return DriverMongo
	}
	return c.Driver
}

// MongoConfig keeps legacy MongoDB connection settings while migration is in progress.
type MongoConfig struct {
	Host           string
	Port           int
	Username       string
	Password       string
	Database       string
	ReplicaSetName string
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
