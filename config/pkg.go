package config

import (
	"context"
	"errors"
	"fmt"

	"github.com/sol-armada/sol-bot/database/postgresql"
	"github.com/sol-armada/sol-bot/database/sqlc/dbgen"
)

var configQueries *dbgen.Queries

func Setup() error {
	pg := postgresql.Get()
	if pg == nil || pg.Pool == nil || pg.Queries == nil {
		return errors.New("postgresql client not initialized")
	}
	configQueries = pg.Queries
	return nil
}

func GetConfig(config string) (any, error) {
	switch config {
	case "attendance_tags":
		return GetAttendanceTags()
	case "attendance_names":
		return GetAttendanceNames()
	default:
		return nil, errors.New("config " + config + " not found")
	}
}

func SetConfig(config string, value any) error {
	switch config {
	case "attendance_tags":
		tags, err := toStringList(value)
		if err != nil {
			return err
		}
		return replaceAttendanceTags(tags)
	case "attendance_names":
		names, err := toStringList(value)
		if err != nil {
			return err
		}
		return replaceAttendanceNames(names)
	default:
		return errors.New("config " + config + " not found")
	}
}

func GetConfigWithDefault[T any](config string, defaultValue T) (T, error) {
	val, err := GetConfig(config)
	if err != nil {
		return defaultValue, nil
	}
	if typed, ok := val.(T); ok {
		return typed, nil
	}
	return defaultValue, nil
}

func queries() (*dbgen.Queries, error) {
	if configQueries == nil {
		return nil, errors.New("config service not initialized")
	}
	return configQueries, nil
}

func withTx(fn func(qtx *dbgen.Queries) error) error {
	pg := postgresql.Get()
	if pg == nil || pg.Pool == nil || pg.Queries == nil {
		return errors.New("postgresql client not initialized")
	}

	ctx := context.Background()
	tx, err := pg.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := fn(pg.Queries.WithTx(tx)); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func toStringList(value any) ([]string, error) {
	switch v := value.(type) {
	case []string:
		return v, nil
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("expected string config item, got %T", item)
			}
			out = append(out, s)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("expected []string config value, got %T", value)
	}
}
