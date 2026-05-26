package config

import (
	"errors"
)

// Temporary stub: config data has been migrated out of database.
// TODO: Either migrate configs to postgres or maintain them in settings files/in-memory only.
var configStore interface{} = nil

func Setup() error {
	// No-op; configs not stored in database anymore.
	return nil
}

func GetConfig(config string) (any, error) {
	// Stub implementation; return empty or known defaults.
	switch config {
	case "attendance_tags":
		// Return empty array for now; handlers should tolerate this.
		return []string{}, nil
	case "attendance_names":
		return []string{}, nil
	default:
		return nil, errors.New("config " + config + " not found (configs removed from database)")
	}
}

func SetConfig(config string, value any) error {
	// No-op stub.
	_ = config
	_ = value
	return errors.New("SetConfig not supported (configs removed from database)")
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
