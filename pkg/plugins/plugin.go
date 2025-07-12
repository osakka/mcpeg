package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/osakka/mcpeg/internal/registry"
	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
)

// Plugin represents a MCpeg service plugin
type Plugin interface {
	// Metadata
	Name() string
	Version() string
	Description() string

	// MCP Protocol Support
	GetTools() []registry.ToolDefinition
	GetResources() []registry.ResourceDefinition
	GetPrompts() []registry.PromptDefinition

	// Tool execution
	CallTool(ctx context.Context, name string, args json.RawMessage) (interface{}, error)

	// Resource access
	ReadResource(ctx context.Context, uri string) (interface{}, error)
	ListResources(ctx context.Context) ([]registry.ResourceDefinition, error)

	// Prompt access
	GetPrompt(ctx context.Context, name string, args json.RawMessage) (interface{}, error)

	// Lifecycle
	Initialize(ctx context.Context, config PluginConfig) error
	Shutdown(ctx context.Context) error

	// Health
	HealthCheck(ctx context.Context) error
}

// PluginConfig contains plugin configuration
type PluginConfig struct {
	Name    string                 `json:"name"`
	Config  map[string]interface{} `json:"config"`
	Logger  logging.Logger         `json:"-"`
	Metrics metrics.Metrics        `json:"-"`
}

// BasePlugin provides common functionality for all plugins
type BasePlugin struct {
	name        string
	version     string
	description string
	logger      logging.Logger
	metrics     metrics.Metrics
	initialized bool
}

// NewBasePlugin creates a new base plugin
func NewBasePlugin(name, version, description string) *BasePlugin {
	return &BasePlugin{
		name:        name,
		version:     version,
		description: description,
	}
}

func (p *BasePlugin) Name() string        { return p.name }
func (p *BasePlugin) Version() string     { return p.version }
func (p *BasePlugin) Description() string { return p.description }

func (p *BasePlugin) Initialize(ctx context.Context, config PluginConfig) error {
	p.logger = config.Logger.WithComponent("plugin_" + p.name)
	p.metrics = config.Metrics.WithPrefix("plugin_" + p.name)
	p.initialized = true

	p.logger.Info("plugin_initialized",
		"plugin", p.name,
		"version", p.version)

	p.metrics.Inc("plugin_initializations_total")

	return nil
}

func (p *BasePlugin) Shutdown(ctx context.Context) error {
	p.logger.Info("plugin_shutdown",
		"plugin", p.name)

	p.metrics.Inc("plugin_shutdowns_total")
	p.initialized = false

	return nil
}

func (p *BasePlugin) HealthCheck(ctx context.Context) error {
	if !p.initialized {
		return fmt.Errorf("plugin %s not initialized", p.name)
	}
	return nil
}

// Helper methods for plugins
func (p *BasePlugin) LogToolCall(toolName string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}

	p.logger.Info("plugin_tool_called",
		"plugin", p.name,
		"tool", toolName,
		"duration_ms", duration.Milliseconds(),
		"status", status,
		"error", err)

	p.metrics.Inc("plugin_tool_calls_total", "tool", toolName, "status", status)
	p.metrics.Observe("plugin_tool_duration_ms", float64(duration.Milliseconds()), "tool", toolName)
}

func (p *BasePlugin) LogResourceAccess(resourceURI string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}

	p.logger.Info("plugin_resource_accessed",
		"plugin", p.name,
		"resource", resourceURI,
		"duration_ms", duration.Milliseconds(),
		"status", status,
		"error", err)

	p.metrics.Inc("plugin_resource_accesses_total", "resource", resourceURI, "status", status)
	p.metrics.Observe("plugin_resource_duration_ms", float64(duration.Milliseconds()))
}

// PluginManager manages the lifecycle of plugins
type PluginManager struct {
	plugins map[string]Plugin
	logger  logging.Logger
	metrics metrics.Metrics
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(logger logging.Logger, metrics metrics.Metrics) *PluginManager {
	return &PluginManager{
		plugins: make(map[string]Plugin),
		logger:  logger.WithComponent("plugin_manager"),
		metrics: metrics.WithPrefix("plugin_manager"),
	}
}

// RegisterPlugin registers a plugin with the manager
func (pm *PluginManager) RegisterPlugin(plugin Plugin) error {
	name := plugin.Name()

	if _, exists := pm.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}

	pm.plugins[name] = plugin

	pm.logger.Info("plugin_registered",
		"plugin", name,
		"version", plugin.Version(),
		"description", plugin.Description())

	pm.metrics.Inc("plugins_registered_total")
	pm.metrics.Set("plugins_active_count", float64(len(pm.plugins)))

	return nil
}

// InitializePlugin initializes a specific plugin
func (pm *PluginManager) InitializePlugin(ctx context.Context, name string, config PluginConfig) error {
	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	config.Logger = pm.logger
	config.Metrics = pm.metrics

	if err := plugin.Initialize(ctx, config); err != nil {
		pm.logger.Error("plugin_initialization_failed",
			"plugin", name,
			"error", err)
		return fmt.Errorf("failed to initialize plugin %s: %w", name, err)
	}

	return nil
}

// InitializeAllPlugins initializes all registered plugins
func (pm *PluginManager) InitializeAllPlugins(ctx context.Context, configs map[string]PluginConfig) error {
	for name := range pm.plugins {
		config, exists := configs[name]
		if !exists {
			config = PluginConfig{Name: name, Config: make(map[string]interface{})}
		}

		if err := pm.InitializePlugin(ctx, name, config); err != nil {
			return err
		}
	}

	pm.logger.Info("all_plugins_initialized",
		"plugin_count", len(pm.plugins))

	return nil
}

// GetPlugin returns a plugin by name
func (pm *PluginManager) GetPlugin(name string) (Plugin, bool) {
	plugin, exists := pm.plugins[name]
	return plugin, exists
}

// ListPlugins returns all registered plugins
func (pm *PluginManager) ListPlugins() map[string]Plugin {
	result := make(map[string]Plugin)
	for name, plugin := range pm.plugins {
		result[name] = plugin
	}
	return result
}

// ShutdownAllPlugins shuts down all plugins
func (pm *PluginManager) ShutdownAllPlugins(ctx context.Context) error {
	var lastError error

	for name, plugin := range pm.plugins {
		if err := plugin.Shutdown(ctx); err != nil {
			pm.logger.Error("plugin_shutdown_failed",
				"plugin", name,
				"error", err)
			lastError = err
		}
	}

	pm.logger.Info("all_plugins_shutdown")
	return lastError
}

// GetAllTools returns all tools from all plugins
func (pm *PluginManager) GetAllTools() []registry.ToolDefinition {
	var tools []registry.ToolDefinition

	for _, plugin := range pm.plugins {
		pluginTools := plugin.GetTools()
		tools = append(tools, pluginTools...)
	}

	return tools
}

// GetAllResources returns all resources from all plugins
func (pm *PluginManager) GetAllResources() []registry.ResourceDefinition {
	var resources []registry.ResourceDefinition

	for _, plugin := range pm.plugins {
		pluginResources := plugin.GetResources()
		resources = append(resources, pluginResources...)
	}

	return resources
}

// CallTool calls a tool from any plugin
func (pm *PluginManager) CallTool(ctx context.Context, toolName string, args json.RawMessage) (interface{}, error) {
	// Find which plugin has this tool
	for _, plugin := range pm.plugins {
		for _, tool := range plugin.GetTools() {
			if tool.Name == toolName {
				start := time.Now()
				result, err := plugin.CallTool(ctx, toolName, args)
				duration := time.Since(start)

				pm.logger.Info("plugin_tool_called",
					"plugin", plugin.Name(),
					"tool", toolName,
					"duration_ms", duration.Milliseconds(),
					"success", err == nil)

				return result, err
			}
		}
	}

	return nil, fmt.Errorf("tool %s not found in any plugin", toolName)
}
