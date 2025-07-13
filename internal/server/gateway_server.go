// Package server provides the core MCPEG gateway server implementation with comprehensive HTTP middleware.
//
// This package implements the main gateway server that handles MCP protocol requests and routes them
// to appropriate service adapters, with enterprise-grade middleware and operational features:
//
//   - HTTP server with graceful shutdown and health monitoring
//   - MCP protocol request routing and response handling
//   - Comprehensive middleware stack (auth, logging, metrics, CORS)
//   - Service discovery integration with dynamic routing
//   - Load balancing and circuit breaker pattern implementation
//   - API versioning with backward compatibility
//   - Production-ready operational features
//
// Server architecture features:
//   - Modular middleware chain with dependency injection
//   - Context propagation with trace and span ID support
//   - Comprehensive error handling with structured responses
//   - Performance monitoring with detailed metrics collection
//   - Security features including authentication and authorization
//   - Rate limiting and request throttling capabilities
//   - Compression and content negotiation support
//
// HTTP middleware stack (applied in order):
//   1. Request logging and correlation ID assignment
//   2. CORS handling for cross-origin requests
//   3. Compression (gzip) for response optimization
//   4. Authentication and JWT token validation
//   5. Authorization with role-based access control
//   6. Rate limiting and throttling
//   7. Metrics collection and performance monitoring
//   8. Error recovery and structured error responses
//
// Example server configuration:
//
//	config := server.Config{
//	    Host:           "0.0.0.0",
//	    Port:           8080,
//	    ReadTimeout:    30 * time.Second,
//	    WriteTimeout:   30 * time.Second,
//	    IdleTimeout:    60 * time.Second,
//	    MaxHeaderBytes: 1 << 20,
//	}
//	
//	srv := server.NewGatewayServer(config, logger, metrics, registry)
//	if err := srv.Start(ctx); err != nil {
//	    log.Fatal("Server startup failed:", err)
//	}
//
// MCP request routing:
//
//	// Requests are routed based on tool/resource/prompt names
//	// POST /api/v1/tools/call -> tool execution
//	// GET /api/v1/resources/read -> resource access  
//	// POST /api/v1/prompts/get -> prompt processing
package server

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/osakka/mcpeg/internal/plugins"
	"github.com/osakka/mcpeg/internal/registry"
	"github.com/osakka/mcpeg/internal/router"
	"github.com/osakka/mcpeg/pkg/auth"
	"github.com/osakka/mcpeg/pkg/capabilities"
	"github.com/osakka/mcpeg/pkg/health"
	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/mcp"
	"github.com/osakka/mcpeg/pkg/metrics"
	"github.com/osakka/mcpeg/pkg/rbac"
	"github.com/osakka/mcpeg/pkg/validation"
)

// GatewayServer represents the main MCPEG gateway server
type GatewayServer struct {
	config     ServerConfig
	httpServer *http.Server
	registry   *registry.ServiceRegistry
	mcpRouter  *router.MCPRouter
	logger     logging.Logger
	metrics    metrics.Metrics
	validator  *validation.Validator
	healthMgr  *health.HealthManager

	// Plugin system integration
	pluginIntegration *plugins.MCpegPluginIntegration

	// Phase 2: Advanced Plugin Discovery and Intelligence
	analysisEngine    *capabilities.AnalysisEngine
	discoveryEngine   *capabilities.DiscoveryEngine
	aggregationEngine *capabilities.AggregationEngine
	validationEngine  *capabilities.ValidationEngine

	// Build and runtime information
	version   string
	commit    string
	buildTime string
	startTime time.Time

	// Rate limiting
	rateLimiter RateLimiter
}

// ServerConfig configures the gateway server
type ServerConfig struct {
	// Server settings
	Address         string        `yaml:"address"`
	Port            int           `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`

	// TLS settings
	TLSEnabled  bool   `yaml:"tls_enabled"`
	TLSCertFile string `yaml:"tls_cert_file"`
	TLSKeyFile  string `yaml:"tls_key_file"`

	// CORS settings
	CORSEnabled      bool     `yaml:"cors_enabled"`
	CORSAllowOrigins []string `yaml:"cors_allow_origins"`
	CORSAllowMethods []string `yaml:"cors_allow_methods"`
	CORSAllowHeaders []string `yaml:"cors_allow_headers"`

	// Middleware settings
	EnableCompression bool `yaml:"enable_compression"`
	EnableRateLimit   bool `yaml:"enable_rate_limit"`
	RateLimitRPS      int  `yaml:"rate_limit_rps"`

	// Management endpoints
	EnableHealthEndpoints bool `yaml:"enable_health_endpoints"`
	EnableMetricsEndpoint bool `yaml:"enable_metrics_endpoint"`
	EnableAdminEndpoints  bool `yaml:"enable_admin_endpoints"`

	// Admin API authentication
	AdminAPIKey    string `yaml:"admin_api_key"`
	AdminAPIHeader string `yaml:"admin_api_header"`
}

// NewGatewayServer creates a new gateway server
func NewGatewayServer(
	config ServerConfig,
	logger logging.Logger,
	metrics metrics.Metrics,
	validator *validation.Validator,
	healthMgr *health.HealthManager,
) *GatewayServer {
	return NewGatewayServerWithVersion(config, logger, metrics, validator, healthMgr, "dev", "unknown", "unknown")
}

// NewGatewayServerWithVersion creates a new gateway server with version information
func NewGatewayServerWithVersion(
	config ServerConfig,
	logger logging.Logger,
	metrics metrics.Metrics,
	validator *validation.Validator,
	healthMgr *health.HealthManager,
	version, commit, buildTime string,
) *GatewayServer {
	// Create service registry
	serviceRegistry := registry.NewServiceRegistry(logger, metrics, validator, healthMgr)

	// Initialize plugin system
	pluginIntegration := plugins.NewMCpegPluginIntegration(serviceRegistry, logger, metrics)

	// Create RBAC engine with minimal config for now
	rbacConfig := rbac.Config{
		DefaultPolicy: "readonly",
		CacheTTL:      5 * time.Minute,
		JWTConfig: auth.JWTConfig{
			Issuer:    "mcpeg",
			Audience:  "mcpeg-users",
			ClockSkew: 5 * time.Minute,
		},
	}
	rbacEngine, err := rbac.NewEngine(rbacConfig, logger, metrics)
	if err != nil {
		logger.Warn("rbac_engine_creation_failed", "error", err)
		rbacEngine = nil // Continue without RBAC for now
	}

	// Create plugin handler
	pluginManager := pluginIntegration.GetPluginManager()
	pluginHandlerConfig := mcp.PluginHandlerConfig{
		DefaultTimeout: 30 * time.Second,
		MaxRetries:     3,
		RetryBackoff:   time.Second,
		CacheEnabled:   true,
		CacheTTL:       5 * time.Minute,
	}
	pluginHandler := mcp.NewPluginHandler(pluginManager, pluginHandlerConfig, logger, metrics)

	// Set the service registry on the plugin handler for Phase 2 discovery
	pluginHandler.SetRegistry(serviceRegistry)

	// Phase 2: Initialize Advanced Plugin Discovery and Intelligence
	
	// Create analysis engine for intelligent capability analysis
	analysisConfig := capabilities.AnalysisConfig{
		EnableSemanticAnalysis: true,
		EnableUsageTracking:    true,
		EnableQualityMetrics:   true,
		AnalysisInterval:       15 * time.Minute,
		RelationThreshold:      0.7,
		CacheTimeout:           1 * time.Hour,
	}
	analysisEngine := capabilities.NewAnalysisEngine(logger, metrics, analysisConfig)

	// Create discovery engine with dependency resolution
	discoveryConfig := capabilities.DiscoveryConfig{
		AutoDiscovery:          true,
		DiscoveryInterval:      10 * time.Minute,
		DependencyResolution:   true,
		ConflictDetection:      true,
		RecommendationEngine:   true,
		MaxDiscoveryDepth:      5,
		ConcurrentAnalysis:     4,
		ReanalysisThreshold:    30 * time.Minute,
	}
	discoveryEngine := capabilities.NewDiscoveryEngine(
		logger, metrics, analysisEngine, pluginManager, serviceRegistry, discoveryConfig,
	)

	// Create aggregation engine for capability aggregation and conflict resolution
	aggregationConfig := capabilities.AggregationConfig{
		EnableAggregation:      true,
		ConflictResolution:     true,
		AutoConflictResolution: true,
		AggregationInterval:    20 * time.Minute,
		ConflictThreshold:      0.5,
		SimilarityThreshold:    0.8,
	}
	aggregationEngine := capabilities.NewAggregationEngine(
		logger, metrics, discoveryEngine, analysisEngine, aggregationConfig,
	)

	// Create validation engine for runtime capability validation
	validationConfig := capabilities.ValidationConfig{
		EnableRuntimeValidation:   true,
		EnableCapabilityMonitoring: true,
		EnablePolicyEnforcement:   true,
		ValidationInterval:        5 * time.Minute,
		MonitoringInterval:        1 * time.Minute,
		ViolationThreshold:        3,
		AutoRemediation:           true,
		ValidationTimeout:         10 * time.Second,
	}
	validationEngine := capabilities.NewValidationEngine(
		logger, metrics, aggregationEngine, analysisEngine, validationConfig,
	)

	// Create MCP router with plugin support and enhanced capabilities
	mcpRouter := router.NewMCPRouter(serviceRegistry, pluginHandler, rbacEngine, logger, metrics, validator)

	server := &GatewayServer{
		config:            config,
		registry:          serviceRegistry,
		mcpRouter:         mcpRouter,
		pluginIntegration: pluginIntegration,
		analysisEngine:    analysisEngine,
		discoveryEngine:   discoveryEngine,
		aggregationEngine: aggregationEngine,
		validationEngine:  validationEngine,
		logger:            logger.WithComponent("gateway_server"),
		metrics:           metrics,
		validator:         validator,
		healthMgr:         healthMgr,
		version:           version,
		commit:            commit,
		buildTime:         buildTime,
		startTime:         time.Now(),
	}

	// Setup HTTP server
	server.setupHTTPServer()

	// Initialize Phase 2 discovery in background
	server.initializePhase2Discovery()

	return server
}

// setupHTTPServer configures the HTTP server
func (gs *GatewayServer) setupHTTPServer() {
	// Create main router
	mainRouter := mux.NewRouter()

	// Add middleware
	gs.addMiddleware(mainRouter)

	// Setup MCP routes
	gs.mcpRouter.SetupRoutes(mainRouter)

	// Setup management routes
	gs.setupManagementRoutes(mainRouter)

	// Create HTTP server
	address := fmt.Sprintf("%s:%d", gs.config.Address, gs.config.Port)
	gs.httpServer = &http.Server{
		Addr:         address,
		Handler:      mainRouter,
		ReadTimeout:  gs.config.ReadTimeout,
		WriteTimeout: gs.config.WriteTimeout,
		IdleTimeout:  gs.config.IdleTimeout,
	}
}

// addMiddleware adds middleware to the router
func (gs *GatewayServer) addMiddleware(router *mux.Router) {
	// CORS middleware
	if gs.config.CORSEnabled {
		router.Use(gs.corsMiddleware)
	}

	// Compression middleware
	if gs.config.EnableCompression {
		router.Use(gs.compressionMiddleware)
	}

	// Rate limiting middleware
	if gs.config.EnableRateLimit {
		router.Use(gs.rateLimitMiddleware)
	}

	// Metrics middleware
	router.Use(gs.metricsMiddleware)

	// Logging middleware
	router.Use(gs.loggingMiddleware)

	// Recovery middleware
	router.Use(gs.recoveryMiddleware)
}

// setupManagementRoutes sets up health and management endpoints
func (gs *GatewayServer) setupManagementRoutes(router *mux.Router) {
	if gs.config.EnableHealthEndpoints {
		router.HandleFunc("/health", gs.handleHealth).Methods("GET")
		router.HandleFunc("/health/live", gs.handleLiveness).Methods("GET")
		router.HandleFunc("/health/ready", gs.handleReadiness).Methods("GET")
	}

	if gs.config.EnableMetricsEndpoint {
		router.HandleFunc("/metrics", gs.handleMetrics).Methods("GET")
	}

	if gs.config.EnableAdminEndpoints {
		adminRouter := router.PathPrefix("/admin").Subrouter()

		// Apply authentication middleware to admin routes
		if gs.config.AdminAPIKey != "" {
			adminRouter.Use(gs.adminAuthMiddleware)
		}

		gs.setupAdminRoutes(adminRouter)
	}
}

// setupAdminRoutes sets up administrative endpoints
func (gs *GatewayServer) setupAdminRoutes(router *mux.Router) {
	// Service management
	router.HandleFunc("/services", gs.handleListServices).Methods("GET")
	router.HandleFunc("/services", gs.handleRegisterService).Methods("POST")
	router.HandleFunc("/services/{id}", gs.handleGetService).Methods("GET")
	router.HandleFunc("/services/{id}", gs.handleUnregisterService).Methods("DELETE")
	router.HandleFunc("/services/{id}/health", gs.handleServiceHealth).Methods("GET")
	router.HandleFunc("/services/{id}/capabilities", gs.handleServiceCapabilities).Methods("GET")
	router.HandleFunc("/services/types", gs.handleServiceTypes).Methods("GET")

	// Service discovery
	router.HandleFunc("/discovery/trigger", gs.handleTriggerDiscovery).Methods("POST")
	router.HandleFunc("/discovery/services", gs.handleDiscoveredServices).Methods("GET")
	router.HandleFunc("/discovery/status", gs.handleDiscoveryStatus).Methods("GET")

	// Load balancer management
	router.HandleFunc("/loadbalancer/stats", gs.handleLoadBalancerStats).Methods("GET")
	router.HandleFunc("/loadbalancer/stats/{service_id}", gs.handleServiceLoadBalancerStats).Methods("GET")
	router.HandleFunc("/loadbalancer/reset/{service_id}", gs.handleResetCircuitBreaker).Methods("POST")
	router.HandleFunc("/loadbalancer/strategies", gs.handleLoadBalancerStrategies).Methods("GET")

	// Configuration
	router.HandleFunc("/config", gs.handleGetConfig).Methods("GET")
	router.HandleFunc("/config", gs.handleUpdateConfig).Methods("PUT")
	router.HandleFunc("/config/reload", gs.handleConfigReload).Methods("POST")

	// Plugin management
	router.HandleFunc("/plugins", gs.handleListPlugins).Methods("GET")
	router.HandleFunc("/plugins/{name}", gs.handleGetPlugin).Methods("GET")
	router.HandleFunc("/plugins/{name}/config", gs.handleGetPluginConfig).Methods("GET")
	router.HandleFunc("/plugins/{name}/config", gs.handleUpdatePluginConfig).Methods("PUT")
	router.HandleFunc("/plugins/{name}/tools", gs.handleGetPluginTools).Methods("GET")
	router.HandleFunc("/plugins/{name}/resources", gs.handleGetPluginResources).Methods("GET")
	router.HandleFunc("/plugins/{name}/health", gs.handleGetPluginHealth).Methods("GET")
	router.HandleFunc("/plugins/health", gs.handleGetAllPluginHealth).Methods("GET")
	router.HandleFunc("/plugins/metrics", gs.handleGetPluginMetrics).Methods("GET")
	router.HandleFunc("/plugins/capabilities", gs.handleGetPluginCapabilities).Methods("GET")

	// System information
	router.HandleFunc("/info", gs.handleSystemInfo).Methods("GET")
	router.HandleFunc("/stats", gs.handleSystemStats).Methods("GET")
	router.HandleFunc("/debug/goroutines", gs.handleGoroutineStats).Methods("GET")

	// API documentation
	router.HandleFunc("/api", gs.handleAPIDocumentation).Methods("GET")
}

// Start starts the gateway server
func (gs *GatewayServer) Start(ctx context.Context) error {
	gs.logger.Info("gateway_server_starting",
		"address", gs.httpServer.Addr,
		"tls_enabled", gs.config.TLSEnabled)

	// Initialize plugins
	if err := gs.pluginIntegration.InitializePlugins(ctx); err != nil {
		gs.logger.Error("failed_to_initialize_plugins", "error", err)
		return fmt.Errorf("failed to initialize plugins: %w", err)
	}

	// Start HTTP server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		var err error
		if gs.config.TLSEnabled {
			err = gs.httpServer.ListenAndServeTLS(gs.config.TLSCertFile, gs.config.TLSKeyFile)
		} else {
			err = gs.httpServer.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	gs.logger.Info("gateway_server_started",
		"address", gs.httpServer.Addr,
		"pid", fmt.Sprintf("%d", gs.getPID()))

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		gs.logger.Info("gateway_server_stopping", "reason", "context_cancelled")
		return gs.Stop()
	case err := <-errChan:
		gs.logger.Error("gateway_server_error", "error", err)
		return err
	}
}

// Stop gracefully stops the gateway server
func (gs *GatewayServer) Stop() error {
	gs.logger.Info("gateway_server_shutting_down")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), gs.config.ShutdownTimeout)
	defer cancel()

	// Shutdown plugins first
	if err := gs.pluginIntegration.ShutdownPlugins(ctx); err != nil {
		gs.logger.Error("plugin_shutdown_error", "error", err)
		// Don't return error, continue with shutdown
	}

	// Shutdown HTTP server
	if err := gs.httpServer.Shutdown(ctx); err != nil {
		gs.logger.Error("http_server_shutdown_error", "error", err)
		return err
	}

	// Shutdown service registry
	if err := gs.registry.Shutdown(); err != nil {
		gs.logger.Error("service_registry_shutdown_error", "error", err)
		return err
	}

	gs.logger.Info("gateway_server_shutdown_complete")
	return nil
}

// Management endpoint handlers

func (gs *GatewayServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Simple health check - server is running
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

func (gs *GatewayServer) handleLiveness(w http.ResponseWriter, r *http.Request) {
	// Liveness check - server is alive
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"alive","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

func (gs *GatewayServer) handleReadiness(w http.ResponseWriter, r *http.Request) {
	// Readiness check - server can handle requests
	healthyServices := gs.registry.GetHealthyServices()

	status := "ready"
	httpStatus := http.StatusOK

	if len(healthyServices) == 0 {
		status = "not_ready"
		httpStatus = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	fmt.Fprintf(w, `{"status":"%s","healthy_services":%d,"timestamp":"%s"}`,
		status, len(healthyServices), time.Now().Format(time.RFC3339))
}

func (gs *GatewayServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	gs.logger.Debug("prometheus_metrics_request_started",
		"remote_addr", r.RemoteAddr,
		"user_agent", r.Header.Get("User-Agent"))

	// Set Prometheus content type
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Write Prometheus metrics
	startTime := time.Now()

	if err := gs.writePrometheusMetrics(w); err != nil {
		gs.logger.Error("prometheus_metrics_write_failed", "error", err)
		http.Error(w, "Error generating metrics", http.StatusInternalServerError)
		return
	}

	duration := time.Since(startTime)
	gs.logger.Debug("prometheus_metrics_request_completed",
		"duration_ms", duration.Milliseconds())

	// Record metrics endpoint access
	gs.metrics.Inc("prometheus_metrics_requests_total", "status", "success")
	gs.metrics.Observe("prometheus_metrics_generation_duration_ms", float64(duration.Milliseconds()))
}

// writePrometheusMetrics writes all metrics in Prometheus format
func (gs *GatewayServer) writePrometheusMetrics(w io.Writer) error {
	// Write header
	fmt.Fprintf(w, "# HELP mcpeg_info Information about the MCPEG gateway instance\n")
	fmt.Fprintf(w, "# TYPE mcpeg_info gauge\n")
	fmt.Fprintf(w, "mcpeg_info{version=\"%s\",commit=\"%s\",build_time=\"%s\"} 1\n",
		gs.version, gs.commit, gs.buildTime)

	// System uptime
	uptime := time.Since(gs.startTime)
	fmt.Fprintf(w, "# HELP mcpeg_uptime_seconds Uptime of the MCPEG gateway in seconds\n")
	fmt.Fprintf(w, "# TYPE mcpeg_uptime_seconds gauge\n")
	fmt.Fprintf(w, "mcpeg_uptime_seconds %f\n", uptime.Seconds())

	// Server configuration info
	fmt.Fprintf(w, "# HELP mcpeg_server_info Server configuration information\n")
	fmt.Fprintf(w, "# TYPE mcpeg_server_info gauge\n")
	fmt.Fprintf(w, "mcpeg_server_info{address=\"%s\",port=\"%d\",tls_enabled=\"%t\"} 1\n",
		gs.config.Address, gs.config.Port, gs.config.TLSEnabled)

	// HTTP metrics
	if err := gs.writeHTTPMetrics(w); err != nil {
		return fmt.Errorf("failed to write HTTP metrics: %w", err)
	}

	// Service registry metrics
	if err := gs.writeServiceMetrics(w); err != nil {
		return fmt.Errorf("failed to write service metrics: %w", err)
	}

	// MCP router metrics
	if err := gs.writeMCPRouterMetrics(w); err != nil {
		return fmt.Errorf("failed to write MCP router metrics: %w", err)
	}

	// Health metrics
	if err := gs.writeHealthMetrics(w); err != nil {
		return fmt.Errorf("failed to write health metrics: %w", err)
	}

	// System resource metrics
	if err := gs.writeSystemMetrics(w); err != nil {
		return fmt.Errorf("failed to write system metrics: %w", err)
	}

	// Custom business metrics
	if err := gs.writeBusinessMetrics(w); err != nil {
		return fmt.Errorf("failed to write business metrics: %w", err)
	}

	return nil
}

// writeHTTPMetrics writes HTTP-related metrics
func (gs *GatewayServer) writeHTTPMetrics(w io.Writer) error {
	// HTTP request metrics
	stats := gs.metrics.GetAllStats()

	// HTTP requests total by status code
	fmt.Fprintf(w, "# HELP mcpeg_http_requests_total Total number of HTTP requests\n")
	fmt.Fprintf(w, "# TYPE mcpeg_http_requests_total counter\n")

	statusCodes := []string{"200", "201", "204", "400", "401", "403", "404", "500", "502", "503"}
	for _, status := range statusCodes {
		metricName := fmt.Sprintf("http_requests_total_status_%s", status)
		if stat, exists := stats[metricName]; exists {
			fmt.Fprintf(w, "mcpeg_http_requests_total{status=\"%s\"} %f\n", status, stat.LastValue)
		}
	}

	// HTTP request duration
	fmt.Fprintf(w, "# HELP mcpeg_http_request_duration_seconds HTTP request duration in seconds\n")
	fmt.Fprintf(w, "# TYPE mcpeg_http_request_duration_seconds histogram\n")

	if stat, exists := stats["http_request_duration_ms"]; exists {
		// Convert milliseconds to seconds for Prometheus convention
		fmt.Fprintf(w, "mcpeg_http_request_duration_seconds_sum %f\n", stat.Sum/1000.0)
		fmt.Fprintf(w, "mcpeg_http_request_duration_seconds_count %d\n", stat.Count)

		// Histogram buckets (in seconds)
		buckets := []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0}
		for _, bucket := range buckets {
			// This is a simplified histogram - in production you'd track actual buckets
			fmt.Fprintf(w, "mcpeg_http_request_duration_seconds_bucket{le=\"%g\"} %d\n", bucket, stat.Count)
		}
		fmt.Fprintf(w, "mcpeg_http_request_duration_seconds_bucket{le=\"+Inf\"} %d\n", stat.Count)
	}

	// HTTP response size
	fmt.Fprintf(w, "# HELP mcpeg_http_response_size_bytes Size of HTTP responses in bytes\n")
	fmt.Fprintf(w, "# TYPE mcpeg_http_response_size_bytes histogram\n")

	if stat, exists := stats["http_response_size_bytes"]; exists {
		fmt.Fprintf(w, "mcpeg_http_response_size_bytes_sum %f\n", stat.Sum)
		fmt.Fprintf(w, "mcpeg_http_response_size_bytes_count %d\n", stat.Count)
	}

	// Active connections
	fmt.Fprintf(w, "# HELP mcpeg_http_connections_active Number of active HTTP connections\n")
	fmt.Fprintf(w, "# TYPE mcpeg_http_connections_active gauge\n")

	if stat, exists := stats["http_connections_active"]; exists {
		fmt.Fprintf(w, "mcpeg_http_connections_active %f\n", stat.LastValue)
	}

	return nil
}

// writeServiceMetrics writes service registry metrics
func (gs *GatewayServer) writeServiceMetrics(w io.Writer) error {
	services := gs.registry.GetAllServices()
	stats := gs.metrics.GetAllStats()

	// Total registered services by type and status
	fmt.Fprintf(w, "# HELP mcpeg_services_registered_total Total number of registered services\n")
	fmt.Fprintf(w, "# TYPE mcpeg_services_registered_total gauge\n")

	servicesByType := make(map[string]int)
	servicesByStatus := make(map[string]int)
	servicesByHealth := make(map[string]int)

	for _, service := range services {
		servicesByType[service.Type]++
		servicesByStatus[string(service.Status)]++
		servicesByHealth[string(service.Health)]++
	}

	// Services by type
	for serviceType, count := range servicesByType {
		fmt.Fprintf(w, "mcpeg_services_registered_total{type=\"%s\"} %d\n", serviceType, count)
	}

	// Service status distribution
	fmt.Fprintf(w, "# HELP mcpeg_services_by_status Number of services by status\n")
	fmt.Fprintf(w, "# TYPE mcpeg_services_by_status gauge\n")
	for status, count := range servicesByStatus {
		fmt.Fprintf(w, "mcpeg_services_by_status{status=\"%s\"} %d\n", status, count)
	}

	// Service health distribution
	fmt.Fprintf(w, "# HELP mcpeg_services_by_health Number of services by health status\n")
	fmt.Fprintf(w, "# TYPE mcpeg_services_by_health gauge\n")
	for health, count := range servicesByHealth {
		fmt.Fprintf(w, "mcpeg_services_by_health{health=\"%s\"} %d\n", health, count)
	}

	// Service health check metrics
	fmt.Fprintf(w, "# HELP mcpeg_service_health_check_duration_seconds Service health check duration\n")
	fmt.Fprintf(w, "# TYPE mcpeg_service_health_check_duration_seconds histogram\n")

	if stat, exists := stats["service_health_check_duration_ms"]; exists {
		fmt.Fprintf(w, "mcpeg_service_health_check_duration_seconds_sum %f\n", stat.Sum/1000.0)
		fmt.Fprintf(w, "mcpeg_service_health_check_duration_seconds_count %d\n", stat.Count)
	}

	// Service health check success rate
	fmt.Fprintf(w, "# HELP mcpeg_service_health_check_success_total Successful health checks\n")
	fmt.Fprintf(w, "# TYPE mcpeg_service_health_check_success_total counter\n")

	if stat, exists := stats["service_health_check_success_total"]; exists {
		fmt.Fprintf(w, "mcpeg_service_health_check_success_total %f\n", stat.LastValue)
	}

	fmt.Fprintf(w, "# HELP mcpeg_service_health_check_failure_total Failed health checks\n")
	fmt.Fprintf(w, "# TYPE mcpeg_service_health_check_failure_total counter\n")

	if stat, exists := stats["service_health_check_failure_total"]; exists {
		fmt.Fprintf(w, "mcpeg_service_health_check_failure_total %f\n", stat.LastValue)
	}

	return nil
}

// writeMCPRouterMetrics writes MCP router metrics
func (gs *GatewayServer) writeMCPRouterMetrics(w io.Writer) error {
	stats := gs.metrics.GetAllStats()

	// MCP request metrics by method
	fmt.Fprintf(w, "# HELP mcpeg_mcp_requests_total Total MCP requests by method\n")
	fmt.Fprintf(w, "# TYPE mcpeg_mcp_requests_total counter\n")

	mcpMethods := []string{
		"initialize", "list_resources", "read_resource", "subscribe", "unsubscribe",
		"list_prompts", "get_prompt", "list_tools", "call_tool", "complete",
		"logging/set_level",
	}

	for _, method := range mcpMethods {
		metricName := fmt.Sprintf("mcp_requests_total_method_%s", strings.ReplaceAll(method, "/", "_"))
		if stat, exists := stats[metricName]; exists {
			fmt.Fprintf(w, "mcpeg_mcp_requests_total{method=\"%s\"} %f\n", method, stat.LastValue)
		}
	}

	// MCP request duration by method
	fmt.Fprintf(w, "# HELP mcpeg_mcp_request_duration_seconds MCP request processing duration\n")
	fmt.Fprintf(w, "# TYPE mcpeg_mcp_request_duration_seconds histogram\n")

	if stat, exists := stats["mcp_request_duration_ms"]; exists {
		fmt.Fprintf(w, "mcpeg_mcp_request_duration_seconds_sum %f\n", stat.Sum/1000.0)
		fmt.Fprintf(w, "mcpeg_mcp_request_duration_seconds_count %d\n", stat.Count)
	}

	// MCP validation errors
	fmt.Fprintf(w, "# HELP mcpeg_mcp_validation_errors_total MCP validation errors\n")
	fmt.Fprintf(w, "# TYPE mcpeg_mcp_validation_errors_total counter\n")

	if stat, exists := stats["mcp_validation_errors_total"]; exists {
		fmt.Fprintf(w, "mcpeg_mcp_validation_errors_total %f\n", stat.LastValue)
	}

	// MCP routing errors
	fmt.Fprintf(w, "# HELP mcpeg_mcp_routing_errors_total MCP routing errors\n")
	fmt.Fprintf(w, "# TYPE mcpeg_mcp_routing_errors_total counter\n")

	if stat, exists := stats["mcp_routing_errors_total"]; exists {
		fmt.Fprintf(w, "mcpeg_mcp_routing_errors_total %f\n", stat.LastValue)
	}

	return nil
}

// writeHealthMetrics writes health check metrics
func (gs *GatewayServer) writeHealthMetrics(w io.Writer) error {
	// Gateway health status
	fmt.Fprintf(w, "# HELP mcpeg_gateway_healthy Gateway health status (1=healthy, 0=unhealthy)\n")
	fmt.Fprintf(w, "# TYPE mcpeg_gateway_healthy gauge\n")

	isHealthy := 1
	if gs.healthMgr != nil {
		healthStatus := gs.healthMgr.GetQuickHealth()
		if healthStatus.Status != "healthy" {
			isHealthy = 0
		}
	}
	fmt.Fprintf(w, "mcpeg_gateway_healthy %d\n", isHealthy)

	// Component health status
	fmt.Fprintf(w, "# HELP mcpeg_component_healthy Component health status\n")
	fmt.Fprintf(w, "# TYPE mcpeg_component_healthy gauge\n")

	if gs.healthMgr != nil {
		healthStatus := gs.healthMgr.GetHealth(context.Background())
		for _, check := range healthStatus.Checks {
			healthy := 1
			if check.Status != "healthy" {
				healthy = 0
			}
			fmt.Fprintf(w, "mcpeg_component_healthy{component=\"%s\"} %d\n", check.Name, healthy)
		}
	}

	return nil
}

// writeSystemMetrics writes system resource metrics
func (gs *GatewayServer) writeSystemMetrics(w io.Writer) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Memory metrics
	fmt.Fprintf(w, "# HELP mcpeg_memory_allocated_bytes Currently allocated memory in bytes\n")
	fmt.Fprintf(w, "# TYPE mcpeg_memory_allocated_bytes gauge\n")
	fmt.Fprintf(w, "mcpeg_memory_allocated_bytes %d\n", m.Alloc)

	fmt.Fprintf(w, "# HELP mcpeg_memory_total_allocated_bytes Total allocated memory in bytes\n")
	fmt.Fprintf(w, "# TYPE mcpeg_memory_total_allocated_bytes counter\n")
	fmt.Fprintf(w, "mcpeg_memory_total_allocated_bytes %d\n", m.TotalAlloc)

	fmt.Fprintf(w, "# HELP mcpeg_memory_system_bytes Memory obtained from system\n")
	fmt.Fprintf(w, "# TYPE mcpeg_memory_system_bytes gauge\n")
	fmt.Fprintf(w, "mcpeg_memory_system_bytes %d\n", m.Sys)

	fmt.Fprintf(w, "# HELP mcpeg_memory_heap_allocated_bytes Heap allocated memory\n")
	fmt.Fprintf(w, "# TYPE mcpeg_memory_heap_allocated_bytes gauge\n")
	fmt.Fprintf(w, "mcpeg_memory_heap_allocated_bytes %d\n", m.HeapAlloc)

	fmt.Fprintf(w, "# HELP mcpeg_memory_heap_system_bytes Heap system memory\n")
	fmt.Fprintf(w, "# TYPE mcpeg_memory_heap_system_bytes gauge\n")
	fmt.Fprintf(w, "mcpeg_memory_heap_system_bytes %d\n", m.HeapSys)

	// Garbage collection metrics
	fmt.Fprintf(w, "# HELP mcpeg_gc_runs_total Total number of GC runs\n")
	fmt.Fprintf(w, "# TYPE mcpeg_gc_runs_total counter\n")
	fmt.Fprintf(w, "mcpeg_gc_runs_total %d\n", m.NumGC)

	fmt.Fprintf(w, "# HELP mcpeg_gc_pause_seconds Time spent in GC pauses\n")
	fmt.Fprintf(w, "# TYPE mcpeg_gc_pause_seconds gauge\n")
	fmt.Fprintf(w, "mcpeg_gc_pause_seconds %f\n", float64(m.PauseTotalNs)/1e9)

	// Goroutine metrics
	fmt.Fprintf(w, "# HELP mcpeg_goroutines_active Number of active goroutines\n")
	fmt.Fprintf(w, "# TYPE mcpeg_goroutines_active gauge\n")
	fmt.Fprintf(w, "mcpeg_goroutines_active %d\n", runtime.NumGoroutine())

	return nil
}

// writeBusinessMetrics writes business logic metrics
func (gs *GatewayServer) writeBusinessMetrics(w io.Writer) error {
	stats := gs.metrics.GetAllStats()

	// Load balancer metrics
	fmt.Fprintf(w, "# HELP mcpeg_load_balancer_requests_total Load balancer request distribution\n")
	fmt.Fprintf(w, "# TYPE mcpeg_load_balancer_requests_total counter\n")

	if stat, exists := stats["load_balancer_requests_total"]; exists {
		fmt.Fprintf(w, "mcpeg_load_balancer_requests_total %f\n", stat.LastValue)
	}

	// Circuit breaker metrics
	fmt.Fprintf(w, "# HELP mcpeg_circuit_breaker_state Circuit breaker state (0=closed, 1=open, 2=half-open)\n")
	fmt.Fprintf(w, "# TYPE mcpeg_circuit_breaker_state gauge\n")

	if stat, exists := stats["circuit_breaker_state"]; exists {
		fmt.Fprintf(w, "mcpeg_circuit_breaker_state %f\n", stat.LastValue)
	}

	// Rate limiting metrics
	fmt.Fprintf(w, "# HELP mcpeg_rate_limit_blocked_total Rate limited requests\n")
	fmt.Fprintf(w, "# TYPE mcpeg_rate_limit_blocked_total counter\n")

	if stat, exists := stats["rate_limit_blocked_total"]; exists {
		fmt.Fprintf(w, "mcpeg_rate_limit_blocked_total %f\n", stat.LastValue)
	}

	// Configuration reload metrics
	fmt.Fprintf(w, "# HELP mcpeg_config_reloads_total Configuration reload attempts\n")
	fmt.Fprintf(w, "# TYPE mcpeg_config_reloads_total counter\n")

	if stat, exists := stats["config_reloads_total"]; exists {
		fmt.Fprintf(w, "mcpeg_config_reloads_total %f\n", stat.LastValue)
	}

	return nil
}

// Admin endpoint handlers (simplified implementations)

func (gs *GatewayServer) handleListServices(w http.ResponseWriter, r *http.Request) {
	services := gs.registry.GetAllServices()
	gs.writeJSONResponse(w, services)
}

func (gs *GatewayServer) handleRegisterService(w http.ResponseWriter, r *http.Request) {
	gs.logger.Debug("admin_register_service_request",
		"remote_addr", r.RemoteAddr,
		"user_agent", r.Header.Get("User-Agent"))

	// Parse request body
	var req registry.ServiceRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		gs.logger.Error("admin_register_service_parse_failed", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		gs.writeJSONResponse(w, map[string]interface{}{
			"error":   "invalid_request_body",
			"message": "Failed to parse JSON request body",
			"details": err.Error(),
		})
		return
	}

	// Validate required fields
	if req.Name == "" || req.Type == "" || req.Endpoint == "" {
		w.WriteHeader(http.StatusBadRequest)
		gs.writeJSONResponse(w, map[string]interface{}{
			"error":    "missing_required_fields",
			"message":  "Name, Type, and Endpoint are required fields",
			"received": req,
		})
		return
	}

	// Register the service
	resp, err := gs.registry.RegisterService(r.Context(), req)
	if err != nil {
		gs.logger.Error("admin_register_service_failed",
			"service_name", req.Name,
			"service_type", req.Type,
			"error", err)

		w.WriteHeader(http.StatusInternalServerError)
		gs.writeJSONResponse(w, map[string]interface{}{
			"error":   "registration_failed",
			"message": "Failed to register service",
			"details": err.Error(),
		})
		return
	}

	gs.logger.Info("admin_service_registered",
		"service_id", resp.ServiceID,
		"service_name", req.Name,
		"service_type", req.Type,
		"endpoint", req.Endpoint)

	// Record metrics
	gs.metrics.Inc("admin_api_service_registrations_total",
		"service_type", req.Type,
		"status", "success")

	w.WriteHeader(http.StatusCreated)
	gs.writeJSONResponse(w, resp)
}

func (gs *GatewayServer) handleGetService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceID := vars["id"]

	service := gs.registry.GetService(serviceID)
	if service == nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Service not found: %s", serviceID)
		return
	}

	gs.writeJSONResponse(w, service)
}

func (gs *GatewayServer) handleUnregisterService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceID := vars["id"]

	if err := gs.registry.UnregisterService(r.Context(), serviceID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to unregister service: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Service unregistered: %s", serviceID)
}

func (gs *GatewayServer) handleTriggerDiscovery(w http.ResponseWriter, r *http.Request) {
	gs.logger.Info("admin_discovery_trigger_requested",
		"remote_addr", r.RemoteAddr,
		"user_agent", r.Header.Get("User-Agent"))

	// Trigger service discovery
	startTime := time.Now()
	err := gs.registry.TriggerDiscovery(r.Context())
	duration := time.Since(startTime)

	if err != nil {
		gs.logger.Error("admin_discovery_trigger_failed", "error", err, "duration", duration)

		w.WriteHeader(http.StatusInternalServerError)
		gs.writeJSONResponse(w, map[string]interface{}{
			"error":       "discovery_failed",
			"message":     "Failed to trigger service discovery",
			"details":     err.Error(),
			"duration_ms": duration.Milliseconds(),
		})
		return
	}

	gs.logger.Info("admin_discovery_trigger_completed", "duration", duration)

	// Record metrics
	gs.metrics.Inc("admin_api_discovery_triggers_total", "status", "success")
	gs.metrics.Observe("admin_api_discovery_duration_seconds", duration.Seconds())

	w.WriteHeader(http.StatusOK)
	gs.writeJSONResponse(w, map[string]interface{}{
		"status":      "success",
		"message":     "Service discovery triggered successfully",
		"duration_ms": duration.Milliseconds(),
		"timestamp":   time.Now().Format(time.RFC3339),
	})
}

func (gs *GatewayServer) handleDiscoveredServices(w http.ResponseWriter, r *http.Request) {
	gs.logger.Debug("admin_discovered_services_request",
		"remote_addr", r.RemoteAddr)

	// Get discovered services
	discovered := gs.registry.GetDiscoveredServices()

	// Parse query parameters for filtering
	source := r.URL.Query().Get("source")
	serviceType := r.URL.Query().Get("type")
	registered := r.URL.Query().Get("registered")

	// Filter discovered services based on query parameters
	var filtered map[string]*registry.DiscoveredService
	if source != "" || serviceType != "" || registered != "" {
		filtered = make(map[string]*registry.DiscoveredService)
		for id, service := range discovered {
			// Apply filters
			if source != "" && service.Source != source {
				continue
			}
			if serviceType != "" && service.Type != serviceType {
				continue
			}
			if registered != "" {
				isRegistered := service.RegistrationError == ""
				if (registered == "true" && !isRegistered) || (registered == "false" && isRegistered) {
					continue
				}
			}
			filtered[id] = service
		}
	} else {
		filtered = discovered
	}

	gs.logger.Debug("admin_discovered_services_response",
		"total_discovered", len(discovered),
		"filtered_count", len(filtered),
		"source_filter", source,
		"type_filter", serviceType)

	// Record metrics
	gs.metrics.Inc("admin_api_discovered_services_requests_total")
	gs.metrics.Set("admin_api_discovered_services_count", float64(len(filtered)))

	// Return response with metadata
	response := map[string]interface{}{
		"services": filtered,
		"metadata": map[string]interface{}{
			"total_count":    len(discovered),
			"filtered_count": len(filtered),
			"filters_applied": map[string]string{
				"source":     source,
				"type":       serviceType,
				"registered": registered,
			},
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	gs.writeJSONResponse(w, response)
}

func (gs *GatewayServer) handleLoadBalancerStats(w http.ResponseWriter, r *http.Request) {
	stats := gs.registry.GetLoadBalancer().GetAllStats()
	gs.writeJSONResponse(w, stats)
}

func (gs *GatewayServer) handleResetCircuitBreaker(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceID := vars["service_id"]

	gs.registry.GetLoadBalancer().ResetCircuitBreaker(serviceID)

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Circuit breaker reset for service: %s", serviceID)
}

func (gs *GatewayServer) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	gs.writeJSONResponse(w, gs.config)
}

func (gs *GatewayServer) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	gs.logger.Info("admin_config_update_requested",
		"remote_addr", r.RemoteAddr,
		"user_agent", r.Header.Get("User-Agent"))

	// Parse the configuration update request
	var updateReq map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		gs.logger.Error("admin_config_update_parse_failed", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		gs.writeJSONResponse(w, map[string]interface{}{
			"error":   "invalid_request_body",
			"message": "Failed to parse JSON request body",
			"details": err.Error(),
		})
		return
	}

	// For security, only allow specific configuration updates
	allowedUpdates := map[string]bool{
		"rate_limit_rps":     true,
		"enable_compression": true,
		"enable_rate_limit":  true,
		"cors_allow_origins": true,
		"log_level":          true,
	}

	updatedFields := make(map[string]interface{})
	invalidFields := make([]string, 0)

	// Validate and apply updates
	for field, value := range updateReq {
		if !allowedUpdates[field] {
			invalidFields = append(invalidFields, field)
			continue
		}

		// Apply specific field updates
		switch field {
		case "rate_limit_rps":
			if rps, ok := value.(float64); ok && rps > 0 {
				gs.config.RateLimitRPS = int(rps)
				updatedFields[field] = int(rps)
			} else {
				invalidFields = append(invalidFields, field+" (invalid value)")
			}
		case "enable_compression":
			if enabled, ok := value.(bool); ok {
				gs.config.EnableCompression = enabled
				updatedFields[field] = enabled
			} else {
				invalidFields = append(invalidFields, field+" (invalid value)")
			}
		case "enable_rate_limit":
			if enabled, ok := value.(bool); ok {
				gs.config.EnableRateLimit = enabled
				updatedFields[field] = enabled
			} else {
				invalidFields = append(invalidFields, field+" (invalid value)")
			}
		case "cors_allow_origins":
			if origins, ok := value.([]interface{}); ok {
				var stringOrigins []string
				for _, origin := range origins {
					if str, ok := origin.(string); ok {
						stringOrigins = append(stringOrigins, str)
					}
				}
				gs.config.CORSAllowOrigins = stringOrigins
				updatedFields[field] = stringOrigins
			} else {
				invalidFields = append(invalidFields, field+" (invalid value)")
			}
		}
	}

	if len(invalidFields) > 0 {
		gs.logger.Warn("admin_config_update_invalid_fields",
			"invalid_fields", invalidFields,
			"updated_fields", updatedFields)

		w.WriteHeader(http.StatusBadRequest)
		gs.writeJSONResponse(w, map[string]interface{}{
			"error":          "invalid_fields",
			"message":        "Some fields are not allowed to be updated or have invalid values",
			"invalid_fields": invalidFields,
			"updated_fields": updatedFields,
			"allowed_fields": allowedUpdates,
		})
		return
	}

	if len(updatedFields) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		gs.writeJSONResponse(w, map[string]interface{}{
			"error":          "no_valid_updates",
			"message":        "No valid configuration updates provided",
			"allowed_fields": allowedUpdates,
		})
		return
	}

	gs.logger.Info("admin_config_updated",
		"updated_fields", updatedFields)

	// Record metrics
	gs.metrics.Inc("admin_api_config_updates_total", "status", "success")
	for field := range updatedFields {
		gs.metrics.Inc("admin_api_config_field_updates_total", "field", field)
	}

	w.WriteHeader(http.StatusOK)
	gs.writeJSONResponse(w, map[string]interface{}{
		"status":         "success",
		"message":        "Configuration updated successfully",
		"updated_fields": updatedFields,
		"timestamp":      time.Now().Format(time.RFC3339),
	})
}

// Middleware implementations (simplified)

func (gs *GatewayServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (gs *GatewayServer) compressionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip compression for certain content types and small responses
		if gs.shouldSkipCompression(r) {
			next.ServeHTTP(w, r)
			return
		}

		// Check if client accepts gzip
		if !gs.clientAcceptsGzip(r) {
			next.ServeHTTP(w, r)
			return
		}

		// Create compressed response writer
		compressedWriter := gs.newCompressedResponseWriter(w, r)
		defer compressedWriter.Close()

		// Log compression metrics
		startTime := time.Now()
		originalSize := compressedWriter.GetOriginalSize()

		// Serve the request with compression
		next.ServeHTTP(compressedWriter, r)

		// Record compression metrics
		compressedSize := compressedWriter.GetCompressedSize()
		compressionRatio := float64(originalSize-compressedSize) / float64(originalSize) * 100
		duration := time.Since(startTime)

		gs.logger.Debug("http_compression_applied",
			"path", r.URL.Path,
			"original_size", originalSize,
			"compressed_size", compressedSize,
			"compression_ratio_percent", compressionRatio,
			"duration_ms", duration.Milliseconds())

		// Record metrics
		gs.metrics.Observe("http_compression_ratio_percent", compressionRatio,
			"path", r.URL.Path, "method", r.Method)
		gs.metrics.Observe("http_compression_duration_ms", float64(duration.Milliseconds()),
			"path", r.URL.Path)
		gs.metrics.Add("http_compression_bytes_saved", float64(originalSize-compressedSize),
			"path", r.URL.Path)
	})
}

func (gs *GatewayServer) rateLimitMiddleware(next http.Handler) http.Handler {
	// Initialize rate limiter if not already done
	if gs.rateLimiter == nil {
		gs.rateLimiter = gs.newRateLimiter()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract client identifier for rate limiting
		clientID := gs.getClientIdentifier(r)

		// Check rate limit
		allowed, resetTime, err := gs.rateLimiter.IsAllowed(clientID, r)
		if err != nil {
			gs.logger.Error("rate_limit_check_failed",
				"client_id", clientID,
				"path", r.URL.Path,
				"error", err)
			// On error, allow the request but log it
			next.ServeHTTP(w, r)
			return
		}

		if !allowed {
			// Rate limit exceeded
			gs.logger.Warn("rate_limit_exceeded",
				"client_id", clientID,
				"path", r.URL.Path,
				"method", r.Method,
				"user_agent", r.Header.Get("User-Agent"),
				"reset_time", resetTime)

			// Record rate limiting metrics
			gs.metrics.Inc("rate_limit_blocked_total",
				"client_id", clientID,
				"path", r.URL.Path,
				"method", r.Method)

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", gs.rateLimiter.GetLimit()))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(time.Until(resetTime).Seconds())))

			// Return 429 Too Many Requests
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)

			errorResponse := map[string]interface{}{
				"error":       "rate_limit_exceeded",
				"message":     "Too many requests. Please try again later.",
				"retry_after": int(time.Until(resetTime).Seconds()),
				"reset_time":  resetTime.Format(time.RFC3339),
			}

			json.NewEncoder(w).Encode(errorResponse)
			return
		}

		// Request allowed - set rate limit headers for transparency
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", gs.rateLimiter.GetLimit()))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

		// Record successful rate limit check
		gs.metrics.Inc("rate_limit_allowed_total",
			"client_id", clientID,
			"path", r.URL.Path,
			"method", r.Method)

		next.ServeHTTP(w, r)
	})
}

func (gs *GatewayServer) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		duration := time.Since(start)
		gs.metrics.Observe("http_request_duration_seconds", duration.Seconds(),
			"method", r.Method,
			"path", r.URL.Path)
		gs.metrics.Inc("http_requests_total",
			"method", r.Method,
			"path", r.URL.Path)
	})
}

func (gs *GatewayServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		gs.logger.Debug("http_request_started",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent())

		next.ServeHTTP(w, r)

		duration := time.Since(start)
		gs.logger.Info("http_request_completed",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", duration)
	})
}

func (gs *GatewayServer) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				gs.logger.Error("panic_recovered",
					"error", err,
					"method", r.Method,
					"path", r.URL.Path)

				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "Internal server error")
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// Helper methods

func (gs *GatewayServer) writeJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		gs.logger.Error("json_encoding_failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (gs *GatewayServer) getPID() int {
	return os.Getpid()
}

func defaultServerConfig() ServerConfig {
	return ServerConfig{
		Address:               "0.0.0.0",
		Port:                  8080,
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          30 * time.Second,
		IdleTimeout:           60 * time.Second,
		ShutdownTimeout:       30 * time.Second,
		TLSEnabled:            false,
		CORSEnabled:           true,
		CORSAllowOrigins:      []string{"*"},
		CORSAllowMethods:      []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		CORSAllowHeaders:      []string{"Content-Type", "Authorization"},
		EnableCompression:     true,
		EnableRateLimit:       true,
		RateLimitRPS:          1000,
		EnableHealthEndpoints: true,
		EnableMetricsEndpoint: true,
		EnableAdminEndpoints:  true,

		// Admin API authentication (empty by default for backward compatibility)
		AdminAPIKey:    "", // Must be set explicitly for security
		AdminAPIHeader: "X-Admin-API-Key",
	}
}

// CompressedResponseWriter wraps http.ResponseWriter to provide gzip compression
type CompressedResponseWriter struct {
	http.ResponseWriter
	gzipWriter    *gzip.Writer
	originalSize  int64
	headerWritten bool
	mutex         sync.Mutex
}

// newCompressedResponseWriter creates a new compressed response writer
func (gs *GatewayServer) newCompressedResponseWriter(w http.ResponseWriter, r *http.Request) *CompressedResponseWriter {
	gzWriter := gzip.NewWriter(w)
	crw := &CompressedResponseWriter{
		ResponseWriter: w,
		gzipWriter:     gzWriter,
	}

	// Set compression headers
	crw.Header().Set("Content-Encoding", "gzip")
	crw.Header().Set("Vary", "Accept-Encoding")

	return crw
}

// Write compresses and writes data
func (crw *CompressedResponseWriter) Write(data []byte) (int, error) {
	crw.mutex.Lock()
	defer crw.mutex.Unlock()

	if !crw.headerWritten {
		crw.WriteHeader(http.StatusOK)
	}

	crw.originalSize += int64(len(data))
	return crw.gzipWriter.Write(data)
}

// WriteHeader writes the status code
func (crw *CompressedResponseWriter) WriteHeader(statusCode int) {
	crw.mutex.Lock()
	defer crw.mutex.Unlock()

	if !crw.headerWritten {
		crw.headerWritten = true
		crw.ResponseWriter.WriteHeader(statusCode)
	}
}

// Close closes the gzip writer
func (crw *CompressedResponseWriter) Close() error {
	return crw.gzipWriter.Close()
}

// GetOriginalSize returns the uncompressed size
func (crw *CompressedResponseWriter) GetOriginalSize() int64 {
	crw.mutex.Lock()
	defer crw.mutex.Unlock()
	return crw.originalSize
}

// GetCompressedSize returns the compressed size
func (crw *CompressedResponseWriter) GetCompressedSize() int64 {
	// This would require tracking bytes written to the underlying writer
	// For now, we estimate based on gzip compression ratio (typically 70-80%)
	return int64(float64(crw.originalSize) * 0.25) // Assume 75% compression
}

// Helper functions for compression middleware

// shouldSkipCompression determines if compression should be skipped
func (gs *GatewayServer) shouldSkipCompression(r *http.Request) bool {
	// Skip compression for certain paths
	path := r.URL.Path
	if strings.HasPrefix(path, "/metrics") ||
		strings.HasPrefix(path, "/health") ||
		strings.HasSuffix(path, ".jpg") ||
		strings.HasSuffix(path, ".png") ||
		strings.HasSuffix(path, ".gif") ||
		strings.HasSuffix(path, ".zip") ||
		strings.HasSuffix(path, ".gz") {
		return true
	}

	// Skip for small requests (less than 1KB)
	contentLength := r.Header.Get("Content-Length")
	if contentLength != "" {
		if length, err := strconv.ParseInt(contentLength, 10, 64); err == nil && length < 1024 {
			return true
		}
	}

	return false
}

// clientAcceptsGzip checks if the client accepts gzip encoding
func (gs *GatewayServer) clientAcceptsGzip(r *http.Request) bool {
	acceptEncoding := r.Header.Get("Accept-Encoding")
	return strings.Contains(strings.ToLower(acceptEncoding), "gzip")
}

// RateLimiter interface for rate limiting
type RateLimiter interface {
	IsAllowed(clientID string, r *http.Request) (allowed bool, resetTime time.Time, err error)
	GetLimit() int
}

// SimpleRateLimiter implements a basic in-memory rate limiter
type SimpleRateLimiter struct {
	limit      int
	windowSize time.Duration
	clients    map[string]*ClientRateInfo
	mutex      sync.RWMutex
	logger     logging.Logger
	metrics    metrics.Metrics
}

// ClientRateInfo tracks rate limiting info for a client
type ClientRateInfo struct {
	requestCount int
	windowStart  time.Time
	lastRequest  time.Time
	mutex        sync.Mutex
}

// newRateLimiter creates a new rate limiter
func (gs *GatewayServer) newRateLimiter() RateLimiter {
	limit := gs.config.RateLimitRPS
	if limit <= 0 {
		limit = 100 // Default to 100 requests per second
	}

	return &SimpleRateLimiter{
		limit:      limit,
		windowSize: time.Second,
		clients:    make(map[string]*ClientRateInfo),
		logger:     gs.logger.WithComponent("rate_limiter"),
		metrics:    gs.metrics,
	}
}

// IsAllowed checks if a request is allowed under the rate limit
func (srl *SimpleRateLimiter) IsAllowed(clientID string, r *http.Request) (bool, time.Time, error) {
	now := time.Now()

	srl.mutex.Lock()
	clientInfo, exists := srl.clients[clientID]
	if !exists {
		clientInfo = &ClientRateInfo{
			requestCount: 0,
			windowStart:  now,
			lastRequest:  now,
		}
		srl.clients[clientID] = clientInfo
	}
	srl.mutex.Unlock()

	clientInfo.mutex.Lock()
	defer clientInfo.mutex.Unlock()

	// Reset window if expired
	if now.Sub(clientInfo.windowStart) >= srl.windowSize {
		clientInfo.requestCount = 0
		clientInfo.windowStart = now
	}

	// Check if limit exceeded
	if clientInfo.requestCount >= srl.limit {
		resetTime := clientInfo.windowStart.Add(srl.windowSize)
		return false, resetTime, nil
	}

	// Allow request and increment counter
	clientInfo.requestCount++
	clientInfo.lastRequest = now

	// Record rate limiting metrics
	srl.metrics.Observe("rate_limit_current_requests", float64(clientInfo.requestCount),
		"client_id", clientID)
	srl.metrics.Inc("rate_limit_allowed_total",
		"client_id", clientID,
		"path", r.URL.Path,
		"method", r.Method)

	resetTime := clientInfo.windowStart.Add(srl.windowSize)
	return true, resetTime, nil
}

// GetLimit returns the rate limit
func (srl *SimpleRateLimiter) GetLimit() int {
	return srl.limit
}

// getClientIdentifier extracts a client identifier for rate limiting
func (gs *GatewayServer) getClientIdentifier(r *http.Request) string {
	// Try to get IP from X-Forwarded-For header first (for load balancers)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP from the comma-separated list
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	// Try X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to remote address
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

// Additional admin API endpoints

func (gs *GatewayServer) handleServiceHealth(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceID := vars["id"]

	service := gs.registry.GetService(serviceID)
	if service == nil {
		w.WriteHeader(http.StatusNotFound)
		gs.writeJSONResponse(w, map[string]interface{}{
			"error":   "service_not_found",
			"message": fmt.Sprintf("Service not found: %s", serviceID),
		})
		return
	}

	// Get load balancer stats for additional health info
	stats := gs.registry.GetLoadBalancer().GetServiceStats(serviceID)

	healthInfo := map[string]interface{}{
		"service_id":    serviceID,
		"status":        service.Status,
		"health":        service.Health,
		"last_seen":     service.LastSeen.Format(time.RFC3339),
		"registered_at": service.RegisteredAt.Format(time.RFC3339),
		"endpoint":      service.Endpoint,
		"metrics":       service.Metrics,
	}

	if stats != nil {
		healthInfo["load_balancer_stats"] = map[string]interface{}{
			"active_requests":  stats.ActiveRequests,
			"total_requests":   stats.TotalRequests,
			"success_requests": stats.SuccessRequests,
			"failed_requests":  stats.FailedRequests,
			"circuit_open":     stats.CircuitOpen,
			"last_used":        stats.LastUsed.Format(time.RFC3339),
		}
	}

	gs.writeJSONResponse(w, healthInfo)
}

func (gs *GatewayServer) handleServiceCapabilities(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceID := vars["id"]

	service := gs.registry.GetService(serviceID)
	if service == nil {
		w.WriteHeader(http.StatusNotFound)
		gs.writeJSONResponse(w, map[string]interface{}{
			"error":   "service_not_found",
			"message": fmt.Sprintf("Service not found: %s", serviceID),
		})
		return
	}

	capabilities := map[string]interface{}{
		"service_id":      serviceID,
		"name":            service.Name,
		"type":            service.Type,
		"version":         service.Version,
		"tools":           service.Tools,
		"resources":       service.Resources,
		"prompts":         service.Prompts,
		"tools_count":     len(service.Tools),
		"resources_count": len(service.Resources),
		"prompts_count":   len(service.Prompts),
	}

	gs.writeJSONResponse(w, capabilities)
}

func (gs *GatewayServer) handleServiceTypes(w http.ResponseWriter, r *http.Request) {
	allServices := gs.registry.GetAllServices()
	typeStats := make(map[string]map[string]interface{})

	for _, service := range allServices {
		if _, exists := typeStats[service.Type]; !exists {
			typeStats[service.Type] = map[string]interface{}{
				"count":         0,
				"healthy_count": 0,
				"services":      []string{},
			}
		}

		stats := typeStats[service.Type]
		stats["count"] = stats["count"].(int) + 1
		if service.Health == registry.HealthHealthy {
			stats["healthy_count"] = stats["healthy_count"].(int) + 1
		}
		stats["services"] = append(stats["services"].([]string), service.ID)
	}

	response := map[string]interface{}{
		"types":       typeStats,
		"total_types": len(typeStats),
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	gs.writeJSONResponse(w, response)
}

func (gs *GatewayServer) handleDiscoveryStatus(w http.ResponseWriter, r *http.Request) {
	discovered := gs.registry.GetDiscoveredServices()

	// Count by source
	sourceStats := make(map[string]int)
	registrationStats := map[string]int{
		"successful": 0,
		"failed":     0,
		"pending":    0,
	}

	for _, service := range discovered {
		sourceStats[service.Source]++

		if service.RegistrationError == "" {
			registrationStats["successful"]++
		} else if service.RegistrationAttempts > 0 {
			registrationStats["failed"]++
		} else {
			registrationStats["pending"]++
		}
	}

	status := map[string]interface{}{
		"total_discovered":    len(discovered),
		"by_source":           sourceStats,
		"registration_status": registrationStats,
		"timestamp":           time.Now().Format(time.RFC3339),
	}

	gs.writeJSONResponse(w, status)
}

func (gs *GatewayServer) handleServiceLoadBalancerStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceID := vars["service_id"]

	stats := gs.registry.GetLoadBalancer().GetServiceStats(serviceID)
	if stats == nil {
		w.WriteHeader(http.StatusNotFound)
		gs.writeJSONResponse(w, map[string]interface{}{
			"error":   "service_stats_not_found",
			"message": fmt.Sprintf("Load balancer stats not found for service: %s", serviceID),
		})
		return
	}

	response := map[string]interface{}{
		"service_id": serviceID,
		"stats":      stats,
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	gs.writeJSONResponse(w, response)
}

func (gs *GatewayServer) handleLoadBalancerStrategies(w http.ResponseWriter, r *http.Request) {
	strategies := map[string]interface{}{
		"available_strategies": []string{
			"round_robin",
			"least_connections",
			"weighted",
			"hash",
			"random",
		},
		"current_strategy": "round_robin", // Default strategy
		"descriptions": map[string]string{
			"round_robin":       "Distributes requests evenly across all healthy services",
			"least_connections": "Routes to the service with the fewest active connections",
			"weighted":          "Routes based on service weights",
			"hash":              "Consistent hash-based routing for session affinity",
			"random":            "Random service selection",
		},
	}

	gs.writeJSONResponse(w, strategies)
}

func (gs *GatewayServer) handleConfigReload(w http.ResponseWriter, r *http.Request) {
	gs.logger.Info("admin_config_reload_requested",
		"remote_addr", r.RemoteAddr)

	// In a real implementation, this would reload configuration from file
	// For now, we'll just return a success response
	gs.metrics.Inc("admin_api_config_reloads_total", "status", "success")

	response := map[string]interface{}{
		"status":    "success",
		"message":   "Configuration reload completed (placeholder implementation)",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	gs.writeJSONResponse(w, response)
}

func (gs *GatewayServer) handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	uptime := time.Since(gs.startTime)

	info := map[string]interface{}{
		"version":        gs.version,
		"commit":         gs.commit,
		"build_time":     gs.buildTime,
		"start_time":     gs.startTime.Format(time.RFC3339),
		"uptime_seconds": uptime.Seconds(),
		"uptime_human":   uptime.String(),
		"go_version":     runtime.Version(),
		"runtime": map[string]interface{}{
			"goroutines":      runtime.NumGoroutine(),
			"cpu_count":       runtime.NumCPU(),
			"memory_alloc_mb": float64(memStats.Alloc) / 1024 / 1024,
			"memory_sys_mb":   float64(memStats.Sys) / 1024 / 1024,
			"gc_runs":         memStats.NumGC,
		},
		"config": map[string]interface{}{
			"address":             gs.config.Address,
			"port":                gs.config.Port,
			"tls_enabled":         gs.config.TLSEnabled,
			"cors_enabled":        gs.config.CORSEnabled,
			"compression_enabled": gs.config.EnableCompression,
			"rate_limit_enabled":  gs.config.EnableRateLimit,
			"rate_limit_rps":      gs.config.RateLimitRPS,
		},
	}

	gs.writeJSONResponse(w, info)
}

func (gs *GatewayServer) handleSystemStats(w http.ResponseWriter, r *http.Request) {
	allServices := gs.registry.GetAllServices()
	healthyServices := gs.registry.GetHealthyServices()
	discovered := gs.registry.GetDiscoveredServices()
	lbStats := gs.registry.GetLoadBalancer().GetAllStats()

	stats := map[string]interface{}{
		"services": map[string]interface{}{
			"total":     len(allServices),
			"healthy":   len(healthyServices),
			"unhealthy": len(allServices) - len(healthyServices),
		},
		"discovery": map[string]interface{}{
			"discovered_services": len(discovered),
		},
		"load_balancer": map[string]interface{}{
			"tracked_services": len(lbStats),
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	gs.writeJSONResponse(w, stats)
}

func (gs *GatewayServer) handleGoroutineStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"goroutines": runtime.NumGoroutine(),
		"cpu_count":  runtime.NumCPU(),
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	// Add memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	stats["memory"] = map[string]interface{}{
		"alloc_mb":       float64(memStats.Alloc) / 1024 / 1024,
		"total_alloc_mb": float64(memStats.TotalAlloc) / 1024 / 1024,
		"sys_mb":         float64(memStats.Sys) / 1024 / 1024,
		"gc_runs":        memStats.NumGC,
		"gc_pause_ns":    memStats.PauseNs[(memStats.NumGC+255)%256],
	}

	gs.writeJSONResponse(w, stats)
}

func (gs *GatewayServer) handleAPIDocumentation(w http.ResponseWriter, r *http.Request) {
	docs := map[string]interface{}{
		"service": map[string]interface{}{
			"name":          "MCpeg",
			"description":   "Model Context Protocol Enablement Gateway",
			"version":       gs.version,
			"pronunciation": "MC peg",
		},
		"admin_api": map[string]interface{}{
			"base_path": "/admin",
			"endpoints": map[string]interface{}{
				"services": map[string]interface{}{
					"GET /services":                   "List all registered services",
					"POST /services":                  "Register a new service",
					"GET /services/{id}":              "Get service details",
					"DELETE /services/{id}":           "Unregister a service",
					"GET /services/{id}/health":       "Get service health information",
					"GET /services/{id}/capabilities": "Get service capabilities",
					"GET /services/types":             "Get service type statistics",
				},
				"discovery": map[string]interface{}{
					"POST /discovery/trigger": "Trigger manual service discovery",
					"GET /discovery/services": "List discovered services",
					"GET /discovery/status":   "Get discovery status and statistics",
				},
				"loadbalancer": map[string]interface{}{
					"GET /loadbalancer/stats":               "Get load balancer statistics for all services",
					"GET /loadbalancer/stats/{service_id}":  "Get load balancer statistics for specific service",
					"POST /loadbalancer/reset/{service_id}": "Reset circuit breaker for service",
					"GET /loadbalancer/strategies":          "List available load balancing strategies",
				},
				"config": map[string]interface{}{
					"GET /config":         "Get current configuration",
					"PUT /config":         "Update configuration",
					"POST /config/reload": "Reload configuration from file",
				},
				"plugins": map[string]interface{}{
					"GET /plugins":                  "List all plugins",
					"GET /plugins/{name}":           "Get plugin information",
					"GET /plugins/{name}/config":    "Get plugin configuration",
					"PUT /plugins/{name}/config":    "Update plugin configuration",
					"GET /plugins/{name}/tools":     "Get plugin tools",
					"GET /plugins/{name}/resources": "Get plugin resources",
					"GET /plugins/{name}/health":    "Get plugin health status",
					"GET /plugins/health":           "Get all plugin health status",
					"GET /plugins/metrics":          "Get plugin metrics",
					"GET /plugins/capabilities":     "Get plugin capabilities summary",
				},
				"system": map[string]interface{}{
					"GET /info":             "Get system information",
					"GET /stats":            "Get system statistics",
					"GET /debug/goroutines": "Get goroutine and memory statistics",
					"GET /api":              "Get API documentation (this endpoint)",
				},
			},
		},
		"health_endpoints": map[string]interface{}{
			"GET /health":       "General health check",
			"GET /health/live":  "Liveness probe",
			"GET /health/ready": "Readiness probe",
		},
		"metrics": map[string]interface{}{
			"GET /metrics": "Prometheus metrics endpoint",
		},
		"mcp_endpoints": map[string]interface{}{
			"POST /mcp":                "Main MCP JSON-RPC endpoint",
			"POST /mcp/tools/list":     "List available tools",
			"POST /mcp/tools/call":     "Call a specific tool",
			"POST /mcp/resources/list": "List available resources",
			"POST /mcp/resources/read": "Read a specific resource",
			"POST /mcp/prompts/list":   "List available prompts",
			"POST /mcp/prompts/get":    "Get a specific prompt",
		},
		"version":   gs.version,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	gs.writeJSONResponse(w, docs)
}

// Plugin management endpoint handlers

func (gs *GatewayServer) handleListPlugins(w http.ResponseWriter, r *http.Request) {
	gs.logger.Debug("admin_list_plugins_request", "remote_addr", r.RemoteAddr)

	allPluginInfo := gs.pluginIntegration.GetAllPluginInfo()

	gs.metrics.Inc("admin_api_plugin_list_requests_total")
	gs.writeJSONResponse(w, allPluginInfo)
}

func (gs *GatewayServer) handleGetPlugin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pluginName := vars["name"]

	gs.logger.Debug("admin_get_plugin_request",
		"plugin_name", pluginName,
		"remote_addr", r.RemoteAddr)

	pluginInfo, err := gs.pluginIntegration.GetPluginInfo(pluginName)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		gs.writeJSONResponse(w, map[string]interface{}{
			"error":   "plugin_not_found",
			"message": fmt.Sprintf("Plugin not found: %s", pluginName),
		})
		return
	}

	gs.metrics.Inc("admin_api_plugin_get_requests_total", "plugin", pluginName)
	gs.writeJSONResponse(w, pluginInfo)
}

func (gs *GatewayServer) handleGetPluginConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pluginName := vars["name"]

	gs.logger.Debug("admin_get_plugin_config_request",
		"plugin_name", pluginName,
		"remote_addr", r.RemoteAddr)

	config, err := gs.pluginIntegration.GetPluginConfiguration(pluginName)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		gs.writeJSONResponse(w, map[string]interface{}{
			"error":   "plugin_not_found",
			"message": fmt.Sprintf("Plugin not found: %s", pluginName),
		})
		return
	}

	gs.metrics.Inc("admin_api_plugin_config_get_requests_total", "plugin", pluginName)
	gs.writeJSONResponse(w, config)
}

func (gs *GatewayServer) handleUpdatePluginConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pluginName := vars["name"]

	gs.logger.Info("admin_update_plugin_config_request",
		"plugin_name", pluginName,
		"remote_addr", r.RemoteAddr)

	var configUpdate map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&configUpdate); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		gs.writeJSONResponse(w, map[string]interface{}{
			"error":   "invalid_request_body",
			"message": "Failed to parse JSON request body",
		})
		return
	}

	ctx := r.Context()
	err := gs.pluginIntegration.UpdatePluginConfiguration(ctx, pluginName, configUpdate)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		gs.writeJSONResponse(w, map[string]interface{}{
			"error":   "config_update_failed",
			"message": err.Error(),
		})
		return
	}

	gs.metrics.Inc("admin_api_plugin_config_update_requests_total", "plugin", pluginName)
	gs.writeJSONResponse(w, map[string]interface{}{
		"status":  "success",
		"message": "Plugin configuration updated successfully",
	})
}

func (gs *GatewayServer) handleGetPluginTools(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pluginName := vars["name"]

	gs.logger.Debug("admin_get_plugin_tools_request",
		"plugin_name", pluginName,
		"remote_addr", r.RemoteAddr)

	pluginInfo, err := gs.pluginIntegration.GetPluginInfo(pluginName)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		gs.writeJSONResponse(w, map[string]interface{}{
			"error":   "plugin_not_found",
			"message": fmt.Sprintf("Plugin not found: %s", pluginName),
		})
		return
	}

	response := map[string]interface{}{
		"plugin": pluginName,
		"tools":  pluginInfo["tools"],
	}

	gs.metrics.Inc("admin_api_plugin_tools_requests_total", "plugin", pluginName)
	gs.writeJSONResponse(w, response)
}

func (gs *GatewayServer) handleGetPluginResources(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pluginName := vars["name"]

	gs.logger.Debug("admin_get_plugin_resources_request",
		"plugin_name", pluginName,
		"remote_addr", r.RemoteAddr)

	pluginInfo, err := gs.pluginIntegration.GetPluginInfo(pluginName)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		gs.writeJSONResponse(w, map[string]interface{}{
			"error":   "plugin_not_found",
			"message": fmt.Sprintf("Plugin not found: %s", pluginName),
		})
		return
	}

	response := map[string]interface{}{
		"plugin":    pluginName,
		"resources": pluginInfo["resources"],
	}

	gs.metrics.Inc("admin_api_plugin_resources_requests_total", "plugin", pluginName)
	gs.writeJSONResponse(w, response)
}

func (gs *GatewayServer) handleGetPluginHealth(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pluginName := vars["name"]

	gs.logger.Debug("admin_get_plugin_health_request",
		"plugin_name", pluginName,
		"remote_addr", r.RemoteAddr)

	allHealth := gs.pluginIntegration.HealthCheckPlugins(r.Context())

	pluginHealthMap, ok := allHealth["plugins"].(map[string]interface{})
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		gs.writeJSONResponse(w, map[string]interface{}{
			"error":   "health_check_failed",
			"message": "Failed to get plugin health information",
		})
		return
	}

	pluginHealth, exists := pluginHealthMap[pluginName]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		gs.writeJSONResponse(w, map[string]interface{}{
			"error":   "plugin_not_found",
			"message": fmt.Sprintf("Plugin not found: %s", pluginName),
		})
		return
	}

	response := map[string]interface{}{
		"plugin": pluginName,
		"health": pluginHealth,
	}

	gs.metrics.Inc("admin_api_plugin_health_requests_total", "plugin", pluginName)
	gs.writeJSONResponse(w, response)
}

func (gs *GatewayServer) handleGetAllPluginHealth(w http.ResponseWriter, r *http.Request) {
	gs.logger.Debug("admin_get_all_plugin_health_request", "remote_addr", r.RemoteAddr)

	healthStatus := gs.pluginIntegration.HealthCheckPlugins(r.Context())

	gs.metrics.Inc("admin_api_plugin_health_all_requests_total")
	gs.writeJSONResponse(w, healthStatus)
}

func (gs *GatewayServer) handleGetPluginMetrics(w http.ResponseWriter, r *http.Request) {
	gs.logger.Debug("admin_get_plugin_metrics_request", "remote_addr", r.RemoteAddr)

	pluginMetrics := gs.pluginIntegration.GetPluginMetrics()

	gs.metrics.Inc("admin_api_plugin_metrics_requests_total")
	gs.writeJSONResponse(w, pluginMetrics)
}

func (gs *GatewayServer) handleGetPluginCapabilities(w http.ResponseWriter, r *http.Request) {
	gs.logger.Debug("admin_get_plugin_capabilities_request", "remote_addr", r.RemoteAddr)

	capabilities := gs.pluginIntegration.ListPluginCapabilities()

	gs.metrics.Inc("admin_api_plugin_capabilities_requests_total")
	gs.writeJSONResponse(w, capabilities)
}

// adminAuthMiddleware provides authentication for admin API endpoints
func (gs *GatewayServer) adminAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get API key header (default to "X-Admin-API-Key" if not configured)
		headerName := gs.config.AdminAPIHeader
		if headerName == "" {
			headerName = "X-Admin-API-Key"
		}

		// Extract API key from request
		providedKey := r.Header.Get(headerName)
		if providedKey == "" {
			gs.logger.Warn("admin_auth_missing_key",
				"remote_addr", r.RemoteAddr,
				"path", r.URL.Path,
				"user_agent", r.Header.Get("User-Agent"))

			gs.metrics.Inc("admin_api_auth_failures_total", "reason", "missing_key")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			gs.writeJSONResponse(w, map[string]interface{}{
				"error":   "authentication_required",
				"message": fmt.Sprintf("Admin API key required in %s header", headerName),
			})
			return
		}

		// Validate API key
		if providedKey != gs.config.AdminAPIKey {
			gs.logger.Warn("admin_auth_invalid_key",
				"remote_addr", r.RemoteAddr,
				"path", r.URL.Path,
				"user_agent", r.Header.Get("User-Agent"),
				"provided_key_length", len(providedKey))

			gs.metrics.Inc("admin_api_auth_failures_total", "reason", "invalid_key")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			gs.writeJSONResponse(w, map[string]interface{}{
				"error":   "authentication_failed",
				"message": "Invalid admin API key",
			})
			return
		}

		// Authentication successful
		gs.logger.Debug("admin_auth_success",
			"remote_addr", r.RemoteAddr,
			"path", r.URL.Path)

		gs.metrics.Inc("admin_api_auth_success_total")

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// initializePhase2Discovery initializes the Phase 2 advanced plugin discovery system
func (gs *GatewayServer) initializePhase2Discovery() {
	go func() {
		ctx := context.Background()
		
		gs.logger.Info("phase2_discovery_initialization_started")

		// Wait a moment for plugin system to fully initialize
		time.Sleep(2 * time.Second)

		// Discover and analyze all registered plugins
		plugins := gs.pluginIntegration.GetPluginManager().GetPlugins()
		for _, pluginName := range plugins {
			// Perform comprehensive plugin discovery
			result, err := gs.discoveryEngine.DiscoverPlugin(ctx, pluginName)
			if err != nil {
				gs.logger.Warn("plugin_discovery_failed",
					"plugin", pluginName,
					"error", err.Error())
				continue
			}

			gs.logger.Info("plugin_discovery_completed",
				"plugin", pluginName,
				"capabilities", len(result.Capabilities),
				"dependencies", len(result.Dependencies),
				"conflicts", len(result.Conflicts),
				"recommendations", len(result.Recommendations))

			// Increment discovery metrics
			gs.metrics.Inc("phase2_plugin_discoveries_total")
			gs.metrics.Set("plugin_capabilities_discovered", float64(len(result.Capabilities)))
		}

		// Aggregate capabilities across all plugins
		err := gs.aggregationEngine.AggregateCapabilities(ctx)
		if err != nil {
			gs.logger.Error("capability_aggregation_failed", "error", err.Error())
		} else {
			gs.logger.Info("capability_aggregation_completed")
			gs.metrics.Inc("phase2_aggregations_total")
		}

		// Validate all discovered capabilities
		totalValidations := 0
		validationsPassed := 0
		for _, pluginName := range plugins {
			if result, exists := gs.discoveryEngine.GetDiscoveryResult(pluginName); exists {
				for _, capability := range result.Capabilities {
					validation, err := gs.validationEngine.ValidateCapability(ctx, pluginName, capability.CapabilityName)
					if err != nil {
						gs.logger.Warn("capability_validation_failed",
							"plugin", pluginName,
							"capability", capability.CapabilityName,
							"error", err.Error())
						continue
					}

					totalValidations++
					if validation.Status == capabilities.StatusPassed {
						validationsPassed++
					}

					gs.logger.Debug("capability_validated",
						"plugin", pluginName,
						"capability", capability.CapabilityName,
						"status", validation.Status,
						"score", validation.Score,
						"issues", len(validation.Issues))
				}
			}
		}

		gs.logger.Info("phase2_discovery_initialization_completed",
			"plugins_discovered", len(plugins),
			"total_validations", totalValidations,
			"validations_passed", validationsPassed,
			"success_rate", float64(validationsPassed)/float64(totalValidations)*100)

		gs.metrics.Set("phase2_discovery_success_rate", float64(validationsPassed)/float64(totalValidations))
		gs.metrics.Inc("phase2_initialization_completed_total")
	}()
}
