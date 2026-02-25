package hybridconfig

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sol-armada/sol-bot/stores"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
)

// HybridConfig manages configuration with TOML defaults and MongoDB overrides
type HybridConfig struct {
	defaults    *viper.Viper
	overrides   map[string]ConfigOverride
	configStore *stores.ConfigsStore
	mu          sync.RWMutex
	ctx         context.Context
	reloadChan  chan struct{}
	stopChan    chan struct{}
	lastReload  time.Time
}

// ConfigOverride represents a configuration override stored in MongoDB
type ConfigOverride struct {
	Name      string    `bson:"name" json:"name"`
	Value     any       `bson:"value" json:"value"`
	Override  bool      `bson:"override" json:"override"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
	UpdatedBy string    `bson:"updated_by" json:"updated_by"`
}

var (
	instance *HybridConfig
	once     sync.Once
)

// Setup initializes the hybrid config system
func Setup(ctx context.Context) error {
	var err error
	once.Do(func() {
		instance = &HybridConfig{
			defaults:   viper.New(),
			overrides:  make(map[string]ConfigOverride),
			ctx:        ctx,
			reloadChan: make(chan struct{}, 1),
			stopChan:   make(chan struct{}),
		}

		// Load defaults.toml from filesystem
		execPath, pathErr := os.Executable()
		if pathErr != nil {
			err = fmt.Errorf("failed to get executable path: %w", pathErr)
			return
		}
		execDir := filepath.Dir(execPath)
		defaultsPath := filepath.Join(execDir, "defaults.toml")

		// Try current directory if not found near executable
		if _, statErr := os.Stat(defaultsPath); statErr != nil {
			defaultsPath = "defaults.toml"
		}

		defaultsData, readErr := os.ReadFile(defaultsPath)
		if readErr != nil {
			err = fmt.Errorf("failed to read defaults.toml: %w", readErr)
			return
		}

		instance.defaults.SetConfigType("toml")
		if readErr := instance.defaults.ReadConfig(bytes.NewReader(defaultsData)); readErr != nil {
			err = fmt.Errorf("failed to parse defaults.toml: %w", readErr)
			return
		}

		// Get config store
		storesClient := stores.Get()
		cs, ok := storesClient.GetConfigsStore()
		if !ok {
			err = fmt.Errorf("config store not found")
			return
		}
		instance.configStore = cs

		// Load overrides from MongoDB
		if loadErr := instance.loadOverrides(); loadErr != nil {
			err = loadErr
			return
		}

		// Start background reload watcher
		go instance.watchReloads()
	})

	return err
}

// Get returns the singleton instance
func Get() *HybridConfig {
	if instance == nil {
		panic("hybridconfig not initialized, call Setup() first")
	}
	return instance
}

// loadOverrides loads all config overrides from MongoDB
func (h *HybridConfig) loadOverrides() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	cursor, err := h.configStore.Find(h.ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("failed to load config overrides: %w", err)
	}
	defer cursor.Close(h.ctx)

	newOverrides := make(map[string]ConfigOverride)
	for cursor.Next(h.ctx) {
		var override ConfigOverride
		if err := cursor.Decode(&override); err != nil {
			continue
		}
		if override.Override {
			newOverrides[override.Name] = override
		}
	}

	h.overrides = newOverrides
	h.lastReload = time.Now()
	return nil
}

// watchReloads watches for reload signals and reloads configs
func (h *HybridConfig) watchReloads() {
	ticker := time.NewTicker(30 * time.Second) // Check for updates every 30s
	defer ticker.Stop()

	for {
		select {
		case <-h.stopChan:
			return
		case <-h.reloadChan:
			_ = h.loadOverrides()
		case <-ticker.C:
			_ = h.loadOverrides()
		}
	}
}

// Reload triggers an immediate reload of overrides from MongoDB
func (h *HybridConfig) Reload() {
	select {
	case h.reloadChan <- struct{}{}:
	default:
	}
}

// Shutdown stops the background reload watcher
func (h *HybridConfig) Shutdown() {
	close(h.stopChan)
}

// GetString gets a string config value with override precedence
func (h *HybridConfig) GetString(key string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if override, ok := h.overrides[key]; ok {
		if str, ok := override.Value.(string); ok {
			return str
		}
	}
	return h.defaults.GetString(key)
}

// GetBool gets a boolean config value with override precedence
func (h *HybridConfig) GetBool(key string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if override, ok := h.overrides[key]; ok {
		if b, ok := override.Value.(bool); ok {
			return b
		}
	}
	return h.defaults.GetBool(key)
}

// GetInt gets an integer config value with override precedence
func (h *HybridConfig) GetInt(key string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if override, ok := h.overrides[key]; ok {
		if i, ok := override.Value.(int); ok {
			return i
		}
		if i, ok := override.Value.(int32); ok {
			return int(i)
		}
		if i, ok := override.Value.(int64); ok {
			return int(i)
		}
	}
	return h.defaults.GetInt(key)
}

// GetStringSlice gets a string slice config value with override precedence
func (h *HybridConfig) GetStringSlice(key string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if override, ok := h.overrides[key]; ok {
		if slice, ok := override.Value.([]string); ok {
			return slice
		}
		if slice, ok := override.Value.([]any); ok {
			result := make([]string, 0, len(slice))
			for _, v := range slice {
				if str, ok := v.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	return h.defaults.GetStringSlice(key)
}

// IsSet checks if a key is set in either overrides or defaults
func (h *HybridConfig) IsSet(key string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if _, ok := h.overrides[key]; ok {
		return true
	}
	return h.defaults.IsSet(key)
}

// SetOverride sets a config override in MongoDB
func (h *HybridConfig) SetOverride(key string, value any, updatedBy string) error {
	override := bson.M{
		"name":       key,
		"value":      value,
		"override":   true,
		"updated_at": time.Now(),
		"updated_by": updatedBy,
	}

	if err := h.configStore.UpsertOverride(override); err != nil {
		return fmt.Errorf("failed to set config override: %w", err)
	}

	// Trigger immediate reload
	h.Reload()
	return nil
}

// RemoveOverride removes a config override from MongoDB (reverts to default)
func (h *HybridConfig) RemoveOverride(key string) error {
	override := bson.M{
		"name":       key,
		"override":   false,
		"updated_at": time.Now(),
	}

	if err := h.configStore.UpsertOverride(override); err != nil {
		return fmt.Errorf("failed to remove config override: %w", err)
	}

	// Trigger immediate reload
	h.Reload()
	return nil
}

// GetAllSettings returns all settings as a nested map with overrides applied
func (h *HybridConfig) GetAllSettings() map[string]any {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := h.defaults.AllSettings()

	// Apply overrides
	for key, override := range h.overrides {
		setNestedValue(result, key, override.Value)
	}

	return result
}

// GetAllOverrides returns all active overrides
func (h *HybridConfig) GetAllOverrides() []ConfigOverride {
	h.mu.RLock()
	defer h.mu.RUnlock()

	overrides := make([]ConfigOverride, 0, len(h.overrides))
	for _, override := range h.overrides {
		overrides = append(overrides, override)
	}
	return overrides
}

// GetDefault gets the default value for a key (ignoring overrides)
func (h *HybridConfig) GetDefault(key string) any {
	return h.defaults.Get(key)
}

// setNestedValue sets a value in a nested map using dot notation
func setNestedValue(m map[string]any, key string, value any) {
	keys := splitKey(key)
	current := m

	for i := 0; i < len(keys)-1; i++ {
		k := keys[i]
		if _, ok := current[k]; !ok {
			current[k] = make(map[string]any)
		}
		if next, ok := current[k].(map[string]any); ok {
			current = next
		} else {
			return
		}
	}

	current[keys[len(keys)-1]] = value
}

// splitKey splits a dot-notation key into parts
func splitKey(key string) []string {
	var parts []string
	current := ""
	for _, ch := range key {
		if ch == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
