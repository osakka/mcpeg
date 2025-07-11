package health

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/osakka/mcpeg/pkg/logging"
)

// HTTPHandler provides HTTP endpoints for health checks
type HTTPHandler struct {
	healthManager *HealthManager
	logger        logging.Logger
}

// NewHTTPHandler creates a new health check HTTP handler
func NewHTTPHandler(healthManager *HealthManager, logger logging.Logger) *HTTPHandler {
	return &HTTPHandler{
		healthManager: healthManager,
		logger:        logger.WithComponent("health_http"),
	}
}

// RegisterRoutes registers health check endpoints with an HTTP mux
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/health/live", h.handleLiveness)
	mux.HandleFunc("/health/ready", h.handleReadiness)
	mux.HandleFunc("/health/detailed", h.handleDetailedHealth)
}

// handleHealth provides basic health status
func (h *HTTPHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	start := time.Now()
	health := h.healthManager.GetQuickHealth()
	
	// Set appropriate HTTP status based on health
	statusCode := h.getHTTPStatusCode(health.Status)
	
	// Add response time to context
	health.Context["response_time_ms"] = time.Since(start).Milliseconds()
	
	h.writeHealthResponse(w, statusCode, health)
	
	h.logger.Debug("health_check_request",
		"status", health.Status,
		"status_code", statusCode,
		"response_time_ms", time.Since(start).Milliseconds(),
		"user_agent", r.UserAgent(),
		"remote_addr", r.RemoteAddr)
}

// handleLiveness provides Kubernetes-style liveness probe
func (h *HTTPHandler) handleLiveness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	// Liveness check: Is the application running?
	// This should only fail if the application is completely broken
	isAlive := h.healthManager.IsHealthy()
	
	if isAlive {
		h.writeSimpleResponse(w, http.StatusOK, map[string]interface{}{
			"status": "alive",
			"timestamp": time.Now(),
			"uptime": time.Since(h.healthManager.startTime),
		})
	} else {
		h.writeSimpleResponse(w, http.StatusServiceUnavailable, map[string]interface{}{
			"status": "dead",
			"timestamp": time.Now(),
			"message": "Application is not responsive",
		})
	}
}

// handleReadiness provides Kubernetes-style readiness probe
func (h *HTTPHandler) handleReadiness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	// Readiness check: Is the application ready to serve traffic?
	// This should fail if critical dependencies are unavailable
	isReady := h.healthManager.IsReady()
	
	if isReady {
		h.writeSimpleResponse(w, http.StatusOK, map[string]interface{}{
			"status": "ready",
			"timestamp": time.Now(),
			"message": "Application is ready to serve traffic",
		})
	} else {
		h.writeSimpleResponse(w, http.StatusServiceUnavailable, map[string]interface{}{
			"status": "not_ready",
			"timestamp": time.Now(),
			"message": "Application is not ready to serve traffic",
		})
	}
}

// handleDetailedHealth provides comprehensive health information
func (h *HTTPHandler) handleDetailedHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	start := time.Now()
	
	// Check if full health check is requested
	fullCheck := r.URL.Query().Get("full") == "true"
	
	var health OverallHealth
	if fullCheck {
		health = h.healthManager.GetHealth(r.Context())
	} else {
		health = h.healthManager.GetQuickHealth()
	}
	
	// Add additional context for detailed view
	health.Context["request_type"] = map[string]interface{}{
		"full_check": fullCheck,
		"response_time_ms": time.Since(start).Milliseconds(),
		"endpoint": "/health/detailed",
	}
	
	// Include debug information if requested
	if r.URL.Query().Get("debug") == "true" {
		health.Context["debug"] = map[string]interface{}{
			"go_version": r.Header.Get("Go-Version"),
			"user_agent": r.UserAgent(),
			"remote_addr": r.RemoteAddr,
			"query_params": r.URL.Query(),
		}
	}
	
	statusCode := h.getHTTPStatusCode(health.Status)
	h.writeHealthResponse(w, statusCode, health)
	
	h.logger.Info("detailed_health_check_request",
		"status", health.Status,
		"status_code", statusCode,
		"full_check", fullCheck,
		"response_time_ms", time.Since(start).Milliseconds(),
		"checks_count", len(health.Checks),
		"user_agent", r.UserAgent())
}

// getHTTPStatusCode maps health status to HTTP status codes
func (h *HTTPHandler) getHTTPStatusCode(status HealthStatus) int {
	switch status {
	case StatusHealthy:
		return http.StatusOK
	case StatusDegraded:
		return http.StatusOK // Still serving traffic, but with warnings
	case StatusUnhealthy:
		return http.StatusServiceUnavailable
	case StatusUnknown:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// writeHealthResponse writes a health response with proper headers
func (h *HTTPHandler) writeHealthResponse(w http.ResponseWriter, statusCode int, health OverallHealth) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("X-Health-Status", string(health.Status))
	w.Header().Set("X-Health-Timestamp", health.Timestamp.Format(time.RFC3339))
	w.Header().Set("X-Health-Version", health.Version)
	
	// Add custom health headers
	w.Header().Set("X-Health-Checks-Total", strconv.Itoa(health.Summary.Total))
	w.Header().Set("X-Health-Checks-Healthy", strconv.Itoa(health.Summary.Healthy))
	w.Header().Set("X-Health-Uptime", health.Uptime.String())
	
	w.WriteHeader(statusCode)
	
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(health); err != nil {
		h.logger.Error("failed_to_encode_health_response",
			"error", err,
			"status", health.Status)
		
		// Fallback to simple error response
		h.writeError(w, http.StatusInternalServerError, "Failed to encode health response")
	}
}

// writeSimpleResponse writes a simple JSON response
func (h *HTTPHandler) writeSimpleResponse(w http.ResponseWriter, statusCode int, data map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	
	w.WriteHeader(statusCode)
	
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(data); err != nil {
		h.logger.Error("failed_to_encode_simple_response",
			"error", err,
			"status_code", statusCode)
	}
}

// writeError writes an error response
func (h *HTTPHandler) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	response := map[string]interface{}{
		"error": message,
		"timestamp": time.Now(),
		"status_code": statusCode,
	}
	
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(response); err != nil {
		h.logger.Error("failed_to_encode_error_response",
			"error", err,
			"original_message", message,
			"status_code", statusCode)
	}
}

// HealthMiddleware provides middleware for adding health information to responses
func (h *HTTPHandler) HealthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add health information to response headers
		quickHealth := h.healthManager.GetQuickHealth()
		
		w.Header().Set("X-Service-Health", string(quickHealth.Status))
		w.Header().Set("X-Service-Version", quickHealth.Version)
		w.Header().Set("X-Service-Uptime", quickHealth.Uptime.String())
		
		// Continue with the next handler
		next.ServeHTTP(w, r)
	})
}

// PrometheusHandler provides Prometheus-compatible metrics
func (h *HTTPHandler) PrometheusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	health := h.healthManager.GetQuickHealth()
	
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	
	// Generate Prometheus metrics format
	metrics := h.generatePrometheusMetrics(health)
	w.Write([]byte(metrics))
}

// generatePrometheusMetrics converts health data to Prometheus format
func (h *HTTPHandler) generatePrometheusMetrics(health OverallHealth) string {
	metrics := ""
	
	// Overall health status (0=unknown, 1=healthy, 2=degraded, 3=unhealthy)
	statusValue := 0
	switch health.Status {
	case StatusHealthy:
		statusValue = 1
	case StatusDegraded:
		statusValue = 2
	case StatusUnhealthy:
		statusValue = 3
	}
	
	metrics += fmt.Sprintf("# HELP mcpeg_health_status Overall health status of the service\n")
	metrics += fmt.Sprintf("# TYPE mcpeg_health_status gauge\n")
	metrics += fmt.Sprintf("mcpeg_health_status{version=\"%s\"} %d\n", health.Version, statusValue)
	
	// System uptime
	metrics += fmt.Sprintf("# HELP mcpeg_uptime_seconds Service uptime in seconds\n")
	metrics += fmt.Sprintf("# TYPE mcpeg_uptime_seconds counter\n")
	metrics += fmt.Sprintf("mcpeg_uptime_seconds{version=\"%s\"} %.0f\n", health.Version, health.Uptime.Seconds())
	
	// Health check counts
	metrics += fmt.Sprintf("# HELP mcpeg_health_checks_total Total number of health checks\n")
	metrics += fmt.Sprintf("# TYPE mcpeg_health_checks_total gauge\n")
	metrics += fmt.Sprintf("mcpeg_health_checks_total{status=\"total\"} %d\n", health.Summary.Total)
	metrics += fmt.Sprintf("mcpeg_health_checks_total{status=\"healthy\"} %d\n", health.Summary.Healthy)
	metrics += fmt.Sprintf("mcpeg_health_checks_total{status=\"degraded\"} %d\n", health.Summary.Degraded)
	metrics += fmt.Sprintf("mcpeg_health_checks_total{status=\"unhealthy\"} %d\n", health.Summary.Unhealthy)
	metrics += fmt.Sprintf("mcpeg_health_checks_total{status=\"critical\"} %d\n", health.Summary.Critical)
	
	// Individual check statuses
	metrics += fmt.Sprintf("# HELP mcpeg_health_check_status Status of individual health checks\n")
	metrics += fmt.Sprintf("# TYPE mcpeg_health_check_status gauge\n")
	
	for _, check := range health.Checks {
		checkStatusValue := 0
		switch check.Status {
		case StatusHealthy:
			checkStatusValue = 1
		case StatusDegraded:
			checkStatusValue = 2
		case StatusUnhealthy:
			checkStatusValue = 3
		}
		
		metrics += fmt.Sprintf("mcpeg_health_check_status{name=\"%s\",critical=\"%t\"} %d\n", 
			check.Name, check.Critical, checkStatusValue)
	}
	
	return metrics
}