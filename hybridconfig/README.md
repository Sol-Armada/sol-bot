# Hybrid Configuration System

The hybrid configuration system combines the safety of file-based defaults with the flexibility of database-driven runtime overrides.

## Architecture

```
┌─────────────┐
│defaults.toml│ (Version controlled baseline)
└──────┬──────┘
       │
       v
┌─────────────────┐      ┌──────────────┐
│ HybridConfig    │◄─────┤MongoDB Store │ (Runtime overrides)
│                 │      └──────────────┘
│ - Load defaults │
│ - Merge overrides│
│ - Hot reload    │
└────────┬────────┘
         │
         v
  ┌──────────────┐
  │  Dashboard   │ (Web UI for editing)
  └──────────────┘
```

## Usage

### Initialization

```go
import "github.com/sol-armada/sol-bot/hybridconfig"

// Initialize during service startup
if err := hybridconfig.Setup(ctx); err != nil {
    log.Fatal("Failed to setup hybrid config:", err)
}
defer hybridconfig.Get().Shutdown()
```

### Reading Configuration

```go
config := hybridconfig.Get()

// Get values (checks overrides first, then defaults)
botToken := config.GetString("discord.bot_token")
debugMode := config.GetBool("log.debug")
port := config.GetInt("dashboard.port")
allowedRoles := config.GetStringSlice("features.attendance.allowed_roles")

// Check if key exists
if config.IsSet("features.monitor.enable") {
    // ...
}

// Get all settings as nested map
allSettings := config.GetAllSettings()

// Get default value (ignoring overrides)
defaultPort := config.GetDefault("dashboard.port")
```

### Setting Overrides

```go
config := hybridconfig.Get()

// Set an override
err := config.SetOverride("features.monitor.enable", true, "admin")
// Changes are saved to MongoDB and applied immediately

// Remove override (revert to default)
err := config.RemoveOverride("features.monitor.enable")
```

### Dashboard Interface

Access the configuration page at: `http://localhost:8080/configs`

Features:
- **View all configs** - See current values, defaults, and override status
- **Edit values** - Click "Edit" to change a config value
- **Reset to default** - Remove overrides and revert to defaults.toml
- **Search** - Filter configurations by key name
- **Export** - Download all configs as JSON
- **Audit trail** - See who changed what and when

## Configuration Precedence

1. **MongoDB Overrides** (highest priority)
   - Set via dashboard or `SetOverride()` API
   - Stored with audit trail (who, when)
   
2. **defaults.toml** (fallback)
   - Version controlled safe defaults
   - Loaded from filesystem at startup

3. **Hardcoded defaults** (last resort)
   - Built into the code

## Hot Reload

The system automatically polls MongoDB every 30 seconds for config changes. You can also trigger immediate reload:

```go
config := hybridconfig.Get()
config.Reload() // Immediately sync with MongoDB
```

## File Structure

```
/
├── defaults.toml              # Default configuration values
├── settings.toml              # Environment-specific settings (not touched by hybrid system)
├── hybridconfig/
│   └── pkg.go                 # Hybrid config implementation
└── dashboard/
    ├── templates/
    │   └── configs.html       # Config management UI
    └── handlers.go            # Config API endpoints
```

## MongoDB Schema

Configs are stored in the `configs` collection:

```json
{
  "name": "features.monitor.enable",
  "value": true,
  "override": true,
  "updated_at": "2026-02-22T10:30:00Z",
  "updated_by": "admin"
}
```

## Best Practices

1. **Keep defaults.toml generic** - No sensitive values, no IDs
2. **Use environment-specific settings.toml** - For deployment secrets
3. **Document config keys** - Add comments in defaults.toml
4. **Test defaults** - Ensure service works with defaults.toml alone
5. **Audit changes** - Always provide meaningful `updated_by` identifier
6. **Export regularly** - Backup overrides using export function

## API Endpoints

- `GET /configs` - Configuration management page
- `POST /api/configs/update` - Update config override
- `POST /api/configs/reset` - Reset config to default
- `GET /api/configs/export` - Export all configs as JSON

## Disaster Recovery

If MongoDB is unavailable:
- Service starts with defaults.toml values
- Dashboard shows "Database: Disconnected"
- Overrides are unavailable but service continues

To recover overrides:
1. Restore MongoDB from backup
2. Or import previously exported JSON
3. Service will auto-reload on next check (30s)

## Migration from Old System

Current code using `settings` package will continue to work. Gradually migrate:

```go
// Old way (still works)
import "github.com/sol-armada/sol-bot/settings"
token := settings.GetString("discord.bot_token")

// New way (with override support)
import "github.com/sol-armada/sol-bot/hybridconfig"
token := hybridconfig.Get().GetString("discord.bot_token")
```

## Security Notes

- Dashboard has no authentication - keep it on internal network only
- Sensitive values in settings.toml are NOT exposed to hybrid system
- defaults.toml should never contain secrets (it's version controlled)
- MongoDB overrides are stored in plaintext - encrypt connection in production
