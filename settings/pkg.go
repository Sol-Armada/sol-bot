package settings

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/sol-armada/sol-bot/stores"
	"go.mongodb.org/mongo-driver/bson"
)

// getFromMongoDB retrieves a config value from MongoDB
func getFromMongoDB(key string) (any, bool) {
	client := stores.Get()
	if client == nil {
		return nil, false
	}

	configStore, ok := client.GetConfigsStore()
	if !ok {
		return nil, false
	}

	result := configStore.Get(strings.ToLower(key))
	if result.Err() != nil {
		return nil, false
	}

	var doc bson.M
	if err := result.Decode(&doc); err != nil {
		return nil, false
	}

	value, ok := doc["value"]
	return value, ok
}

func IsSet(key string) bool {
	if _, ok := getFromMongoDB(key); ok {
		return true
	}
	return false
}

func AllSettings() map[string]any {
	var allSettings map[string]any
	client := stores.Get()
	if client != nil {
		if configStore, ok := client.GetConfigsStore(); ok {
			ctx := context.Background()
			cursor, err := configStore.GetAll()
			if err == nil {
				defer cursor.Close(ctx)

				for cursor.Next(ctx) {
					var doc bson.M
					if err := cursor.Decode(&doc); err == nil {
						if name, ok := doc["name"].(string); ok {
							if value, ok := doc["value"]; ok {
								allSettings[name] = value
							}
						}
					}
				}
			}
		}
	}
	return allSettings
}

func getBool(key string) (bool, bool) {
	if value, ok := getFromMongoDB(key); ok {
		if b, ok := value.(bool); ok {
			return b, true
		}
	}
	return false, false
}

func GetBool(key string) bool {
	if value, ok := getFromMongoDB(key); ok {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return false
}

func GetBoolWithDefault(key string, val bool) bool {
	if value, ok := getBool(key); ok {
		return value
	}
	return val
}

func GetInt(key string) (int, bool) {
	// Check MongoDB first
	if value, ok := getFromMongoDB(key); ok {
		switch v := value.(type) {
		case int:
			return v, true
		case int32:
			return int(v), true
		case int64:
			return int(v), true
		case float64:
			return int(v), true
		}
	}
	return 0, false
}

func GetIntWithDefault(key string, val int) int {
	if value, ok := GetInt(key); ok {
		return value
	}
	return val
}

func GetConfigSlice(key string) []string {
	if value, ok := getFromMongoDB(key); ok {
		switch v := value.(type) {
		case []string:
			return v
		case []any:
			result := make([]string, 0, len(v))
			for _, item := range v {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		case bson.A:
			result := make([]string, 0, len(v))
			for _, item := range v {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}

	return nil
}

// GetConfig retrieves a raw config value from MongoDB

// SetConfig stores a config value to MongoDB
func SetConfig(config string, value any) error {
	client := stores.Get()
	if client == nil {
		return fmt.Errorf("stores not initialized")
	}

	configStore, ok := client.GetConfigsStore()
	if !ok {
		return fmt.Errorf("config store not found")
	}

	return configStore.Upsert(config, value)
}

// GetConfigWithDefault retrieves a typed config value with a default fallback
func GetConfigWithDefault[T any](config string, defaultValue T) (T, error) {
	client := stores.Get()
	if client == nil {
		return defaultValue, nil
	}

	configStore, ok := client.GetConfigsStore()
	if !ok {
		return defaultValue, nil
	}

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

// Attendance-specific config functions (consolidated from config package)

// GetAttendanceTags retrieves the list of attendance tags
func GetAttendanceTags() ([]string, error) {
	client := stores.Get()
	if client == nil {
		return nil, fmt.Errorf("stores not initialized")
	}

	c, ok := client.GetConfigsStore()
	if !ok {
		return nil, fmt.Errorf("could not get configs store")
	}

	res := c.Get("attendance_tags")
	if res.Err() != nil {
		return nil, res.Err()
	}

	var out []string
	if err := res.Decode(&out); err != nil {
		return nil, err
	}

	return out, nil
}

// NewAttendanceTag adds a new attendance tag
func NewAttendanceTag(tag string) error {
	client := stores.Get()
	if client == nil {
		return fmt.Errorf("stores not initialized")
	}

	c, ok := client.GetConfigsStore()
	if !ok {
		return fmt.Errorf("could not get configs store")
	}

	tags, err := GetAttendanceTags()
	if err != nil && err.Error() != "mongo: no documents in result" {
		return err
	}

	if slices.Contains(tags, tag) {
		return nil
	}

	tags = append(tags, strings.ToUpper(strings.ReplaceAll(tag, " ", "-")))
	return c.Upsert("attendance_tags", tags)
}

// GetAttendanceNames retrieves the list of attendance names
func GetAttendanceNames() ([]string, error) {
	raw, ok := getFromMongoDB("attendance_names")
	if !ok {
		return nil, fmt.Errorf("could not get attendance names")
	}

	names, ok := raw.(bson.A)
	if !ok {
		return nil, fmt.Errorf("could not convert attendance names to []string")
	}

	var out []string
	for _, name := range names {
		out = append(out, name.(string))
	}

	return out, nil
}

// ValidAttendanceName checks if a name is valid for attendance
func ValidAttendanceName(name string) (bool, error) {
	names, err := GetAttendanceNames()
	if err != nil {
		return false, err
	}

	name = strings.ToLower(name)
	for _, n := range names {
		if strings.ToLower(n) == name {
			return true, nil
		}
	}

	return false, nil
}

func getString(key string) (string, bool) {
	if value, ok := getFromMongoDB(key); ok {
		if s, ok := value.(string); ok {
			return s, true
		}
	}
	return "", false
}

func GetString(key string) string {
	if value, ok := getFromMongoDB(key); ok {
		if s, ok := value.(string); ok {
			return s
		}
	}
	return ""
}

func GetStringWithDefault(key string, val string) string {
	if value, ok := getString(key); ok {
		return value
	}
	return val
}

// NewAttendanceName adds a new attendance name
func NewAttendanceName(name string) error {
	client := stores.Get()
	if client == nil {
		return fmt.Errorf("stores not initialized")
	}

	c, ok := client.GetConfigsStore()
	if !ok {
		return fmt.Errorf("could not get configs store")
	}

	names, err := GetAttendanceNames()
	if err != nil && err.Error() != "mongo: no documents in result" {
		return err
	}

	if slices.Contains(names, name) {
		return nil
	}

	names = append(names, name)
	return c.Upsert("attendance_names", names)
}

// RemoveAttendanceName removes an attendance name
func RemoveAttendanceName(name string) error {
	client := stores.Get()
	if client == nil {
		return fmt.Errorf("stores not initialized")
	}

	c, ok := client.GetConfigsStore()
	if !ok {
		return fmt.Errorf("could not get configs store")
	}

	names, err := GetAttendanceNames()
	if err != nil && err.Error() != "mongo: no documents in result" {
		return err
	}

	for i, n := range names {
		if n == name {
			names = append(names[:i], names[i+1:]...)
			break
		}
	}

	return c.Upsert("attendance_names", names)
}
