package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
)

// ServiceDiscovery handles automatic discovery of MCP services
type ServiceDiscovery struct {
	registry *ServiceRegistry
	logger   logging.Logger
	metrics  metrics.Metrics
	config   DiscoveryConfig
	
	// Discovery mechanisms
	dns      *DNSDiscovery
	consul   *ConsulDiscovery
	k8s      *KubernetesDiscovery
	static   *StaticDiscovery
	
	// Discovery state
	discovered map[string]*DiscoveredService
	mutex      sync.RWMutex
}

// DiscoveryConfig configures service discovery behavior
type DiscoveryConfig struct {
	// DNS-based discovery
	DNSEnabled     bool     `yaml:"dns_enabled"`
	DNSDomains     []string `yaml:"dns_domains"`
	DNSServiceName string   `yaml:"dns_service_name"`
	
	// Consul discovery
	ConsulEnabled bool   `yaml:"consul_enabled"`
	ConsulAddress string `yaml:"consul_address"`
	ConsulService string `yaml:"consul_service"`
	
	// Kubernetes discovery
	K8sEnabled    bool   `yaml:"k8s_enabled"`
	K8sNamespace  string `yaml:"k8s_namespace"`
	K8sLabelSelector string `yaml:"k8s_label_selector"`
	
	// Static configuration
	StaticEnabled  bool               `yaml:"static_enabled"`
	StaticServices []StaticServiceDef `yaml:"static_services"`
	
	// Discovery behavior
	DiscoveryTimeout time.Duration `yaml:"discovery_timeout"`
	RetryInterval   time.Duration `yaml:"retry_interval"`
	MaxRetries      int           `yaml:"max_retries"`
}

// DiscoveredService represents a service discovered through service discovery
type DiscoveredService struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Address     string                 `json:"address"`
	Port        int                    `json:"port"`
	Protocol    string                 `json:"protocol"`
	Metadata    map[string]interface{} `json:"metadata"`
	Tags        []string               `json:"tags"`
	DiscoveredAt time.Time             `json:"discovered_at"`
	Source      string                 `json:"source"`
	
	// Registration attempt tracking
	RegistrationAttempts int       `json:"registration_attempts"`
	LastRegistrationTry  time.Time `json:"last_registration_try"`
	RegistrationError    string    `json:"registration_error,omitempty"`
}

// StaticServiceDef defines a statically configured service
type StaticServiceDef struct {
	Name     string                 `yaml:"name"`
	Type     string                 `yaml:"type"`
	Endpoint string                 `yaml:"endpoint"`
	Metadata map[string]interface{} `yaml:"metadata"`
	Tags     []string               `yaml:"tags"`
}

// NewServiceDiscovery creates a new service discovery manager
func NewServiceDiscovery(registry *ServiceRegistry, logger logging.Logger, metrics metrics.Metrics) *ServiceDiscovery {
	sd := &ServiceDiscovery{
		registry:   registry,
		logger:     logger.WithComponent("service_discovery"),
		metrics:    metrics,
		config:     defaultDiscoveryConfig(),
		discovered: make(map[string]*DiscoveredService),
	}
	
	// Initialize discovery mechanisms
	if sd.config.DNSEnabled {
		sd.dns = NewDNSDiscovery(sd.config, logger, metrics)
	}
	
	if sd.config.ConsulEnabled {
		sd.consul = NewConsulDiscovery(sd.config, logger, metrics)
	}
	
	if sd.config.K8sEnabled {
		sd.k8s = NewKubernetesDiscovery(sd.config, logger, metrics)
	}
	
	if sd.config.StaticEnabled {
		sd.static = NewStaticDiscovery(sd.config, logger, metrics)
	}
	
	return sd
}

// DiscoverServices performs service discovery across all configured mechanisms
func (sd *ServiceDiscovery) DiscoverServices(ctx context.Context) error {
	start := time.Now()
	
	sd.logger.Info("service_discovery_started",
		"dns_enabled", sd.config.DNSEnabled,
		"consul_enabled", sd.config.ConsulEnabled,
		"k8s_enabled", sd.config.K8sEnabled,
		"static_enabled", sd.config.StaticEnabled)
	
	var allDiscovered []*DiscoveredService
	
	// DNS discovery
	if sd.dns != nil {
		if services, err := sd.dns.Discover(ctx); err == nil {
			allDiscovered = append(allDiscovered, services...)
			sd.logger.Debug("dns_discovery_completed", "services_found", len(services))
		} else {
			sd.logger.Warn("dns_discovery_failed", "error", err)
		}
	}
	
	// Consul discovery
	if sd.consul != nil {
		if services, err := sd.consul.Discover(ctx); err == nil {
			allDiscovered = append(allDiscovered, services...)
			sd.logger.Debug("consul_discovery_completed", "services_found", len(services))
		} else {
			sd.logger.Warn("consul_discovery_failed", "error", err)
		}
	}
	
	// Kubernetes discovery
	if sd.k8s != nil {
		if services, err := sd.k8s.Discover(ctx); err == nil {
			allDiscovered = append(allDiscovered, services...)
			sd.logger.Debug("k8s_discovery_completed", "services_found", len(services))
		} else {
			sd.logger.Warn("k8s_discovery_failed", "error", err)
		}
	}
	
	// Static discovery
	if sd.static != nil {
		if services, err := sd.static.Discover(ctx); err == nil {
			allDiscovered = append(allDiscovered, services...)
			sd.logger.Debug("static_discovery_completed", "services_found", len(services))
		} else {
			sd.logger.Warn("static_discovery_failed", "error", err)
		}
	}
	
	// Process discovered services
	newServices := 0
	for _, discovered := range allDiscovered {
		if sd.processDiscoveredService(ctx, discovered) {
			newServices++
		}
	}
	
	duration := time.Since(start)
	
	sd.logger.Info("service_discovery_completed",
		"total_discovered", len(allDiscovered),
		"new_services", newServices,
		"duration", duration)
	
	// Record metrics
	sd.metrics.Set("service_discovery_duration_seconds", duration.Seconds())
	sd.metrics.Set("services_discovered_total", float64(len(allDiscovered)))
	sd.metrics.Set("new_services_registered", float64(newServices))
	
	return nil
}

// processDiscoveredService processes a newly discovered service
func (sd *ServiceDiscovery) processDiscoveredService(ctx context.Context, discovered *DiscoveredService) bool {
	sd.mutex.Lock()
	defer sd.mutex.Unlock()
	
	// Check if we've already discovered this service
	if existing, exists := sd.discovered[discovered.ID]; exists {
		existing.DiscoveredAt = time.Now()
		return false
	}
	
	// Add to discovered services
	sd.discovered[discovered.ID] = discovered
	
	// Attempt to register the service
	if err := sd.registerDiscoveredService(ctx, discovered); err != nil {
		sd.logger.Warn("failed_to_register_discovered_service",
			"service_id", discovered.ID,
			"name", discovered.Name,
			"endpoint", fmt.Sprintf("%s://%s:%d", discovered.Protocol, discovered.Address, discovered.Port),
			"source", discovered.Source,
			"error", err)
		
		discovered.RegistrationError = err.Error()
		discovered.RegistrationAttempts++
		discovered.LastRegistrationTry = time.Now()
		
		return false
	}
	
	sd.logger.Info("discovered_service_registered",
		"service_id", discovered.ID,
		"name", discovered.Name,
		"type", discovered.Type,
		"endpoint", fmt.Sprintf("%s://%s:%d", discovered.Protocol, discovered.Address, discovered.Port),
		"source", discovered.Source)
	
	return true
}

// registerDiscoveredService attempts to register a discovered service
func (sd *ServiceDiscovery) registerDiscoveredService(ctx context.Context, discovered *DiscoveredService) error {
	// First, probe the service to get its capabilities
	capabilities, err := sd.probeServiceCapabilities(ctx, discovered)
	if err != nil {
		return fmt.Errorf("failed to probe service capabilities: %w", err)
	}
	
	// Create registration request
	request := ServiceRegistrationRequest{
		Name:        discovered.Name,
		Type:        discovered.Type,
		Version:     getStringFromMetadata(discovered.Metadata, "version", "unknown"),
		Description: getStringFromMetadata(discovered.Metadata, "description", ""),
		Endpoint:    fmt.Sprintf("%s://%s:%d", discovered.Protocol, discovered.Address, discovered.Port),
		Protocol:    discovered.Protocol,
		Tools:       capabilities.Tools,
		Resources:   capabilities.Resources,
		Prompts:     capabilities.Prompts,
		Metadata:    discovered.Metadata,
		Tags:        discovered.Tags,
	}
	
	// Register with the registry
	_, err = sd.registry.RegisterService(ctx, request)
	return err
}

// probeServiceCapabilities probes a service to discover its capabilities
func (sd *ServiceDiscovery) probeServiceCapabilities(ctx context.Context, discovered *DiscoveredService) (*ServiceCapabilitiesProbe, error) {
	endpoint := fmt.Sprintf("%s://%s:%d", discovered.Protocol, discovered.Address, discovered.Port)
	
	// Try to fetch capabilities from the service
	client := &http.Client{Timeout: 10 * time.Second}
	
	// Try standard MCP adapter endpoints
	capabilities := &ServiceCapabilitiesProbe{}
	
	// Probe for tools
	if tools, err := sd.probeTools(ctx, client, endpoint); err == nil {
		capabilities.Tools = tools
	}
	
	// Probe for resources
	if resources, err := sd.probeResources(ctx, client, endpoint); err == nil {
		capabilities.Resources = resources
	}
	
	// Probe for prompts
	if prompts, err := sd.probePrompts(ctx, client, endpoint); err == nil {
		capabilities.Prompts = prompts
	}
	
	return capabilities, nil
}

// ServiceCapabilitiesProbe represents probed service capabilities
type ServiceCapabilitiesProbe struct {
	Tools     []ToolDefinition     `json:"tools"`
	Resources []ResourceDefinition `json:"resources"`
	Prompts   []PromptDefinition   `json:"prompts"`
}

// probeTools probes a service for available tools
func (sd *ServiceDiscovery) probeTools(ctx context.Context, client *http.Client, endpoint string) ([]ToolDefinition, error) {
	url := endpoint + "/tools"
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	
	var result struct {
		Tools []ToolDefinition `json:"tools"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	return result.Tools, nil
}

// probeResources probes a service for available resources
func (sd *ServiceDiscovery) probeResources(ctx context.Context, client *http.Client, endpoint string) ([]ResourceDefinition, error) {
	url := endpoint + "/resources"
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	
	var result struct {
		Resources []ResourceDefinition `json:"resources"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	return result.Resources, nil
}

// probePrompts probes a service for available prompts
func (sd *ServiceDiscovery) probePrompts(ctx context.Context, client *http.Client, endpoint string) ([]PromptDefinition, error) {
	url := endpoint + "/prompts"
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	
	var result struct {
		Prompts []PromptDefinition `json:"prompts"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	return result.Prompts, nil
}

// GetDiscoveredServices returns all discovered services
func (sd *ServiceDiscovery) GetDiscoveredServices() map[string]*DiscoveredService {
	sd.mutex.RLock()
	defer sd.mutex.RUnlock()
	
	result := make(map[string]*DiscoveredService)
	for id, service := range sd.discovered {
		result[id] = service
	}
	return result
}

// DNS Discovery Implementation
type DNSDiscovery struct {
	config  DiscoveryConfig
	logger  logging.Logger
	metrics metrics.Metrics
}

func NewDNSDiscovery(config DiscoveryConfig, logger logging.Logger, metrics metrics.Metrics) *DNSDiscovery {
	return &DNSDiscovery{
		config:  config,
		logger:  logger.WithComponent("dns_discovery"),
		metrics: metrics,
	}
}

func (d *DNSDiscovery) Discover(ctx context.Context) ([]*DiscoveredService, error) {
	var services []*DiscoveredService
	
	for _, domain := range d.config.DNSDomains {
		serviceName := fmt.Sprintf("%s.%s", d.config.DNSServiceName, domain)
		
		// Perform SRV lookup
		_, srvRecords, err := net.LookupSRV("", "", serviceName)
		if err != nil {
			d.logger.Debug("dns_srv_lookup_failed", "service", serviceName, "error", err)
			continue
		}
		
		for _, srv := range srvRecords {
			service := &DiscoveredService{
				ID:          fmt.Sprintf("dns-%s-%d", srv.Target, srv.Port),
				Name:        strings.TrimSuffix(srv.Target, "."),
				Type:        "mcp_adapter",
				Address:     strings.TrimSuffix(srv.Target, "."),
				Port:        int(srv.Port),
				Protocol:    "http",
				DiscoveredAt: time.Now(),
				Source:      "dns",
				Metadata: map[string]interface{}{
					"priority": srv.Priority,
					"weight":   srv.Weight,
					"domain":   domain,
				},
			}
			
			services = append(services, service)
		}
	}
	
	return services, nil
}

// Static Discovery Implementation
type StaticDiscovery struct {
	config  DiscoveryConfig
	logger  logging.Logger
	metrics metrics.Metrics
}

func NewStaticDiscovery(config DiscoveryConfig, logger logging.Logger, metrics metrics.Metrics) *StaticDiscovery {
	return &StaticDiscovery{
		config:  config,
		logger:  logger.WithComponent("static_discovery"),
		metrics: metrics,
	}
}

func (s *StaticDiscovery) Discover(ctx context.Context) ([]*DiscoveredService, error) {
	var services []*DiscoveredService
	
	for _, staticSvc := range s.config.StaticServices {
		// Parse endpoint to get address and port
		address, port, protocol := parseEndpoint(staticSvc.Endpoint)
		
		service := &DiscoveredService{
			ID:          fmt.Sprintf("static-%s", staticSvc.Name),
			Name:        staticSvc.Name,
			Type:        staticSvc.Type,
			Address:     address,
			Port:        port,
			Protocol:    protocol,
			DiscoveredAt: time.Now(),
			Source:      "static",
			Metadata:    staticSvc.Metadata,
			Tags:        staticSvc.Tags,
		}
		
		services = append(services, service)
	}
	
	return services, nil
}

// Consul Discovery (placeholder)
type ConsulDiscovery struct {
	config  DiscoveryConfig
	logger  logging.Logger
	metrics metrics.Metrics
}

func NewConsulDiscovery(config DiscoveryConfig, logger logging.Logger, metrics metrics.Metrics) *ConsulDiscovery {
	return &ConsulDiscovery{
		config:  config,
		logger:  logger.WithComponent("consul_discovery"),
		metrics: metrics,
	}
}

func (c *ConsulDiscovery) Discover(ctx context.Context) ([]*DiscoveredService, error) {
	c.logger.Debug("consul_discovery_started", "address", c.config.ConsulAddress)
	
	client := &http.Client{Timeout: c.config.DiscoveryTimeout}
	
	// Query Consul for healthy services
	consulURL := fmt.Sprintf("http://%s/v1/health/service/%s?passing=true", 
		c.config.ConsulAddress, c.config.ConsulService)
	
	req, err := http.NewRequestWithContext(ctx, "GET", consulURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Consul request: %w", err)
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Consul request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Consul returned HTTP %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Consul response: %w", err)
	}
	
	var consulServices []ConsulHealthCheck
	if err := json.Unmarshal(body, &consulServices); err != nil {
		return nil, fmt.Errorf("failed to parse Consul response: %w", err)
	}
	
	var services []*DiscoveredService
	for _, consulSvc := range consulServices {
		service := &DiscoveredService{
			ID:       fmt.Sprintf("consul-%s-%s", consulSvc.Service.Service, consulSvc.Service.ID),
			Name:     consulSvc.Service.Service,
			Type:     getConsulServiceType(consulSvc.Service.Tags),
			Address:  consulSvc.Service.Address,
			Port:     consulSvc.Service.Port,
			Protocol: getConsulProtocol(consulSvc.Service.Tags),
			Tags:     consulSvc.Service.Tags,
			DiscoveredAt: time.Now(),
			Source:   "consul",
			Metadata: map[string]interface{}{
				"consul_id":      consulSvc.Service.ID,
				"consul_meta":    consulSvc.Service.Meta,
				"datacenter":     consulSvc.Node.Datacenter,
				"node":          consulSvc.Node.Node,
				"node_address":  consulSvc.Node.Address,
			},
		}
		
		// Use node address if service address is empty
		if service.Address == "" {
			service.Address = consulSvc.Node.Address
		}
		
		services = append(services, service)
	}
	
	c.logger.Debug("consul_discovery_completed", 
		"services_found", len(services),
		"consul_response_count", len(consulServices))
	
	return services, nil
}

// Kubernetes Discovery (placeholder)
type KubernetesDiscovery struct {
	config  DiscoveryConfig
	logger  logging.Logger
	metrics metrics.Metrics
}

func NewKubernetesDiscovery(config DiscoveryConfig, logger logging.Logger, metrics metrics.Metrics) *KubernetesDiscovery {
	return &KubernetesDiscovery{
		config:  config,
		logger:  logger.WithComponent("k8s_discovery"),
		metrics: metrics,
	}
}

func (k *KubernetesDiscovery) Discover(ctx context.Context) ([]*DiscoveredService, error) {
	k.logger.Debug("k8s_discovery_started", 
		"namespace", k.config.K8sNamespace,
		"label_selector", k.config.K8sLabelSelector)
	
	// Check if running in Kubernetes
	if !k.isRunningInKubernetes() {
		k.logger.Debug("not_running_in_kubernetes", "skip_discovery", true)
		return []*DiscoveredService{}, nil
	}
	
	// Get Kubernetes API client
	client := &http.Client{Timeout: k.config.DiscoveryTimeout}
	
	// Get service account token
	token, err := k.getServiceAccountToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get service account token: %w", err)
	}
	
	// Query Kubernetes API for services
	apiURL := fmt.Sprintf("https://kubernetes.default.svc/api/v1/namespaces/%s/services", k.config.K8sNamespace)
	if k.config.K8sLabelSelector != "" {
		values := url.Values{}
		values.Set("labelSelector", k.config.K8sLabelSelector)
		apiURL += "?" + values.Encode()
	}
	
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes API request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Kubernetes API request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Kubernetes API returned HTTP %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Kubernetes API response: %w", err)
	}
	
	var k8sResponse KubernetesServiceList
	if err := json.Unmarshal(body, &k8sResponse); err != nil {
		return nil, fmt.Errorf("failed to parse Kubernetes API response: %w", err)
	}
	
	var services []*DiscoveredService
	for _, k8sSvc := range k8sResponse.Items {
		// Skip services without MCP annotation
		if !isK8sMCPService(k8sSvc) {
			continue
		}
		
		for _, port := range k8sSvc.Spec.Ports {
			service := &DiscoveredService{
				ID:       fmt.Sprintf("k8s-%s-%s-%d", k8sSvc.Metadata.Namespace, k8sSvc.Metadata.Name, port.Port),
				Name:     k8sSvc.Metadata.Name,
				Type:     getK8sServiceType(k8sSvc.Metadata.Annotations),
				Address:  k8sSvc.Spec.ClusterIP,
				Port:     int(port.Port),
				Protocol: getK8sProtocol(k8sSvc.Metadata.Annotations, port.Name),
				Tags:     getK8sTags(k8sSvc.Metadata.Labels),
				DiscoveredAt: time.Now(),
				Source:   "kubernetes",
				Metadata: map[string]interface{}{
					"namespace":   k8sSvc.Metadata.Namespace,
					"labels":      k8sSvc.Metadata.Labels,
					"annotations": k8sSvc.Metadata.Annotations,
					"port_name":   port.Name,
					"port_protocol": port.Protocol,
					"selector":    k8sSvc.Spec.Selector,
				},
			}
			
			services = append(services, service)
		}
	}
	
	k.logger.Debug("k8s_discovery_completed", 
		"services_found", len(services),
		"k8s_services_checked", len(k8sResponse.Items))
	
	return services, nil
}

// Helper functions

func parseEndpoint(endpoint string) (address string, port int, protocol string) {
	// Simple endpoint parsing
	// Format: protocol://address:port
	parts := strings.Split(endpoint, "://")
	if len(parts) == 2 {
		protocol = parts[0]
		hostPort := parts[1]
		
		if strings.Contains(hostPort, ":") {
			hostPortParts := strings.Split(hostPort, ":")
			address = hostPortParts[0]
			if len(hostPortParts) > 1 {
				if p, err := fmt.Sscanf(hostPortParts[1], "%d", &port); p == 1 && err == nil {
					return address, port, protocol
				}
			}
		} else {
			address = hostPort
			if protocol == "https" {
				port = 443
			} else {
				port = 80
			}
		}
	}
	
	return address, port, protocol
}

func getStringFromMetadata(metadata map[string]interface{}, key, defaultValue string) string {
	if value, exists := metadata[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}

// Consul data structures
type ConsulHealthCheck struct {
	Node    ConsulNode    `json:"Node"`
	Service ConsulService `json:"Service"`
	Checks  []ConsulCheck `json:"Checks"`
}

type ConsulNode struct {
	ID         string            `json:"ID"`
	Node       string            `json:"Node"`
	Address    string            `json:"Address"`
	Datacenter string            `json:"Datacenter"`
	TaggedAddresses map[string]interface{} `json:"TaggedAddresses"`
	Meta       map[string]string `json:"Meta"`
}

type ConsulService struct {
	ID      string            `json:"ID"`
	Service string            `json:"Service"`
	Tags    []string          `json:"Tags"`
	Meta    map[string]string `json:"Meta"`
	Port    int               `json:"Port"`
	Address string            `json:"Address"`
	Weights struct {
		Passing int `json:"Passing"`
		Warning int `json:"Warning"`
	} `json:"Weights"`
}

type ConsulCheck struct {
	Node        string   `json:"Node"`
	CheckID     string   `json:"CheckID"`
	Name        string   `json:"Name"`
	Status      string   `json:"Status"`
	Notes       string   `json:"Notes"`
	Output      string   `json:"Output"`
	ServiceID   string   `json:"ServiceID"`
	ServiceName string   `json:"ServiceName"`
	ServiceTags []string `json:"ServiceTags"`
}

// Kubernetes data structures
type KubernetesServiceList struct {
	APIVersion string                `json:"apiVersion"`
	Kind       string                `json:"kind"`
	Metadata   KubernetesListMeta    `json:"metadata"`
	Items      []KubernetesService   `json:"items"`
}

type KubernetesListMeta struct {
	SelfLink        string `json:"selfLink"`
	ResourceVersion string `json:"resourceVersion"`
}

type KubernetesService struct {
	APIVersion string                    `json:"apiVersion"`
	Kind       string                    `json:"kind"`
	Metadata   KubernetesObjectMeta      `json:"metadata"`
	Spec       KubernetesServiceSpec     `json:"spec"`
	Status     KubernetesServiceStatus   `json:"status"`
}

type KubernetesObjectMeta struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	UID         string            `json:"uid"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

type KubernetesServiceSpec struct {
	Type                     string                     `json:"type"`
	Selector                 map[string]string          `json:"selector"`
	ClusterIP               string                     `json:"clusterIP"`
	ClusterIPs              []string                   `json:"clusterIPs"`
	Ports                   []KubernetesServicePort    `json:"ports"`
	ExternalIPs             []string                   `json:"externalIPs"`
	LoadBalancerIP          string                     `json:"loadBalancerIP"`
	LoadBalancerSourceRanges []string                  `json:"loadBalancerSourceRanges"`
}

type KubernetesServicePort struct {
	Name       string `json:"name"`
	Protocol   string `json:"protocol"`
	Port       int32  `json:"port"`
	TargetPort interface{} `json:"targetPort"`
	NodePort   int32  `json:"nodePort"`
}

type KubernetesServiceStatus struct {
	LoadBalancer struct {
		Ingress []struct {
			IP       string `json:"ip"`
			Hostname string `json:"hostname"`
		} `json:"ingress"`
	} `json:"loadBalancer"`
}

// Helper functions for Consul discovery
func getConsulServiceType(tags []string) string {
	for _, tag := range tags {
		if strings.HasPrefix(tag, "mcp-type:") {
			return strings.TrimPrefix(tag, "mcp-type:")
		}
	}
	// Default to mcp_adapter if no specific type tag found
	return "mcp_adapter"
}

func getConsulProtocol(tags []string) string {
	for _, tag := range tags {
		if strings.HasPrefix(tag, "protocol:") {
			return strings.TrimPrefix(tag, "protocol:")
		}
		if tag == "https" || tag == "tls" {
			return "https"
		}
	}
	return "http"
}

// Helper functions for Kubernetes discovery
func (k *KubernetesDiscovery) isRunningInKubernetes() bool {
	// Check if service account token exists
	if _, err := k.getServiceAccountToken(); err != nil {
		return false
	}
	
	// Check if Kubernetes API is accessible
	return true
}

func (k *KubernetesDiscovery) getServiceAccountToken() (string, error) {
	tokenPath := "/var/run/secrets/kubernetes.io/serviceaccount/token"
	token, err := os.ReadFile(tokenPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(token)), nil
}

func isK8sMCPService(svc KubernetesService) bool {
	// Check for MCP annotation
	if annotations := svc.Metadata.Annotations; annotations != nil {
		if mcp, exists := annotations["mcpeg.io/mcp-service"]; exists && mcp == "true" {
			return true
		}
		if _, exists := annotations["mcpeg.io/service-type"]; exists {
			return true
		}
	}
	
	// Check for MCP label
	if labels := svc.Metadata.Labels; labels != nil {
		if app, exists := labels["app"]; exists && (app == "mcp-adapter" || strings.Contains(app, "mcp")) {
			return true
		}
		if _, exists := labels["mcpeg.io/mcp-service"]; exists {
			return true
		}
	}
	
	return false
}

func getK8sServiceType(annotations map[string]string) string {
	if annotations != nil {
		if serviceType, exists := annotations["mcpeg.io/service-type"]; exists {
			return serviceType
		}
	}
	return "mcp_adapter"
}

func getK8sProtocol(annotations map[string]string, portName string) string {
	if annotations != nil {
		if protocol, exists := annotations["mcpeg.io/protocol"]; exists {
			return protocol
		}
	}
	
	// Infer from port name
	if strings.Contains(strings.ToLower(portName), "https") || 
	   strings.Contains(strings.ToLower(portName), "tls") {
		return "https"
	}
	
	return "http"
}

func getK8sTags(labels map[string]string) []string {
	var tags []string
	
	if labels != nil {
		// Convert specific labels to tags
		for key, value := range labels {
			if strings.HasPrefix(key, "mcpeg.io/tag-") {
				tag := strings.TrimPrefix(key, "mcpeg.io/tag-")
				if value == "true" {
					tags = append(tags, tag)
				} else {
					tags = append(tags, fmt.Sprintf("%s:%s", tag, value))
				}
			}
		}
		
		// Add app label as tag if present
		if app, exists := labels["app"]; exists {
			tags = append(tags, fmt.Sprintf("app:%s", app))
		}
		
		// Add version label as tag if present
		if version, exists := labels["version"]; exists {
			tags = append(tags, fmt.Sprintf("version:%s", version))
		}
	}
	
	return tags
}

func defaultDiscoveryConfig() DiscoveryConfig {
	return DiscoveryConfig{
		DNSEnabled:       false,
		DNSDomains:       []string{"local"},
		DNSServiceName:   "mcp-adapter",
		ConsulEnabled:    false,
		ConsulAddress:    "localhost:8500",
		ConsulService:    "mcp-adapter",
		K8sEnabled:       false,
		K8sNamespace:     "default",
		K8sLabelSelector: "app=mcp-adapter",
		StaticEnabled:    true,
		StaticServices:   []StaticServiceDef{},
		DiscoveryTimeout: 30 * time.Second,
		RetryInterval:   60 * time.Second,
		MaxRetries:      3,
	}
}