# Hybrid Configuration System - Usage Guide

The hybrid configuration system has been fully integrated into sol-bot. All existing code using the `settings` package will automatically benefit from the hybrid config system.

## How It Works

1. **Startup**: During initialization, the system loads `defaults.toml` and MongoDB overrides
2. **Settings Wrapper**: The `settings` package now automatically delegates to `hybridconfig`
3. **Hot Reload**: Config changes in MongoDB are synced every 30 seconds
4. **Backward Compatible**: All existing code using `settings.*` functions works unchanged

## Using Configurations Anywhere

### Option 1: Using Settings Package (Recommended for existing code)

All existing code continues to work without changes:

```go
import "github.com/sol-armada/sol-bot/settings"

// Get config values - automatically uses hybrid config if available
token := settings.GetString("discord.bot_token")
debugMode := settings.GetBool("log.debug")
port := settings.GetIntWithDefault("dashboard.port", 8080)
roles := settings.GetStringSlice("features.attendance.allowed_roles")
```

### Option 2: Using HybridConfig Directly (Recommended for new code)

For new services or packages, you can use hybridconfig directly:

```go
import "github.com/sol-armada/sol-bot/hybridconfig"

func MyFunction() {
    config := hybridconfig.Get()
    
    // Read values
    enabled := config.GetBool("features.my_feature.enable")
    apiKey := config.GetString("my_feature.api_key")
    timeout := config.GetInt("my_feature.timeout")
    allowedUsers := config.GetStringSlice("my_feature.allowed_users")
    
    // Check if key exists
    if config.IsSet("my_feature.custom_setting") {
        // ...
    }
}
```

## Managing Configurations

### Via Dashboard (Recommended)

1. Navigate to `http://localhost:8080/configs`
2. Search for the config you want to change
3. Click "Edit" and enter the new value
4. Provide your name/identifier for audit trail
5. Changes apply immediately (within 30 seconds)

### Via Code

```go
import "github.com/sol-armada/sol-bot/hybridconfig"

config := hybridconfig.Get()

// Set an override
err := config.SetOverride("features.new_feature.enable", true, "admin")

// Remove override (revert to default)
err := config.RemoveOverride("features.new_feature.enable")

// Force immediate reload from MongoDB
config.Reload()

// Get all current settings
allSettings := config.GetAllSettings()

// Get all overrides with audit info
overrides := config.GetAllOverrides()
```

## Adding New Configuration Keys

### 1. Add to defaults.toml

```toml
[features.my_new_feature]
enable = false
api_endpoint = "https://api.example.com"
timeout = 30
allowed_users = []
```

### 2. Use in Your Code

```go
import "github.com/sol-armada/sol-bot/settings"

func InitMyFeature() error {
    if !settings.GetBool("features.my_new_feature.enable") {
        return nil // Feature disabled
    }
    
    endpoint := settings.GetString("features.my_new_feature.api_endpoint")
    timeout := settings.GetInt("features.my_new_feature.timeout")
    users := settings.GetStringSlice("features.my_new_feature.allowed_users")
    
    // Initialize feature...
}
```

### 3. (Optional) Add to Dashboard

Edit `dashboard/metrics.go` and add to `collectConfigMetrics`:

```go
configKeys := []struct {
    Key  string
    Type string
}{
    // ... existing configs ...
    {"features.my_new_feature.enable", "boolean"},
    {"features.my_new_feature.api_endpoint", "string"},
    {"features.my_new_feature.timeout", "string"},
    {"features.my_new_feature.allowed_users", "array"},
}
```

## Configuration Precedence

Values are resolved in this order:

1. **MongoDB Override** (if set via dashboard or API) ✅ Highest Priority
2. **defaults.toml** (baseline defaults) ↓
3. **Environment-specific settings.toml** (for secrets) ↓
4. **Code default** (fallback in GetStringWithDefault, etc.) ✅ Lowest Priority

## Type Conversion

The system handles type conversion automatically:

```go
// Boolean
enabled := config.GetBool("feature.enable") // "true", "false", true, false

// Integer
port := config.GetInt("server.port") // "8080", 8080, int32(8080), int64(8080)

// String
name := config.GetString("app.name") // Any value converted to string

// String Array
roles := config.GetStringSlice("allowed_roles") // ["role1", "role2"]
```

## Hot Reload Behavior

- Config changes in MongoDB are synced every **30 seconds**
- You can trigger immediate reload: `hybridconfig.Get().Reload()`
- All running code will see updated values on next access
- No service restart required

## Best Practices

### ✅ DO

- Use `settings` package for existing features (backward compatible)
- Use `hybridconfig.Get()` for new services
- Add meaningful defaults to `defaults.toml`
- Provide your name/ID when updating configs via dashboard
- Export configs regularly as backup
- Keep sensitive values in environment-specific `settings.toml` files

### ❌ DON'T

- Don't put secrets in `defaults.toml` (it's version controlled)
- Don't modify `settings.toml` files at runtime (use MongoDB overrides)
- Don't cache config values long-term (defeats hot reload)
- Don't update configs without audit trail (`updated_by` field)

## Examples

### Feature Toggle

```go
func ProcessEvent(event Event) error {
    if !settings.GetBool("features.new_processor.enable") {
        return nil // Feature disabled, skip
    }
    
    // Process event with new logic
    return processWithNewLogic(event)
}
```

### Channel ID Configuration

```go
func SendNotification(msg string) error {
    channelID := settings.GetString("features.notifications.channel_id")
    if channelID == "" {
        return errors.New("notification channel not configured")
    }
    
    return bot.SendMessage(channelID, msg)
}
```

### Dynamic Rate Limits

```go
func RateLimit() int {
    limit := settings.GetIntWithDefault("features.api.rate_limit", 100)
    return limit
}
```

### Role-Based Access Control

```go
func IsAllowed(userRoles []string) bool {
    allowedRoles := settings.GetStringSlice("features.admin_panel.allowed_roles")
    
    for _, userRole := range userRoles {
        for _, allowedRole := range allowedRoles {
            if userRole == allowedRole {
                return true
            }
        }
    }
    return false
}
```

## Troubleshooting

### Config changes not applying

1. Check dashboard shows "Database: Connected"
2. Wait 30 seconds for auto-sync
3. Or trigger manual reload: `hybridconfig.Get().Reload()`
4. Check override is marked with green "✓ Override" badge

### Config not found

1. Check key exists in `defaults.toml`
2. Use `IsSet()` to verify: `settings.IsSet("my.key")`
3. Check for typos (keys are case-sensitive)

### Type conversion errors

1. Use correct getter for type: `GetBool`, `GetInt`, `GetString`, `GetStringSlice`
2. For ambiguous types, use `GetString` and parse manually
3. Check MongoDB value is correct type

## Migration Checklist

When migrating code to use hybrid config:

- [ ] Add config keys to `defaults.toml`
- [ ] Replace hardcoded values with `settings.Get*()` calls
- [ ] Add config keys to dashboard (optional)
- [ ] Test with both default and override values
- [ ] Document config keys in feature README
- [ ] Remove old environment variable usage (if any)

## Summary

The hybrid config system is now the **single source of configuration** for sol-bot:

- ✅ All existing code automatically uses it via `settings` package
- ✅ New code can use `hybridconfig.Get()` directly
- ✅ Hot reload without restart
- ✅ Web UI for easy management
- ✅ Full audit trail
- ✅ Disaster recovery with `defaults.toml` backup

**No code changes required** - everything using `settings` package already works with the hybrid system!
