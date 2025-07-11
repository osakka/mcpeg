# Health Check System

The MCPEG health check system provides comprehensive monitoring and reporting of system health with LLM-optimized diagnostics.

## Features

### Core Health Monitoring
- **Multi-level health checks**: System, service, and component level monitoring
- **Real-time status reporting**: Immediate health status with detailed context
- **Background monitoring**: Continuous health assessment with configurable intervals
- **LLM-optimized logging**: Complete context for 100% troubleshooting capability

### Health Check Types
- **Memory Usage**: Monitors system memory consumption with configurable thresholds
- **Goroutine Count**: Tracks goroutine leaks and concurrent operation health
- **System Load**: Monitors system performance and resource utilization
- **Service Health**: Checks individual MCP service availability and responsiveness
- **Database Health**: Validates database connectivity and query performance
- **External API Health**: Monitors external service dependencies

### HTTP Endpoints

#### Basic Health Check
```
GET /health
```
Returns overall system health status with summary information.

#### Liveness Probe (Kubernetes-compatible)
```
GET /health/live
```
Indicates if the application is running and responsive.

#### Readiness Probe (Kubernetes-compatible)
```
GET /health/ready
```
Indicates if the application is ready to serve traffic.

#### Detailed Health Check
```
GET /health/detailed?full=true&debug=true
```
Returns comprehensive health information with all check details.

#### Prometheus Metrics
```
GET /metrics
```
Returns health metrics in Prometheus format.

## Health Status Levels

- **Healthy**: All systems operating normally
- **Degraded**: Some non-critical issues detected, service still functional
- **Unhealthy**: Critical issues detected, service may not function properly
- **Unknown**: Unable to determine health status

## Configuration

### Health Check Configuration
```yaml
health:
  default_timeout: "30s"
  global_timeout: "60s"
  quick_check_interval: "10s"
  full_check_interval: "60s"
  max_consecutive_failures: 3
  failure_retry_delay: "5s"
  degraded_threshold: 0.8    # 80% healthy
  unhealthy_threshold: 0.6   # 60% healthy
  include_detailed_diagnostics: true
  generate_suggestions: true

  # Memory check configuration
  memory:
    warning_threshold: 0.80   # 80%
    critical_threshold: 0.95  # 95%

  # Goroutine check configuration
  goroutines:
    warning_threshold: 1000
    critical_threshold: 5000
```

### Service Health Checks
Register service-specific health checks:

```go
// Register a database health check
dbChecker := health.NewDatabaseHealthChecker(
    "mysql", 
    "mysql://user:pass@host:3306/db",
    30*time.Second,
    logger,
)
healthManager.RegisterChecker(dbChecker)

// Register an external API health check
apiChecker := health.NewExternalAPIHealthChecker(
    "weather_api",
    "https://api.weather.com/health",
    10*time.Second,
    logger,
)
healthManager.RegisterChecker(apiChecker)
```

## Usage Examples

### Basic Health Check
```bash
curl -X GET http://localhost:8080/health
```

Response:
```json
{
  "status": "healthy",
  "timestamp": "2025-07-11T10:30:00Z",
  "version": "1.0.0",
  "uptime": "24h30m15s",
  "summary": {
    "total": 5,
    "healthy": 5,
    "degraded": 0,
    "unhealthy": 0,
    "critical": 0
  },
  "suggestions": []
}
```

### Detailed Health Check
```bash
curl -X GET "http://localhost:8080/health/detailed?full=true"
```

Response includes all individual check results with detailed diagnostics.

### Kubernetes Probes
```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: mcpeg
    livenessProbe:
      httpGet:
        path: /health/live
        port: 8080
      initialDelaySeconds: 30
      periodSeconds: 10
      timeoutSeconds: 5
      failureThreshold: 3
    
    readinessProbe:
      httpGet:
        path: /health/ready
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 5
      timeoutSeconds: 3
      failureThreshold: 2
```

## LLM-Optimized Features

### Complete Context Logging
Every health check failure includes:
- Detailed error information with root cause analysis
- System context at time of failure
- Suggested remediation actions
- Historical failure patterns
- Resource utilization data

### Actionable Suggestions
Health check failures automatically generate:
- Immediate troubleshooting steps
- Configuration recommendations
- Monitoring suggestions
- Escalation procedures

### Diagnostic Information
Detailed health responses include:
- Component dependencies
- Resource thresholds and current usage
- Performance metrics
- Error patterns and frequencies

## Monitoring Integration

### Prometheus Metrics
The health system exports metrics for:
- Overall health status
- Individual check statuses
- Check execution durations
- Failure rates and patterns
- System resource utilization

### Alerting
Configure alerts based on:
- Health status changes
- Critical component failures
- Degraded performance patterns
- Resource threshold breaches

## Error Recovery

### Automatic Recovery
The health system includes:
- Retry mechanisms for transient failures
- Circuit breaker patterns for unstable components
- Graceful degradation strategies
- Automatic service restart triggers

### Failure Isolation
Failed health checks:
- Don't affect other checks
- Include isolation context
- Provide recovery recommendations
- Maintain service availability data

This health check system ensures MCPEG maintains high availability while providing comprehensive visibility into system health for both human operators and LLM-based troubleshooting.