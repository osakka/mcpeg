package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
)

// HealthStatus represents the overall health state
type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusDegraded  HealthStatus = "degraded"
	StatusUnhealthy HealthStatus = "unhealthy"
	StatusUnknown   HealthStatus = "unknown"
)

// CheckResult represents the result of a single health check
type CheckResult struct {
	Name        string                 `json:"name"`
	Status      HealthStatus           `json:"status"`
	Message     string                 `json:"message"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Duration    time.Duration          `json:"duration"`
	Timestamp   time.Time              `json:"timestamp"`
	Error       string                 `json:"error,omitempty"`
	Critical    bool                   `json:"critical"`
	Suggestions []string               `json:"suggestions,omitempty"`
}

// OverallHealth represents the complete system health
type OverallHealth struct {
	Status      HealthStatus           `json:"status"`
	Timestamp   time.Time              `json:"timestamp"`
	Version     string                 `json:"version"`
	Uptime      time.Duration          `json:"uptime"`
	Checks      []CheckResult          `json:"checks"`
	Summary     HealthSummary          `json:"summary"`
	Context     map[string]interface{} `json:"context"`
	Suggestions []string               `json:"suggestions"`
}

// HealthSummary provides aggregated health information
type HealthSummary struct {
	Total     int `json:"total"`
	Healthy   int `json:"healthy"`
	Degraded  int `json:"degraded"`
	Unhealthy int `json:"unhealthy"`
	Critical  int `json:"critical"`
	Warnings  int `json:"warnings"`
}

// HealthChecker defines the interface for health check implementations
type HealthChecker interface {
	Check(ctx context.Context) CheckResult
	Name() string
	IsCritical() bool
	Interval() time.Duration
}

// HealthManager coordinates all health checks and provides health status
type HealthManager struct {
	checkers     []HealthChecker
	lastResults  map[string]CheckResult
	resultsMutex sync.RWMutex
	logger       logging.Logger
	metrics      metrics.Metrics
	startTime    time.Time
	version      string

	// Configuration
	config HealthConfig

	// Background monitoring
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// HealthConfig configures health check behavior
type HealthConfig struct {
	// Global timeouts
	DefaultTimeout time.Duration `yaml:"default_timeout"`
	GlobalTimeout  time.Duration `yaml:"global_timeout"`

	// Check intervals
	QuickCheckInterval time.Duration `yaml:"quick_check_interval"`
	FullCheckInterval  time.Duration `yaml:"full_check_interval"`

	// Failure handling
	MaxConsecutiveFailures int           `yaml:"max_consecutive_failures"`
	FailureRetryDelay      time.Duration `yaml:"failure_retry_delay"`

	// Degradation thresholds
	DegradedThreshold  float64 `yaml:"degraded_threshold"`
	UnhealthyThreshold float64 `yaml:"unhealthy_threshold"`

	// LLM optimization
	IncludeDetailedDiagnostics bool `yaml:"include_detailed_diagnostics"`
	GenerateSuggestions        bool `yaml:"generate_suggestions"`
}

// NewHealthManager creates a comprehensive health management system
func NewHealthManager(logger logging.Logger, metrics metrics.Metrics, version string) *HealthManager {
	ctx, cancel := context.WithCancel(context.Background())

	hm := &HealthManager{
		checkers:    make([]HealthChecker, 0),
		lastResults: make(map[string]CheckResult),
		logger:      logger.WithComponent("health_manager"),
		metrics:     metrics,
		startTime:   time.Now(),
		version:     version,
		ctx:         ctx,
		cancel:      cancel,
		config:      defaultHealthConfig(),
	}

	// Register core health checks
	hm.registerCoreChecks()

	// Start background monitoring
	hm.startBackgroundMonitoring()

	return hm
}

// RegisterChecker adds a health checker to the system
func (hm *HealthManager) RegisterChecker(checker HealthChecker) {
	hm.checkers = append(hm.checkers, checker)

	hm.logger.Info("health_checker_registered",
		"name", checker.Name(),
		"critical", checker.IsCritical(),
		"interval", checker.Interval(),
		"total_checkers", len(hm.checkers))
}

// GetHealth performs all health checks and returns overall status
func (hm *HealthManager) GetHealth(ctx context.Context) OverallHealth {
	start := time.Now()

	// Create context with global timeout
	checkCtx, cancel := context.WithTimeout(ctx, hm.config.GlobalTimeout)
	defer cancel()

	// Run all health checks concurrently
	results := hm.runAllChecks(checkCtx)

	// Calculate overall status
	overall := hm.calculateOverallHealth(results)
	overall.Context = map[string]interface{}{
		"check_duration":    time.Since(start),
		"concurrent_checks": len(hm.checkers),
		"system_load":       hm.getSystemLoad(),
		"memory_usage":      hm.getMemoryUsage(),
	}

	// Store results for monitoring
	hm.updateStoredResults(results)

	// Record metrics
	hm.recordHealthMetrics(overall)

	// Log health status with LLM context
	hm.logHealthStatus(overall)

	return overall
}

// GetQuickHealth returns cached results for fast health checks
func (hm *HealthManager) GetQuickHealth() OverallHealth {
	hm.resultsMutex.RLock()
	defer hm.resultsMutex.RUnlock()

	// Use cached results
	results := make([]CheckResult, 0, len(hm.lastResults))
	for _, result := range hm.lastResults {
		results = append(results, result)
	}

	return hm.calculateOverallHealth(results)
}

// IsHealthy returns true if the system is healthy or degraded
func (hm *HealthManager) IsHealthy() bool {
	health := hm.GetQuickHealth()
	return health.Status == StatusHealthy || health.Status == StatusDegraded
}

// IsReady returns true if all critical components are healthy
func (hm *HealthManager) IsReady() bool {
	hm.resultsMutex.RLock()
	defer hm.resultsMutex.RUnlock()

	for _, result := range hm.lastResults {
		if result.Critical && result.Status != StatusHealthy {
			return false
		}
	}
	return true
}

// Shutdown gracefully stops health monitoring
func (hm *HealthManager) Shutdown() {
	hm.logger.Info("health_manager_shutting_down")
	hm.cancel()
	hm.wg.Wait()
	hm.logger.Info("health_manager_shutdown_complete")
}

// runAllChecks executes all registered health checks
func (hm *HealthManager) runAllChecks(ctx context.Context) []CheckResult {
	results := make(chan CheckResult, len(hm.checkers))

	// Run checks concurrently
	for _, checker := range hm.checkers {
		go func(c HealthChecker) {
			start := time.Now()

			// Create check-specific context with timeout
			checkCtx, cancel := context.WithTimeout(ctx, hm.config.DefaultTimeout)
			defer cancel()

			result := c.Check(checkCtx)
			result.Duration = time.Since(start)
			result.Timestamp = time.Now()

			// Add LLM debugging context
			if hm.config.IncludeDetailedDiagnostics {
				result.Details = hm.addDiagnosticContext(result.Details, c)
			}

			// Generate suggestions for failures
			if hm.config.GenerateSuggestions && result.Status != StatusHealthy {
				result.Suggestions = hm.generateSuggestions(result, c)
			}

			results <- result
		}(checker)
	}

	// Collect all results
	checkResults := make([]CheckResult, 0, len(hm.checkers))
	for i := 0; i < len(hm.checkers); i++ {
		select {
		case result := <-results:
			checkResults = append(checkResults, result)
		case <-ctx.Done():
			// Timeout - add failed check for missing results
			checkResults = append(checkResults, CheckResult{
				Name:      "timeout_check",
				Status:    StatusUnhealthy,
				Message:   "Health check timed out",
				Error:     ctx.Err().Error(),
				Critical:  true,
				Timestamp: time.Now(),
			})
		}
	}

	return checkResults
}

// calculateOverallHealth determines the system's overall health status
func (hm *HealthManager) calculateOverallHealth(results []CheckResult) OverallHealth {
	summary := HealthSummary{Total: len(results)}
	var status HealthStatus = StatusHealthy
	var suggestions []string

	for _, result := range results {
		switch result.Status {
		case StatusHealthy:
			summary.Healthy++
		case StatusDegraded:
			summary.Degraded++
			if status == StatusHealthy {
				status = StatusDegraded
			}
		case StatusUnhealthy:
			summary.Unhealthy++
			status = StatusUnhealthy
			if result.Critical {
				summary.Critical++
			}
		}

		// Collect suggestions
		suggestions = append(suggestions, result.Suggestions...)
	}

	// Calculate health percentage
	healthyPercentage := float64(summary.Healthy) / float64(summary.Total)

	// Adjust status based on thresholds
	if healthyPercentage < hm.config.UnhealthyThreshold {
		status = StatusUnhealthy
	} else if healthyPercentage < hm.config.DegradedThreshold {
		if status == StatusHealthy {
			status = StatusDegraded
		}
	}

	return OverallHealth{
		Status:      status,
		Timestamp:   time.Now(),
		Version:     hm.version,
		Uptime:      time.Since(hm.startTime),
		Checks:      results,
		Summary:     summary,
		Suggestions: deduplicateStrings(suggestions),
	}
}

// updateStoredResults updates the cached results for quick access
func (hm *HealthManager) updateStoredResults(results []CheckResult) {
	hm.resultsMutex.Lock()
	defer hm.resultsMutex.Unlock()

	for _, result := range results {
		hm.lastResults[result.Name] = result
	}
}

// recordHealthMetrics records health check metrics
func (hm *HealthManager) recordHealthMetrics(health OverallHealth) {
	labels := []string{
		"status", string(health.Status),
		"version", health.Version,
	}

	hm.metrics.Set("health_status", map[string]float64{
		string(StatusHealthy):   0,
		string(StatusDegraded):  0,
		string(StatusUnhealthy): 0,
	}[string(health.Status)], labels...)

	hm.metrics.Set("health_checks_total", float64(health.Summary.Total), labels...)
	hm.metrics.Set("health_checks_healthy", float64(health.Summary.Healthy), labels...)
	hm.metrics.Set("health_checks_degraded", float64(health.Summary.Degraded), labels...)
	hm.metrics.Set("health_checks_unhealthy", float64(health.Summary.Unhealthy), labels...)
	hm.metrics.Set("health_checks_critical", float64(health.Summary.Critical), labels...)
	hm.metrics.Set("system_uptime_seconds", health.Uptime.Seconds(), labels...)

	// Record individual check results
	for _, check := range health.Checks {
		checkLabels := []string{
			"check_name", check.Name,
			"status", string(check.Status),
			"critical", fmt.Sprintf("%t", check.Critical),
		}

		hm.metrics.Set("health_check_duration_seconds", check.Duration.Seconds(), checkLabels...)
		hm.metrics.Inc("health_check_executions_total", checkLabels...)

		if check.Status != StatusHealthy {
			hm.metrics.Inc("health_check_failures_total", checkLabels...)
		}
	}
}

// logHealthStatus logs health status with full LLM context
func (hm *HealthManager) logHealthStatus(health OverallHealth) {
	logLevel := "info"
	message := "health_check_completed"

	if health.Status == StatusUnhealthy {
		logLevel = "error"
		message = "system_unhealthy"
	} else if health.Status == StatusDegraded {
		logLevel = "warn"
		message = "system_degraded"
	}

	fields := []interface{}{
		"overall_status", health.Status,
		"uptime", health.Uptime,
		"version", health.Version,
		"total_checks", health.Summary.Total,
		"healthy_checks", health.Summary.Healthy,
		"degraded_checks", health.Summary.Degraded,
		"unhealthy_checks", health.Summary.Unhealthy,
		"critical_failures", health.Summary.Critical,
		"health_percentage", float64(health.Summary.Healthy) / float64(health.Summary.Total) * 100,
		"suggestions", health.Suggestions,
		"context", health.Context,
	}

	// Add detailed check information for failures
	if health.Status != StatusHealthy {
		failedChecks := []map[string]interface{}{}
		for _, check := range health.Checks {
			if check.Status != StatusHealthy {
				failedChecks = append(failedChecks, map[string]interface{}{
					"name":        check.Name,
					"status":      check.Status,
					"message":     check.Message,
					"error":       check.Error,
					"critical":    check.Critical,
					"duration":    check.Duration,
					"suggestions": check.Suggestions,
					"details":     check.Details,
				})
			}
		}
		fields = append(fields, "failed_checks", failedChecks)
	}

	switch logLevel {
	case "error":
		hm.logger.Error(message, fields...)
	case "warn":
		hm.logger.Warn(message, fields...)
	default:
		hm.logger.Info(message, fields...)
	}
}

// startBackgroundMonitoring starts continuous health monitoring
func (hm *HealthManager) startBackgroundMonitoring() {
	hm.wg.Add(2)

	// Quick health checks (frequent)
	go func() {
		defer hm.wg.Done()
		ticker := time.NewTicker(hm.config.QuickCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Run only critical checks for quick monitoring
				hm.runQuickChecks()
			case <-hm.ctx.Done():
				return
			}
		}
	}()

	// Full health checks (less frequent)
	go func() {
		defer hm.wg.Done()
		ticker := time.NewTicker(hm.config.FullCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Run all checks
				health := hm.GetHealth(hm.ctx)

				// Alert on status changes
				hm.detectStatusChanges(health)

			case <-hm.ctx.Done():
				return
			}
		}
	}()
}

// runQuickChecks runs only critical health checks for fast monitoring
func (hm *HealthManager) runQuickChecks() {
	ctx, cancel := context.WithTimeout(hm.ctx, hm.config.DefaultTimeout)
	defer cancel()

	for _, checker := range hm.checkers {
		if checker.IsCritical() {
			go func(c HealthChecker) {
				result := c.Check(ctx)
				result.Timestamp = time.Now()

				hm.resultsMutex.Lock()
				hm.lastResults[c.Name()] = result
				hm.resultsMutex.Unlock()

				// Alert on critical failures
				if result.Status == StatusUnhealthy {
					hm.logger.Error("critical_health_check_failed",
						"check_name", result.Name,
						"message", result.Message,
						"error", result.Error,
						"suggestions", result.Suggestions)
				}
			}(checker)
		}
	}
}

// detectStatusChanges alerts on health status transitions
func (hm *HealthManager) detectStatusChanges(current OverallHealth) {
	// This would compare with previous status and alert on changes
	// Implementation would depend on alerting system integration
	hm.logger.Debug("health_status_checked",
		"status", current.Status,
		"healthy_percentage", float64(current.Summary.Healthy)/float64(current.Summary.Total)*100)
}

// registerCoreChecks registers essential system health checks
func (hm *HealthManager) registerCoreChecks() {
	// Memory usage check
	hm.RegisterChecker(&MemoryHealthChecker{
		logger: hm.logger.WithComponent("memory_health"),
		config: MemoryCheckConfig{
			WarningThreshold:  0.80, // 80%
			CriticalThreshold: 0.95, // 95%
		},
	})

	// Goroutine count check
	hm.RegisterChecker(&GoroutineHealthChecker{
		logger: hm.logger.WithComponent("goroutine_health"),
		config: GoroutineCheckConfig{
			WarningThreshold:  1000,
			CriticalThreshold: 5000,
		},
	})

	// System load check
	hm.RegisterChecker(&SystemLoadHealthChecker{
		logger: hm.logger.WithComponent("system_load_health"),
	})
}

// Helper functions
func (hm *HealthManager) addDiagnosticContext(existing map[string]interface{}, checker HealthChecker) map[string]interface{} {
	if existing == nil {
		existing = make(map[string]interface{})
	}

	existing["check_interval"] = checker.Interval()
	existing["is_critical"] = checker.IsCritical()
	existing["system_time"] = time.Now()
	existing["uptime"] = time.Since(hm.startTime)

	return existing
}

func (hm *HealthManager) generateSuggestions(result CheckResult, checker HealthChecker) []string {
	suggestions := []string{}

	switch result.Status {
	case StatusUnhealthy:
		suggestions = append(suggestions,
			fmt.Sprintf("Check %s component configuration", checker.Name()),
			"Review recent error logs",
			"Verify resource availability",
			"Consider restarting the service")

	case StatusDegraded:
		suggestions = append(suggestions,
			fmt.Sprintf("Monitor %s component closely", checker.Name()),
			"Check for resource constraints",
			"Review performance metrics")
	}

	return suggestions
}

func (hm *HealthManager) getSystemLoad() float64 {
	// Implementation would read system load average
	return 0.0
}

func (hm *HealthManager) getMemoryUsage() map[string]interface{} {
	// Implementation would read memory statistics
	return map[string]interface{}{
		"used_mb":      0,
		"available_mb": 0,
		"percentage":   0.0,
	}
}

func defaultHealthConfig() HealthConfig {
	return HealthConfig{
		DefaultTimeout:             30 * time.Second,
		GlobalTimeout:              60 * time.Second,
		QuickCheckInterval:         10 * time.Second,
		FullCheckInterval:          60 * time.Second,
		MaxConsecutiveFailures:     3,
		FailureRetryDelay:          5 * time.Second,
		DegradedThreshold:          0.8, // 80% healthy
		UnhealthyThreshold:         0.6, // 60% healthy
		IncludeDetailedDiagnostics: true,
		GenerateSuggestions:        true,
	}
}

func deduplicateStrings(slice []string) []string {
	keys := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}
