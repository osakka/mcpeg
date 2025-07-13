// Package registry provides comprehensive service discovery and management for MCPEG gateway operations.
//
// This package implements the core service registry that manages all registered MCP service adapters,
// providing dynamic service discovery, health monitoring, and intelligent request routing:
//
//   - Service registration and lifecycle management
//   - Dynamic service discovery with capability analysis
//   - Health monitoring with circuit breaker pattern implementation
//   - Load balancing with multiple algorithm support
//   - Service metadata and capability management
//   - Performance metrics collection and monitoring
//   - Security and access control integration
//
// The service registry supports enterprise-grade features:
//   - Multi-protocol service support (HTTP, gRPC, WebSocket)
//   - Circuit breaker pattern for fault tolerance
//   - Service versioning and rolling updates
//   - Comprehensive metrics and observability
//   - Tag-based service filtering and selection
//   - Configuration management per service
//   - Background health monitoring with automatic recovery
//
// Service registration workflow:
//   1. Service discovery identifies available services
//   2. Capability analysis determines service features
//   3. Health checks validate service readiness
//   4. Load balancer configuration for traffic distribution
//   5. Circuit breaker initialization for fault tolerance
//   6. Continuous monitoring and status updates
//
// Example service registration:
//
//	service := &RegisteredService{
//	    ID:          "service-001",
//	    Name:        "file-processor",
//	    Type:        "mcp-adapter",
//	    Version:     "1.0.0",
//	    Endpoint:    "http://localhost:8080",
//	    Protocol:    "http",
//	    Tools:       []ToolDefinition{...},
//	    Resources:   []ResourceDefinition{...},
//	    Status:      StatusActive,
//	}
//	
//	err := registry.RegisterService(ctx, service)
//	if err != nil {
//	    log.Printf("Service registration failed: %v", err)
//	}
//
// Service discovery and routing:
//
//	services := registry.DiscoverServices(ctx, ServiceFilter{
//	    Type: "mcp-adapter",
//	    Tags: []string{"file-processing"},
//	    HealthStatus: HealthHealthy,
//	})
//	
//	service := registry.SelectService(services, LoadBalanceRoundRobin)
package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/osakka/mcpeg/pkg/errors"
	"github.com/osakka/mcpeg/pkg/health"
	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
	"github.com/osakka/mcpeg/pkg/validation"
)

// ServiceRegistry manages all registered MCP service adapters
type ServiceRegistry struct {
	services     map[string]*RegisteredService
	byType       map[string][]*RegisteredService
	capabilities map[string]*ServiceCapabilities
	mutex        sync.RWMutex

	logger    logging.Logger
	metrics   metrics.Metrics
	validator *validation.Validator
	health    *health.HealthManager

	config       RegistryConfig
	discovery    *ServiceDiscovery
	loadBalancer *LoadBalancer

	// Circuit breaker configuration
	maxFailures int

	// Background monitoring
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// RegisteredService represents a service registered with the gateway
type RegisteredService struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`

	// Network configuration
	Endpoint string `json:"endpoint"`
	Protocol string `json:"protocol"`

	// Service capabilities
	Tools     []ToolDefinition     `json:"tools"`
	Resources []ResourceDefinition `json:"resources"`
	Prompts   []PromptDefinition   `json:"prompts"`

	// Operational status
	Status       ServiceStatus `json:"status"`
	Health       HealthStatus  `json:"health"`
	LastSeen     time.Time     `json:"last_seen"`
	RegisteredAt time.Time     `json:"registered_at"`

	// Configuration and metadata
	Configuration map[string]interface{} `json:"configuration,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Tags          []string               `json:"tags,omitempty"`

	// Performance metrics
	Metrics ServiceMetrics `json:"metrics"`

	// Security and access control
	Security ServiceSecurity `json:"security"`

	// Runtime state
	mutex      sync.RWMutex
	client     *http.Client
	lastHealth time.Time

	// Circuit breaker state
	FailureCount int `json:"failure_count"`
}

// ServiceStatus represents the operational status of a service
type ServiceStatus string

const (
	StatusRegistering ServiceStatus = "registering"
	StatusActive      ServiceStatus = "active"
	StatusInactive    ServiceStatus = "inactive"
	StatusError       ServiceStatus = "error"
	StatusDraining    ServiceStatus = "draining"
	StatusMaintenance ServiceStatus = "maintenance"
	StatusUnavailable ServiceStatus = "unavailable"
)

// HealthStatus represents the health status of a service
type HealthStatus string

const (
	HealthHealthy   HealthStatus = "healthy"
	HealthDegraded  HealthStatus = "degraded"
	HealthUnhealthy HealthStatus = "unhealthy"
	HealthUnknown   HealthStatus = "unknown"
)

// ServiceCapabilities defines what a service can do
type ServiceCapabilities struct {
	Tools     ToolCapabilities     `json:"tools"`
	Resources ResourceCapabilities `json:"resources"`
	Prompts   PromptCapabilities   `json:"prompts"`
	Features  []string             `json:"features"`
}

// ToolCapabilities defines tool-related capabilities
type ToolCapabilities struct {
	Count             int      `json:"count"`
	Categories        []string `json:"categories"`
	SupportsAsync     bool     `json:"supports_async"`
	MaxConcurrency    int      `json:"max_concurrency"`
	SupportsStreaming bool     `json:"supports_streaming"`
}

// ResourceCapabilities defines resource-related capabilities
type ResourceCapabilities struct {
	Count                int      `json:"count"`
	Types                []string `json:"types"`
	SupportsSubscription bool     `json:"supports_subscription"`
	SupportsStreaming    bool     `json:"supports_streaming"`
	SupportsWatch        bool     `json:"supports_watch"`
}

// PromptCapabilities defines prompt-related capabilities
type PromptCapabilities struct {
	Count              int      `json:"count"`
	Categories         []string `json:"categories"`
	SupportsTemplating bool     `json:"supports_templating"`
	SupportsArguments  bool     `json:"supports_arguments"`
}

// ToolDefinition defines a tool provided by a service
type ToolDefinition struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Category     string                 `json:"category,omitempty"`
	InputSchema  map[string]interface{} `json:"input_schema"`
	OutputSchema map[string]interface{} `json:"output_schema,omitempty"`
	Examples     []ToolExample          `json:"examples,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ResourceDefinition defines a resource provided by a service
type ResourceDefinition struct {
	URI         string                 `json:"uri"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Type        string                 `json:"type,omitempty"`
	MimeType    string                 `json:"mime_type,omitempty"`
	Size        int64                  `json:"size,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// PromptDefinition defines a prompt template provided by a service
type PromptDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category,omitempty"`
	Arguments   []PromptArgument       `json:"arguments,omitempty"`
	Template    string                 `json:"template,omitempty"`
	Examples    []PromptExample        `json:"examples,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ToolExample provides an example of tool usage
type ToolExample struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Input       interface{} `json:"input"`
	Output      interface{} `json:"output"`
}

// PromptArgument defines a prompt template argument
type PromptArgument struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
}

// PromptExample provides an example of prompt usage
type PromptExample struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Arguments   map[string]interface{} `json:"arguments"`
	Output      string                 `json:"output"`
}

// ServiceMetrics tracks service performance
type ServiceMetrics struct {
	RequestCount    uint64        `json:"request_count"`
	ErrorCount      uint64        `json:"error_count"`
	AverageLatency  time.Duration `json:"average_latency"`
	LastRequestTime time.Time     `json:"last_request_time"`
	Uptime          time.Duration `json:"uptime"`
	ErrorRate       float64       `json:"error_rate"`
}

// ServiceSecurity defines security settings for a service
type ServiceSecurity struct {
	AuthRequired   bool       `json:"auth_required"`
	AllowedClients []string   `json:"allowed_clients,omitempty"`
	RequiredScopes []string   `json:"required_scopes,omitempty"`
	RateLimit      *RateLimit `json:"rate_limit,omitempty"`
}

// RateLimit defines rate limiting for a service
type RateLimit struct {
	RequestsPerMinute int           `json:"requests_per_minute"`
	Burst             int           `json:"burst"`
	WindowSize        time.Duration `json:"window_size"`
}

// RegistryConfig configures the service registry
type RegistryConfig struct {
	// Discovery settings
	DiscoveryEnabled    bool          `yaml:"discovery_enabled"`
	DiscoveryInterval   time.Duration `yaml:"discovery_interval"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`

	// Service validation
	ValidateOnRegister  bool          `yaml:"validate_on_register"`
	RequireHealthCheck  bool          `yaml:"require_health_check"`
	MaxRegistrationTime time.Duration `yaml:"max_registration_time"`

	// Load balancing
	LoadBalancingEnabled  bool   `yaml:"load_balancing_enabled"`
	LoadBalancingStrategy string `yaml:"load_balancing_strategy"`

	// Security
	RequireAuthentication bool `yaml:"require_authentication"`
	AllowSelfRegistration bool `yaml:"allow_self_registration"`

	// Cleanup and maintenance
	InactiveServiceTimeout time.Duration `yaml:"inactive_service_timeout"`
	CleanupInterval        time.Duration `yaml:"cleanup_interval"`
}

// ServiceRegistrationRequest represents a service registration request
type ServiceRegistrationRequest struct {
	Name          string                 `json:"name" validate:"required"`
	Type          string                 `json:"type" validate:"required"`
	Version       string                 `json:"version" validate:"required"`
	Description   string                 `json:"description,omitempty"`
	Endpoint      string                 `json:"endpoint" validate:"required,url"`
	Protocol      string                 `json:"protocol" validate:"required"`
	Tools         []ToolDefinition       `json:"tools,omitempty"`
	Resources     []ResourceDefinition   `json:"resources,omitempty"`
	Prompts       []PromptDefinition     `json:"prompts,omitempty"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Tags          []string               `json:"tags,omitempty"`
	Security      ServiceSecurity        `json:"security,omitempty"`
}

// ServiceRegistrationResponse represents the response to a registration request
type ServiceRegistrationResponse struct {
	ServiceID   string        `json:"service_id"`
	Status      ServiceStatus `json:"status"`
	Message     string        `json:"message,omitempty"`
	Timestamp   time.Time     `json:"timestamp"`
	GatewayInfo GatewayInfo   `json:"gateway_info"`
}

// GatewayInfo provides information about the gateway
type GatewayInfo struct {
	Version             string   `json:"version"`
	SupportedProtocols  []string `json:"supported_protocols"`
	Features            []string `json:"features"`
	HealthCheckEndpoint string   `json:"health_check_endpoint"`
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry(logger logging.Logger, metrics metrics.Metrics, validator *validation.Validator, healthManager *health.HealthManager) *ServiceRegistry {
	ctx, cancel := context.WithCancel(context.Background())

	registry := &ServiceRegistry{
		services:     make(map[string]*RegisteredService),
		byType:       make(map[string][]*RegisteredService),
		capabilities: make(map[string]*ServiceCapabilities),
		logger:       logger.WithComponent("service_registry"),
		metrics:      metrics,
		validator:    validator,
		health:       healthManager,
		config:       defaultRegistryConfig(),
		maxFailures:  5, // Default max failures before marking service unavailable
		ctx:          ctx,
		cancel:       cancel,
	}

	// Initialize discovery and load balancing
	registry.discovery = NewServiceDiscovery(registry, logger, metrics)
	registry.loadBalancer = NewLoadBalancer(registry, logger, metrics)

	// Start background processes
	registry.startBackgroundProcesses()

	return registry
}

// GetLoadBalancer returns the load balancer instance
func (sr *ServiceRegistry) GetLoadBalancer() *LoadBalancer {
	return sr.loadBalancer
}

// RegisterService registers a new service with the gateway
func (sr *ServiceRegistry) RegisterService(ctx context.Context, req ServiceRegistrationRequest) (*ServiceRegistrationResponse, error) {
	start := time.Now()

	sr.logger.Info("service_registration_started",
		"name", req.Name,
		"type", req.Type,
		"version", req.Version,
		"endpoint", req.Endpoint,
		"tools_count", len(req.Tools),
		"resources_count", len(req.Resources),
		"prompts_count", len(req.Prompts))

	// Validate registration request
	if sr.config.ValidateOnRegister {
		if result := sr.validator.ValidateStruct(ctx, req); !result.Valid {
			return nil, errors.ValidationError("service_registry", "register_service",
				"Invalid registration request", map[string]interface{}{
					"errors":   result.Errors,
					"warnings": result.Warnings,
					"request":  req,
				})
		}
	}

	// Generate unique service ID
	serviceID := sr.generateServiceID(req.Name, req.Type)

	// Check if service already exists
	if existing := sr.GetService(serviceID); existing != nil {
		return nil, errors.ValidationError("service_registry", "register_service",
			fmt.Sprintf("Service already registered: %s", serviceID), map[string]interface{}{
				"existing_service": existing,
				"new_request":      req,
			})
	}

	// Create registered service
	service := &RegisteredService{
		ID:            serviceID,
		Name:          req.Name,
		Type:          req.Type,
		Version:       req.Version,
		Description:   req.Description,
		Endpoint:      req.Endpoint,
		Protocol:      req.Protocol,
		Tools:         req.Tools,
		Resources:     req.Resources,
		Prompts:       req.Prompts,
		Status:        StatusRegistering,
		Health:        HealthUnknown,
		RegisteredAt:  time.Now(),
		LastSeen:      time.Now(),
		Configuration: req.Configuration,
		Metadata:      req.Metadata,
		Tags:          req.Tags,
		Security:      req.Security,
		client:        &http.Client{Timeout: 30 * time.Second},
	}

	// Perform health check if required
	if sr.config.RequireHealthCheck {
		if err := sr.performHealthCheck(ctx, service); err != nil {
			return nil, errors.UnavailableError("service_registry", "register_service", err, map[string]interface{}{
				"service_id":         serviceID,
				"endpoint":           req.Endpoint,
				"health_check_error": err.Error(),
			})
		}
	}

	// Add service to registry
	sr.mutex.Lock()
	sr.services[serviceID] = service
	sr.addServiceByType(service)
	sr.updateCapabilities(service)
	sr.mutex.Unlock()

	// Update service status
	service.Status = StatusActive
	service.Health = HealthHealthy

	// Record metrics
	sr.recordRegistrationMetrics(service, time.Since(start))

	// Create response
	response := &ServiceRegistrationResponse{
		ServiceID: serviceID,
		Status:    StatusActive,
		Message:   "Service registered successfully",
		Timestamp: time.Now(),
		GatewayInfo: GatewayInfo{
			Version:             "1.0.0",
			SupportedProtocols:  []string{"http", "https"},
			Features:            []string{"load_balancing", "health_checks", "metrics"},
			HealthCheckEndpoint: "/health",
		},
	}

	sr.logger.Info("service_registration_completed",
		"service_id", serviceID,
		"name", req.Name,
		"type", req.Type,
		"version", req.Version,
		"endpoint", req.Endpoint,
		"status", service.Status,
		"health", service.Health,
		"registration_time", time.Since(start),
		"total_services", len(sr.services))

	return response, nil
}

// GetService retrieves a service by ID
func (sr *ServiceRegistry) GetService(serviceID string) *RegisteredService {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	return sr.services[serviceID]
}

// GetServicesByType retrieves all services of a specific type
func (sr *ServiceRegistry) GetServicesByType(serviceType string) []*RegisteredService {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	services := sr.byType[serviceType]
	result := make([]*RegisteredService, len(services))
	copy(result, services)
	return result
}

// GetAllServices retrieves all registered services
func (sr *ServiceRegistry) GetAllServices() map[string]*RegisteredService {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	result := make(map[string]*RegisteredService)
	for id, service := range sr.services {
		result[id] = service
	}
	return result
}

// GetHealthyServices retrieves all healthy services
func (sr *ServiceRegistry) GetHealthyServices() []*RegisteredService {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	var healthy []*RegisteredService
	for _, service := range sr.services {
		if service.Health == HealthHealthy && service.Status == StatusActive {
			healthy = append(healthy, service)
		}
	}
	return healthy
}

// UnregisterService removes a service from the registry
func (sr *ServiceRegistry) UnregisterService(ctx context.Context, serviceID string) error {
	sr.logger.Info("service_unregistration_started", "service_id", serviceID)

	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	service, exists := sr.services[serviceID]
	if !exists {
		return errors.ValidationError("service_registry", "unregister_service",
			fmt.Sprintf("Service not found: %s", serviceID), map[string]interface{}{
				"service_id": serviceID,
			})
	}

	// Update service status to draining
	service.Status = StatusDraining

	// Remove from registry
	delete(sr.services, serviceID)
	sr.removeServiceByType(service)
	sr.updateCapabilitiesAfterRemoval(service)

	sr.logger.Info("service_unregistration_completed",
		"service_id", serviceID,
		"name", service.Name,
		"type", service.Type,
		"uptime", time.Since(service.RegisteredAt),
		"remaining_services", len(sr.services))

	return nil
}

// GetCapabilities returns the aggregated capabilities of all services
func (sr *ServiceRegistry) GetCapabilities() map[string]*ServiceCapabilities {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	result := make(map[string]*ServiceCapabilities)
	for serviceType, caps := range sr.capabilities {
		result[serviceType] = caps
	}
	return result
}

// SelectService selects a service for handling a request (load balancing)
func (sr *ServiceRegistry) SelectService(serviceType string, criteria SelectionCriteria) (*RegisteredService, error) {
	services := sr.GetServicesByType(serviceType)
	if len(services) == 0 {
		return nil, errors.ValidationError("service_registry", "select_service",
			fmt.Sprintf("No services available for type: %s", serviceType), map[string]interface{}{
				"service_type": serviceType,
				"criteria":     criteria,
			})
	}

	// Filter healthy services
	var healthy []*RegisteredService
	for _, service := range services {
		if service.Health == HealthHealthy && service.Status == StatusActive {
			healthy = append(healthy, service)
		}
	}

	if len(healthy) == 0 {
		return nil, errors.UnavailableError("service_registry", "select_service",
			fmt.Errorf("no healthy services available"), map[string]interface{}{
				"service_type":   serviceType,
				"total_services": len(services),
				"criteria":       criteria,
			})
	}

	// Use load balancer to select service
	return sr.loadBalancer.SelectService(healthy, criteria)
}

// SelectionCriteria defines criteria for service selection
type SelectionCriteria struct {
	PreferredRegion string                 `json:"preferred_region,omitempty"`
	Tags            []string               `json:"tags,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	LoadBalancing   string                 `json:"load_balancing,omitempty"`
}

// TriggerDiscovery manually triggers service discovery
func (sr *ServiceRegistry) TriggerDiscovery(ctx context.Context) error {
	if sr.discovery == nil {
		return fmt.Errorf("service discovery not enabled")
	}

	sr.logger.Info("manual_service_discovery_triggered")
	return sr.discovery.DiscoverServices(ctx)
}

// GetDiscoveredServices returns all discovered services
func (sr *ServiceRegistry) GetDiscoveredServices() map[string]*DiscoveredService {
	if sr.discovery == nil {
		return make(map[string]*DiscoveredService)
	}

	return sr.discovery.GetDiscoveredServices()
}

// Shutdown gracefully shuts down the service registry
func (sr *ServiceRegistry) Shutdown() error {
	sr.logger.Info("service_registry_shutting_down")

	sr.cancel()
	sr.wg.Wait()

	sr.logger.Info("service_registry_shutdown_complete")
	return nil
}

// Helper methods

func (sr *ServiceRegistry) generateServiceID(name, serviceType string) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s-%s-%d", serviceType, name, timestamp)
}

func (sr *ServiceRegistry) addServiceByType(service *RegisteredService) {
	services := sr.byType[service.Type]
	sr.byType[service.Type] = append(services, service)
}

func (sr *ServiceRegistry) removeServiceByType(service *RegisteredService) {
	services := sr.byType[service.Type]
	for i, s := range services {
		if s.ID == service.ID {
			sr.byType[service.Type] = append(services[:i], services[i+1:]...)
			break
		}
	}
}

func (sr *ServiceRegistry) updateCapabilities(service *RegisteredService) {
	caps, exists := sr.capabilities[service.Type]
	if !exists {
		caps = &ServiceCapabilities{}
		sr.capabilities[service.Type] = caps
	}

	// Update tool capabilities
	caps.Tools.Count += len(service.Tools)
	for _, tool := range service.Tools {
		if tool.Category != "" && !contains(caps.Tools.Categories, tool.Category) {
			caps.Tools.Categories = append(caps.Tools.Categories, tool.Category)
		}
	}

	// Update resource capabilities
	caps.Resources.Count += len(service.Resources)
	for _, resource := range service.Resources {
		if resource.Type != "" && !contains(caps.Resources.Types, resource.Type) {
			caps.Resources.Types = append(caps.Resources.Types, resource.Type)
		}
	}

	// Update prompt capabilities
	caps.Prompts.Count += len(service.Prompts)
	for _, prompt := range service.Prompts {
		if prompt.Category != "" && !contains(caps.Prompts.Categories, prompt.Category) {
			caps.Prompts.Categories = append(caps.Prompts.Categories, prompt.Category)
		}
	}
}

func (sr *ServiceRegistry) updateCapabilitiesAfterRemoval(service *RegisteredService) {
	// This is a simplified implementation
	// In practice, you'd need to recalculate capabilities for the service type
	caps := sr.capabilities[service.Type]
	if caps != nil {
		caps.Tools.Count -= len(service.Tools)
		caps.Resources.Count -= len(service.Resources)
		caps.Prompts.Count -= len(service.Prompts)
	}
}

func (sr *ServiceRegistry) performHealthCheck(ctx context.Context, service *RegisteredService) error {
	startTime := time.Now()
	sr.logger.Debug("service_health_check_started",
		"service_id", service.ID,
		"service_name", service.Name,
		"endpoint", service.Endpoint)

	// Skip health checks for plugin endpoints - they are managed internally
	if strings.HasPrefix(service.Endpoint, "plugin://") {
		sr.logger.Debug("skipping_health_check_for_plugin",
			"service_id", service.ID,
			"service_name", service.Name,
			"endpoint", service.Endpoint)
		return sr.updateServiceHealth(service, HealthHealthy, nil, time.Since(startTime))
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			IdleConnTimeout:   3 * time.Second,
		},
	}

	// Construct health check URL
	healthURL := sr.buildHealthCheckURL(service)

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		sr.logger.Error("service_health_check_request_creation_failed",
			"service_id", service.ID,
			"health_url", healthURL,
			"error", err)
		return sr.updateServiceHealth(service, HealthUnhealthy, err, time.Since(startTime))
	}

	// Set user agent and other headers
	req.Header.Set("User-Agent", "MCPEG-HealthChecker/1.0")
	req.Header.Set("Accept", "application/json, text/plain, */*")

	// Add authentication if configured
	if err := sr.addHealthCheckAuth(req, service); err != nil {
		sr.logger.Warn("service_health_check_auth_failed",
			"service_id", service.ID,
			"error", err)
	}

	// Perform the health check request
	resp, err := client.Do(req)
	if err != nil {
		sr.logger.Error("service_health_check_request_failed",
			"service_id", service.ID,
			"health_url", healthURL,
			"error", err)
		return sr.updateServiceHealth(service, HealthUnhealthy, err, time.Since(startTime))
	}
	defer resp.Body.Close()

	// Validate response
	if err := sr.validateHealthCheckResponse(resp, service); err != nil {
		sr.logger.Error("service_health_check_validation_failed",
			"service_id", service.ID,
			"status_code", resp.StatusCode,
			"error", err)
		return sr.updateServiceHealth(service, HealthUnhealthy, err, time.Since(startTime))
	}

	// Health check successful
	duration := time.Since(startTime)
	sr.logger.Debug("service_health_check_successful",
		"service_id", service.ID,
		"status_code", resp.StatusCode,
		"response_time_ms", duration.Milliseconds())

	return sr.updateServiceHealth(service, HealthHealthy, nil, duration)
}

// buildHealthCheckURL constructs the health check URL for a service
func (sr *ServiceRegistry) buildHealthCheckURL(service *RegisteredService) string {
	// Use custom health path if specified in metadata
	healthPath := "/health" // default
	if customPath, ok := service.Metadata["health_path"].(string); ok && customPath != "" {
		healthPath = customPath
	}

	// Check if endpoint already has a scheme
	if strings.Contains(service.Endpoint, "://") {
		// Endpoint already has full URL - just append health path
		return service.Endpoint + healthPath
	}

	// Determine protocol for endpoints without scheme
	protocol := "http"
	if useTLS, ok := service.Metadata["tls"]; ok && useTLS == "true" {
		protocol = "https"
	}

	// Build URL with protocol prefix
	return fmt.Sprintf("%s://%s%s", protocol, service.Endpoint, healthPath)
}

// addHealthCheckAuth adds authentication headers if configured
func (sr *ServiceRegistry) addHealthCheckAuth(req *http.Request, service *RegisteredService) error {
	// API Key authentication
	if apiKey, ok := service.Metadata["health_api_key"].(string); ok && apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
		return nil
	}

	// Bearer token authentication
	if token, ok := service.Metadata["health_token"].(string); ok && token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}

	// Basic authentication
	if username, ok := service.Metadata["health_username"].(string); ok && username != "" {
		password, _ := service.Metadata["health_password"].(string)
		req.SetBasicAuth(username, password)
		return nil
	}

	// Custom header authentication
	if headerName, ok := service.Metadata["health_header_name"].(string); ok && headerName != "" {
		headerValue, _ := service.Metadata["health_header_value"].(string)
		if headerValue == "" {
			return fmt.Errorf("health check header value not specified")
		}
		req.Header.Set(headerName, headerValue)
		return nil
	}

	return nil
}

// validateHealthCheckResponse validates the health check response
func (sr *ServiceRegistry) validateHealthCheckResponse(resp *http.Response, service *RegisteredService) error {
	// Check status code
	expectedStatus := sr.getExpectedHealthStatus(service)
	if !sr.isValidHealthStatus(resp.StatusCode, expectedStatus) {
		return fmt.Errorf("unexpected status code: %d, expected one of %v", resp.StatusCode, expectedStatus)
	}

	// Read response body for additional validation
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096)) // Limit to 4KB
	if err != nil {
		sr.logger.Warn("service_health_check_body_read_failed",
			"service_id", service.ID,
			"error", err)
		// Don't fail health check if we can't read body but status is good
		return nil
	}

	// Validate response content if configured
	return sr.validateHealthResponseContent(body, service)
}

// getExpectedHealthStatus returns expected status codes for health checks
func (sr *ServiceRegistry) getExpectedHealthStatus(service *RegisteredService) []int {
	// Check if custom status codes are configured
	if statusStr, ok := service.Metadata["health_expected_status"].(string); ok && statusStr != "" {
		return sr.parseStatusCodes(statusStr)
	}

	// Default expected status codes
	return []int{200, 204}
}

// parseStatusCodes parses comma-separated status codes
func (sr *ServiceRegistry) parseStatusCodes(statusStr string) []int {
	var codes []int
	parts := strings.Split(statusStr, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if code, err := strconv.Atoi(part); err == nil {
			codes = append(codes, code)
		}
	}

	if len(codes) == 0 {
		return []int{200} // fallback
	}

	return codes
}

// isValidHealthStatus checks if status code is in expected list
func (sr *ServiceRegistry) isValidHealthStatus(statusCode int, expected []int) bool {
	for _, code := range expected {
		if statusCode == code {
			return true
		}
	}
	return false
}

// validateHealthResponseContent validates response body content
func (sr *ServiceRegistry) validateHealthResponseContent(body []byte, service *RegisteredService) error {
	if len(body) == 0 {
		return nil // Empty body is OK
	}

	// Check for expected response content
	if expectedContent, ok := service.Metadata["health_expected_content"].(string); ok && expectedContent != "" {
		bodyStr := string(body)
		if !strings.Contains(bodyStr, expectedContent) {
			return fmt.Errorf("response body does not contain expected content: %s", expectedContent)
		}
	}

	// Try to parse as JSON for structured health responses
	if strings.HasPrefix(string(body), "{") {
		return sr.validateJSONHealthResponse(body, service)
	}

	return nil
}

// validateJSONHealthResponse validates JSON health check responses
func (sr *ServiceRegistry) validateJSONHealthResponse(body []byte, service *RegisteredService) error {
	var healthResp map[string]interface{}

	if err := json.Unmarshal(body, &healthResp); err != nil {
		// JSON parsing failed, but that's OK for health checks
		sr.logger.Debug("service_health_response_json_parse_failed",
			"service_id", service.ID,
			"error", err)
		return nil
	}

	// Check status field in JSON response
	if status, ok := healthResp["status"].(string); ok {
		expectedJSONStatus, _ := service.Metadata["health_expected_json_status"].(string)
		if expectedJSONStatus == "" {
			expectedJSONStatus = "ok,healthy,up" // common status values
		}

		validStatuses := strings.Split(expectedJSONStatus, ",")
		statusValid := false
		for _, validStatus := range validStatuses {
			if strings.EqualFold(strings.TrimSpace(validStatus), status) {
				statusValid = true
				break
			}
		}

		if !statusValid {
			return fmt.Errorf("JSON status field indicates unhealthy: %s", status)
		}
	}

	// Log detailed health response for debugging
	sr.logger.Debug("service_health_json_response_received",
		"service_id", service.ID,
		"response", string(body))

	return nil
}

// updateServiceHealth updates service health status and metrics
func (sr *ServiceRegistry) updateServiceHealth(service *RegisteredService, health HealthStatus, err error, duration time.Duration) error {
	service.Health = health
	service.lastHealth = time.Now()

	// Record health check metrics
	sr.recordHealthCheckMetrics(service, health, duration, err)

	// Update failure count for circuit breaker logic
	if health == HealthUnhealthy {
		service.FailureCount++
		if service.FailureCount >= sr.maxFailures {
			service.Status = StatusUnavailable
			sr.logger.Warn("service_marked_unavailable_due_to_health_failures",
				"service_id", service.ID,
				"failure_count", service.FailureCount,
				"max_failures", sr.maxFailures)
		}
	} else {
		service.FailureCount = 0
		if service.Status == StatusUnavailable {
			service.Status = StatusActive
			sr.logger.Info("service_recovered_from_health_failures",
				"service_id", service.ID)
		}
	}

	return err
}

// recordHealthCheckMetrics records metrics for health check operations
func (sr *ServiceRegistry) recordHealthCheckMetrics(service *RegisteredService, health HealthStatus, duration time.Duration, err error) {
	labels := []string{
		"service_id", service.ID,
		"service_name", service.Name,
		"service_type", service.Type,
		"health_status", string(health),
	}

	// Record health check duration
	sr.metrics.Observe("service_health_check_duration_ms", float64(duration.Milliseconds()), labels...)

	// Record health check result
	if health == HealthHealthy {
		sr.metrics.Inc("service_health_check_success_total", labels...)
	} else {
		sr.metrics.Inc("service_health_check_failure_total", labels...)
	}

	// Record specific error types
	if err != nil {
		errorType := "unknown"
		if strings.Contains(err.Error(), "timeout") {
			errorType = "timeout"
		} else if strings.Contains(err.Error(), "connection") {
			errorType = "connection"
		} else if strings.Contains(err.Error(), "status code") {
			errorType = "status_code"
		}

		errorLabels := append(labels, "error_type", errorType)
		sr.metrics.Inc("service_health_check_error_total", errorLabels...)
	}
}

func (sr *ServiceRegistry) recordRegistrationMetrics(service *RegisteredService, duration time.Duration) {
	labels := []string{
		"service_type", service.Type,
		"status", string(service.Status),
	}

	sr.metrics.Inc("service_registrations_total", labels...)
	sr.metrics.Set("service_registration_duration_seconds", duration.Seconds(), labels...)
	sr.metrics.Set("services_registered_total", float64(len(sr.services)))
}

func (sr *ServiceRegistry) startBackgroundProcesses() {
	if sr.config.DiscoveryEnabled {
		sr.wg.Add(1)
		go sr.runDiscovery()
	}

	sr.wg.Add(1)
	go sr.runHealthChecks()

	sr.wg.Add(1)
	go sr.runCleanup()
}

func (sr *ServiceRegistry) runDiscovery() {
	defer sr.wg.Done()
	ticker := time.NewTicker(sr.config.DiscoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sr.discovery.DiscoverServices(sr.ctx)
		case <-sr.ctx.Done():
			return
		}
	}
}

func (sr *ServiceRegistry) runHealthChecks() {
	defer sr.wg.Done()
	ticker := time.NewTicker(sr.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sr.performAllHealthChecks()
		case <-sr.ctx.Done():
			return
		}
	}
}

func (sr *ServiceRegistry) runCleanup() {
	defer sr.wg.Done()
	ticker := time.NewTicker(sr.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sr.cleanupInactiveServices()
		case <-sr.ctx.Done():
			return
		}
	}
}

func (sr *ServiceRegistry) performAllHealthChecks() {
	services := sr.GetAllServices()

	for _, service := range services {
		if err := sr.performHealthCheck(sr.ctx, service); err != nil {
			sr.logger.Warn("service_health_check_failed",
				"service_id", service.ID,
				"name", service.Name,
				"endpoint", service.Endpoint,
				"error", err)

			service.Health = HealthUnhealthy
		}
	}
}

func (sr *ServiceRegistry) cleanupInactiveServices() {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	cutoff := time.Now().Add(-sr.config.InactiveServiceTimeout)

	for id, service := range sr.services {
		if service.LastSeen.Before(cutoff) {
			sr.logger.Info("removing_inactive_service",
				"service_id", id,
				"name", service.Name,
				"last_seen", service.LastSeen,
				"inactive_duration", time.Since(service.LastSeen))

			delete(sr.services, id)
			sr.removeServiceByType(service)
		}
	}
}

func defaultRegistryConfig() RegistryConfig {
	return RegistryConfig{
		DiscoveryEnabled:       true,
		DiscoveryInterval:      60 * time.Second,
		HealthCheckInterval:    30 * time.Second,
		ValidateOnRegister:     true,
		RequireHealthCheck:     true,
		MaxRegistrationTime:    30 * time.Second,
		LoadBalancingEnabled:   true,
		LoadBalancingStrategy:  "round_robin",
		RequireAuthentication:  false,
		AllowSelfRegistration:  true,
		InactiveServiceTimeout: 300 * time.Second,
		CleanupInterval:        120 * time.Second,
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
