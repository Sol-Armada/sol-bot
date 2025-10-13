package config

import (
	"errors"

	"github.com/sol-armada/sol-bot/stores"
)

var configStore *stores.ConfigsStore

func Setup() error {
	storesClient := stores.Get()
	cs, ok := storesClient.GetConfigsStore()
	if !ok {
		return errors.New("config store not found")
	}
	configStore = cs

	return nil
}

func GetConfig(config string) (any, error) {
	res := configStore.Get(config)
	if res.Err() != nil {
		return "", res.Err()
	}

	var out map[string]any

	if err := res.Decode(&out); err != nil {
		return "", err
	}

	return out["value"], nil
}

func SetConfig(config string, value any) error {
	return configStore.Upsert(config, value)
}

func GetConfigWithDefault[T any](config string, defaultValue T) (T, error) {
	res := configStore.Get(config)
	if res.Err() != nil {
		return defaultValue, nil
	}

	var decoded map[string]any

	if err := res.Decode(&decoded); err != nil {
		return defaultValue, err
	}

	val, ok := decoded["value"].(T)
	if !ok {
		return defaultValue, nil
	}

	return val, nil
}
