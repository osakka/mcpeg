# Service Adapters

This package contains the adapter implementations for various backend services.

## Adapter Interface

All adapters implement the `ServiceAdapter` interface:

```go
type ServiceAdapter interface {
    // Metadata
    Name() string
    Type() string
    
    // Lifecycle
    Initialize(config ServiceConfig) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    
    // MCP Integration
    GetTools() []mcp.Tool
    GetResources() []mcp.Resource
    GetPrompts() []mcp.Prompt
    
    // Health
    HealthCheck(ctx context.Context) error
    GetMetrics() AdapterMetrics
}
```

## Service Types

### Database Adapters
- `mysql` - MySQL/MariaDB adapter
- `postgresql` - PostgreSQL adapter
- `mongodb` - MongoDB adapter

### Messaging Adapters
- `kafka` - Apache Kafka adapter
- `rabbitmq` - RabbitMQ adapter

### Infrastructure Adapters
- `kubernetes` - Kubernetes API adapter
- `docker` - Docker API adapter
- `vault` - HashiCorp Vault adapter

### API Adapters
- `rest` - Generic REST API adapter
- `graphql` - GraphQL adapter

### Script Adapters
- `script` - Local script/binary execution

## Adding a New Adapter

1. Create a new package under `adapter/`
2. Implement the `ServiceAdapter` interface
3. Register in the service registry
4. Add configuration schema
5. Write comprehensive tests
6. Document MCP tools/resources provided

## Adapter Isolation

Each adapter:
- Runs in its own goroutine
- Has its own circuit breaker
- Manages its own connections
- Cannot affect other adapters
- Logs with its own component name

## Configuration

Adapters receive configuration through the `ServiceConfig` struct:

```go
type ServiceConfig struct {
    Enabled         bool
    Type           string
    Driver         string
    CircuitBreaker CircuitBreakerConfig
    Pool           PoolConfig
    Custom         map[string]interface{}
}
```