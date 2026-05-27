package settings

import (
	"strings"

	"github.com/spf13/viper"
)

type Setting struct {
	*viper.Viper
}

var setting *Setting

func init() {
	setting = &Setting{viper.New()}
}

func Group(key string) *Setting {
	return &Setting{setting.Sub(key)}
}

func Set(key string, value any) {
	setting.Set(key, value)
}

func SetConfigName(in string) {
	setting.Set("ENVIRONMENT", in)
	setting.SetConfigName(in)
}

func SetConfigFile(in string) {
	setting.SetConfigFile(in)
}

func SetDefault(key string, value any) {
	setting.SetDefault(key, value)
}

func SetEnvPrefix(prefix string) {
	setting.SetEnvPrefix(prefix)
}

func SetEnvKeyReplacer(replacer *strings.Replacer) {
	setting.SetEnvKeyReplacer(replacer)
}

func AutomaticEnv() {
	setting.AutomaticEnv()
}

func BindEnv(key string, envs ...string) error {
	if len(envs) == 0 {
		return setting.BindEnv(key)
	}
	return setting.BindEnv(key, envs[0])
}

func AddConfigPath(in string) {
	setting.AddConfigPath(in)
}

func ReadInConfig() error {
	return setting.ReadInConfig()
}

func IsSet(key string) bool {
	return setting.IsSet(key)
}

func AllSettings() map[string]any {
	return setting.AllSettings()
}

func GetStringWithDefault(key string, val string) string {
	if !setting.IsSet(key) {
		return val
	}
	return setting.GetString(key)
}

func GetIntWithDefault(key string, val int) int {
	if !setting.IsSet(key) {
		return val
	}
	return setting.GetInt(key)
}

func GetString(key string) string {
	return setting.GetString(key)
}

func GetBool(key string) bool {
	return setting.GetBool(key)
}

func GetBoolWithDefault(key string, val bool) bool {
	if !setting.IsSet(key) {
		return val
	}
	return setting.GetBool(key)
}

func GetInt(key string) int {
	return setting.GetInt(key)
}

func GetStringMapString(key string) map[string]string {
	return setting.GetStringMapString(key)
}

func GetIntSlice(key string) []int {
	return setting.GetIntSlice(key)
}

func GetStringSlice(key string) []string {
	return setting.GetStringSlice(key)
}
