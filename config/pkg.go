package config

import (
	"errors"

	"github.com/sol-armada/sol-bot/stores"
)

// var defaults = map[string]interface{}{
// 	"attendance_tags":          []string{},
// 	"attendance_names":         []string{},
// 	"attendance_allowed_roles": []string{},

// 	"dkp_name": "StarCoin",
// }

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

func GetConfig(config string) (interface{}, error) {
	res := configStore.Get(config)
	if res.Err() != nil {
		return "", res.Err()
	}

	var out map[string]interface{}

	if err := res.Decode(&out); err != nil {
		return "", err
	}

	return out["value"], nil
}

func SetConfig(config string, value interface{}) error {
	return configStore.Upsert(config, value)
}
