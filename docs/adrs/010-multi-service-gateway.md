# ADR-010: Multi-Service Gateway Architecture

## Status

Proposed

## Context

MCPEG needs to provide MCP access to various backend services. We face a fundamental design choice:
1. Single service per gateway instance (ultra-lightweight)
2. Multiple services per gateway instance (more complex but practical)

Services we need to support include:
- Databases (MySQL, PostgreSQL, MongoDB)
- Message queues (Kafka, RabbitMQ)
- Orchestration (Kubernetes, Docker)
- Service mesh (Istio)
- Secrets management (Vault)
- Issue tracking (Jira)
- Local scripts and binaries
- External APIs (weather, geocoding, etc.)

## Decision

We will implement MCPEG as a **multi-service gateway** that can handle multiple backend services in a single instance, while keeping the architecture lightweight and maintainable.

## Design Principles to Navigate the Rabbit Hole

### 1. Service Isolation
Each service adapter runs in its own goroutine with isolated:
- Configuration namespace
- Circuit breaker
- Connection pool
- Error handling

```go
type ServiceAdapter interface {
    Name() string
    Initialize(config ServiceConfig) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    GetTools() []Tool
    GetResources() []Resource
}
```

### 2. Lazy Loading
Services are only initialized when:
- Explicitly enabled in configuration
- First request arrives for that service
- Dependencies are satisfied

### 3. Graceful Degradation
If one service fails:
- Other services continue operating
- Failed service enters circuit breaker state
- Clear error messages indicate which service failed

### 4. Configuration Hierarchy
```yaml
# Global settings
global:
  timeout: 30s
  memory_limit: 512MB

# Service-specific overrides
services:
  mysql:
    enabled: true
    timeout: 60s  # Override for slow queries
    config:
      host: localhost
      port: 3306
  
  weather:
    enabled: true
    # Uses global timeout
    config:
      api_key: ${WEATHER_API_KEY}
```

## Implementation Strategy

### Phase 1: Core Multi-Service Framework
```go
type Gateway struct {
    services   map[string]ServiceAdapter
    router     *MCPRouter
    config     *Config
    logger     logging.Logger
    pool       *concurrency.WorkerPool
    breakers   *concurrency.CircuitBreakerGroup
}
```

### Phase 2: Service Registry
```go
type ServiceRegistry struct {
    factories map[string]ServiceFactory
}

type ServiceFactory func(config ServiceConfig) (ServiceAdapter, error)

// Register built-in services
registry.Register("mysql", NewMySQLAdapter)
registry.Register("rest", NewRESTAdapter)
registry.Register("script", NewScriptAdapter)
```

### Phase 3: Dynamic Service Management
- Enable/disable services at runtime
- Hot reload configuration
- Service health monitoring
- Automatic recovery

## Configuration Examples

### Simple Configuration
```yaml
services:
  mysql:
    enabled: true
    dsn: "user:pass@tcp(localhost:3306)/dbname"
```

### Advanced Configuration
```yaml
services:
  mysql:
    enabled: true
    type: database
    driver: mysql
    pool:
      max_connections: 10
      idle_connections: 2
    circuit_breaker:
      threshold: 5
      timeout: 60s
    config:
      dsn: ${MYSQL_DSN}
      timeout: 30s
    
  vault:
    enabled: true
    type: secrets
    driver: hashicorp-vault
    config:
      address: https://vault.service.consul:8200
      token: ${VAULT_TOKEN}
      namespace: myapp
    
  weather:
    enabled: false  # Can enable without restart
    type: rest
    config:
      base_url: https://api.weather.gov
      cache_ttl: 5m
```

## Consequences

### Positive
- Single deployment for all integrations
- Shared infrastructure reduces overhead
- Dynamic service management
- Better resource utilization
- Unified logging and monitoring
- Simpler operational model

### Negative
- More complex codebase
- Potential for service interference
- Need careful resource management
- Configuration complexity grows
- Single point of failure risk

### Mitigations
- Strong service isolation patterns
- Comprehensive testing per service
- Resource limits per service
- Circuit breakers prevent cascade failures
- Clear service boundaries in code

## Graceful Navigation Strategies

1. **Start Simple**: Begin with 2-3 core services (MySQL, REST, Script)
2. **Prove Patterns**: Ensure isolation works before adding services
3. **Monitor Everything**: Detailed per-service metrics
4. **Fail Fast**: Clear errors when service misbehaves
5. **Document Clearly**: Each service adapter has own docs

## Future Considerations

- Plugin system for external service adapters
- Service marketplace for community adapters
- Automatic service discovery
- Multi-tenant support
- Service composition (combining multiple services)

## References
- [Microservices Gateway Pattern](https://microservices.io/patterns/apigateway.html)
- [Circuit Breaker Pattern](https://martinfowler.com/bliki/CircuitBreaker.html)
- [Go Plugin System](https://golang.org/pkg/plugin/) (for future plugin support)