# LLM-Optimized Logging Guidelines

## Core Principle

Every log entry must contain sufficient information for an LLM to understand:
1. **What** happened
2. **Why** it happened
3. **When** it happened (with precise timing)
4. **Where** in the system it happened
5. **How** to fix it if something went wrong

## Logging Rules

### 1. Never Log Partial Information
❌ **Bad**:
```go
log.Error("Request failed")
```

✅ **Good**:
```go
log.Error("request_failed", 
    "trace_id", traceID,
    "operation", "rest_adapter.execute",
    "error_type", "connection_timeout",
    "endpoint", "https://api.example.com/search",
    "timeout_ms", 5000,
    "retry_attempts", 3,
    "circuit_breaker_state", "open",
    "suggested_fix", "increase timeout or check backend health")
```

### 2. Log Decision Points
Every conditional branch should log why that path was taken:

```go
if err != nil {
    log.Debug("error_path_taken",
        "condition", "err != nil",
        "error_type", fmt.Sprintf("%T", err),
        "error_message", err.Error(),
        "recovery_strategy", "retry_with_backoff",
        "context", map[string]interface{}{
            "previous_attempts": attempts,
            "max_attempts": maxAttempts,
            "backoff_ms": backoffMs,
        })
    // Handle error...
} else {
    log.Debug("success_path_taken",
        "condition", "err == nil",
        "response_time_ms", responseTime,
        "response_size_bytes", len(response))
}
```

### 3. Log Complete State Transitions
```go
log.Info("state_transition",
    "component", "circuit_breaker",
    "adapter_id", adapterID,
    "previous_state", "closed",
    "new_state", "open",
    "trigger", "consecutive_failures",
    "failure_count", 5,
    "threshold", 5,
    "recovery_time_seconds", 60,
    "affected_operations", []string{"search_items", "get_item"})
```

### 4. Include Contextual Breadcrumbs
```go
type RequestContext struct {
    TraceID     string
    SpanID      string
    Breadcrumbs []Breadcrumb
}

func (rc *RequestContext) AddBreadcrumb(operation string, data map[string]interface{}) {
    rc.Breadcrumbs = append(rc.Breadcrumbs, Breadcrumb{
        Timestamp: time.Now(),
        Operation: operation,
        Data:      data,
    })
}

// When logging an error, include breadcrumbs
log.Error("operation_failed",
    "breadcrumbs", rc.Breadcrumbs,
    "final_error", err)
```

### 5. Log Performance Metrics Inline
```go
start := time.Now()
result, err := operation()
duration := time.Since(start)

log.Info("operation_completed",
    "operation", "transform_response",
    "duration_ms", duration.Milliseconds(),
    "duration_percentile", calculatePercentile(duration),
    "input_size_bytes", len(input),
    "output_size_bytes", len(result),
    "memory_allocated_bytes", memStats.Alloc - startMem,
    "goroutines_created", runtime.NumGoroutine() - startGoroutines)
```

### 6. Log Configuration Context
When operations depend on configuration:
```go
log.Info("operation_executed",
    "operation", "rest_call",
    "config_context", map[string]interface{}{
        "timeout_ms": config.Timeout,
        "retry_policy": config.RetryPolicy,
        "circuit_breaker_enabled": config.CircuitBreaker.Enabled,
        "rate_limit": config.RateLimit,
    },
    "config_version", config.Version,
    "config_loaded_at", config.LoadedAt)
```

## Structured Log Fields

### Required Fields for Every Log
```go
type LogEntry struct {
    Timestamp   time.Time              `json:"timestamp"`
    Level       string                 `json:"level"`
    Component   string                 `json:"component"`
    Operation   string                 `json:"operation"`
    TraceID     string                 `json:"trace_id"`
    SpanID      string                 `json:"span_id"`
    Message     string                 `json:"message"`
    Data        map[string]interface{} `json:"data"`
}
```

### Component Naming Convention
Use dot notation for component hierarchy:
- `mcp.server`
- `mcp.handler.resources`
- `adapter.rest`
- `adapter.rest.auth`
- `config.loader`
- `validation.compliance`

### Operation Naming Convention
Use descriptive operation names:
- `parse_request`
- `validate_schema`
- `transform_to_backend`
- `execute_http_call`
- `transform_from_backend`
- `send_response`

## Error Logging Pattern

```go
func LogError(err error, context map[string]interface{}) {
    entry := map[string]interface{}{
        "error_type": fmt.Sprintf("%T", err),
        "error_message": err.Error(),
        "stack_trace": debug.Stack(),
        "recovery_possible": isRecoverable(err),
        "suggested_actions": getSuggestedActions(err),
    }
    
    // Add all context
    for k, v := range context {
        entry[k] = v
    }
    
    // Add error chain if wrapped
    if wrapped, ok := err.(interface{ Unwrap() error }); ok {
        entry["error_chain"] = buildErrorChain(wrapped)
    }
    
    log.Error("error_occurred", entry)
}
```

## LLM-Friendly Format Examples

### Success Case
```json
{
  "timestamp": "2025-07-11T10:30:45.123Z",
  "level": "INFO",
  "component": "adapter.rest",
  "operation": "execute_request",
  "trace_id": "req-123",
  "message": "REST API call successful",
  "data": {
    "method": "GET",
    "url": "https://api.example.com/items",
    "status_code": 200,
    "duration_ms": 145,
    "response_size_bytes": 2048,
    "cache_hit": false,
    "rate_limit_remaining": 95
  }
}
```

### Failure Case
```json
{
  "timestamp": "2025-07-11T10:31:00.456Z",
  "level": "ERROR",
  "component": "adapter.rest",
  "operation": "execute_request",
  "trace_id": "req-124",
  "message": "REST API call failed",
  "data": {
    "method": "GET",
    "url": "https://api.example.com/items",
    "error_type": "timeout",
    "timeout_ms": 5000,
    "attempts": 3,
    "backoff_pattern": [100, 200, 400],
    "circuit_breaker_triggered": true,
    "last_successful_call": "2025-07-11T10:25:00Z",
    "suggested_actions": [
      "Check backend service health",
      "Increase timeout to 10000ms",
      "Enable request queuing"
    ],
    "related_errors_last_hour": 15
  }
}
```

## Implementation Checklist

- [ ] Every function logs entry and exit
- [ ] Every error includes recovery attempts
- [ ] Every decision point logs why
- [ ] Every external call logs full context
- [ ] Every state change logs before/after
- [ ] Every configuration use logs values
- [ ] Every performance metric includes percentiles
- [ ] Every log includes trace correlation

## Testing Logging

Write tests that verify logging completeness:

```go
func TestLoggingCompleteness(t *testing.T) {
    logs := CaptureLogsT(t, func() {
        // Execute operation
    })
    
    // Verify all required fields
    require.Contains(t, logs[0], "trace_id")
    require.Contains(t, logs[0], "component")
    require.Contains(t, logs[0], "operation")
    
    // Verify LLM can understand the error
    if logs[0]["level"] == "ERROR" {
        require.Contains(t, logs[0]["data"], "error_type")
        require.Contains(t, logs[0]["data"], "suggested_actions")
        require.Contains(t, logs[0]["data"], "recovery_attempted")
    }
}
```