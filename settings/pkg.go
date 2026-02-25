package settings

import (
	"github.com/spf13/viper"
)

type Setting struct {
	*viper.Viper
}

// HybridConfigInterface defines the interface for hybrid config
type HybridConfigInterface interface {
	GetString(key string) string
	GetBool(key string) bool
	GetInt(key string) int
	GetStringSlice(key string) []string
	IsSet(key string) bool
	GetAllSettings() map[string]any
}

var setting *Setting
var hybridConfig HybridConfigInterface

func init() {
	Reset()
}

func Reset() {
	setting = &Setting{viper.New()}
}

// InitWithHybridConfig sets up the settings package to use hybrid config as backend
func InitWithHybridConfig(hc HybridConfigInterface) {
	hybridConfig = hc
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

func AddConfigPath(in string) {
	setting.AddConfigPath(in)
}

func ReadInConfig() error {
	return setting.ReadInConfig()
}

func IsSet(key string) bool {
	// Use hybrid config if available
	if hybridConfig != nil {
		return hybridConfig.IsSet(key)
	}
	return setting.IsSet(key)
}

func AllSettings() map[string]any {
	// Use hybrid config if available
	if hybridConfig != nil {
		return hybridConfig.GetAllSettings()
	}
	return setting.AllSettings()
}

func GetStringWithDefault(key string, val string) string {
	// Use hybrid config if available
	if hybridConfig != nil {
		if hybridConfig.IsSet(key) {
			return hybridConfig.GetString(key)
		}
		return val
	}
	
	if !setting.IsSet(key) {
		return val
	}
	return setting.GetString(key)
}

func GetIntWithDefault(key string, val int) int {
	// Use hybrid config if available
	if hybridConfig != nil {
		if hybridConfig.IsSet(key) {
			return hybridConfig.GetInt(key)
		}
		return val
	}
	
	if !setting.IsSet(key) {
		return val
	}
	return setting.GetInt(key)
}

func GetString(key string) string {
	// Use hybrid config if available
	if hybridConfig != nil {
		return hybridConfig.GetString(key)
	}
	return setting.GetString(key)
}

func GetBool(key string) bool {
	// Use hybrid config if available
	if hybridConfig != nil {
		return hybridConfig.GetBool(key)
	}
	return setting.GetBool(key)
}

func GetBoolWithDefault(key string, val bool) bool {
	// Use hybrid config if available
	if hybridConfig != nil {
		if hybridConfig.IsSet(key) {
			return hybridConfig.GetBool(key)
		}
		return val
	}
	
	if !setting.IsSet(key) {
		return val
	}
	return setting.GetBool(key)
}

func GetInt(key string) int {
	// Use hybrid config if available
	if hybridConfig != nil {
		return hybridConfig.GetInt(key)
	}
	return setting.GetInt(key)
}

func GetStringMapString(key string) map[string]string {
	return setting.GetStringMapString(key)
}

func GetIntSlice(key string) []int {
	return setting.GetIntSlice(key)
}

func GetStringSlice(key string) []string {
	// Use hybrid config if available
	if hybridConfig != nil {
		return hybridConfig.GetStringSlice(key)
	}
	return setting.GetStringSlice(key)
}
