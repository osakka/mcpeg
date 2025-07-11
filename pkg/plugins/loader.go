package plugins

import (
	"context"
	"fmt"
	"time"

	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
	"github.com/osakka/mcpeg/internal/registry"
)

// PluginLoader manages loading and registration of all plugins
type PluginLoader struct {
	manager *PluginManager
	logger  logging.Logger
	metrics metrics.Metrics
}

// NewPluginLoader creates a new plugin loader
func NewPluginLoader(logger logging.Logger, metrics metrics.Metrics) *PluginLoader {
	return &PluginLoader{
		manager: NewPluginManager(logger, metrics),
		logger:  logger.WithComponent("plugin_loader"),
		metrics: metrics.WithPrefix("plugin_loader"),
	}
}

// LoadAllPlugins loads and registers all built-in plugins
func (pl *PluginLoader) LoadAllPlugins(ctx context.Context, configs map[string]PluginConfig) error {
	pl.logger.Info("loading_built_in_plugins")
	
	// Register built-in plugins
	plugins := []Plugin{
		NewMemoryService(),
		NewGitService(),
		NewEditorService(),
	}
	
	// Register each plugin
	for _, plugin := range plugins {
		if err := pl.manager.RegisterPlugin(plugin); err != nil {
			pl.logger.Error("failed_to_register_plugin",
				"plugin", plugin.Name(),
				"error", err)
			return fmt.Errorf("failed to register plugin %s: %w", plugin.Name(), err)
		}
		
		pl.logger.Info("plugin_registered",
			"plugin", plugin.Name(),
			"version", plugin.Version(),
			"description", plugin.Description())
	}
	
	// Initialize all plugins
	if err := pl.manager.InitializeAllPlugins(ctx, configs); err != nil {
		return fmt.Errorf("failed to initialize plugins: %w", err)
	}
	
	pl.metrics.Set("plugins_loaded_count", float64(len(plugins)))
	pl.logger.Info("all_plugins_loaded", "count", len(plugins))
	
	return nil
}

// GetPluginManager returns the plugin manager
func (pl *PluginLoader) GetPluginManager() *PluginManager {
	return pl.manager
}

// CreateRegisteredServices converts plugins to registered services for the gateway
func (pl *PluginLoader) CreateRegisteredServices() []*registry.RegisteredService {
	var services []*registry.RegisteredService
	
	for _, plugin := range pl.manager.ListPlugins() {
		service := &registry.RegisteredService{
			ID:          fmt.Sprintf("plugin_%s", plugin.Name()),
			Name:        plugin.Name(),
			Type:        "mcp_plugin",
			Version:     plugin.Version(),
			Description: plugin.Description(),
			Endpoint:    fmt.Sprintf("plugin://%s", plugin.Name()),
			Protocol:    "mcp",
			
			// MCP capabilities
			Tools:     plugin.GetTools(),
			Resources: plugin.GetResources(),
			Prompts:   plugin.GetPrompts(),
			
			// Status
			Status:       registry.StatusActive,
			Health:       registry.HealthHealthy,
			RegisteredAt: time.Now(),
			LastSeen:     time.Now(),
			
			// Metadata
			Metadata: map[string]interface{}{
				"plugin_type":    "built_in",
				"plugin_version": plugin.Version(),
				"capabilities": map[string]interface{}{
					"tools_count":     len(plugin.GetTools()),
					"resources_count": len(plugin.GetResources()),
					"prompts_count":   len(plugin.GetPrompts()),
				},
			},
			Tags: []string{"plugin", "built_in", plugin.Name()},
			
			// Metrics
			Metrics: registry.ServiceMetrics{
				RequestCount: 0,
				ErrorCount:   0,
			},
			
			// Security
			Security: registry.ServiceSecurity{
				AuthRequired: false,
			},
		}
		
		services = append(services, service)
	}
	
	return services
}

// GetDefaultPluginConfigs returns default configurations for all plugins
func (pl *PluginLoader) GetDefaultPluginConfigs() map[string]PluginConfig {
	return map[string]PluginConfig{
		"memory": {
			Name: "memory",
			Config: map[string]interface{}{
				"data_dir":       "./data",
				"auto_save":      true,
				"max_keys":       10000,
				"default_ttl":    3600, // 1 hour
			},
		},
		"git": {
			Name: "git",
			Config: map[string]interface{}{
				"working_dir":   ".",
				"git_path":      "git",
				"auto_detect":   true,
				"safe_mode":     true, // Require confirmation for destructive operations
			},
		},
		"editor": {
			Name: "editor",
			Config: map[string]interface{}{
				"working_dir":      ".",
				"max_file_size":    10485760, // 10MB
				"backup_enabled":   true,
				"allowed_extensions": []string{
					".go", ".js", ".ts", ".py", ".java", ".cpp", ".c", ".h",
					".md", ".txt", ".json", ".yaml", ".yml", ".xml", 
					".html", ".css", ".sql", ".sh", ".env",
				},
			},
		},
	}
}

// ShutdownAllPlugins shuts down all loaded plugins
func (pl *PluginLoader) ShutdownAllPlugins(ctx context.Context) error {
	pl.logger.Info("shutting_down_all_plugins")
	
	if err := pl.manager.ShutdownAllPlugins(ctx); err != nil {
		pl.logger.Error("failed_to_shutdown_plugins", "error", err)
		return err
	}
	
	pl.logger.Info("all_plugins_shutdown")
	return nil
}

// GetPluginInfo returns information about a specific plugin
func (pl *PluginLoader) GetPluginInfo(name string) (map[string]interface{}, error) {
	plugin, exists := pl.manager.GetPlugin(name)
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}
	
	return map[string]interface{}{
		"name":        plugin.Name(),
		"version":     plugin.Version(),
		"description": plugin.Description(),
		"tools":       plugin.GetTools(),
		"resources":   plugin.GetResources(),
		"prompts":     plugin.GetPrompts(),
		"capabilities": map[string]interface{}{
			"tools_count":     len(plugin.GetTools()),
			"resources_count": len(plugin.GetResources()),
			"prompts_count":   len(plugin.GetPrompts()),
		},
	}, nil
}

// GetAllPluginInfo returns information about all loaded plugins
func (pl *PluginLoader) GetAllPluginInfo() map[string]interface{} {
	plugins := pl.manager.ListPlugins()
	info := make(map[string]interface{})
	
	for name, plugin := range plugins {
		info[name] = map[string]interface{}{
			"name":        plugin.Name(),
			"version":     plugin.Version(),
			"description": plugin.Description(),
			"capabilities": map[string]interface{}{
				"tools_count":     len(plugin.GetTools()),
				"resources_count": len(plugin.GetResources()),
				"prompts_count":   len(plugin.GetPrompts()),
			},
		}
	}
	
	return map[string]interface{}{
		"total_plugins": len(plugins),
		"plugins":       info,
	}
}

// CallPluginTool calls a tool from any loaded plugin
func (pl *PluginLoader) CallPluginTool(ctx context.Context, toolName string, args []byte) (interface{}, error) {
	result, err := pl.manager.CallTool(ctx, toolName, args)
	if err != nil {
		pl.metrics.Inc("plugin_tool_errors_total", "tool", toolName)
		return nil, err
	}
	
	pl.metrics.Inc("plugin_tool_calls_total", "tool", toolName)
	return result, nil
}

// GetPluginResource gets a resource from any loaded plugin
func (pl *PluginLoader) GetPluginResource(ctx context.Context, pluginName, resourceURI string) (interface{}, error) {
	plugin, exists := pl.manager.GetPlugin(pluginName)
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", pluginName)
	}
	
	result, err := plugin.ReadResource(ctx, resourceURI)
	if err != nil {
		pl.metrics.Inc("plugin_resource_errors_total", "plugin", pluginName, "resource", resourceURI)
		return nil, err
	}
	
	pl.metrics.Inc("plugin_resource_accesses_total", "plugin", pluginName, "resource", resourceURI)
	return result, nil
}

// GetPluginPrompt gets a prompt from any loaded plugin
func (pl *PluginLoader) GetPluginPrompt(ctx context.Context, pluginName, promptName string, args []byte) (interface{}, error) {
	plugin, exists := pl.manager.GetPlugin(pluginName)
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", pluginName)
	}
	
	result, err := plugin.GetPrompt(ctx, promptName, args)
	if err != nil {
		pl.metrics.Inc("plugin_prompt_errors_total", "plugin", pluginName, "prompt", promptName)
		return nil, err
	}
	
	pl.metrics.Inc("plugin_prompt_calls_total", "plugin", pluginName, "prompt", promptName)
	return result, nil
}

// HealthCheckAllPlugins checks the health of all loaded plugins
func (pl *PluginLoader) HealthCheckAllPlugins(ctx context.Context) map[string]error {
	plugins := pl.manager.ListPlugins()
	results := make(map[string]error)
	
	for name, plugin := range plugins {
		err := plugin.HealthCheck(ctx)
		results[name] = err
		
		if err != nil {
			pl.metrics.Inc("plugin_health_check_failures_total", "plugin", name)
			pl.logger.Warn("plugin_health_check_failed",
				"plugin", name,
				"error", err)
		} else {
			pl.metrics.Inc("plugin_health_check_successes_total", "plugin", name)
		}
	}
	
	return results
}

// Helper methods

func (pl *PluginLoader) getCurrentTime() interface{} {
	// Return current time in a format compatible with the registry
	// This is a placeholder - the actual implementation would depend on 
	// how the registry expects time to be formatted
	return "2023-01-01T00:00:00Z"
}

// PluginServiceAdapter adapts the plugin system to work with the MCpeg service registry
type PluginServiceAdapter struct {
	loader *PluginLoader
	logger logging.Logger
}

// NewPluginServiceAdapter creates a new plugin service adapter
func NewPluginServiceAdapter(loader *PluginLoader, logger logging.Logger) *PluginServiceAdapter {
	return &PluginServiceAdapter{
		loader: loader,
		logger: logger.WithComponent("plugin_adapter"),
	}
}

// HandleToolCall handles MCP tool calls for plugins
func (psa *PluginServiceAdapter) HandleToolCall(ctx context.Context, toolName string, args []byte) (interface{}, error) {
	psa.logger.Debug("plugin_tool_call_received",
		"tool", toolName,
		"args_size", len(args))
	
	result, err := psa.loader.CallPluginTool(ctx, toolName, args)
	if err != nil {
		psa.logger.Error("plugin_tool_call_failed",
			"tool", toolName,
			"error", err)
		return nil, err
	}
	
	psa.logger.Debug("plugin_tool_call_completed",
		"tool", toolName,
		"success", true)
	
	return result, nil
}

// HandleResourceRequest handles MCP resource requests for plugins
func (psa *PluginServiceAdapter) HandleResourceRequest(ctx context.Context, pluginName, resourceURI string) (interface{}, error) {
	psa.logger.Debug("plugin_resource_request_received",
		"plugin", pluginName,
		"resource", resourceURI)
	
	result, err := psa.loader.GetPluginResource(ctx, pluginName, resourceURI)
	if err != nil {
		psa.logger.Error("plugin_resource_request_failed",
			"plugin", pluginName,
			"resource", resourceURI,
			"error", err)
		return nil, err
	}
	
	psa.logger.Debug("plugin_resource_request_completed",
		"plugin", pluginName,
		"resource", resourceURI,
		"success", true)
	
	return result, nil
}

// HandlePromptRequest handles MCP prompt requests for plugins
func (psa *PluginServiceAdapter) HandlePromptRequest(ctx context.Context, pluginName, promptName string, args []byte) (interface{}, error) {
	psa.logger.Debug("plugin_prompt_request_received",
		"plugin", pluginName,
		"prompt", promptName,
		"args_size", len(args))
	
	result, err := psa.loader.GetPluginPrompt(ctx, pluginName, promptName, args)
	if err != nil {
		psa.logger.Error("plugin_prompt_request_failed",
			"plugin", pluginName,
			"prompt", promptName,
			"error", err)
		return nil, err
	}
	
	psa.logger.Debug("plugin_prompt_request_completed",
		"plugin", pluginName,
		"prompt", promptName,
		"success", true)
	
	return result, nil
}