package mcp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/osakka/mcpeg/internal/registry"
	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
	"github.com/osakka/mcpeg/pkg/plugins"
)

// PluginDiscovery provides advanced plugin discovery mechanisms for MCP integration
type PluginDiscovery struct {
	pluginManager *plugins.PluginManager
	registry      *registry.ServiceRegistry
	logger        logging.Logger
	metrics       metrics.Metrics
	config        PluginDiscoveryConfig

	// Discovery state
	discoveredPlugins map[string]*DiscoveredPlugin
	capabilities      map[string]*EnhancedPluginCapabilities
	dependencies      map[string]*PluginDependencyGraph
	mutex             sync.RWMutex
}

// PluginDiscoveryConfig configures plugin discovery behavior
type PluginDiscoveryConfig struct {
	// Discovery mechanisms
	BuiltInDiscovery    bool `yaml:"builtin_discovery"`
	ExternalDiscovery   bool `yaml:"external_discovery"`
	RegistryDiscovery   bool `yaml:"registry_discovery"`
	FileSystemDiscovery bool `yaml:"filesystem_discovery"`

	// Discovery paths
	PluginDirectories []string      `yaml:"plugin_directories"`
	ScanInterval      time.Duration `yaml:"scan_interval"`
	DiscoveryTimeout  time.Duration `yaml:"discovery_timeout"`

	// Capability analysis
	DeepCapabilityAnalysis bool          `yaml:"deep_capability_analysis"`
	CapabilityCaching      bool          `yaml:"capability_caching"`
	CacheTTL               time.Duration `yaml:"cache_ttl"`

	// Dependency resolution
	DependencyResolution bool   `yaml:"dependency_resolution"`
	ResolutionStrategy   string `yaml:"resolution_strategy"` // "eager", "lazy", "on-demand"

	// Filtering and prioritization
	EnableCapabilityFiltering bool           `yaml:"enable_capability_filtering"`
	CapabilityRequirements    []string       `yaml:"capability_requirements"`
	PluginPriorities          map[string]int `yaml:"plugin_priorities"`
}

// DiscoveredPlugin represents a discovered plugin with enhanced metadata
type DiscoveredPlugin struct {
	// Basic plugin information
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Type        string `json:"type"` // "builtin", "external", "registry"

	// Discovery metadata
	Source       string                 `json:"source"`
	DiscoveredAt time.Time              `json:"discovered_at"`
	Location     string                 `json:"location"`
	LoadStatus   string                 `json:"load_status"`
	Metadata     map[string]interface{} `json:"metadata"`
	Tags         []string               `json:"tags"`

	// Enhanced capabilities
	Capabilities *EnhancedPluginCapabilities `json:"capabilities"`
	Dependencies *PluginDependencyInfo       `json:"dependencies"`

	// Health and status
	Health        *PluginHealthStatus `json:"health"`
	LastChecked   time.Time           `json:"last_checked"`
	CheckInterval time.Duration       `json:"check_interval"`
}

// EnhancedPluginCapabilities provides detailed plugin capability information
type EnhancedPluginCapabilities struct {
	// Standard capabilities
	Tools     []ToolCapability     `json:"tools"`
	Resources []ResourceCapability `json:"resources"`
	Prompts   []PromptCapability   `json:"prompts"`

	// Extended capabilities
	API                *APICapability         `json:"api,omitempty"`
	Storage            *StorageCapability     `json:"storage,omitempty"`
	Network            *NetworkCapability     `json:"network,omitempty"`
	FileSystem         *FileSystemCapability  `json:"filesystem,omitempty"`
	Security           *SecurityCapability    `json:"security,omitempty"`
	Integration        *IntegrationCapability `json:"integration,omitempty"`
	CustomCapabilities map[string]interface{} `json:"custom_capabilities,omitempty"`

	// Capability metrics
	ToolCount     int     `json:"tool_count"`
	ResourceCount int     `json:"resource_count"`
	PromptCount   int     `json:"prompt_count"`
	TotalCalls    int64   `json:"total_calls"`
	ErrorRate     float64 `json:"error_rate"`
}

// ToolCapability provides enhanced tool capability information
type ToolCapability struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	InputSchema map[string]interface{} `json:"input_schema"`

	// Enhanced metadata
	Complexity    string                `json:"complexity"`     // "simple", "medium", "complex"
	ExecutionTime string                `json:"execution_time"` // "fast", "medium", "slow"
	Dependencies  []string              `json:"dependencies"`
	Prerequisites []string              `json:"prerequisites"`
	Tags          []string              `json:"tags"`
	Examples      []ToolExampleEnhanced `json:"examples"`

	// Capability requirements
	RequiredPermissions  []string              `json:"required_permissions"`
	ResourceRequirements *ResourceRequirements `json:"resource_requirements"`
}

// ResourceCapability provides enhanced resource capability information
type ResourceCapability struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mime_type"`

	// Enhanced metadata
	Size         int64    `json:"size,omitempty"`
	AccessLevel  string   `json:"access_level"` // "read", "write", "admin"
	UpdateFreq   string   `json:"update_freq"`  // "static", "dynamic", "realtime"
	Dependencies []string `json:"dependencies"`
	Tags         []string `json:"tags"`

	// Resource requirements
	RequiredPermissions []string `json:"required_permissions"`
}

// PromptCapability provides enhanced prompt capability information
type PromptCapability struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`

	// Enhanced metadata
	Complexity   string          `json:"complexity"`  // "simple", "medium", "complex"
	InputType    string          `json:"input_type"`  // "none", "simple", "complex"
	OutputType   string          `json:"output_type"` // "text", "structured", "binary"
	Dependencies []string        `json:"dependencies"`
	Tags         []string        `json:"tags"`
	Examples     []PromptExample `json:"examples"`

	// Prompt requirements
	RequiredPermissions []string `json:"required_permissions"`
}

// Additional capability types
type APICapability struct {
	Endpoints   []APIEndpoint `json:"endpoints"`
	Protocols   []string      `json:"protocols"`
	AuthMethods []string      `json:"auth_methods"`
}

type StorageCapability struct {
	Persistent   bool     `json:"persistent"`
	StorageTypes []string `json:"storage_types"`
	MaxSize      int64    `json:"max_size"`
}

type NetworkCapability struct {
	OutboundAccess bool     `json:"outbound_access"`
	Protocols      []string `json:"protocols"`
	Domains        []string `json:"allowed_domains"`
}

type FileSystemCapability struct {
	ReadAccess   bool     `json:"read_access"`
	WriteAccess  bool     `json:"write_access"`
	AllowedPaths []string `json:"allowed_paths"`
	MaxFileSize  int64    `json:"max_file_size"`
}

type SecurityCapability struct {
	RequiresAuth      bool     `json:"requires_auth"`
	PermissionLevel   string   `json:"permission_level"`
	SandboxMode       bool     `json:"sandbox_mode"`
	AllowedOperations []string `json:"allowed_operations"`
}

type IntegrationCapability struct {
	MCPVersion      string   `json:"mcp_version"`
	PluginAPI       string   `json:"plugin_api"`
	InterPluginComm bool     `json:"inter_plugin_comm"`
	EventHandling   bool     `json:"event_handling"`
	HookSupport     []string `json:"hook_support"`
}

// Dependency and requirements types
type PluginDependencyInfo struct {
	Required  []PluginDependency `json:"required"`
	Optional  []PluginDependency `json:"optional"`
	Conflicts []string           `json:"conflicts"`
}

type PluginDependency struct {
	Name       string            `json:"name"`
	Version    string            `json:"version"`
	Type       string            `json:"type"` // "plugin", "service", "library"
	Repository string            `json:"repository,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type PluginDependencyGraph struct {
	Plugin       string                       `json:"plugin"`
	Dependencies map[string]*PluginDependency `json:"dependencies"`
	Dependents   []string                     `json:"dependents"`
	ResolvedAt   time.Time                    `json:"resolved_at"`
	Status       string                       `json:"status"` // "resolved", "unresolved", "conflict"
}

type ResourceRequirements struct {
	Memory  string `json:"memory,omitempty"`
	CPU     string `json:"cpu,omitempty"`
	Disk    string `json:"disk,omitempty"`
	Network string `json:"network,omitempty"`
	Timeout string `json:"timeout,omitempty"`
}

type PluginHealthStatus struct {
	Status     string        `json:"status"` // "healthy", "degraded", "unhealthy"
	LastCheck  time.Time     `json:"last_check"`
	Uptime     time.Duration `json:"uptime"`
	CallCount  int64         `json:"call_count"`
	ErrorCount int64         `json:"error_count"`
	AvgLatency time.Duration `json:"avg_latency"`
	Issues     []string      `json:"issues,omitempty"`
}

type ToolExampleEnhanced struct {
	Description string                 `json:"description"`
	Input       map[string]interface{} `json:"input"`
	Output      interface{}            `json:"output,omitempty"`
}

type PromptExample struct {
	Description string      `json:"description"`
	Input       interface{} `json:"input,omitempty"`
	Output      string      `json:"output"`
}

type APIEndpoint struct {
	Path    string   `json:"path"`
	Methods []string `json:"methods"`
	Params  []string `json:"params,omitempty"`
}

// NewPluginDiscovery creates a new enhanced plugin discovery manager
func NewPluginDiscovery(
	pluginManager *plugins.PluginManager,
	registry *registry.ServiceRegistry,
	logger logging.Logger,
	metrics metrics.Metrics,
) *PluginDiscovery {
	return &PluginDiscovery{
		pluginManager:     pluginManager,
		registry:          registry,
		logger:            logger.WithComponent("plugin_discovery"),
		metrics:           metrics,
		config:            defaultPluginDiscoveryConfig(),
		discoveredPlugins: make(map[string]*DiscoveredPlugin),
		capabilities:      make(map[string]*EnhancedPluginCapabilities),
		dependencies:      make(map[string]*PluginDependencyGraph),
	}
}

// DiscoverPlugins performs comprehensive plugin discovery
func (pd *PluginDiscovery) DiscoverPlugins(ctx context.Context) error {
	start := time.Now()
	pd.logger.Info("advanced_plugin_discovery_started")

	var discoveredCount int

	// 1. Built-in plugin discovery
	if pd.config.BuiltInDiscovery {
		if count, err := pd.discoverBuiltInPlugins(ctx); err == nil {
			discoveredCount += count
			pd.logger.Debug("builtin_plugin_discovery_completed", "plugins_found", count)
		} else {
			pd.logger.Warn("builtin_plugin_discovery_failed", "error", err)
		}
	}

	// 2. External plugin discovery
	if pd.config.ExternalDiscovery {
		if count, err := pd.discoverExternalPlugins(ctx); err == nil {
			discoveredCount += count
			pd.logger.Debug("external_plugin_discovery_completed", "plugins_found", count)
		} else {
			pd.logger.Warn("external_plugin_discovery_failed", "error", err)
		}
	}

	// 3. Registry-based discovery
	if pd.config.RegistryDiscovery {
		if count, err := pd.discoverRegistryPlugins(ctx); err == nil {
			discoveredCount += count
			pd.logger.Debug("registry_plugin_discovery_completed", "plugins_found", count)
		} else {
			pd.logger.Warn("registry_plugin_discovery_failed", "error", err)
		}
	}

	// 4. File system discovery
	if pd.config.FileSystemDiscovery {
		if count, err := pd.discoverFileSystemPlugins(ctx); err == nil {
			discoveredCount += count
			pd.logger.Debug("filesystem_plugin_discovery_completed", "plugins_found", count)
		} else {
			pd.logger.Warn("filesystem_plugin_discovery_failed", "error", err)
		}
	}

	// 5. Capability analysis
	if pd.config.DeepCapabilityAnalysis {
		if err := pd.analyzePluginCapabilities(ctx); err != nil {
			pd.logger.Warn("plugin_capability_analysis_failed", "error", err)
		}
	}

	// 6. Dependency resolution
	if pd.config.DependencyResolution {
		if err := pd.resolvePluginDependencies(ctx); err != nil {
			pd.logger.Warn("plugin_dependency_resolution_failed", "error", err)
		}
	}

	duration := time.Since(start)
	pd.logger.Info("advanced_plugin_discovery_completed",
		"total_discovered", discoveredCount,
		"duration", duration,
		"builtin_enabled", pd.config.BuiltInDiscovery,
		"external_enabled", pd.config.ExternalDiscovery,
		"registry_enabled", pd.config.RegistryDiscovery,
		"filesystem_enabled", pd.config.FileSystemDiscovery)

	// Record metrics
	pd.metrics.Set("plugin_discovery_duration_seconds", duration.Seconds())
	pd.metrics.Set("plugins_discovered_total", float64(discoveredCount))

	return nil
}

// discoverBuiltInPlugins discovers and analyzes built-in plugins
func (pd *PluginDiscovery) discoverBuiltInPlugins(ctx context.Context) (int, error) {
	plugins := pd.pluginManager.ListPlugins()
	var count int

	for name, plugin := range plugins {
		discovered := &DiscoveredPlugin{
			ID:           fmt.Sprintf("builtin-%s", name),
			Name:         plugin.Name(),
			Version:      plugin.Version(),
			Description:  plugin.Description(),
			Type:         "builtin",
			Source:       "builtin",
			DiscoveredAt: time.Now(),
			Location:     "internal",
			LoadStatus:   "loaded",
			Tags:         []string{"builtin", "core"},
			Metadata: map[string]interface{}{
				"plugin_type": "builtin",
				"integrated":  true,
			},
		}

		// Analyze plugin capabilities
		if capabilities, err := pd.analyzePluginCapabilitiesDeep(plugin); err == nil {
			discovered.Capabilities = capabilities
		}

		// Check plugin health
		if health, err := pd.checkPluginHealth(ctx, plugin); err == nil {
			discovered.Health = health
		}

		pd.mutex.Lock()
		pd.discoveredPlugins[discovered.ID] = discovered
		pd.mutex.Unlock()

		count++

		pd.logger.Debug("builtin_plugin_discovered",
			"plugin_id", discovered.ID,
			"name", discovered.Name,
			"version", discovered.Version,
			"tool_count", len(plugin.GetTools()),
			"resource_count", len(plugin.GetResources()),
			"prompt_count", len(plugin.GetPrompts()))
	}

	return count, nil
}

// discoverExternalPlugins discovers external plugins via HTTP probing
func (pd *PluginDiscovery) discoverExternalPlugins(ctx context.Context) (int, error) {
	// This would probe external plugin endpoints
	// For now, return 0 as this requires external plugin infrastructure
	pd.logger.Debug("external_plugin_discovery_skipped", "reason", "no_external_plugins_configured")
	return 0, nil
}

// discoverRegistryPlugins discovers plugins through the service registry
func (pd *PluginDiscovery) discoverRegistryPlugins(ctx context.Context) (int, error) {
	services := pd.registry.GetAllServices()
	var count int

	for _, service := range services {
		// Only process MCP plugin services
		if service.Type != "mcp_plugin" {
			continue
		}

		discovered := &DiscoveredPlugin{
			ID:           fmt.Sprintf("registry-%s", service.ID),
			Name:         service.Name,
			Version:      service.Version,
			Description:  service.Description,
			Type:         "registry",
			Source:       "service_registry",
			DiscoveredAt: time.Now(),
			Location:     service.Endpoint,
			LoadStatus:   string(service.Status),
			Tags:         service.Tags,
			Metadata:     service.Metadata,
		}

		// Extract capabilities from registry information
		capabilities := &EnhancedPluginCapabilities{
			ToolCount:     len(service.Tools),
			ResourceCount: len(service.Resources),
			PromptCount:   len(service.Prompts),
		}

		for _, tool := range service.Tools {
			capabilities.Tools = append(capabilities.Tools, ToolCapability{
				Name:          tool.Name,
				Description:   tool.Description,
				Category:      tool.Category,
				InputSchema:   tool.InputSchema,
				Complexity:    "medium", // Default assumption
				ExecutionTime: "medium",
			})
		}

		for _, resource := range service.Resources {
			capabilities.Resources = append(capabilities.Resources, ResourceCapability{
				URI:         resource.URI,
				Name:        resource.Name,
				Description: resource.Description,
				MimeType:    resource.MimeType,
				AccessLevel: "read", // Default assumption
			})
		}

		for _, prompt := range service.Prompts {
			capabilities.Prompts = append(capabilities.Prompts, PromptCapability{
				Name:        prompt.Name,
				Description: prompt.Description,
				Category:    prompt.Category,
				Complexity:  "medium", // Default assumption
			})
		}

		discovered.Capabilities = capabilities

		pd.mutex.Lock()
		pd.discoveredPlugins[discovered.ID] = discovered
		pd.mutex.Unlock()

		count++
	}

	return count, nil
}

// discoverFileSystemPlugins discovers plugins in configured directories
func (pd *PluginDiscovery) discoverFileSystemPlugins(ctx context.Context) (int, error) {
	// This would scan filesystem directories for plugin manifests
	// For now, return 0 as this requires filesystem plugin infrastructure
	pd.logger.Debug("filesystem_plugin_discovery_skipped", "reason", "no_plugin_directories_configured")
	return 0, nil
}

// analyzePluginCapabilities performs deep capability analysis
func (pd *PluginDiscovery) analyzePluginCapabilities(ctx context.Context) error {
	pd.mutex.RLock()
	plugins := make([]*DiscoveredPlugin, 0, len(pd.discoveredPlugins))
	for _, plugin := range pd.discoveredPlugins {
		plugins = append(plugins, plugin)
	}
	pd.mutex.RUnlock()

	for _, discoveredPlugin := range plugins {
		if discoveredPlugin.Type == "builtin" {
			// Get the actual plugin instance for deep analysis
			if plugin, exists := pd.pluginManager.GetPlugin(discoveredPlugin.Name); exists {
				if capabilities, err := pd.analyzePluginCapabilitiesDeep(plugin); err == nil {
					pd.mutex.Lock()
					pd.capabilities[discoveredPlugin.ID] = capabilities
					discoveredPlugin.Capabilities = capabilities
					pd.mutex.Unlock()
				}
			}
		}
	}

	return nil
}

// resolvePluginDependencies resolves plugin dependencies
func (pd *PluginDiscovery) resolvePluginDependencies(ctx context.Context) error {
	pd.logger.Debug("plugin_dependency_resolution_started", "strategy", pd.config.ResolutionStrategy)

	// For built-in plugins, dependencies are resolved at compile time
	// This would be more relevant for external or dynamically loaded plugins

	return nil
}

// analyzePluginCapabilitiesDeep performs deep capability analysis of a plugin
func (pd *PluginDiscovery) analyzePluginCapabilitiesDeep(plugin plugins.Plugin) (*EnhancedPluginCapabilities, error) {
	capabilities := &EnhancedPluginCapabilities{
		Tools:     make([]ToolCapability, 0),
		Resources: make([]ResourceCapability, 0),
		Prompts:   make([]PromptCapability, 0),
	}

	// Analyze tools
	for _, tool := range plugin.GetTools() {
		toolCap := ToolCapability{
			Name:        tool.Name,
			Description: tool.Description,
			Category:    tool.Category,
			InputSchema: tool.InputSchema,
			Examples:    make([]ToolExampleEnhanced, 0),
		}

		// Enhance with intelligent analysis
		toolCap.Complexity = pd.analyzeToolComplexity(tool)
		toolCap.ExecutionTime = pd.analyzeToolExecutionTime(tool)
		toolCap.RequiredPermissions = pd.analyzeToolPermissions(tool)

		// Convert examples
		for _, example := range tool.Examples {
			input := make(map[string]interface{})
			if inputMap, ok := example.Input.(map[string]interface{}); ok {
				input = inputMap
			}

			toolCap.Examples = append(toolCap.Examples, ToolExampleEnhanced{
				Description: example.Description,
				Input:       input,
			})
		}

		capabilities.Tools = append(capabilities.Tools, toolCap)
	}

	// Analyze resources
	for _, resource := range plugin.GetResources() {
		resourceCap := ResourceCapability{
			URI:         resource.URI,
			Name:        resource.Name,
			Description: resource.Description,
			MimeType:    resource.MimeType,
			AccessLevel: pd.analyzeResourceAccessLevel(resource),
			UpdateFreq:  pd.analyzeResourceUpdateFrequency(resource),
		}

		capabilities.Resources = append(capabilities.Resources, resourceCap)
	}

	// Analyze prompts
	for _, prompt := range plugin.GetPrompts() {
		promptCap := PromptCapability{
			Name:        prompt.Name,
			Description: prompt.Description,
			Category:    prompt.Category,
			Complexity:  pd.analyzePromptComplexity(prompt),
			InputType:   "simple", // Default assumption
			OutputType:  "text",   // Default assumption
		}

		capabilities.Prompts = append(capabilities.Prompts, promptCap)
	}

	// Set counts
	capabilities.ToolCount = len(capabilities.Tools)
	capabilities.ResourceCount = len(capabilities.Resources)
	capabilities.PromptCount = len(capabilities.Prompts)

	// Analyze plugin-level capabilities
	capabilities.Integration = &IntegrationCapability{
		MCPVersion:      "2025-03-26",
		PluginAPI:       "1.0",
		InterPluginComm: false, // Default for built-in plugins
		EventHandling:   false,
		HookSupport:     []string{},
	}

	return capabilities, nil
}

// checkPluginHealth checks the health status of a plugin
func (pd *PluginDiscovery) checkPluginHealth(ctx context.Context, plugin plugins.Plugin) (*PluginHealthStatus, error) {
	start := time.Now()

	// Perform health check
	err := plugin.HealthCheck(ctx)

	health := &PluginHealthStatus{
		LastCheck:  time.Now(),
		CallCount:  0, // Would be tracked by metrics
		ErrorCount: 0, // Would be tracked by metrics
		AvgLatency: time.Since(start),
		Issues:     []string{},
	}

	if err != nil {
		health.Status = "unhealthy"
		health.Issues = append(health.Issues, err.Error())
	} else {
		health.Status = "healthy"
	}

	return health, nil
}

// Analysis helper methods
func (pd *PluginDiscovery) analyzeToolComplexity(tool registry.ToolDefinition) string {
	// Analyze input schema complexity
	if inputSchema, ok := tool.InputSchema["properties"].(map[string]interface{}); ok {
		if len(inputSchema) > 5 {
			return "complex"
		} else if len(inputSchema) > 2 {
			return "medium"
		}
	}
	return "simple"
}

func (pd *PluginDiscovery) analyzeToolExecutionTime(tool registry.ToolDefinition) string {
	// Heuristic analysis based on tool name and category
	name := strings.ToLower(tool.Name)
	category := strings.ToLower(tool.Category)

	if strings.Contains(name, "list") || strings.Contains(name, "get") || strings.Contains(name, "status") {
		return "fast"
	}

	if strings.Contains(category, "git") || strings.Contains(name, "clone") || strings.Contains(name, "push") {
		return "slow"
	}

	return "medium"
}

func (pd *PluginDiscovery) analyzeToolPermissions(tool registry.ToolDefinition) []string {
	permissions := []string{}
	name := strings.ToLower(tool.Name)

	if strings.Contains(name, "delete") || strings.Contains(name, "remove") || strings.Contains(name, "clear") {
		permissions = append(permissions, "write", "admin")
	} else if strings.Contains(name, "store") || strings.Contains(name, "save") || strings.Contains(name, "create") {
		permissions = append(permissions, "write")
	} else {
		permissions = append(permissions, "read")
	}

	return permissions
}

func (pd *PluginDiscovery) analyzeResourceAccessLevel(resource registry.ResourceDefinition) string {
	name := strings.ToLower(resource.Name)

	if strings.Contains(name, "dump") || strings.Contains(name, "debug") {
		return "admin"
	} else if strings.Contains(name, "stats") || strings.Contains(name, "info") {
		return "read"
	}

	return "read"
}

func (pd *PluginDiscovery) analyzeResourceUpdateFrequency(resource registry.ResourceDefinition) string {
	name := strings.ToLower(resource.Name)

	if strings.Contains(name, "stats") || strings.Contains(name, "status") {
		return "dynamic"
	} else if strings.Contains(name, "info") || strings.Contains(name, "metadata") {
		return "static"
	}

	return "dynamic"
}

func (pd *PluginDiscovery) analyzePromptComplexity(prompt registry.PromptDefinition) string {
	if strings.Contains(strings.ToLower(prompt.Description), "complex") ||
		strings.Contains(strings.ToLower(prompt.Description), "advanced") {
		return "complex"
	}
	return "simple"
}

// GetDiscoveredPlugins returns all discovered plugins
func (pd *PluginDiscovery) GetDiscoveredPlugins() map[string]*DiscoveredPlugin {
	pd.mutex.RLock()
	defer pd.mutex.RUnlock()

	result := make(map[string]*DiscoveredPlugin)
	for id, plugin := range pd.discoveredPlugins {
		result[id] = plugin
	}
	return result
}

// GetPluginsByCapability returns plugins filtered by capability requirements
func (pd *PluginDiscovery) GetPluginsByCapability(requirements []string) []*DiscoveredPlugin {
	pd.mutex.RLock()
	defer pd.mutex.RUnlock()

	var filtered []*DiscoveredPlugin

	for _, plugin := range pd.discoveredPlugins {
		if pd.pluginMatchesRequirements(plugin, requirements) {
			filtered = append(filtered, plugin)
		}
	}

	return filtered
}

// pluginMatchesRequirements checks if a plugin matches capability requirements
func (pd *PluginDiscovery) pluginMatchesRequirements(plugin *DiscoveredPlugin, requirements []string) bool {
	if plugin.Capabilities == nil {
		return false
	}

	for _, req := range requirements {
		switch req {
		case "memory":
			if plugin.Name != "memory" {
				return false
			}
		case "git":
			if plugin.Name != "git" {
				return false
			}
		case "editor":
			if plugin.Name != "editor" {
				return false
			}
		case "storage":
			if plugin.Capabilities.Storage == nil {
				return false
			}
		case "network":
			if plugin.Capabilities.Network == nil {
				return false
			}
		}
	}

	return true
}

// GetPluginDependencies returns dependency information for all plugins
func (pd *PluginDiscovery) GetPluginDependencies() map[string]*PluginDependencyGraph {
	pd.mutex.RLock()
	defer pd.mutex.RUnlock()

	result := make(map[string]*PluginDependencyGraph)
	for id, deps := range pd.dependencies {
		result[id] = deps
	}
	return result
}

func defaultPluginDiscoveryConfig() PluginDiscoveryConfig {
	return PluginDiscoveryConfig{
		BuiltInDiscovery:          true,
		ExternalDiscovery:         false,
		RegistryDiscovery:         true,
		FileSystemDiscovery:       false,
		PluginDirectories:         []string{"./plugins", "/opt/mcpeg/plugins"},
		ScanInterval:              5 * time.Minute,
		DiscoveryTimeout:          30 * time.Second,
		DeepCapabilityAnalysis:    true,
		CapabilityCaching:         true,
		CacheTTL:                  10 * time.Minute,
		DependencyResolution:      true,
		ResolutionStrategy:        "eager",
		EnableCapabilityFiltering: false,
		CapabilityRequirements:    []string{},
		PluginPriorities:          map[string]int{},
	}
}
