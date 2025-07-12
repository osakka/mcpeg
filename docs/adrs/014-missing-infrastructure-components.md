# ADR-014: Missing Infrastructure Components

## Status

**ACCEPTED** - *2025-07-11*

## Context

We have established memory, logging, metrics, and configuration as core infrastructure. However, production systems typically require additional boilerplate components that we haven't addressed yet.

## Missing Infrastructure Analysis

### 1. **Error Handling & Recovery** 🔥 CRITICAL
We need standardized error handling across all components:
- Error categorization and propagation
- Automatic retry mechanisms with backoff
- Graceful degradation strategies
- Error correlation across services

### 2. **Health Checks & Readiness** 🔥 CRITICAL  
Production systems need health monitoring:
- Liveness probes (is the service running?)
- Readiness probes (can it handle requests?)
- Dependency health checks
- Health check aggregation

### 3. **Security Infrastructure** 🔥 CRITICAL
- Authentication framework
- Authorization/RBAC
- Request validation and sanitization  
- Rate limiting and DDoS protection
- Audit trails

### 4. **API Generation & Routing** (Your question!)
- Schema-driven router generation
- Automatic validation from schemas
- OpenAPI spec generation
- Client SDK generation

### 5. **Context Propagation** 
- Request context across service boundaries
- Distributed tracing
- Correlation IDs
- Timeout propagation

### 6. **Event System**
- Internal pub/sub for component communication
- Service lifecycle events
- Configuration change events
- Health state changes

### 7. **Validation Framework**
- Schema validation (JSON Schema, Go struct validation)
- Business rule validation
- Input sanitization
- Output validation

### 8. **Cache Infrastructure**
- Multi-level caching (in-memory, distributed)
- Cache invalidation strategies
- Cache warming
- Cache metrics

### 9. **Background Jobs/Workers**
- Task queue system
- Scheduled jobs (cron-like)
- Long-running background processes
- Job monitoring and retries

### 10. **Resource Management**
- Connection pooling
- Resource limits enforcement
- Cleanup mechanisms
- Resource lifecycle management

## Decision

Implement these in priority order:

**Phase 1: Critical Infrastructure**
1. Error Handling & Recovery
2. Health Checks & Readiness
3. API Generation & Routing
4. Security Infrastructure

**Phase 2: Operational Infrastructure**  
5. Context Propagation
6. Validation Framework
7. Event System

**Phase 3: Performance Infrastructure**
8. Cache Infrastructure  
9. Background Jobs
10. Resource Management

## Implementation Approach

### 1. Error Handling Framework

```go
// pkg/errors/errors.go
package errors

type ErrorCategory string

const (
    CategoryValidation    ErrorCategory = "validation"
    CategoryAuthentication ErrorCategory = "authentication" 
    CategoryAuthorization  ErrorCategory = "authorization"
    CategoryRateLimit     ErrorCategory = "rate_limit"
    CategoryTimeout       ErrorCategory = "timeout"
    CategoryUnavailable   ErrorCategory = "unavailable"
    CategoryInternal      ErrorCategory = "internal"
)

type MCPError struct {
    Code        int               `json:"code"`
    Message     string            `json:"message"`
    Category    ErrorCategory     `json:"category"`
    Service     string            `json:"service"`
    Operation   string            `json:"operation"`
    Context     map[string]interface{} `json:"context"`
    Suggestions []string          `json:"suggestions"`
    Retryable   bool              `json:"retryable"`
    Cause       error             `json:"cause,omitempty"`
    TraceID     string            `json:"trace_id"`
    Timestamp   time.Time         `json:"timestamp"`
}

func (e *MCPError) Error() string {
    return fmt.Sprintf("[%s] %s: %s", e.Service, e.Category, e.Message)
}

// Standardized error constructors
func ValidationError(service, operation, message string, context map[string]interface{}) *MCPError
func TimeoutError(service, operation string, timeout time.Duration) *MCPError
func UnavailableError(service string, cause error) *MCPError
```

### 2. Health Check System

```go
// pkg/health/health.go
package health

type HealthStatus string

const (
    StatusHealthy   HealthStatus = "healthy"
    StatusDegraded  HealthStatus = "degraded" 
    StatusUnhealthy HealthStatus = "unhealthy"
)

type HealthChecker interface {
    Name() string
    Check(ctx context.Context) HealthResult
}

type HealthResult struct {
    Status      HealthStatus           `json:"status"`
    Message     string                 `json:"message"`
    Details     map[string]interface{} `json:"details"`
    Duration    time.Duration          `json:"duration"`
    Timestamp   time.Time             `json:"timestamp"`
}

type HealthManager struct {
    checkers []HealthChecker
    cache    map[string]HealthResult
    interval time.Duration
}

// Health endpoints
// GET /health - overall health
// GET /health/live - liveness probe
// GET /health/ready - readiness probe  
// GET /health/detailed - detailed status
```

### 3. API Generation Framework

```go
// pkg/codegen/router.go
package codegen

// Generate router from MCP schema
type RouterGenerator struct {
    schema MCPSchema
    config GeneratorConfig
}

func (rg *RouterGenerator) GenerateRouter() (string, error) {
    // Generate Go code for:
    // - Route handlers
    // - Validation middleware
    // - Error handling
    // - Metrics collection
    // - Logging integration
}

// Generated router example:
func GeneratedRouter(adapters map[string]adapter.ServiceAdapter) *mux.Router {
    r := mux.NewRouter()
    
    // Auto-generated from schema
    r.HandleFunc("/mcp/v1/tools/call", 
        withMetrics(
            withValidation(
                withAuth(
                    handleToolCall(adapters)))))
    
    return r
}
```

### 4. Security Framework

```go
// pkg/security/auth.go
package security

type AuthProvider interface {
    Authenticate(ctx context.Context, token string) (*Principal, error)
    Authorize(ctx context.Context, principal *Principal, resource, action string) error
}

type Principal struct {
    ID          string            `json:"id"`
    Type        string            `json:"type"` // "user", "service", "api_key"
    Permissions []Permission      `json:"permissions"`
    Metadata    map[string]interface{} `json:"metadata"`
}

type Permission struct {
    Resource string   `json:"resource"` // "mysql:query", "vault:read"
    Actions  []string `json:"actions"`  // ["read", "write", "execute"]
}

// Security middleware
func WithAuth(authProvider AuthProvider) Middleware
func WithRateLimit(config RateLimitConfig) Middleware
func WithValidation(schema ValidationSchema) Middleware
```

## Configuration Integration

All new components follow our configuration pattern:

```yaml
# mcpeg.yaml additions
infrastructure:
  error_handling:
    retry:
      max_attempts: 3
      backoff: "exponential"
      initial_delay: "100ms"
      max_delay: "10s"
    
    circuit_breaker:
      enabled: true
      failure_threshold: 5
      recovery_timeout: "30s"
  
  health:
    interval: "30s"
    timeout: "5s"
    endpoints:
      live: "/health/live"
      ready: "/health/ready"
      detailed: "/health"
  
  security:
    auth:
      provider: "jwt"  # jwt, api_key, mtls, none
      jwt:
        secret: "${JWT_SECRET}"
        expiry: "24h"
    
    rate_limit:
      enabled: true
      requests_per_minute: 100
      burst: 10
    
    validation:
      strict_mode: true
      sanitize_input: true
  
  api_generation:
    auto_generate: true
    output_path: "internal/generated"
    formats: ["go", "openapi", "client_sdk"]
  
  context:
    tracing:
      enabled: true
      provider: "jaeger"  # jaeger, zipkin, none
      endpoint: "${JAEGER_ENDPOINT}"
    
    timeout:
      default: "30s"
      max: "5m"
```

## LLM-Optimized Error Context

All errors include LLM-friendly context:

```json
{
  "error": {
    "code": -32001,
    "message": "Database connection timeout",
    "category": "timeout", 
    "service": "mysql",
    "operation": "query_database",
    "context": {
      "query": "SELECT * FROM users WHERE...",
      "timeout_ms": 30000,
      "connection_pool_status": "exhausted",
      "active_connections": 10,
      "queued_requests": 15
    },
    "suggestions": [
      "Increase connection pool size",
      "Optimize slow queries",
      "Add query timeout limits",
      "Consider read replicas"
    ],
    "retryable": true,
    "health_impact": "service_degraded",
    "related_metrics": {
      "avg_query_time_ms": 2500,
      "connection_pool_utilization": 100,
      "error_rate_last_5min": 0.15
    }
  }
}
```

## Missing Component Priority

**Absolutely Critical (Can't ship without):**
1. Error Handling Framework
2. Health Checks  
3. Security Infrastructure
4. API Generation

**Important (Should have for production):**
5. Context Propagation
6. Validation Framework
7. Event System

**Nice to Have (Can add later):**
8. Advanced Caching
9. Background Jobs
10. Resource Management

## Consequences

### Positive
- Complete production-ready infrastructure
- Consistent patterns across all components  
- LLM-optimized troubleshooting for everything
- API-first development fully realized

### Negative
- Significant implementation work
- More complexity to manage
- Learning curve for developers

### Neutral
- Standard production system components
- Well-understood patterns

## Implementation Strategy

Build incrementally, each component as infrastructure:
1. Start with error handling (needed by everything)
2. Add health checks (needed for deployment)
3. Implement security (needed for production)
4. Build API generation (enables rapid development)

Each component follows our established patterns:
- LLM-optimized logging
- Comprehensive metrics
- Configuration-driven
- XVC methodology