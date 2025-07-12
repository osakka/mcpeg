# MCPEG Configuration

This directory contains all configuration files for MCPEG.

## Structure

```
config/
├── mcpeg.yaml           # Main configuration
├── services/            # Service-specific configurations
│   ├── mysql.yaml      # MySQL database adapter
│   ├── vault.yaml      # HashiCorp Vault adapter
│   ├── weather.yaml    # Weather API adapter
│   └── script.yaml     # Script execution adapter
└── secrets/             # Local secrets (development only)
    └── api-keys.yaml   # API keys and tokens
```

## Configuration Loading

MCPEG loads configuration in this order (later overrides earlier):

1. **Base Configuration**: `mcpeg.yaml`
2. **Environment Override**: `mcpeg-{environment}.yaml` (if exists)
3. **Environment Variables**: `${VAR_NAME:default_value}`
4. **Command Line Flags**: `--config-override key=value`

## Environment Variables

Configuration supports environment variable substitution:

```yaml
connection:
  host: "${MYSQL_HOST:localhost}"      # Default to localhost
  password: "${MYSQL_PASSWORD}"        # Required, no default
  timeout: "${MYSQL_TIMEOUT:30s}"     # Default to 30s
```

## Service Configuration

Each service has its own configuration file in the `services/` directory:

- **mysql.yaml**: Database connection, tools, resources, prompts
- **vault.yaml**: Vault connection, secret management
- **weather.yaml**: Weather API configuration and mappings
- **script.yaml**: Script execution configuration and safety

## Security

### Secrets Management

**Development**: Store in `secrets/api-keys.yaml` (NOT committed to Git)
**Production**: Use environment variables or external secret stores

```yaml
# secrets/api-keys.yaml (development only)
weather_api_key: "your-api-key"
vault_token: "your-vault-token"
```

### File Permissions

Ensure proper file permissions:
```bash
chmod 600 secrets/*.yaml    # Secrets readable only by owner
chmod 644 *.yaml           # Config files readable by group
chmod 644 services/*.yaml  # Service configs readable by group
```

## Configuration Validation

MCPEG validates configuration on startup:

- Schema validation against JSON schemas
- Service dependency checks
- Connection testing (if enabled)
- Secret availability verification

## Hot Reload

Configuration can be reloaded without restart:

```bash
# Send SIGHUP to reload configuration
kill -HUP $(cat mcpeg.pid)

# Or use the management API
curl -X POST http://localhost:8080/v1/admin/reload
```

## Environment-Specific Configurations

Create environment-specific overrides:

```
config/
├── mcpeg.yaml              # Base configuration
├── mcpeg-development.yaml  # Development overrides
├── mcpeg-staging.yaml      # Staging overrides
└── mcpeg-production.yaml   # Production overrides
```

Example override:
```yaml
# mcpeg-production.yaml
global:
  log_level: "info"          # Override debug in base config

transports:
  http:
    enabled: true            # Enable HTTP in production
    port: 8080

monitoring:
  profiling:
    enabled: false           # Disable profiling in production
```

## Configuration Examples

### Minimal Configuration
```yaml
version: "1.0"
services:
  mysql:
    enabled: true
    config_file: "services/mysql.yaml"
```

### Production Configuration
```yaml
version: "1.0"
metadata:
  name: "mcpeg-prod"
  environment: "production"

global:
  timeout: "30s"
  log_level: "info"

transports:
  stdio:
    enabled: true
  http:
    enabled: true
    port: 8080

services:
  mysql:
    enabled: true
    config_file: "services/mysql.yaml"
  vault:
    enabled: true
    config_file: "services/vault.yaml"

storage:
  runtime:
    type: "sqlite"
    path: "/var/lib/mcpeg/runtime.db"
  logs:
    type: "file"
    path: "/var/log/mcpeg"

security:
  secrets:
    provider: "vault"
    vault_path: "secret/mcpeg"
```

## Troubleshooting

### Common Issues

1. **Configuration not found**: Check file paths and permissions
2. **Environment variables not resolved**: Verify variable names and defaults
3. **Service won't start**: Check service-specific configuration
4. **Secrets not loading**: Verify secrets provider configuration

### Debug Configuration

Enable configuration debugging:
```yaml
development:
  debug_config: true    # Log loaded configuration (masks secrets)
```

### Validation

Validate configuration without starting:
```bash
mcpeg validate --config config/mcpeg.yaml
```