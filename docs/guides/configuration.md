# Configuration Guide

Complete guide to configuring MCpeg for different environments and use cases.

## Overview

MCpeg uses a hierarchical configuration system that supports:
- **YAML Configuration Files** - Primary configuration method
- **Environment Variables** - Override any configuration value
- **Command Line Flags** - Runtime configuration options
- **Development Overrides** - Special settings for development mode

## Configuration Hierarchy

Configuration is applied in this order (later overrides earlier):

1. **Default Values** - Built-in defaults
2. **Configuration File** - YAML file specified with `--config`
3. **Environment Variables** - `MCPEG_` prefixed variables
4. **Command Line Flags** - Runtime flags
5. **Development Mode** - Special overrides when `--dev` flag is used

## Configuration Files

### Basic Structure

```yaml
# config/example.yaml
server:
  address: "0.0.0.0:8080"
  tls:
    enabled: false
    cert_path: ""
    key_path: ""
  timeouts:
    read: "10s"
    write: "10s"
    idle: "60s"

logging:
  level: "info"
  format: "json"
  output: "stdout"

auth:
  jwt:
    enabled: false
    secret_key: ""
    issuer: "mcpeg"
    expiration: "24h"

plugins:
  memory:
    enabled: true
    data_file: "data/memory_storage.json"
  git:
    enabled: true
    working_dir: "."
  editor:
    enabled: true
    max_file_size: 10485760

metrics:
  enabled: true
  path: "/metrics"
  
health:
  enabled: true
  path: "/health"
```

### Development Configuration

```yaml
# config/development.yaml
server:
  address: "0.0.0.0:8080"
  tls:
    enabled: false

logging:
  level: "debug"
  format: "json"

development:
  enabled: true
  hot_reload: true
  debug_endpoints: true

plugins:
  memory:
    enabled: true
    data_file: "data/memory_storage.json"
  git:
    enabled: true
    working_dir: "."
  editor:
    enabled: true
    max_file_size: 10485760
    allowed_extensions:
      - ".go"
      - ".js"
      - ".py"
      - ".md"
      - ".yaml"
      - ".yml"
      - ".json"
```

### Production Configuration

```yaml
# config/production.yaml
server:
  address: "0.0.0.0:8080"
  tls:
    enabled: true
    cert_path: "/etc/ssl/certs/mcpeg.pem"
    key_path: "/etc/ssl/private/mcpeg.key"
  timeouts:
    read: "30s"
    write: "30s"
    idle: "120s"

logging:
  level: "info"
  format: "json"
  output: "/var/log/mcpeg/gateway.log"

auth:
  jwt:
    enabled: true
    secret_key: "${MCPEG_JWT_SECRET}"
    issuer: "mcpeg-production"
    expiration: "8h"

plugins:
  memory:
    enabled: true
    data_file: "/var/lib/mcpeg/memory.json"
  git:
    enabled: true
    working_dir: "/var/lib/mcpeg/workspace"
  editor:
    enabled: true
    max_file_size: 52428800  # 50MB
    base_path: "/var/lib/mcpeg/workspace"

metrics:
  enabled: true
  path: "/metrics"
  
health:
  enabled: true
  path: "/health"

rbac:
  enabled: true
  policy_file: "/etc/mcpeg/rbac.yaml"
```

## Environment Variables

All configuration values can be overridden with environment variables using the `MCPEG_` prefix and underscore-separated paths:

### Server Configuration
```bash
export MCPEG_SERVER_ADDRESS="0.0.0.0:9000"
export MCPEG_SERVER_TLS_ENABLED="true"
export MCPEG_SERVER_TLS_CERT_PATH="/path/to/cert.pem"
export MCPEG_SERVER_TLS_KEY_PATH="/path/to/key.pem"
```

### Logging Configuration
```bash
export MCPEG_LOGGING_LEVEL="debug"
export MCPEG_LOGGING_FORMAT="text"
export MCPEG_LOGGING_OUTPUT="/var/log/mcpeg.log"
```

### Authentication Configuration
```bash
export MCPEG_AUTH_JWT_ENABLED="true"
export MCPEG_AUTH_JWT_SECRET_KEY="your-secret-key"
export MCPEG_AUTH_JWT_ISSUER="mcpeg"
export MCPEG_AUTH_JWT_EXPIRATION="24h"
```

### Plugin Configuration
```bash
export MCPEG_PLUGINS_MEMORY_ENABLED="true"
export MCPEG_PLUGINS_MEMORY_DATA_FILE="/custom/path/memory.json"
export MCPEG_PLUGINS_GIT_ENABLED="true"
export MCPEG_PLUGINS_GIT_WORKING_DIR="/workspace"
export MCPEG_PLUGINS_EDITOR_ENABLED="true"
export MCPEG_PLUGINS_EDITOR_MAX_FILE_SIZE="10485760"
```

## Command Line Flags

### Common Flags

```bash
# Configuration file
mcpeg gateway --config config/production.yaml

# Development mode
mcpeg gateway --dev

# Log level
mcpeg gateway --log-level debug

# Server address
mcpeg gateway --address 0.0.0.0:9000

# Enable TLS
mcpeg gateway --tls --cert-path /path/to/cert.pem --key-path /path/to/key.pem

# Daemon mode
mcpeg gateway --daemon

# Plugin management
mcpeg gateway --list-plugins
mcpeg gateway --disable-plugin memory
mcpeg gateway --enable-plugin git
```

### Validation and Testing

```bash
# Validate configuration
mcpeg validate --config config/production.yaml

# Dry run (validate without starting)
mcpeg gateway --config config/production.yaml --dry-run

# Test specific plugin
mcpeg gateway --test-plugin memory
```

## Plugin Configuration

### Memory Plugin

```yaml
plugins:
  memory:
    enabled: true
    data_file: "data/memory_storage.json"
    cache_size: 1000
    persist_interval: "5m"
    max_key_size: 1024
    max_value_size: 1048576  # 1MB
    compression: true
```

### Git Plugin

```yaml
plugins:
  git:
    enabled: true
    working_dir: "."
    max_repo_size: 1073741824  # 1GB
    allowed_commands:
      - "status"
      - "diff"
      - "log"
      - "commit"
      - "branch"
    timeout: "30s"
```

### Editor Plugin

```yaml
plugins:
  editor:
    enabled: true
    max_file_size: 10485760  # 10MB
    base_path: "."
    allowed_extensions:
      - ".go"
      - ".js"
      - ".py"
      - ".md"
      - ".yaml"
      - ".yml"
      - ".json"
      - ".txt"
    forbidden_paths:
      - "/etc"
      - "/var"
      - "/usr"
    backup_enabled: true
    backup_dir: "backups"
```

## Security Configuration

### JWT Authentication

```yaml
auth:
  jwt:
    enabled: true
    secret_key: "your-256-bit-secret-key"
    issuer: "mcpeg"
    expiration: "24h"
    algorithm: "HS256"
    claims:
      - "user_id"
      - "permissions"
```

### TLS Configuration

```yaml
server:
  tls:
    enabled: true
    cert_path: "/etc/ssl/certs/mcpeg.pem"
    key_path: "/etc/ssl/private/mcpeg.key"
    min_version: "1.2"
    cipher_suites:
      - "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
      - "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
```

### RBAC Configuration

```yaml
rbac:
  enabled: true
  policy_file: "/etc/mcpeg/rbac.yaml"
  default_policy: "readonly"
  admin_users:
    - "admin@example.com"
```

RBAC Policy File (`rbac.yaml`):
```yaml
policies:
  - name: "readonly"
    permissions:
      - "tools:read"
      - "resources:read"
      - "prompts:read"
  
  - name: "editor"
    permissions:
      - "tools:*"
      - "resources:read"
      - "prompts:*"
  
  - name: "admin"
    permissions:
      - "*"

roles:
  - name: "viewer"
    policies: ["readonly"]
  
  - name: "developer"
    policies: ["editor"]
  
  - name: "administrator"
    policies: ["admin"]
```

## Performance Configuration

### Resource Limits

```yaml
server:
  max_concurrent_requests: 100
  request_timeout: "30s"
  read_timeout: "10s"
  write_timeout: "10s"
  idle_timeout: "60s"
  max_header_bytes: 1048576  # 1MB

performance:
  worker_pool_size: 10
  queue_size: 1000
  gc_target_percentage: 75
  max_memory_usage: 1073741824  # 1GB
```

### Caching Configuration

```yaml
cache:
  enabled: true
  type: "memory"  # memory, redis, memcached
  ttl: "5m"
  max_size: 10000
  
  # Redis configuration (if type: redis)
  redis:
    address: "localhost:6379"
    password: ""
    db: 0
    pool_size: 10
```

## Monitoring Configuration

### Metrics

```yaml
metrics:
  enabled: true
  path: "/metrics"
  format: "prometheus"
  collection_interval: "15s"
  custom_metrics:
    - name: "custom_counter"
      type: "counter"
      help: "Custom counter metric"
```

### Health Checks

```yaml
health:
  enabled: true
  path: "/health"
  interval: "30s"
  timeout: "5s"
  checks:
    - name: "database"
      type: "tcp"
      target: "localhost:5432"
    - name: "external_service"
      type: "http"
      target: "https://api.example.com/health"
```

### Logging Configuration

```yaml
logging:
  level: "info"  # debug, info, warn, error
  format: "json"  # json, text
  output: "stdout"  # stdout, stderr, file path
  
  # File output configuration
  file:
    path: "/var/log/mcpeg/gateway.log"
    max_size: 100  # MB
    max_backups: 3
    max_age: 28  # days
    compress: true
  
  # Custom fields
  fields:
    service: "mcpeg"
    version: "1.0.0"
    environment: "production"
```

## Development Configuration

### Hot Reload

```yaml
development:
  enabled: true
  hot_reload: true
  watch_paths:
    - "config/"
    - "plugins/"
  reload_delay: "1s"
```

### Debug Endpoints

```yaml
development:
  debug_endpoints: true
  endpoints:
    - "/debug/pprof"
    - "/debug/vars"
    - "/debug/config"
```

## Configuration Validation

### Validate Configuration File

```bash
mcpeg validate --config config/production.yaml
```

### Schema Validation

MCpeg validates configuration against a JSON schema:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "server": {
      "type": "object",
      "properties": {
        "address": {
          "type": "string",
          "pattern": "^[0-9.]+:[0-9]+$"
        },
        "tls": {
          "type": "object",
          "properties": {
            "enabled": {"type": "boolean"},
            "cert_path": {"type": "string"},
            "key_path": {"type": "string"}
          }
        }
      },
      "required": ["address"]
    }
  }
}
```

## Environment-Specific Configurations

### Docker Configuration

```yaml
# config/docker.yaml
server:
  address: "0.0.0.0:8080"
  
logging:
  level: "info"
  format: "json"
  output: "stdout"

plugins:
  memory:
    enabled: true
    data_file: "/app/data/memory.json"
  git:
    enabled: true
    working_dir: "/app/workspace"
  editor:
    enabled: true
    base_path: "/app/workspace"
```

### Kubernetes Configuration

```yaml
# config/kubernetes.yaml
server:
  address: "0.0.0.0:8080"
  
logging:
  level: "info"
  format: "json"
  output: "stdout"

auth:
  jwt:
    enabled: true
    secret_key: "${JWT_SECRET}"

plugins:
  memory:
    enabled: true
    data_file: "/data/memory.json"
  
health:
  enabled: true
  path: "/health"
  
metrics:
  enabled: true
  path: "/metrics"
```

## Best Practices

### Configuration Management

1. **Use Environment-Specific Files:** Separate configs for dev, staging, production
2. **Environment Variables for Secrets:** Never commit secrets to version control
3. **Validate Before Deployment:** Always run `mcpeg validate` before deployment
4. **Version Control:** Keep configuration files in version control
5. **Documentation:** Document all custom configuration options

### Security

1. **TLS in Production:** Always enable TLS for production deployments
2. **Strong JWT Secrets:** Use 256-bit random secrets for JWT
3. **Least Privilege:** Configure RBAC with minimal necessary permissions
4. **Regular Rotation:** Rotate secrets and certificates regularly

### Performance

1. **Resource Limits:** Set appropriate limits for memory and CPU
2. **Connection Pooling:** Configure connection pools for external services
3. **Caching:** Enable caching for frequently accessed data
4. **Monitoring:** Set up comprehensive monitoring and alerting

## Troubleshooting

### Common Configuration Issues

1. **Invalid YAML Syntax:**
   ```bash
   mcpeg validate --config config/production.yaml
   ```

2. **Missing Required Fields:**
   ```yaml
   server:
     address: "0.0.0.0:8080"  # Required
   ```

3. **Port Conflicts:**
   ```yaml
   server:
     address: "0.0.0.0:8081"  # Use different port
   ```

4. **File Permissions:**
   ```bash
   chmod 600 config/production.yaml
   chown mcpeg:mcpeg config/production.yaml
   ```

### Configuration Debugging

```bash
# View effective configuration
mcpeg gateway --config config/production.yaml --dry-run

# Debug environment variables
env | grep MCPEG_

# Test configuration
mcpeg validate --config config/production.yaml --verbose
```

## Examples

### Complete Production Configuration

See `config/production.yaml` for a complete production-ready configuration example.

### Docker Compose Configuration

```yaml
version: '3.8'
services:
  mcpeg:
    image: mcpeg/mcpeg:latest
    ports:
      - "8080:8080"
    environment:
      - MCPEG_LOGGING_LEVEL=info
      - MCPEG_AUTH_JWT_ENABLED=true
      - MCPEG_AUTH_JWT_SECRET_KEY=${JWT_SECRET}
    volumes:
      - ./config:/app/config
      - ./data:/app/data
    command: gateway --config /app/config/docker.yaml
```

## Next Steps

- **Installation:** [Installation Guide](installation.md)
- **Usage:** [User Guide](user-guide.md)
- **API Reference:** [API Documentation](../reference/api-reference.md)
- **Plugin Development:** [Plugin Development Guide](plugin-development.md)
- **Troubleshooting:** [Troubleshooting Guide](troubleshooting.md)