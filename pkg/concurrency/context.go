package concurrency

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/yourusername/mcpeg/pkg/logging"
)

// RequestContext provides comprehensive request tracking
type RequestContext struct {
	context.Context
	TraceID         string
	SpanID          string
	StartTime       time.Time
	MemoryStart     uint64
	GoroutineStart  int
	Logger          logging.Logger
	Breadcrumbs     []Breadcrumb
	mu              sync.Mutex
	cancel          context.CancelFunc
	memoryMonitor   *MemoryMonitor
}

// Breadcrumb represents a step in request processing
type Breadcrumb struct {
	Timestamp time.Time              `json:"timestamp"`
	Operation string                 `json:"operation"`
	Duration  time.Duration          `json:"duration_ms"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// NewRequestContext creates a new request context with tracking
func NewRequestContext(parent context.Context, traceID, spanID string, timeout time.Duration, logger logging.Logger, memMonitor *MemoryMonitor) *RequestContext {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	ctx, cancel := context.WithTimeout(parent, timeout)
	
	rc := &RequestContext{
		Context:        ctx,
		TraceID:        traceID,
		SpanID:         spanID,
		StartTime:      time.Now(),
		MemoryStart:    m.Alloc,
		GoroutineStart: runtime.NumGoroutine(),
		Logger:         logger.WithTraceID(traceID).WithSpanID(spanID),
		cancel:         cancel,
		memoryMonitor:  memMonitor,
		Breadcrumbs:    make([]Breadcrumb, 0, 10),
	}
	
	// Add initial breadcrumb
	rc.AddBreadcrumb("request_started", map[string]interface{}{
		"timeout_seconds": timeout.Seconds(),
		"initial_memory_mb": float64(m.Alloc) / (1024 * 1024),
		"initial_goroutines": runtime.NumGoroutine(),
	})
	
	rc.Logger.Info("request_context_created",
		"timeout", timeout,
		"memory_mb", float64(m.Alloc)/(1024*1024),
		"goroutines", runtime.NumGoroutine())
	
	return rc
}

// AddBreadcrumb adds a breadcrumb to the request trail
func (rc *RequestContext) AddBreadcrumb(operation string, data map[string]interface{}) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	
	breadcrumb := Breadcrumb{
		Timestamp: time.Now(),
		Operation: operation,
		Data:      data,
	}
	
	// Calculate duration since last breadcrumb
	if len(rc.Breadcrumbs) > 0 {
		lastBC := rc.Breadcrumbs[len(rc.Breadcrumbs)-1]
		breadcrumb.Duration = breadcrumb.Timestamp.Sub(lastBC.Timestamp)
	}
	
	rc.Breadcrumbs = append(rc.Breadcrumbs, breadcrumb)
	
	rc.Logger.Debug("breadcrumb_added",
		"operation", operation,
		"duration_ms", breadcrumb.Duration.Milliseconds(),
		"data", data)
}

// Complete finalizes the request context and logs comprehensive metrics
func (rc *RequestContext) Complete() {
	defer rc.cancel()
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	duration := time.Since(rc.StartTime)
	memoryDelta := int64(m.Alloc) - int64(rc.MemoryStart)
	goroutineDelta := runtime.NumGoroutine() - rc.GoroutineStart
	
	// Add completion breadcrumb
	rc.AddBreadcrumb("request_completed", map[string]interface{}{
		"total_duration_ms": duration.Milliseconds(),
		"memory_delta_bytes": memoryDelta,
		"goroutine_delta": goroutineDelta,
	})
	
	// Log comprehensive request summary
	rc.Logger.Info("request_completed",
		"duration_ms", duration.Milliseconds(),
		"memory_allocated_bytes", memoryDelta,
		"memory_allocated_mb", float64(memoryDelta)/(1024*1024),
		"goroutine_delta", goroutineDelta,
		"final_goroutines", runtime.NumGoroutine(),
		"breadcrumb_count", len(rc.Breadcrumbs),
		"breadcrumbs", rc.Breadcrumbs)
	
	// Warn on resource issues
	if memoryDelta > 10*1024*1024 { // 10MB
		rc.Logger.Warn("high_request_memory",
			"allocated_mb", float64(memoryDelta)/(1024*1024),
			"trace_id", rc.TraceID,
			"suggested_actions", []string{
				"Review response size",
				"Check for unnecessary buffering",
				"Consider streaming large responses",
			})
	}
	
	if goroutineDelta > 0 {
		rc.Logger.Warn("goroutine_leak",
			"leaked", goroutineDelta,
			"trace_id", rc.TraceID,
			"suggested_actions", []string{
				"Check for missing defer statements",
				"Verify goroutine completion",
				"Review context cancellation",
			})
	}
	
	if duration > 10*time.Second {
		rc.Logger.Warn("slow_request",
			"duration_seconds", duration.Seconds(),
			"trace_id", rc.TraceID,
			"breadcrumbs", rc.Breadcrumbs,
			"suggested_actions", []string{
				"Review backend call latencies",
				"Check for sequential operations that could be parallel",
				"Consider caching frequently accessed data",
			})
	}
}

// WaitForMemory applies memory backpressure if needed
func (rc *RequestContext) WaitForMemory() error {
	if rc.memoryMonitor != nil {
		return rc.memoryMonitor.WaitIfNeeded(rc.Context)
	}
	return nil
}

// LogCheckpoint logs an intermediate checkpoint with current metrics
func (rc *RequestContext) LogCheckpoint(name string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	elapsed := time.Since(rc.StartTime)
	memoryDelta := int64(m.Alloc) - int64(rc.MemoryStart)
	
	rc.AddBreadcrumb("checkpoint_"+name, map[string]interface{}{
		"elapsed_ms": elapsed.Milliseconds(),
		"memory_delta_bytes": memoryDelta,
		"goroutines": runtime.NumGoroutine(),
	})
	
	rc.Logger.Debug("request_checkpoint",
		"checkpoint", name,
		"elapsed_ms", elapsed.Milliseconds(),
		"memory_delta_mb", float64(memoryDelta)/(1024*1024),
		"goroutines", runtime.NumGoroutine())
}

// WithTimeout creates a child context with a new timeout
func (rc *RequestContext) WithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	childCtx, cancel := context.WithTimeout(rc.Context, timeout)
	
	rc.AddBreadcrumb("child_context_created", map[string]interface{}{
		"timeout_seconds": timeout.Seconds(),
	})
	
	return childCtx, cancel
}

// GetElapsedTime returns the time elapsed since request start
func (rc *RequestContext) GetElapsedTime() time.Duration {
	return time.Since(rc.StartTime)
}

// GetBreadcrumbs returns a copy of the breadcrumbs
func (rc *RequestContext) GetBreadcrumbs() []Breadcrumb {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	
	// Return a copy to prevent external modification
	result := make([]Breadcrumb, len(rc.Breadcrumbs))
	copy(result, rc.Breadcrumbs)
	return result
}

// contextKey is used for storing values in context
type contextKey string

const (
	requestContextKey contextKey = "request_context"
)

// WithRequestContext adds a RequestContext to a context
func WithRequestContext(ctx context.Context, rc *RequestContext) context.Context {
	return context.WithValue(ctx, requestContextKey, rc)
}

// GetRequestContext retrieves a RequestContext from a context
func GetRequestContext(ctx context.Context) (*RequestContext, bool) {
	rc, ok := ctx.Value(requestContextKey).(*RequestContext)
	return rc, ok
}

// RequestOption configures a RequestContext
type RequestOption func(*RequestContext)

// WithMemoryMonitor adds a memory monitor to the request context
func WithMemoryMonitor(monitor *MemoryMonitor) RequestOption {
	return func(rc *RequestContext) {
		rc.memoryMonitor = monitor
	}
}

// WithLogger sets a custom logger for the request context
func WithLogger(logger logging.Logger) RequestOption {
	return func(rc *RequestContext) {
		rc.Logger = logger.WithTraceID(rc.TraceID).WithSpanID(rc.SpanID)
	}
}