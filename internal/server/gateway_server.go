package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/osakka/mcpeg/internal/registry"
	"github.com/osakka/mcpeg/internal/router"
	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
	"github.com/osakka/mcpeg/pkg/validation"
	"github.com/osakka/mcpeg/pkg/health"
)

// GatewayServer represents the main MCPEG gateway server
type GatewayServer struct {
	config       ServerConfig
	httpServer   *http.Server
	registry     *registry.ServiceRegistry
	mcpRouter    *router.MCPRouter
	logger       logging.Logger
	metrics      metrics.Metrics
	validator    *validation.Validator
	healthMgr    *health.HealthManager
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
	EnableHealthEndpoints  bool `yaml:"enable_health_endpoints"`
	EnableMetricsEndpoint  bool `yaml:"enable_metrics_endpoint"`
	EnableAdminEndpoints   bool `yaml:"enable_admin_endpoints"`
}

// NewGatewayServer creates a new gateway server
func NewGatewayServer(
	config ServerConfig,
	logger logging.Logger,
	metrics metrics.Metrics,
	validator *validation.Validator,
	healthMgr *health.HealthManager,
) *GatewayServer {
	// Create service registry
	serviceRegistry := registry.NewServiceRegistry(logger, metrics, validator, healthMgr)
	
	// Create MCP router
	mcpRouter := router.NewMCPRouter(serviceRegistry, logger, metrics, validator)
	
	server := &GatewayServer{
		config:    config,
		registry:  serviceRegistry,
		mcpRouter: mcpRouter,
		logger:    logger.WithComponent("gateway_server"),
		metrics:   metrics,
		validator: validator,
		healthMgr: healthMgr,
	}
	
	// Setup HTTP server
	server.setupHTTPServer()
	
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
	
	// Service discovery
	router.HandleFunc("/discovery/trigger", gs.handleTriggerDiscovery).Methods("POST")
	router.HandleFunc("/discovery/services", gs.handleDiscoveredServices).Methods("GET")
	
	// Load balancer management
	router.HandleFunc("/loadbalancer/stats", gs.handleLoadBalancerStats).Methods("GET")
	router.HandleFunc("/loadbalancer/reset/{service_id}", gs.handleResetCircuitBreaker).Methods("POST")
	
	// Configuration
	router.HandleFunc("/config", gs.handleGetConfig).Methods("GET")
	router.HandleFunc("/config", gs.handleUpdateConfig).Methods("PUT")
}

// Start starts the gateway server
func (gs *GatewayServer) Start(ctx context.Context) error {
	gs.logger.Info("gateway_server_starting",
		"address", gs.httpServer.Addr,
		"tls_enabled", gs.config.TLSEnabled)
	
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
	// TODO: Implement Prometheus metrics endpoint
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "# MCPEG Gateway Metrics\n")
	fmt.Fprintf(w, "# TODO: Implement Prometheus metrics\n")
}

// Admin endpoint handlers (simplified implementations)

func (gs *GatewayServer) handleListServices(w http.ResponseWriter, r *http.Request) {
	services := gs.registry.GetAllServices()
	gs.writeJSONResponse(w, services)
}

func (gs *GatewayServer) handleRegisterService(w http.ResponseWriter, r *http.Request) {
	// TODO: Parse registration request and register service
	w.WriteHeader(http.StatusNotImplemented)
	fmt.Fprintf(w, "Service registration via admin API not yet implemented")
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
	// TODO: Trigger manual service discovery
	w.WriteHeader(http.StatusNotImplemented)
	fmt.Fprintf(w, "Manual discovery trigger not yet implemented")
}

func (gs *GatewayServer) handleDiscoveredServices(w http.ResponseWriter, r *http.Request) {
	// TODO: Return discovered services
	w.WriteHeader(http.StatusNotImplemented)
	fmt.Fprintf(w, "Discovered services endpoint not yet implemented")
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
	// TODO: Implement configuration updates
	w.WriteHeader(http.StatusNotImplemented)
	fmt.Fprintf(w, "Configuration updates not yet implemented")
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
	// TODO: Implement gzip compression
	return next
}

func (gs *GatewayServer) rateLimitMiddleware(next http.Handler) http.Handler {
	// TODO: Implement rate limiting
	return next
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
	// TODO: Implement proper JSON encoding with error handling
	fmt.Fprintf(w, "JSON response placeholder")
}

func (gs *GatewayServer) getPID() int {
	// TODO: Get actual process ID
	return 0
}

func defaultServerConfig() ServerConfig {
	return ServerConfig{
		Address:         "0.0.0.0",
		Port:            8080,
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownTimeout: 30 * time.Second,
		TLSEnabled:      false,
		CORSEnabled:     true,
		CORSAllowOrigins: []string{"*"},
		CORSAllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		CORSAllowHeaders: []string{"Content-Type", "Authorization"},
		EnableCompression:     true,
		EnableRateLimit:       true,
		RateLimitRPS:         1000,
		EnableHealthEndpoints: true,
		EnableMetricsEndpoint: true,
		EnableAdminEndpoints:  true,
	}
}