# ADR-008: LLM-Optimized Logging for 100% Troubleshooting

## Status
**ACCEPTED** - *2025-07-11*

## Context

Traditional logging is designed for human operators who can infer context, recognize patterns, and fill in gaps. However, LLMs need explicit, structured, and complete information to troubleshoot effectively. We need a logging system that enables an LLM to:
- Understand the complete system state at any point
- Trace request flow without missing steps
- Diagnose issues without additional context
- Suggest fixes based on log data alone

## Decision

We will implement an LLM-optimized logging system with the following principles:

### 1. Structured Logging with Complete Context
Every log entry must be a self-contained JSON object with:
```json
{
  "timestamp": "2025-07-11T10:30:45.123Z",
  "trace_id": "req-123e4567-e89b-12d3-a456-426614174000",
  "span_id": "span-123",
  "component": "adapter.rest",
  "operation": "http_request",
  "phase": "pre_request",
  "context": {
    "mcp_method": "tools/call",
    "tool_name": "search_items",
    "adapter_id": "example-api",
    "config_version": "1.0.3"
  },
  "input": {
    "original": {"query": "test", "limit": 10},
    "transformed": {"q": "test", "limit": "10"}
  },
  "state": {
    "connection_pool": {"active": 3, "idle": 7, "max": 10},
    "circuit_breaker": "closed",
    "rate_limit": {"remaining": 98, "reset_at": "2025-07-11T11:00:00Z"}
  },
  "metadata": {
    "template_used": "search_query_v1",
    "auth_type": "bearer",
    "timeout_ms": 5000
  }
}
```

### 2. Causality Chain Logging
Every action logs its complete causal chain:
```json
{
  "event": "error",
  "error": {
    "type": "connection_timeout",
    "message": "Request to https://api.example.com timed out after 5000ms",
    "stack_trace": "...",
    "recovery_attempted": true,
    "recovery_result": "circuit_breaker_opened"
  },
  "causality": [
    {"timestamp": "T-5s", "event": "connection_pool_exhausted"},
    {"timestamp": "T-4s", "event": "retry_attempt_1_failed"},
    {"timestamp": "T-2s", "event": "retry_attempt_2_failed"},
    {"timestamp": "T-0s", "event": "circuit_breaker_triggered"}
  ],
  "suggested_actions": [
    "increase_connection_pool_size",
    "check_backend_health",
    "review_timeout_settings"
  ]
}
```

### 3. Configuration Snapshot Logging
Log complete configuration context when relevant:
```json
{
  "event": "config_applied",
  "operation": "adapter_mapping",
  "config_snapshot": {
    "service_id": "example-api",
    "mapping": {
      "mcp_tool": "search_items",
      "backend_endpoint": "GET /items/search",
      "transform_template": "..."
    },
    "diff_from_previous": {
      "changed_fields": ["timeout_ms"],
      "old_values": {"timeout_ms": 3000},
      "new_values": {"timeout_ms": 5000}
    }
  }
}
```

### 4. Request Lifecycle Logging
Complete request tracing with state at each phase:
```json
{
  "lifecycle_phase": "request_complete",
  "trace_id": "req-123",
  "phases": [
    {"phase": "received", "duration_ms": 0.1, "status": "ok"},
    {"phase": "validated", "duration_ms": 0.5, "status": "ok"},
    {"phase": "routed", "duration_ms": 0.2, "status": "ok"},
    {"phase": "transformed", "duration_ms": 1.2, "status": "ok"},
    {"phase": "executed", "duration_ms": 45.3, "status": "ok"},
    {"phase": "response_transformed", "duration_ms": 0.8, "status": "ok"}
  ],
  "total_duration_ms": 48.1,
  "memory_delta_bytes": 1024,
  "goroutines_delta": 0
}
```

### 5. System State Snapshots
Periodic system state logging:
```json
{
  "event": "system_snapshot",
  "timestamp": "2025-07-11T10:30:00Z",
  "resources": {
    "memory": {"heap_mb": 45, "total_mb": 128, "gc_runs": 15},
    "goroutines": {"active": 25, "blocked": 2},
    "connections": {"mcp_clients": 3, "backend_http": 10}
  },
  "adapters": {
    "rest": {"active_requests": 5, "queued": 0},
    "binary": {"processes": 2, "zombies": 0}
  },
  "health_indicators": {
    "overall": "healthy",
    "components": {
      "mcp_server": "healthy",
      "config_manager": "healthy",
      "adapter_rest": "degraded"
    }
  }
}
```

## Implementation Details

### Log Levels Redefined for LLMs
- **TRACE**: Complete data flow including payloads
- **DEBUG**: State changes and decision points
- **INFO**: Significant operations and outcomes
- **WARN**: Anomalies that don't prevent operation
- **ERROR**: Failures with full context and recovery attempts

### Special Log Types
1. **Decision Logs**: Why a particular path was chosen
2. **Validation Logs**: What was checked and why it passed/failed
3. **Performance Logs**: Timing for every operation
4. **Recovery Logs**: What recovery was attempted and why

### Log Correlation
Every log entry includes:
- `trace_id`: Correlates all logs for a request
- `span_id`: Identifies specific operation within request
- `parent_span_id`: Shows operation hierarchy
- `correlation_ids`: Links to related requests/operations

## Consequences

### Positive
- LLM can troubleshoot without human intervention
- Complete system observability
- Self-documenting system behavior
- Enables predictive issue detection
- Facilitates automated root cause analysis

### Negative
- Increased log volume (mitigated by structured storage)
- Performance overhead (mitigated by async logging)
- Sensitive data exposure risk (mitigated by redaction)

### Neutral
- Requires structured log storage (e.g., JSON logs)
- Changes traditional debugging workflows
- Requires log retention strategy

## Example Troubleshooting Scenario

When an LLM sees:
```json
{
  "event": "request_failed",
  "trace_id": "req-789",
  "error": "backend_timeout",
  "context": {
    "similar_failures_last_hour": 15,
    "pattern": "all_failures_to_same_endpoint",
    "backend_response_times": [5000, 5000, 5000],
    "suggested_diagnosis": "backend_endpoint_degraded",
    "recommended_actions": [
      "enable_circuit_breaker",
      "increase_timeout",
      "add_retry_with_backoff"
    ]
  }
}
```

The LLM can immediately understand:
1. What failed and why
2. The pattern of failures
3. The likely root cause
4. Specific remediation steps

## References
- [Structured Logging Best Practices](https://www.datadoghq.com/blog/structured-logging/)
- [OpenTelemetry Semantic Conventions](https://opentelemetry.io/docs/concepts/semantic-conventions/)
- [Causality in Distributed Systems](https://www.microsoft.com/en-us/research/publication/causality-is-simple/)