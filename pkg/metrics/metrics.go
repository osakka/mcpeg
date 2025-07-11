package metrics

import (
	"runtime"
	"sync"
	"time"

	"github.com/osakka/mcpeg/pkg/logging"
)

// Metrics is the core metrics interface
type Metrics interface {
	// Counters - values that only increase
	Inc(name string, labels ...string)
	Add(name string, value float64, labels ...string)
	
	// Gauges - values that can go up and down
	Set(name string, value float64, labels ...string)
	
	// Histograms - track distribution of values
	Observe(name string, value float64, labels ...string)
	Time(name string, labels ...string) Timer
	
	// Utilities
	WithLabels(labels map[string]string) Metrics
	WithPrefix(prefix string) Metrics
	
	// Analytics
	GetStats(name string) MetricStats
	GetAllStats() map[string]MetricStats
}

// Timer tracks operation duration
type Timer interface {
	Duration() time.Duration
	Stop() time.Duration
}

// MetricStats provides statistics for a metric
type MetricStats struct {
	Count       uint64        `json:"count"`
	Sum         float64       `json:"sum"`
	Average     float64       `json:"average"`
	Min         float64       `json:"min"`
	Max         float64       `json:"max"`
	LastValue   float64       `json:"last_value"`
	LastUpdated time.Time     `json:"last_updated"`
	Trend       string        `json:"trend"` // "increasing", "stable", "decreasing"
}

// ComponentMetrics provides standard metrics for any component
type ComponentMetrics struct {
	name    string
	metrics Metrics
	logger  logging.Logger
	
	// Standard metric names
	requestsTotal    string
	durationSeconds  string
	errorsTotal      string
	activeOps        string
	memoryBytes      string
}

// NewComponentMetrics creates metrics for a component
func NewComponentMetrics(name string, metrics Metrics, logger logging.Logger) *ComponentMetrics {
	return &ComponentMetrics{
		name:    name,
		metrics: metrics.WithPrefix(name),
		logger:  logger.WithComponent("metrics." + name),
		
		requestsTotal:   name + "_requests_total",
		durationSeconds: name + "_duration_seconds", 
		errorsTotal:     name + "_errors_total",
		activeOps:       name + "_active_operations",
		memoryBytes:     name + "_memory_bytes",
	}
}

// StartOperation begins tracking an operation
func (cm *ComponentMetrics) StartOperation(operation string) func(error) {
	// Track active operations
	cm.metrics.Inc(cm.activeOps, "operation", operation)
	
	// Start timing
	timer := cm.metrics.Time(cm.durationSeconds, "operation", operation)
	
	// Track memory at start
	var startMem runtime.MemStats
	runtime.ReadMemStats(&startMem)
	
	return func(err error) {
		// Stop timing
		duration := timer.Stop()
		
		// Decrease active operations
		cm.metrics.Add(cm.activeOps, -1, "operation", operation)
		
		// Count request
		cm.metrics.Inc(cm.requestsTotal, "operation", operation)
		
		// Track memory delta
		var endMem runtime.MemStats
		runtime.ReadMemStats(&endMem)
		memDelta := int64(endMem.Alloc) - int64(startMem.Alloc)
		cm.metrics.Observe(cm.memoryBytes, float64(memDelta), "operation", operation)
		
		// Track errors
		success := "true"
		if err != nil {
			cm.metrics.Inc(cm.errorsTotal, "operation", operation, "error_type", getErrorType(err))
			success = "false"
		}
		
		// LLM-optimized logging with metrics context
		cm.logger.Info("operation_metrics",
			"operation", operation,
			"duration_ms", duration.Milliseconds(),
			"memory_delta_bytes", memDelta,
			"success", success,
			"error", err,
			
			// Current performance context
			"requests_per_minute", cm.getRequestsPerMinute(operation),
			"average_duration_ms", cm.getAverageDuration(operation),
			"error_rate_percent", cm.getErrorRate(operation)*100,
			
			// Performance insights
			"performance_trend", cm.getPerformanceTrend(operation),
			"bottleneck_indicator", cm.isBottleneck(duration, operation),
		)
		
		// Trigger alerts if needed
		cm.checkThresholds(operation, duration, err)
	}
}

// Performance analysis helpers
func (cm *ComponentMetrics) getRequestsPerMinute(operation string) float64 {
	stats := cm.metrics.GetStats(cm.requestsTotal)
	if stats.Count == 0 {
		return 0
	}
	
	// Calculate RPM over last minute
	elapsed := time.Since(stats.LastUpdated).Minutes()
	if elapsed == 0 {
		return 0
	}
	
	return float64(stats.Count) / elapsed
}

func (cm *ComponentMetrics) getAverageDuration(operation string) float64 {
	stats := cm.metrics.GetStats(cm.durationSeconds)
	return stats.Average * 1000 // Convert to milliseconds
}

func (cm *ComponentMetrics) getErrorRate(operation string) float64 {
	totalStats := cm.metrics.GetStats(cm.requestsTotal)
	errorStats := cm.metrics.GetStats(cm.errorsTotal)
	
	if totalStats.Count == 0 {
		return 0
	}
	
	return float64(errorStats.Count) / float64(totalStats.Count)
}

func (cm *ComponentMetrics) getPerformanceTrend(operation string) string {
	stats := cm.metrics.GetStats(cm.durationSeconds)
	return stats.Trend
}

func (cm *ComponentMetrics) isBottleneck(duration time.Duration, operation string) bool {
	avgDuration := cm.getAverageDuration(operation)
	return duration.Milliseconds() > int64(avgDuration*2) // 2x average is bottleneck
}

func (cm *ComponentMetrics) checkThresholds(operation string, duration time.Duration, err error) {
	// Check duration threshold
	if duration > 5*time.Second {
		cm.logger.Warn("slow_operation_detected",
			"operation", operation,
			"duration_ms", duration.Milliseconds(),
			"threshold_ms", 5000,
			"suggested_actions", []string{
				"Review operation complexity",
				"Check for blocking operations",
				"Consider async processing",
				"Add caching if appropriate",
			})
	}
	
	// Check error rate threshold
	errorRate := cm.getErrorRate(operation)
	if errorRate > 0.05 { // 5% error rate
		cm.logger.Error("high_error_rate_detected",
			"operation", operation,
			"error_rate_percent", errorRate*100,
			"threshold_percent", 5,
			"suggested_actions", []string{
				"Review error patterns",
				"Check input validation",
				"Verify external dependencies",
				"Consider circuit breaker",
			})
	}
}

// LogPerformanceInsights logs comprehensive performance analysis
func (cm *ComponentMetrics) LogPerformanceInsights() {
	allStats := cm.metrics.GetAllStats()
	
	analysis := cm.analyzePerformance(allStats)
	
	cm.logger.Info("component_performance_analysis",
		"component", cm.name,
		"total_requests", analysis.TotalRequests,
		"requests_per_minute", analysis.RequestsPerMinute,
		"average_latency_ms", analysis.AverageLatency,
		"error_rate_percent", analysis.ErrorRate*100,
		"memory_usage_trend", analysis.MemoryTrend,
		"active_operations", analysis.ActiveOperations,
		
		// LLM-friendly insights
		"health_score", analysis.HealthScore,
		"performance_issues", analysis.Issues,
		"optimization_opportunities", analysis.Optimizations,
		"capacity_recommendations", analysis.Recommendations,
	)
}

// PerformanceAnalysis contains component performance insights
type PerformanceAnalysis struct {
	TotalRequests     uint64    `json:"total_requests"`
	RequestsPerMinute float64   `json:"requests_per_minute"`
	AverageLatency    float64   `json:"average_latency_ms"`
	ErrorRate         float64   `json:"error_rate"`
	MemoryTrend       string    `json:"memory_trend"`
	ActiveOperations  float64   `json:"active_operations"`
	HealthScore       float64   `json:"health_score"`
	Issues            []string  `json:"issues"`
	Optimizations     []string  `json:"optimizations"`
	Recommendations   []string  `json:"recommendations"`
}

func (cm *ComponentMetrics) analyzePerformance(stats map[string]MetricStats) PerformanceAnalysis {
	analysis := PerformanceAnalysis{
		Issues:          []string{},
		Optimizations:   []string{},
		Recommendations: []string{},
	}
	
	// Extract key metrics
	if requestStats, ok := stats[cm.requestsTotal]; ok {
		analysis.TotalRequests = requestStats.Count
		analysis.RequestsPerMinute = cm.calculateRPM(requestStats)
	}
	
	if durationStats, ok := stats[cm.durationSeconds]; ok {
		analysis.AverageLatency = durationStats.Average * 1000 // Convert to ms
	}
	
	if errorStats, ok := stats[cm.errorsTotal]; ok {
		analysis.ErrorRate = float64(errorStats.Count) / float64(analysis.TotalRequests)
	}
	
	if activeStats, ok := stats[cm.activeOps]; ok {
		analysis.ActiveOperations = activeStats.LastValue
	}
	
	if memStats, ok := stats[cm.memoryBytes]; ok {
		analysis.MemoryTrend = memStats.Trend
	}
	
	// Calculate health score (0-100)
	analysis.HealthScore = cm.calculateHealthScore(analysis)
	
	// Identify issues and optimizations
	cm.identifyIssues(&analysis)
	cm.suggestOptimizations(&analysis)
	
	return analysis
}

func (cm *ComponentMetrics) calculateHealthScore(analysis PerformanceAnalysis) float64 {
	score := 100.0
	
	// Penalize high error rate
	if analysis.ErrorRate > 0.01 { // 1%
		score -= analysis.ErrorRate * 100 * 50 // 50% penalty for errors
	}
	
	// Penalize high latency
	if analysis.AverageLatency > 1000 { // 1 second
		score -= (analysis.AverageLatency - 1000) / 100 // Penalty for slow responses
	}
	
	// Penalize many active operations (potential bottleneck)
	if analysis.ActiveOperations > 10 {
		score -= (analysis.ActiveOperations - 10) * 2
	}
	
	if score < 0 {
		score = 0
	}
	
	return score
}

func (cm *ComponentMetrics) identifyIssues(analysis *PerformanceAnalysis) {
	if analysis.ErrorRate > 0.05 {
		analysis.Issues = append(analysis.Issues, "High error rate detected")
	}
	
	if analysis.AverageLatency > 2000 {
		analysis.Issues = append(analysis.Issues, "High response latency")
	}
	
	if analysis.ActiveOperations > 20 {
		analysis.Issues = append(analysis.Issues, "High number of concurrent operations")
	}
	
	if analysis.MemoryTrend == "increasing" {
		analysis.Issues = append(analysis.Issues, "Memory usage increasing over time")
	}
}

func (cm *ComponentMetrics) suggestOptimizations(analysis *PerformanceAnalysis) {
	if analysis.AverageLatency > 1000 {
		analysis.Optimizations = append(analysis.Optimizations, 
			"Consider adding caching",
			"Review algorithm complexity",
			"Add connection pooling")
	}
	
	if analysis.ErrorRate > 0.02 {
		analysis.Optimizations = append(analysis.Optimizations,
			"Improve input validation",
			"Add retry mechanisms",
			"Implement circuit breakers")
	}
	
	if analysis.ActiveOperations > 15 {
		analysis.Optimizations = append(analysis.Optimizations,
			"Implement request queuing",
			"Add rate limiting",
			"Scale horizontally")
	}
}

func (cm *ComponentMetrics) calculateRPM(stats MetricStats) float64 {
	elapsed := time.Since(stats.LastUpdated).Minutes()
	if elapsed == 0 {
		return 0
	}
	return float64(stats.Count) / elapsed
}

// Helper functions
func getErrorType(err error) string {
	if err == nil {
		return "none"
	}
	
	// Categorize common error types
	errStr := err.Error()
	switch {
	case contains(errStr, "timeout"):
		return "timeout"
	case contains(errStr, "connection"):
		return "connection"
	case contains(errStr, "permission"):
		return "permission"
	case contains(errStr, "not found"):
		return "not_found"
	default:
		return "other"
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && 
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// timer implements the Timer interface
type timer struct {
	start time.Time
	end   time.Time
	mu    sync.Mutex
}

func newTimer() *timer {
	return &timer{start: time.Now()}
}

func (t *timer) Duration() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	if t.end.IsZero() {
		return time.Since(t.start)
	}
	return t.end.Sub(t.start)
}

func (t *timer) Stop() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	if t.end.IsZero() {
		t.end = time.Now()
	}
	return t.end.Sub(t.start)
}

// ProductionMetrics implements the Metrics interface with real metric collection
type ProductionMetrics struct {
	stats   map[string]*MetricStats
	mutex   sync.RWMutex
	logger  logging.Logger
	prefix  string
	labels  map[string]string
}

// NewProductionMetrics creates a new production metrics instance
func NewProductionMetrics(logger logging.Logger) *ProductionMetrics {
	return &ProductionMetrics{
		stats:  make(map[string]*MetricStats),
		logger: logger.WithComponent("metrics"),
		labels: make(map[string]string),
	}
}

func (m *ProductionMetrics) Inc(name string, labels ...string) {
	m.Add(name, 1, labels...)
}

func (m *ProductionMetrics) Add(name string, value float64, labels ...string) {
	key := m.buildKey(name, labels)
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	stats, exists := m.stats[key]
	if !exists {
		stats = &MetricStats{
			Min:         value,
			Max:         value,
			LastUpdated: time.Now(),
		}
		m.stats[key] = stats
	}
	
	stats.Count++
	stats.Sum += value
	stats.LastValue = value
	stats.LastUpdated = time.Now()
	
	// Update min/max
	if value < stats.Min || stats.Count == 1 {
		stats.Min = value
	}
	if value > stats.Max || stats.Count == 1 {
		stats.Max = value
	}
	
	// Calculate average
	if stats.Count > 0 {
		stats.Average = stats.Sum / float64(stats.Count)
	}
	
	// Update trend (simplified)
	if stats.Count > 1 {
		if value > stats.Average {
			stats.Trend = "increasing"
		} else if value < stats.Average {
			stats.Trend = "decreasing"
		} else {
			stats.Trend = "stable"
		}
	}
	
	m.logger.Trace("metric_updated", "name", name, "value", value, "labels", labels)
}

func (m *ProductionMetrics) Set(name string, value float64, labels ...string) {
	key := m.buildKey(name, labels)
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	stats, exists := m.stats[key]
	if !exists {
		stats = &MetricStats{
			Count:       1,
			Min:         value,
			Max:         value,
			LastUpdated: time.Now(),
		}
		m.stats[key] = stats
	}
	
	stats.LastValue = value
	stats.LastUpdated = time.Now()
	
	// For gauges, we track the trend differently
	if stats.Count > 0 {
		oldValue := stats.Average
		if value > oldValue {
			stats.Trend = "increasing"
		} else if value < oldValue {
			stats.Trend = "decreasing"
		} else {
			stats.Trend = "stable"
		}
	}
	
	// Update min/max
	if value < stats.Min || stats.Count == 1 {
		stats.Min = value
	}
	if value > stats.Max || stats.Count == 1 {
		stats.Max = value
	}
	
	// For gauges, average is the last value
	stats.Average = value
	
	m.logger.Trace("gauge_updated", "name", name, "value", value, "labels", labels)
}

func (m *ProductionMetrics) Observe(name string, value float64, labels ...string) {
	// For histograms, we use Add to track observations
	m.Add(name, value, labels...)
}

func (m *ProductionMetrics) Time(name string, labels ...string) Timer {
	return &productionTimer{
		start:   time.Now(),
		name:    name,
		labels:  labels,
		metrics: m,
	}
}

func (m *ProductionMetrics) WithLabels(labels map[string]string) Metrics {
	newMetrics := &ProductionMetrics{
		stats:  m.stats, // Share the same stats map
		mutex:  m.mutex,
		logger: m.logger,
		prefix: m.prefix,
		labels: make(map[string]string),
	}
	
	// Copy existing labels
	for k, v := range m.labels {
		newMetrics.labels[k] = v
	}
	
	// Add new labels
	for k, v := range labels {
		newMetrics.labels[k] = v
	}
	
	return newMetrics
}

func (m *ProductionMetrics) WithPrefix(prefix string) Metrics {
	newMetrics := &ProductionMetrics{
		stats:  m.stats, // Share the same stats map
		mutex:  m.mutex,
		logger: m.logger,
		prefix: m.buildPrefix(prefix),
		labels: make(map[string]string),
	}
	
	// Copy labels
	for k, v := range m.labels {
		newMetrics.labels[k] = v
	}
	
	return newMetrics
}

func (m *ProductionMetrics) GetStats(name string) MetricStats {
	key := m.buildKey(name, nil)
	
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	if stats, exists := m.stats[key]; exists {
		return *stats
	}
	
	return MetricStats{}
}

func (m *ProductionMetrics) GetAllStats() map[string]MetricStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	result := make(map[string]MetricStats)
	for key, stats := range m.stats {
		result[key] = *stats
	}
	
	return result
}

func (m *ProductionMetrics) buildKey(name string, labels []string) string {
	key := m.prefix + name
	
	// Add instance labels
	for k, v := range m.labels {
		key += ":" + k + "=" + v
	}
	
	// Add method labels
	for i := 0; i < len(labels); i += 2 {
		if i+1 < len(labels) {
			key += ":" + labels[i] + "=" + labels[i+1]
		}
	}
	
	return key
}

func (m *ProductionMetrics) buildPrefix(prefix string) string {
	if m.prefix != "" {
		return m.prefix + "_" + prefix
	}
	return prefix
}

type productionTimer struct {
	start   time.Time
	name    string
	labels  []string
	metrics *ProductionMetrics
}

func (t *productionTimer) Duration() time.Duration {
	return time.Since(t.start)
}

func (t *productionTimer) Stop() time.Duration {
	duration := time.Since(t.start)
	t.metrics.Observe(t.name, float64(duration.Milliseconds()), t.labels...)
	return duration
}