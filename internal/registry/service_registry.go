package registry

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
	"github.com/osakka/mcpeg/pkg/validation"
	"github.com/osakka/mcpeg/pkg/health"
	"github.com/osakka/mcpeg/pkg/errors"
)

// ServiceRegistry manages all registered MCP service adapters
type ServiceRegistry struct {
	services    map[string]*RegisteredService
	byType      map[string][]*RegisteredService
	capabilities map[string]*ServiceCapabilities
	mutex       sync.RWMutex
	
	logger      logging.Logger
	metrics     metrics.Metrics
	validator   *validation.Validator
	health      *health.HealthManager
	
	config      RegistryConfig
	discovery   *ServiceDiscovery
	loadBalancer *LoadBalancer
	
	// Background monitoring
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// RegisteredService represents a service registered with the gateway
type RegisteredService struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Version      string                 `json:"version"`
	Description  string                 `json:"description,omitempty"`
	
	// Network configuration
	Endpoint     string                 `json:"endpoint"`
	Protocol     string                 `json:"protocol"`
	
	// Service capabilities
	Tools        []ToolDefinition       `json:"tools"`
	Resources    []ResourceDefinition   `json:"resources"`
	Prompts      []PromptDefinition     `json:"prompts"`
	
	// Operational status
	Status       ServiceStatus          `json:"status"`
	Health       HealthStatus           `json:"health"`
	LastSeen     time.Time              `json:"last_seen"`
	RegisteredAt time.Time              `json:"registered_at"`
	
	// Configuration and metadata
	Configuration map[string]interface{} `json:"configuration,omitempty"`
	Metadata     map[string]interface{}  `json:"metadata,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	
	// Performance metrics
	Metrics      ServiceMetrics         `json:"metrics"`
	
	// Security and access control
	Security     ServiceSecurity        `json:"security"`
	
	// Runtime state
	mutex        sync.RWMutex
	client       *http.Client
	lastHealth   time.Time
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
	Count           int      `json:"count"`
	Categories      []string `json:"categories"`
	SupportsAsync   bool     `json:"supports_async"`
	MaxConcurrency  int      `json:"max_concurrency"`
	SupportsStreaming bool   `json:"supports_streaming"`
}

// ResourceCapabilities defines resource-related capabilities
type ResourceCapabilities struct {
	Count             int      `json:"count"`
	Types             []string `json:"types"`
	SupportsSubscription bool  `json:"supports_subscription"`
	SupportsStreaming bool     `json:"supports_streaming"`
	SupportsWatch     bool     `json:"supports_watch"`
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
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category,omitempty"`
	InputSchema map[string]interface{} `json:"input_schema"`
	OutputSchema map[string]interface{} `json:"output_schema,omitempty"`
	Examples    []ToolExample          `json:"examples,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
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
	AuthRequired   bool     `json:"auth_required"`
	AllowedClients []string `json:"allowed_clients,omitempty"`
	RequiredScopes []string `json:"required_scopes,omitempty"`
	RateLimit      *RateLimit `json:"rate_limit,omitempty"`
}

// RateLimit defines rate limiting for a service
type RateLimit struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	Burst            int `json:"burst"`
	WindowSize       time.Duration `json:"window_size"`
}

// RegistryConfig configures the service registry
type RegistryConfig struct {
	// Discovery settings
	DiscoveryEnabled     bool          `yaml:"discovery_enabled"`
	DiscoveryInterval    time.Duration `yaml:"discovery_interval"`
	HealthCheckInterval  time.Duration `yaml:"health_check_interval"`
	
	// Service validation
	ValidateOnRegister   bool          `yaml:"validate_on_register"`
	RequireHealthCheck   bool          `yaml:"require_health_check"`
	MaxRegistrationTime  time.Duration `yaml:"max_registration_time"`
	
	// Load balancing
	LoadBalancingEnabled bool          `yaml:"load_balancing_enabled"`
	LoadBalancingStrategy string       `yaml:"load_balancing_strategy"`
	
	// Security
	RequireAuthentication bool         `yaml:"require_authentication"`
	AllowSelfRegistration bool         `yaml:"allow_self_registration"`
	
	// Cleanup and maintenance
	InactiveServiceTimeout time.Duration `yaml:"inactive_service_timeout"`
	CleanupInterval       time.Duration  `yaml:"cleanup_interval"`
}

// ServiceRegistrationRequest represents a service registration request
type ServiceRegistrationRequest struct {
	Name         string                 `json:"name" validate:"required"`
	Type         string                 `json:"type" validate:"required"`
	Version      string                 `json:"version" validate:"required"`
	Description  string                 `json:"description,omitempty"`
	Endpoint     string                 `json:"endpoint" validate:"required,url"`
	Protocol     string                 `json:"protocol" validate:"required"`
	Tools        []ToolDefinition       `json:"tools,omitempty"`
	Resources    []ResourceDefinition   `json:"resources,omitempty"`
	Prompts      []PromptDefinition     `json:"prompts,omitempty"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	Security     ServiceSecurity        `json:"security,omitempty"`
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
					"errors": result.Errors,
					"warnings": result.Warnings,
					"request": req,
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
				"new_request": req,
			})
	}
	
	// Create registered service
	service := &RegisteredService{
		ID:           serviceID,
		Name:         req.Name,
		Type:         req.Type,
		Version:      req.Version,
		Description:  req.Description,
		Endpoint:     req.Endpoint,
		Protocol:     req.Protocol,
		Tools:        req.Tools,
		Resources:    req.Resources,
		Prompts:      req.Prompts,
		Status:       StatusRegistering,
		Health:       HealthUnknown,
		RegisteredAt: time.Now(),
		LastSeen:     time.Now(),
		Configuration: req.Configuration,
		Metadata:     req.Metadata,
		Tags:         req.Tags,
		Security:     req.Security,
		client:       &http.Client{Timeout: 30 * time.Second},
	}
	
	// Perform health check if required
	if sr.config.RequireHealthCheck {
		if err := sr.performHealthCheck(ctx, service); err != nil {
			return nil, errors.UnavailableError("service_registry", "register_service", err, map[string]interface{}{
				"service_id": serviceID,
				"endpoint": req.Endpoint,
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
			Version:            "1.0.0",
			SupportedProtocols: []string{"http", "https"},
			Features:           []string{"load_balancing", "health_checks", "metrics"},
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
				"criteria": criteria,
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
				"service_type": serviceType,
				"total_services": len(services),
				"criteria": criteria,
			})
	}
	
	// Use load balancer to select service
	return sr.loadBalancer.SelectService(healthy, criteria)
}

// SelectionCriteria defines criteria for service selection
type SelectionCriteria struct {
	PreferredRegion string                 `json:"preferred_region,omitempty"`
	Tags           []string               `json:"tags,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	LoadBalancing  string                 `json:"load_balancing,omitempty"`
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
	// This would make an actual HTTP request to the service's health endpoint
	// For now, we'll simulate a successful health check
	service.Health = HealthHealthy
	service.lastHealth = time.Now()
	return nil
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
		CleanupInterval:       120 * time.Second,
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