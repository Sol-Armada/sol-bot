# Sol-Bot

Discord bot for Sol Armada organization management.

## Configuration

### MongoDB Connection

MongoDB connection settings are read from **environment variables first**, falling back to TOML configuration files if not set.

#### Environment Variables

```bash
# MongoDB Host (default: localhost)
export MONGO_HOST=localhost

# MongoDB Port (default: 27017)
export MONGO_PORT=27017

# MongoDB Username (optional)
export MONGO_USERNAME=your_username

# MongoDB Password (optional, @ symbols will be automatically URL encoded)
export MONGO_PASSWORD=your_password

# MongoDB Database Name (default: org)
export MONGO_DATABASE=solarmada

# MongoDB Replica Set Name (optional)
export MONGO_REPLICA_SET_NAME=rs0
```

**Alternative naming convention** (also supported):
- `MONGODB_HOST` instead of `MONGO_HOST`
- `MONGODB_PORT` instead of `MONGO_PORT`
- etc.

#### TOML Configuration

If environment variables are not set, MongoDB settings can be configured in `settings.toml` or environment-specific files like `settings.staging.toml`:

```toml
[mongo]
host = "localhost"
port = "27017"
database = "solarmada"
```

For sensitive credentials, **always use environment variables** rather than committing them to TOML files.

### Other Configuration

See `settings.example.toml` for all available configuration options.

## Deployment

### Systemd Service

```bash
# Copy service file
sudo cp solbot.service /etc/systemd/system/

# Edit to add MongoDB environment variables
sudo systemctl edit solbot.service

# Enable and start
sudo systemctl enable solbot
sudo systemctl start solbot
```

### Docker Compose

```bash
# Update environment variables in compose.yaml
docker compose up -d
```

## Development

```bash
# Build
go build -o bin/solbot ./cmd/

# Run with environment variables
MONGO_HOST=localhost MONGO_DATABASE=solarmada ./bin/solbot
```

---

## Legacy Deployment Notes

### PowerShell Deployment Example

```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o bin/admin ./cmd/ `
&& ssh root@192.168.71.206 "supervisorctl stop all" `
&& scp ./bin/admin root@192.168.71.206:/opt/admin/admin `
&& scp ./settings.prod.toml root@192.168.71.206:/opt/admin/settings.toml `
&& ssh root@192.168.71.206 "supervisorctl start all"
```
