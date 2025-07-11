package concurrency

import (
	"context"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/yourusername/mcpeg/pkg/logging"
	"golang.org/x/time/rate"
)

// MemoryStatus represents current memory statistics
type MemoryStatus struct {
	Allocated      uint64    `json:"allocated_bytes"`
	Total          uint64    `json:"total_allocated_bytes"`
	System         uint64    `json:"system_bytes"`
	NumGC          uint32    `json:"gc_runs"`
	LastGC         time.Time `json:"last_gc"`
	PauseTotal     uint64    `json:"gc_pause_total_ns"`
	HeapInUse      uint64    `json:"heap_in_use_bytes"`
	StackInUse     uint64    `json:"stack_in_use_bytes"`
	NumGoroutines  int       `json:"goroutines"`
	ThresholdMB    uint64    `json:"threshold_mb"`
	OverThreshold  bool      `json:"over_threshold"`
}

// MemoryMonitor tracks memory usage and applies backpressure
type MemoryMonitor struct {
	thresholdBytes uint64
	gcTriggerBytes uint64
	checkPeriod    time.Duration
	limiter        *rate.Limiter
	logger         logging.Logger
	mu             sync.RWMutex
	lastStatus     MemoryStatus
	stopCh         chan struct{}
	wg             sync.WaitGroup
}

// NewMemoryMonitor creates a new memory monitor
func NewMemoryMonitor(thresholdMB, gcTriggerMB uint64, checkPeriod time.Duration, logger logging.Logger) *MemoryMonitor {
	return &MemoryMonitor{
		thresholdBytes: thresholdMB * 1024 * 1024,
		gcTriggerBytes: gcTriggerMB * 1024 * 1024,
		checkPeriod:    checkPeriod,
		limiter:        rate.NewLimiter(rate.Every(time.Second), 10), // Allow 10 requests/second under pressure
		logger:         logger.WithComponent("memory_monitor"),
		stopCh:         make(chan struct{}),
	}
}

// Start begins monitoring memory usage
func (m *MemoryMonitor) Start(ctx context.Context) {
	m.wg.Add(1)
	go m.monitor(ctx)
}

// Stop halts memory monitoring
func (m *MemoryMonitor) Stop() {
	close(m.stopCh)
	m.wg.Wait()
}

// monitor runs the periodic memory check
func (m *MemoryMonitor) monitor(ctx context.Context) {
	defer m.wg.Done()
	
	ticker := time.NewTicker(m.checkPeriod)
	defer ticker.Stop()

	// Initial check
	m.checkMemory()

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("memory_monitor_stopped", "reason", "context_cancelled")
			return
		case <-m.stopCh:
			m.logger.Info("memory_monitor_stopped", "reason", "stop_requested")
			return
		case <-ticker.C:
			m.checkMemory()
		}
	}
}

// checkMemory performs a memory check and takes action if needed
func (m *MemoryMonitor) checkMemory() {
	status := m.GetStatus()
	
	m.mu.Lock()
	m.lastStatus = status
	m.mu.Unlock()

	// Log memory status for LLM troubleshooting
	m.logger.Debug("memory_check",
		"allocated_mb", status.Allocated/(1024*1024),
		"heap_mb", status.HeapInUse/(1024*1024),
		"goroutines", status.NumGoroutines,
		"gc_runs", status.NumGC,
		"over_threshold", status.OverThreshold)

	// Take action if over threshold
	if status.OverThreshold {
		m.handleHighMemory(status)
	}

	// Trigger GC if over GC threshold
	if status.Allocated > m.gcTriggerBytes {
		before := status.Allocated
		runtime.GC()
		debug.FreeOSMemory()
		
		// Check memory after GC
		afterStatus := m.GetStatus()
		freed := int64(before) - int64(afterStatus.Allocated)
		
		m.logger.Info("gc_triggered",
			"before_mb", before/(1024*1024),
			"after_mb", afterStatus.Allocated/(1024*1024),
			"freed_mb", freed/(1024*1024),
			"gc_duration_ms", afterStatus.LastGC.Sub(status.LastGC).Milliseconds())
	}
}

// handleHighMemory applies backpressure when memory is high
func (m *MemoryMonitor) handleHighMemory(status MemoryStatus) {
	m.logger.Warn("memory_threshold_exceeded",
		"allocated_mb", status.Allocated/(1024*1024),
		"threshold_mb", m.thresholdBytes/(1024*1024),
		"goroutines", status.NumGoroutines,
		"suggested_actions", []string{
			"Reduce concurrent requests",
			"Check for memory leaks",
			"Increase memory limit",
			"Enable request queuing",
		})

	// Apply rate limiting to reduce memory pressure
	m.limiter.SetLimit(rate.Every(time.Second * 2)) // Slow down to 1 request per 2 seconds
	m.limiter.SetBurst(1)
}

// GetStatus returns current memory status
func (m *MemoryMonitor) GetStatus() MemoryStatus {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	status := MemoryStatus{
		Allocated:     stats.Alloc,
		Total:         stats.TotalAlloc,
		System:        stats.Sys,
		NumGC:         stats.NumGC,
		LastGC:        time.Unix(0, int64(stats.LastGC)),
		PauseTotal:    stats.PauseTotalNs,
		HeapInUse:     stats.HeapInuse,
		StackInUse:    stats.StackInuse,
		NumGoroutines: runtime.NumGoroutine(),
		ThresholdMB:   m.thresholdBytes / (1024 * 1024),
		OverThreshold: stats.Alloc > m.thresholdBytes,
	}

	return status
}

// WaitIfNeeded blocks if memory pressure requires backpressure
func (m *MemoryMonitor) WaitIfNeeded(ctx context.Context) error {
	m.mu.RLock()
	overThreshold := m.lastStatus.OverThreshold
	m.mu.RUnlock()

	if overThreshold {
		// Use rate limiter to apply backpressure
		if err := m.limiter.Wait(ctx); err != nil {
			return err
		}
		
		m.logger.Debug("memory_backpressure_applied",
			"allocated_mb", m.lastStatus.Allocated/(1024*1024))
	}

	return nil
}

// GetLastStatus returns the last checked memory status
func (m *MemoryMonitor) GetLastStatus() MemoryStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastStatus
}

// LogDetailedStats logs detailed memory statistics for debugging
func (m *MemoryMonitor) LogDetailedStats() {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	m.logger.Info("memory_detailed_stats",
		"alloc_mb", stats.Alloc/(1024*1024),
		"total_alloc_mb", stats.TotalAlloc/(1024*1024),
		"sys_mb", stats.Sys/(1024*1024),
		"lookups", stats.Lookups,
		"mallocs", stats.Mallocs,
		"frees", stats.Frees,
		"heap_alloc_mb", stats.HeapAlloc/(1024*1024),
		"heap_sys_mb", stats.HeapSys/(1024*1024),
		"heap_idle_mb", stats.HeapIdle/(1024*1024),
		"heap_inuse_mb", stats.HeapInuse/(1024*1024),
		"heap_released_mb", stats.HeapReleased/(1024*1024),
		"heap_objects", stats.HeapObjects,
		"stack_inuse_mb", stats.StackInuse/(1024*1024),
		"stack_sys_mb", stats.StackSys/(1024*1024),
		"mspan_inuse_mb", stats.MSpanInuse/(1024*1024),
		"mcache_inuse_mb", stats.MCacheInuse/(1024*1024),
		"gc_cpu_fraction", stats.GCCPUFraction,
		"num_gc", stats.NumGC,
		"num_forced_gc", stats.NumForcedGC,
		"gc_pause_total_ms", float64(stats.PauseTotalNs)/1e6,
		"goroutines", runtime.NumGoroutine())
}

// RequestMemoryContext tracks memory usage for a single request
type RequestMemoryContext struct {
	StartAlloc     uint64
	StartGoroutines int
	TraceID        string
	logger         logging.Logger
}

// NewRequestMemoryContext creates a memory tracking context for a request
func NewRequestMemoryContext(traceID string, logger logging.Logger) *RequestMemoryContext {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &RequestMemoryContext{
		StartAlloc:      m.Alloc,
		StartGoroutines: runtime.NumGoroutine(),
		TraceID:         traceID,
		logger:          logger,
	}
}

// Complete logs the memory usage for the request
func (rmc *RequestMemoryContext) Complete() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	allocated := int64(m.Alloc) - int64(rmc.StartAlloc)
	goroutineDelta := runtime.NumGoroutine() - rmc.StartGoroutines

	rmc.logger.Info("request_memory_usage",
		"trace_id", rmc.TraceID,
		"allocated_bytes", allocated,
		"allocated_mb", float64(allocated)/(1024*1024),
		"goroutine_delta", goroutineDelta,
		"final_goroutines", runtime.NumGoroutine())

	// Warn if request allocated too much memory
	if allocated > 10*1024*1024 { // 10MB
		rmc.logger.Warn("high_request_memory_usage",
			"trace_id", rmc.TraceID,
			"allocated_mb", float64(allocated)/(1024*1024),
			"suggested_actions", []string{
				"Check for large response bodies",
				"Verify streaming is used for large data",
				"Look for unnecessary data copying",
			})
	}

	// Warn if goroutines leaked
	if goroutineDelta > 0 {
		rmc.logger.Warn("goroutine_leak_detected",
			"trace_id", rmc.TraceID,
			"leaked_goroutines", goroutineDelta,
			"suggested_actions", []string{
				"Check for missing context cancellation",
				"Verify all goroutines have exit conditions",
				"Look for blocked channel operations",
			})
	}
}