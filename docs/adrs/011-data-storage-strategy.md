# ADR-011: Data Storage Strategy

## Status

Proposed

## Context

MCPEG needs to store different types of data:

1. **Configuration Data**: Service definitions, connection details, mappings
2. **Runtime Data**: Metrics, state, circuit breaker status
3. **Cache Data**: Responses, schemas, temporary data
4. **Secrets**: API keys, passwords, certificates
5. **Logs**: Request/response logs, audit trails

We need to decide where and how to store this data, balancing simplicity, security, and operational requirements.

## Decision

We will use a **layered storage approach** starting simple and evolving as needed:

### Phase 1: Configuration-First (Initial Implementation)

**Primary Storage: YAML Configuration Files**
```
config/
├── mcpeg.yaml           # Main configuration
├── services/            # Service-specific configs
│   ├── mysql.yaml
│   ├── vault.yaml
│   └── weather.yaml
└── secrets/             # Local secrets (dev only)
    └── api-keys.yaml
```

**Runtime Storage: In-Memory + Local Files**
- Metrics: In-memory with periodic snapshots
- State: In-memory with optional persistence
- Cache: In-memory with configurable TTL
- Logs: Files with rotation

### Phase 2: Hybrid Approach (Production Ready)

**Configuration: Git + External Secrets**
- Service configs in Git repository
- Secrets in external systems (Vault, K8s secrets)
- Environment-specific overlays

**Runtime Storage: Embedded Database**
- SQLite for metrics, state, cache
- File-based for simplicity
- No external dependencies

### Phase 3: Distributed Storage (Scale Out)

**External Systems for Everything**
- Configuration: Consul, etcd, or K8s ConfigMaps
- Secrets: HashiCorp Vault, AWS Secrets Manager
- Runtime: Redis, PostgreSQL
- Logs: Centralized logging (ELK, Fluentd)

## Initial Implementation Details

### Configuration Schema
```yaml
# mcpeg.yaml
version: "1.0"
metadata:
  name: "mcpeg-prod"
  environment: "production"

global:
  timeout: "30s"
  memory_limit_mb: 512
  log_level: "info"

transports:
  stdio:
    enabled: true
  http:
    enabled: false
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
    type: "memory"
    persistence: true
    snapshot_interval: "5m"
  
  cache:
    type: "memory"
    max_size_mb: 100
    ttl: "10m"
  
  logs:
    type: "file"
    path: "/var/log/mcpeg"
    rotation: "daily"
    retention: "30d"
```

### Service Configuration
```yaml
# services/mysql.yaml
type: "database"
driver: "mysql"

connection:
  host: "${MYSQL_HOST}"
  port: 3306
  database: "${MYSQL_DATABASE}"
  username: "${MYSQL_USER}"
  password: "${MYSQL_PASSWORD}"

pool:
  max_connections: 10
  idle_connections: 2
  connection_lifetime: "1h"

circuit_breaker:
  enabled: true
  failure_threshold: 5
  reset_timeout: "60s"

tools:
  - name: "query_database"
    description: "Execute SQL queries"
    max_rows: 1000
    timeout: "30s"
  
  - name: "get_schema"
    description: "Get table schema information"
    cache_ttl: "1h"

resources:
  - pattern: "mysql://schema/{table}"
    handler: "get_table_schema"
    cache_ttl: "1h"
  
  - pattern: "mysql://data/{table}"
    handler: "get_sample_data"
    max_rows: 100
```

### Secrets Management
```yaml
# secrets/api-keys.yaml (development only)
weather_api_key: "dev-key-12345"
vault_token: "dev-token-67890"

# Production uses environment variables or external secrets
```

## Directory Structure
```
/opt/mcpeg/
├── config/
│   ├── mcpeg.yaml
│   ├── services/
│   │   ├── mysql.yaml
│   │   ├── vault.yaml
│   │   └── weather.yaml
│   └── secrets/
│       └── api-keys.yaml
├── data/
│   ├── runtime/
│   │   ├── metrics.db
│   │   └── state.json
│   ├── cache/
│   │   └── responses/
│   └── logs/
│       ├── mcpeg.log
│       └── audit.log
├── build/
│   └── mcpeg
└── mcpeg.pid
```

## Configuration Loading Strategy

### 1. Hierarchical Loading
```go
type ConfigLoader struct {
    baseDir     string
    environment string
    overrides   map[string]interface{}
}

// Load order:
// 1. Base configuration (mcpeg.yaml)
// 2. Environment overrides (mcpeg-prod.yaml)
// 3. Environment variables
// 4. Command line flags
```

### 2. Environment Variable Substitution
```yaml
connection:
  host: "${MYSQL_HOST:localhost}"      # Default to localhost
  password: "${MYSQL_PASSWORD}"        # Required, no default
  timeout: "${MYSQL_TIMEOUT:30s}"     # Default to 30s
```

### 3. Hot Reload Support
```go
// Configuration can be reloaded without restart
func (s *Server) ReloadConfig() error {
    newConfig, err := LoadConfig(s.configPath)
    if err != nil {
        return err
    }
    
    return s.ApplyConfig(newConfig)
}
```

## Runtime Data Management

### 1. Metrics Storage
```go
type MetricsStore interface {
    Record(metric string, value float64, tags map[string]string)
    Query(metric string, timeRange TimeRange) ([]DataPoint, error)
    Snapshot() error  // Persist to disk
}

// In-memory with periodic snapshots
type MemoryMetricsStore struct {
    data         map[string][]DataPoint
    snapshotPath string
    interval     time.Duration
}
```

### 2. State Management
```go
type StateStore interface {
    Get(key string) (interface{}, error)
    Set(key string, value interface{}) error
    Delete(key string) error
    Persist() error
}

// Circuit breaker states, service health, etc.
```

### 3. Cache Management
```go
type Cache interface {
    Get(key string) (interface{}, bool)
    Set(key string, value interface{}, ttl time.Duration)
    Delete(key string)
    Clear()
    Stats() CacheStats
}
```

## Security Considerations

### 1. Secrets Handling
- Never store secrets in Git
- Use environment variables or external secret stores
- Support secret rotation
- Audit secret access

### 2. Configuration Validation
- Schema validation on startup
- Encrypt sensitive configuration at rest
- Secure file permissions (600 for secrets)

### 3. Runtime Security
- Limit file system access
- Validate all configuration changes
- Audit configuration modifications

## Migration Path

### Phase 1 → Phase 2
- Add SQLite for runtime data
- External secret management
- Git-based configuration deployment

### Phase 2 → Phase 3
- External configuration stores
- Distributed caching
- Centralized logging

## Consequences

### Positive
- **Simple Start**: YAML files are easy to understand and edit
- **No Dependencies**: No external databases required initially
- **Version Control**: Configuration can be tracked in Git
- **Environment Specific**: Easy to have dev/staging/prod configs
- **Hot Reload**: Configuration changes without restart

### Negative
- **File Management**: Many configuration files to manage
- **Secret Security**: Local secrets are not ideal for production
- **Concurrency**: File-based storage has concurrency limitations
- **Backup**: Need to backup configuration and runtime data

### Mitigation Strategies
- Clear documentation on file organization
- Early migration to external secret management
- Use file locks for concurrent access
- Automated backup strategies

## Future Considerations
- Configuration validation schemas
- Configuration templates and inheritance
- Multi-environment configuration management
- Configuration drift detection
- Automated configuration deployment