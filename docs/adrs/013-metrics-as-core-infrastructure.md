# ADR-013: Metrics as Core Infrastructure

## Status

Proposed

## Context

Currently, we have logging as a first-class citizen in MCPEG with LLM-optimized structured logging. We should consider making metrics collection equally foundational, integrated into every component like logging and memory management.

Metrics would provide:
- Performance monitoring
- Usage analytics
- Health monitoring
- Capacity planning
- LLM-debuggable performance insights

## Decision

We will implement metrics as core infrastructure, similar to logging and memory management, with every component automatically emitting metrics.

## Design Principles

### 1. Metrics as First-Class Citizens
Like logging, metrics should be:
- Built into every component
- Automatically collected
- LLM-optimized for troubleshooting
- Zero-configuration for basic usage
- Highly configurable for advanced scenarios

### 2. Three-Tier Metrics Architecture

**Tier 1: Component Metrics** (Built-in)
Every component automatically emits:
```go
type ComponentMetrics struct {
    RequestCount      Counter
    RequestDuration   Histogram
    ErrorCount        Counter
    ActiveOperations  Gauge
    MemoryUsage      Gauge
}
```

**Tier 2: Service Metrics** (Service-specific)
Each adapter emits domain-specific metrics:
```go
// MySQL adapter metrics
type MySQLMetrics struct {
    ComponentMetrics              // Inherit base metrics
    
    ConnectionPoolActive   Gauge
    ConnectionPoolIdle     Gauge
    QueryDuration         Histogram
    SlowQueries           Counter
    DeadlockRetries       Counter
}
```

**Tier 3: Business Metrics** (User-defined)
Custom metrics defined in configuration:
```yaml
custom_metrics:
  - name: "user_query_complexity"
    type: "histogram"
    description: "SQL query complexity score"
    labels: ["user", "database", "table"]
```

## Implementation Architecture

### Core Metrics Infrastructure

```go
// pkg/metrics/metrics.go
package metrics

import (
    "context"
    "time"
    
    "github.com/osakka/mcpeg/pkg/logging"
)

// Metrics is the core metrics interface
type Metrics interface {
    // Counters
    Inc(name string, labels ...string)
    Add(name string, value float64, labels ...string)
    
    // Gauges  
    Set(name string, value float64, labels ...string)
    
    // Histograms
    Observe(name string, value float64, labels ...string)
    Time(name string, labels ...string) func()
    
    // Utilities
    WithLabels(labels map[string]string) Metrics
    WithPrefix(prefix string) Metrics
}

// ComponentMetrics provides standard metrics for any component
type ComponentMetrics struct {
    metrics Metrics
    logger  logging.Logger
    
    // Standard metrics
    requests      Counter
    duration      Histogram
    errors        Counter
    active        Gauge
    memory        Gauge
}

func NewComponentMetrics(name string, metrics Metrics, logger logging.Logger) *ComponentMetrics {
    return &ComponentMetrics{
        metrics: metrics.WithPrefix(name),
        logger:  logger,
        
        requests: metrics.Counter(name + "_requests_total"),
        duration: metrics.Histogram(name + "_duration_seconds"),
        errors:   metrics.Counter(name + "_errors_total"),
        active:   metrics.Gauge(name + "_active_operations"),
        memory:   metrics.Gauge(name + "_memory_bytes"),
    }
}

func (cm *ComponentMetrics) StartOperation(operation string) func(error) {
    cm.active.Inc()
    timer := cm.duration.Timer()
    startMem := getCurrentMemory()
    
    return func(err error) {
        defer cm.active.Dec()
        defer timer()
        
        cm.requests.Inc()
        
        if err != nil {
            cm.errors.Inc()
            cm.logger.Error("operation_failed",
                "operation", operation,
                "error", err,
                "duration_ms", timer.Duration().Milliseconds())
        }
        
        memDelta := getCurrentMemory() - startMem
        cm.memory.Add(float64(memDelta))
        
        cm.logger.Debug("operation_completed",
            "operation", operation,
            "duration_ms", timer.Duration().Milliseconds(),
            "memory_delta_bytes", memDelta,
            "success", err == nil)
    }
}
```

### Adapter Integration Pattern

```go
// Every adapter gets metrics automatically
type MySQLAdapter struct {
    *adapter.BaseAdapter
    
    // Embedded metrics (automatic)
    *metrics.ComponentMetrics
    
    // MySQL-specific metrics
    connectionPool    *metrics.Gauge
    queryDuration     *metrics.Histogram
    slowQueries       *metrics.Counter
}

func NewMySQLAdapter(config Config, logger logging.Logger, metrics metrics.Metrics) *MySQLAdapter {
    base := adapter.NewBaseAdapter("mysql", "database")
    
    return &MySQLAdapter{
        BaseAdapter:       base,
        ComponentMetrics:  metrics.NewComponentMetrics("mysql", metrics, logger),
        
        // MySQL-specific metrics
        connectionPool: metrics.Gauge("mysql_connection_pool_active"),
        queryDuration:  metrics.Histogram("mysql_query_duration_seconds"),
        slowQueries:    metrics.Counter("mysql_slow_queries_total"),
    }
}

func (m *MySQLAdapter) ExecuteTool(ctx context.Context, tool string, params map[string]interface{}) (interface{}, error) {
    // Automatic component metrics
    done := m.StartOperation("execute_tool")
    defer done(nil) // Will be called with actual error
    
    // Tool-specific metrics
    toolTimer := m.queryDuration.Timer()
    defer toolTimer()
    
    result, err := m.executeQuery(params["query"].(string))
    
    // Custom business metrics
    if duration := toolTimer.Duration(); duration > 1*time.Second {
        m.slowQueries.Inc()
        m.logger.Warn("slow_query_detected",
            "query", params["query"],
            "duration_ms", duration.Milliseconds(),
            "suggested_actions", []string{
                "Add database indexes",
                "Optimize query structure", 
                "Consider query caching",
            })
    }
    
    return result, err
}
```

### Configuration Integration

```yaml
# mcpeg.yaml
metrics:
  enabled: true
  
  # Export configuration
  exporters:
    - type: "prometheus"
      enabled: true
      port: 9090
      path: "/metrics"
      
    - type: "datadog"
      enabled: false
      api_key: "${DATADOG_API_KEY}"
      
    - type: "json_logs"
      enabled: true
      interval: "30s"
  
  # Collection settings
  collection:
    interval: "15s"
    cardinality_limit: 10000
    retention: "24h"
  
  # LLM optimization
  llm_integration:
    enabled: true
    include_in_logs: true    # Include metrics in log entries
    alert_thresholds:
      error_rate: 0.05       # 5% error rate triggers alert
      latency_p99: "5s"      # 99th percentile latency threshold
      memory_growth: "100MB" # Memory growth rate threshold

# Service-specific metrics
services:
  mysql:
    metrics:
      custom:
        - name: "query_complexity_score"
          type: "histogram"
          description: "SQL query complexity analysis"
          buckets: [1, 2, 5, 10, 20, 50, 100]
          
        - name: "table_scan_ratio"
          type: "gauge"
          description: "Ratio of table scans vs index usage"
```

### LLM-Optimized Metrics Logging

```go
// Metrics automatically enhance logging
func (cm *ComponentMetrics) LogPerformanceInsight() {
    stats := cm.GetStats()
    
    cm.logger.Info("performance_metrics",
        "requests_per_minute", stats.RequestsPerMinute,
        "average_latency_ms", stats.AverageLatency.Milliseconds(),
        "error_rate_percent", stats.ErrorRate*100,
        "memory_usage_mb", stats.MemoryUsage/(1024*1024),
        "active_operations", stats.ActiveOperations,
        
        // LLM-friendly insights
        "performance_health", stats.HealthScore(),
        "bottleneck_analysis", stats.IdentifyBottlenecks(),
        "optimization_suggestions", stats.GetOptimizationSuggestions(),
        
        // Trending information
        "trend_requests", stats.RequestTrend,      // "increasing", "stable", "decreasing"
        "trend_latency", stats.LatencyTrend,
        "trend_errors", stats.ErrorTrend,
    )
}
```

### Built-in Analytics and Insights

```go
type MetricsAnalyzer struct {
    metrics Metrics
    logger  logging.Logger
}

func (ma *MetricsAnalyzer) AnalyzeSystemHealth() SystemHealthReport {
    report := SystemHealthReport{
        Timestamp: time.Now(),
        Services:  make(map[string]ServiceHealth),
    }
    
    for service := range ma.getServices() {
        health := ma.analyzeServiceHealth(service)
        report.Services[service] = health
        
        // LLM-optimized health reporting
        ma.logger.Info("service_health_analysis",
            "service", service,
            "health_score", health.Score,
            "primary_issues", health.Issues,
            "recommendations", health.Recommendations,
            "trend_analysis", health.Trends)
    }
    
    return report
}
```

## Integration with Existing Infrastructure

### 1. Logging Integration
```go
// Metrics automatically enhance log entries
logger.Info("request_processed",
    "endpoint", "/tools/call",
    "duration_ms", 234,
    "metrics", map[string]interface{}{
        "requests_per_minute": metrics.GetRPM(),
        "average_latency_ms": metrics.GetAvgLatency(),
        "error_rate": metrics.GetErrorRate(),
    })
```

### 2. Memory Management Integration
```go
// Memory monitor uses metrics
type MemoryMonitor struct {
    monitor *concurrency.MemoryMonitor
    metrics metrics.Metrics
}

func (mm *MemoryMonitor) checkMemory() {
    status := mm.monitor.GetStatus()
    
    // Emit metrics
    mm.metrics.Set("memory_allocated_bytes", float64(status.Allocated))
    mm.metrics.Set("memory_heap_bytes", float64(status.HeapInUse))
    mm.metrics.Set("goroutines_active", float64(status.NumGoroutines))
    
    // Trigger alerts via metrics
    if status.OverThreshold {
        mm.metrics.Inc("memory_threshold_exceeded_total")
    }
}
```

### 3. Circuit Breaker Integration
```go
func (cb *CircuitBreaker) recordMetrics() {
    state, status := cb.GetState()
    
    cb.metrics.Set("circuit_breaker_state", float64(state))
    cb.metrics.Set("circuit_breaker_failures", float64(status.Failures))
    cb.metrics.Set("circuit_breaker_successes", float64(status.Successes))
    
    if state == StateOpen {
        cb.metrics.Inc("circuit_breaker_opened_total")
        
        cb.logger.Error("circuit_breaker_analysis",
            "service", cb.name,
            "failure_pattern", cb.analyzeFailurePattern(),
            "estimated_recovery_time", status.TimeUntilReset,
            "impact_assessment", cb.assessImpact())
    }
}
```

## Consequences

### Positive
- **Complete Observability**: Every component automatically provides metrics
- **LLM-Optimized**: Metrics enhance logging with performance context
- **Zero Configuration**: Works out of the box with sensible defaults
- **Performance Insights**: Built-in analysis and recommendations
- **Consistent Pattern**: Same metrics pattern across all components

### Negative
- **Resource Overhead**: Metrics collection uses CPU and memory
- **Complexity**: More moving parts in the system
- **Storage Requirements**: Metrics data needs storage and retention

### Neutral
- **Learning Curve**: Developers need to understand metrics concepts
- **Configuration Options**: Many knobs to tune for advanced users

## Implementation Timeline

**Phase 1**: Core metrics infrastructure and component integration
**Phase 2**: Service-specific metrics for each adapter
**Phase 3**: Advanced analytics and LLM-optimized insights
**Phase 4**: Predictive metrics and anomaly detection

This makes metrics a true peer to logging and memory management!