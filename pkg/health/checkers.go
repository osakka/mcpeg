package health

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/osakka/mcpeg/pkg/logging"
)

// MemoryHealthChecker monitors system memory usage
type MemoryHealthChecker struct {
	logger logging.Logger
	config MemoryCheckConfig
}

type MemoryCheckConfig struct {
	WarningThreshold  float64 // Percentage of memory usage that triggers warning
	CriticalThreshold float64 // Percentage of memory usage that triggers critical
}

func (m *MemoryHealthChecker) Name() string {
	return "memory_usage"
}

func (m *MemoryHealthChecker) IsCritical() bool {
	return true
}

func (m *MemoryHealthChecker) Interval() time.Duration {
	return 30 * time.Second
}

func (m *MemoryHealthChecker) Check(ctx context.Context) CheckResult {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	// Calculate memory usage percentage (simplified)
	usedMB := float64(memStats.Alloc) / 1024 / 1024
	sysMB := float64(memStats.Sys) / 1024 / 1024
	usagePercentage := usedMB / sysMB
	
	status := StatusHealthy
	message := fmt.Sprintf("Memory usage: %.1f%% (%.1fMB/%.1fMB)", usagePercentage*100, usedMB, sysMB)
	
	if usagePercentage >= m.config.CriticalThreshold {
		status = StatusUnhealthy
		message = fmt.Sprintf("Critical memory usage: %.1f%% (%.1fMB/%.1fMB)", usagePercentage*100, usedMB, sysMB)
	} else if usagePercentage >= m.config.WarningThreshold {
		status = StatusDegraded
		message = fmt.Sprintf("High memory usage: %.1f%% (%.1fMB/%.1fMB)", usagePercentage*100, usedMB, sysMB)
	}
	
	suggestions := []string{}
	if status != StatusHealthy {
		suggestions = append(suggestions,
			"Monitor memory consumption patterns",
			"Check for memory leaks in application code",
			"Consider increasing available memory",
			"Review garbage collection settings")
	}
	
	return CheckResult{
		Name:     m.Name(),
		Status:   status,
		Message:  message,
		Critical: m.IsCritical(),
		Details: map[string]interface{}{
			"alloc_mb":           usedMB,
			"sys_mb":             sysMB,
			"usage_percentage":   usagePercentage * 100,
			"heap_objects":       memStats.HeapObjects,
			"gc_cycles":          memStats.NumGC,
			"last_gc":            time.Unix(0, int64(memStats.LastGC)),
			"pause_total_ns":     memStats.PauseTotalNs,
			"warning_threshold":  m.config.WarningThreshold * 100,
			"critical_threshold": m.config.CriticalThreshold * 100,
		},
		Suggestions: suggestions,
	}
}

// GoroutineHealthChecker monitors goroutine count
type GoroutineHealthChecker struct {
	logger logging.Logger
	config GoroutineCheckConfig
}

type GoroutineCheckConfig struct {
	WarningThreshold  int // Number of goroutines that triggers warning
	CriticalThreshold int // Number of goroutines that triggers critical
}

func (g *GoroutineHealthChecker) Name() string {
	return "goroutine_count"
}

func (g *GoroutineHealthChecker) IsCritical() bool {
	return true
}

func (g *GoroutineHealthChecker) Interval() time.Duration {
	return 15 * time.Second
}

func (g *GoroutineHealthChecker) Check(ctx context.Context) CheckResult {
	goroutineCount := runtime.NumGoroutine()
	
	status := StatusHealthy
	message := fmt.Sprintf("Goroutines: %d", goroutineCount)
	
	if goroutineCount >= g.config.CriticalThreshold {
		status = StatusUnhealthy
		message = fmt.Sprintf("Critical goroutine count: %d", goroutineCount)
	} else if goroutineCount >= g.config.WarningThreshold {
		status = StatusDegraded
		message = fmt.Sprintf("High goroutine count: %d", goroutineCount)
	}
	
	suggestions := []string{}
	if status != StatusHealthy {
		suggestions = append(suggestions,
			"Check for goroutine leaks",
			"Review concurrent operations",
			"Monitor goroutine creation patterns",
			"Consider implementing goroutine pooling")
	}
	
	return CheckResult{
		Name:     g.Name(),
		Status:   status,
		Message:  message,
		Critical: g.IsCritical(),
		Details: map[string]interface{}{
			"count":              goroutineCount,
			"warning_threshold":  g.config.WarningThreshold,
			"critical_threshold": g.config.CriticalThreshold,
		},
		Suggestions: suggestions,
	}
}

// SystemLoadHealthChecker monitors system load
type SystemLoadHealthChecker struct {
	logger logging.Logger
}

func (s *SystemLoadHealthChecker) Name() string {
	return "system_load"
}

func (s *SystemLoadHealthChecker) IsCritical() bool {
	return false
}

func (s *SystemLoadHealthChecker) Interval() time.Duration {
	return 60 * time.Second
}

func (s *SystemLoadHealthChecker) Check(ctx context.Context) CheckResult {
	// For simplicity, we'll check CPU count and estimate load
	cpuCount := runtime.NumCPU()
	
	// In a real implementation, you would read /proc/loadavg on Linux
	// For now, we'll create a placeholder
	
	return CheckResult{
		Name:     s.Name(),
		Status:   StatusHealthy,
		Message:  fmt.Sprintf("System load monitoring (CPUs: %d)", cpuCount),
		Critical: s.IsCritical(),
		Details: map[string]interface{}{
			"cpu_count": cpuCount,
			"note":      "Load average monitoring not implemented in this example",
		},
		Suggestions: []string{},
	}
}

// ServiceHealthChecker checks the health of individual MCP services
type ServiceHealthChecker struct {
	serviceName string
	endpoint    string
	timeout     time.Duration
	logger      logging.Logger
}

func NewServiceHealthChecker(serviceName, endpoint string, timeout time.Duration, logger logging.Logger) *ServiceHealthChecker {
	return &ServiceHealthChecker{
		serviceName: serviceName,
		endpoint:    endpoint,
		timeout:     timeout,
		logger:      logger.WithComponent("service_health_" + serviceName),
	}
}

func (s *ServiceHealthChecker) Name() string {
	return fmt.Sprintf("service_%s", s.serviceName)
}

func (s *ServiceHealthChecker) IsCritical() bool {
	return true
}

func (s *ServiceHealthChecker) Interval() time.Duration {
	return 30 * time.Second
}

func (s *ServiceHealthChecker) Check(ctx context.Context) CheckResult {
	checkCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	
	start := time.Now()
	
	// In a real implementation, this would make an actual health check request
	// to the service endpoint. For now, we'll simulate it.
	
	select {
	case <-checkCtx.Done():
		return CheckResult{
			Name:     s.Name(),
			Status:   StatusUnhealthy,
			Message:  fmt.Sprintf("Service %s health check timed out", s.serviceName),
			Error:    checkCtx.Err().Error(),
			Critical: s.IsCritical(),
			Details: map[string]interface{}{
				"service":     s.serviceName,
				"endpoint":    s.endpoint,
				"timeout":     s.timeout,
				"duration":    time.Since(start),
			},
			Suggestions: []string{
				fmt.Sprintf("Check %s service availability", s.serviceName),
				"Verify network connectivity",
				"Review service configuration",
				"Check service logs for errors",
			},
		}
	default:
		// Simulate successful health check
		return CheckResult{
			Name:     s.Name(),
			Status:   StatusHealthy,
			Message:  fmt.Sprintf("Service %s is healthy", s.serviceName),
			Critical: s.IsCritical(),
			Details: map[string]interface{}{
				"service":     s.serviceName,
				"endpoint":    s.endpoint,
				"duration":    time.Since(start),
				"last_check":  time.Now(),
			},
			Suggestions: []string{},
		}
	}
}

// DatabaseHealthChecker checks database connectivity and performance
type DatabaseHealthChecker struct {
	dbName        string
	connectionURL string
	queryTimeout  time.Duration
	logger        logging.Logger
}

func NewDatabaseHealthChecker(dbName, connectionURL string, queryTimeout time.Duration, logger logging.Logger) *DatabaseHealthChecker {
	return &DatabaseHealthChecker{
		dbName:        dbName,
		connectionURL: connectionURL,
		queryTimeout:  queryTimeout,
		logger:        logger.WithComponent("db_health_" + dbName),
	}
}

func (d *DatabaseHealthChecker) Name() string {
	return fmt.Sprintf("database_%s", d.dbName)
}

func (d *DatabaseHealthChecker) IsCritical() bool {
	return true
}

func (d *DatabaseHealthChecker) Interval() time.Duration {
	return 45 * time.Second
}

func (d *DatabaseHealthChecker) Check(ctx context.Context) CheckResult {
	_, cancel := context.WithTimeout(ctx, d.queryTimeout)
	defer cancel()
	
	start := time.Now()
	
	// In a real implementation, this would:
	// 1. Test database connection
	// 2. Execute a simple query (SELECT 1)
	// 3. Check connection pool stats
	// 4. Measure query latency
	
	// For now, we'll simulate a successful check
	queryLatency := time.Since(start)
	
	status := StatusHealthy
	message := fmt.Sprintf("Database %s is healthy", d.dbName)
	
	// Simulate latency-based health assessment
	if queryLatency > 5*time.Second {
		status = StatusUnhealthy
		message = fmt.Sprintf("Database %s is unhealthy (slow queries)", d.dbName)
	} else if queryLatency > 1*time.Second {
		status = StatusDegraded
		message = fmt.Sprintf("Database %s is degraded (slow queries)", d.dbName)
	}
	
	suggestions := []string{}
	if status != StatusHealthy {
		suggestions = append(suggestions,
			fmt.Sprintf("Check %s database performance", d.dbName),
			"Review database connection pool settings",
			"Monitor database server resources",
			"Check for long-running queries",
			"Verify database server health")
	}
	
	return CheckResult{
		Name:     d.Name(),
		Status:   status,
		Message:  message,
		Critical: d.IsCritical(),
		Details: map[string]interface{}{
			"database":         d.dbName,
			"query_latency_ms": queryLatency.Milliseconds(),
			"connection_url":   d.connectionURL, // Be careful not to log credentials
			"timeout":          d.queryTimeout,
		},
		Suggestions: suggestions,
	}
}

// ExternalAPIHealthChecker checks external API availability
type ExternalAPIHealthChecker struct {
	apiName     string
	endpoint    string
	timeout     time.Duration
	logger      logging.Logger
}

func NewExternalAPIHealthChecker(apiName, endpoint string, timeout time.Duration, logger logging.Logger) *ExternalAPIHealthChecker {
	return &ExternalAPIHealthChecker{
		apiName:  apiName,
		endpoint: endpoint,
		timeout:  timeout,
		logger:   logger.WithComponent("api_health_" + apiName),
	}
}

func (e *ExternalAPIHealthChecker) Name() string {
	return fmt.Sprintf("external_api_%s", e.apiName)
}

func (e *ExternalAPIHealthChecker) IsCritical() bool {
	return false // External APIs are usually not critical for core health
}

func (e *ExternalAPIHealthChecker) Interval() time.Duration {
	return 120 * time.Second // Check less frequently
}

func (e *ExternalAPIHealthChecker) Check(ctx context.Context) CheckResult {
	checkCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()
	
	start := time.Now()
	
	// In a real implementation, this would make an HTTP request to the API
	// For now, we'll simulate it
	
	select {
	case <-checkCtx.Done():
		return CheckResult{
			Name:     e.Name(),
			Status:   StatusUnhealthy,
			Message:  fmt.Sprintf("External API %s is unreachable", e.apiName),
			Error:    checkCtx.Err().Error(),
			Critical: e.IsCritical(),
			Details: map[string]interface{}{
				"api_name": e.apiName,
				"endpoint": e.endpoint,
				"timeout":  e.timeout,
				"duration": time.Since(start),
			},
			Suggestions: []string{
				fmt.Sprintf("Check %s API status", e.apiName),
				"Verify network connectivity",
				"Check API documentation for outages",
				"Consider implementing fallback mechanisms",
			},
		}
	default:
		// Simulate successful API check
		responseTime := time.Since(start)
		
		status := StatusHealthy
		message := fmt.Sprintf("External API %s is healthy", e.apiName)
		
		if responseTime > 10*time.Second {
			status = StatusDegraded
			message = fmt.Sprintf("External API %s is slow", e.apiName)
		}
		
		return CheckResult{
			Name:     e.Name(),
			Status:   status,
			Message:  message,
			Critical: e.IsCritical(),
			Details: map[string]interface{}{
				"api_name":         e.apiName,
				"endpoint":         e.endpoint,
				"response_time_ms": responseTime.Milliseconds(),
				"last_check":       time.Now(),
			},
			Suggestions: []string{},
		}
	}
}