# ADR-009: Concurrency and Memory Management Patterns

## Status

**ACCEPTED** - *2025-07-11*

## Context

MCPEG will handle multiple concurrent MCP clients, each potentially making multiple backend calls. We need to ensure:
- Controlled concurrency to prevent resource exhaustion
- Proper memory management to avoid leaks
- Graceful degradation under load
- Fault isolation between requests
- Observable resource usage for LLM troubleshooting

Go provides excellent concurrency primitives (goroutines, channels) and automatic memory management (GC), but we need patterns to use them effectively.

## Decision

We will implement the following utilities and patterns:

### 1. Worker Pool Pattern
Limit concurrent operations with a configurable worker pool:
```go
type WorkerPool struct {
    maxWorkers int
    queue      chan Task
    sem        chan struct{}  // Semaphore for limiting concurrency
    metrics    *PoolMetrics
}
```

### 2. Context-Based Cancellation
Use context.Context for request lifecycle management:
```go
type RequestContext struct {
    context.Context
    TraceID     string
    StartTime   time.Time
    MemoryStart uint64
    Logger      logging.Logger
}
```

### 3. Memory-Aware Scheduling
Monitor memory usage and apply backpressure:
```go
type MemoryMonitor struct {
    threshold   uint64  // Bytes
    checkPeriod time.Duration
    limiter     *rate.Limiter
}
```

### 4. Circuit Breaker Pattern
Prevent cascade failures with circuit breakers per adapter:
```go
type CircuitBreaker struct {
    maxFailures     int
    resetTimeout    time.Duration
    halfOpenMax     int
    state           State
    failures        int
    lastFailureTime time.Time
}
```

### 5. Resource Pools
Pool expensive resources like HTTP clients:
```go
type ClientPool struct {
    clients   chan *http.Client
    factory   ClientFactory
    maxIdle   int
    maxActive int
}
```

## Implementation Details

### Worker Pool Implementation
```go
// pkg/concurrency/worker_pool.go
func (wp *WorkerPool) Submit(ctx context.Context, task Task) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    case wp.sem <- struct{}{}:
        // Acquired semaphore
        go func() {
            defer func() {
                <-wp.sem // Release semaphore
                if r := recover(); r != nil {
                    wp.logger.Error("worker_panic", 
                        "panic", r,
                        "stack", debug.Stack())
                }
            }()
            
            wp.executeTask(ctx, task)
        }()
        return nil
    default:
        return ErrPoolFull
    }
}
```

### Memory Monitoring
```go
// pkg/concurrency/memory.go
func (m *MemoryMonitor) CheckMemory() (MemoryStatus, error) {
    var stats runtime.MemStats
    runtime.ReadMemStats(&stats)
    
    status := MemoryStatus{
        Allocated:   stats.Alloc,
        Total:       stats.TotalAlloc,
        System:      stats.Sys,
        NumGC:       stats.NumGC,
        LastGC:      time.Unix(0, int64(stats.LastGC)),
        PauseTotal:  stats.PauseTotalNs,
        HeapInUse:   stats.HeapInuse,
        StackInUse:  stats.StackInuse,
    }
    
    if stats.Alloc > m.threshold {
        // Trigger GC and apply backpressure
        runtime.GC()
        m.limiter.Reserve()
    }
    
    return status, nil
}
```

### Request Context Pattern
```go
// internal/mcp/context.go
func NewRequestContext(parent context.Context, traceID string) *RequestContext {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    
    ctx, cancel := context.WithTimeout(parent, 30*time.Second)
    
    return &RequestContext{
        Context:     ctx,
        TraceID:     traceID,
        StartTime:   time.Now(),
        MemoryStart: m.Alloc,
        Logger:      logging.New("request").WithTraceID(traceID),
        cancel:      cancel,
    }
}

func (rc *RequestContext) Complete() {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    
    rc.Logger.Info("request_complete",
        "duration_ms", time.Since(rc.StartTime).Milliseconds(),
        "memory_allocated", m.Alloc - rc.MemoryStart,
        "goroutines", runtime.NumGoroutine())
    
    rc.cancel()
}
```

### Graceful Shutdown
```go
// internal/server/shutdown.go
func (s *Server) Shutdown(ctx context.Context) error {
    s.logger.Info("shutdown_initiated")
    
    // Stop accepting new requests
    s.acceptor.Stop()
    
    // Wait for in-flight requests with timeout
    done := make(chan struct{})
    go func() {
        s.wg.Wait()
        close(done)
    }()
    
    select {
    case <-done:
        s.logger.Info("shutdown_complete", "clean", true)
        return nil
    case <-ctx.Done():
        s.logger.Warn("shutdown_timeout", "forced", true)
        return ctx.Err()
    }
}
```

## Consequences

### Positive
- Prevents resource exhaustion under load
- Provides graceful degradation
- Enables detailed performance monitoring
- Isolates failures between requests
- Memory leaks are detectable and preventable
- LLM can observe resource usage patterns

### Negative
- Additional complexity in request handling
- Slight performance overhead from monitoring
- Need to tune pool sizes and thresholds

### Neutral
- Requires configuration of limits
- Changes request flow to be resource-aware

## Configuration

```yaml
concurrency:
  worker_pool:
    max_workers: 100
    queue_size: 1000
  
  memory:
    threshold_mb: 512
    gc_trigger_mb: 256
    check_interval: "10s"
  
  circuit_breaker:
    failure_threshold: 5
    reset_timeout: "60s"
    half_open_max: 3
  
  timeouts:
    request: "30s"
    shutdown: "30s"
    idle: "120s"
```

## Monitoring

All concurrency utilities will log detailed metrics:

```json
{
  "component": "worker_pool",
  "operation": "pool_status",
  "data": {
    "active_workers": 45,
    "queued_tasks": 12,
    "completed_tasks": 1523,
    "failed_tasks": 3,
    "avg_task_duration_ms": 234,
    "memory_per_worker_mb": 2.3
  }
}
```

## References
- [Go Concurrency Patterns](https://go.dev/blog/pipelines)
- [Uber's Go Style Guide - Concurrency](https://github.com/uber-go/guide/blob/master/style.md#concurrency)
- [Circuit Breaker Pattern](https://martinfowler.com/bliki/CircuitBreaker.html)