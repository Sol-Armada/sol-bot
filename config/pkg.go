package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	*viper.Viper
}

var config *Config

func init() {
	Reset()
}

func Reset() {
	config = &Config{viper.New()}
}

func Group(key string) *Config {
	return &Config{config.Sub(key)}
}

func Set(key string, value interface{}) {
	config.Set(key, value)
}

func SetConfigName(in string) {
	config.Set("ENVIRONMENT", in)
	config.SetConfigName(in)
}

func AddConfigPath(in string) {
	config.AddConfigPath(in)
}

func ReadInConfig() error {
	return config.ReadInConfig()
}

func IsSet(key string) bool {
	return config.IsSet(key)
}

func AllSettings() map[string]interface{} {
	return config.AllSettings()
}

func GetStringWithDefault(key string, val string) string {
	if !config.IsSet(key) {
		return val
	}
	return config.GetString(key)
}

func GetIntWithDefault(key string, val int) int {
	if !config.IsSet(key) {
		return val
	}
	return config.GetInt(key)
}

func GetString(key string) string {
	return config.GetString(key)
}

func GetBool(key string) bool {
	return config.GetBool(key)
}

func GetInt(key string) int {
	return config.GetInt(key)
}

func GetStringMapString(key string) map[string]string {
	return config.GetStringMapString(key)
}

func GetIntSlice(key string) []int {
	return config.GetIntSlice(key)
}

func GetStringSlice(key string) []string {
	return config.GetStringSlice(key)
}
