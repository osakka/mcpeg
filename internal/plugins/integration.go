package plugins

import (
	"context"
	"fmt"
	"time"

	"github.com/osakka/mcpeg/internal/registry"
	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
	"github.com/osakka/mcpeg/pkg/plugins"
)

// MCpegPluginIntegration integrates plugins with the MCpeg gateway
type MCpegPluginIntegration struct {
	loader   *plugins.PluginLoader
	adapter  *plugins.PluginServiceAdapter
	registry *registry.ServiceRegistry
	logger   logging.Logger
	metrics  metrics.Metrics
}

// NewMCpegPluginIntegration creates a new plugin integration
func NewMCpegPluginIntegration(
	serviceRegistry *registry.ServiceRegistry,
	logger logging.Logger,
	metricsCollector metrics.Metrics,
) *MCpegPluginIntegration {
	loader := plugins.NewPluginLoader(logger, metricsCollector)
	adapter := plugins.NewPluginServiceAdapter(loader, logger)

	return &MCpegPluginIntegration{
		loader:   loader,
		adapter:  adapter,
		registry: serviceRegistry,
		logger:   logger.WithComponent("plugin_integration"),
		metrics:  metricsCollector.WithPrefix("plugin_integration"),
	}
}

// InitializePlugins loads and registers all plugins with the gateway
func (mpi *MCpegPluginIntegration) InitializePlugins(ctx context.Context) error {
	mpi.logger.Info("initializing_mcpeg_plugins")

	// Get default plugin configurations
	configs := mpi.loader.GetDefaultPluginConfigs()

	// Load all built-in plugins
	if err := mpi.loader.LoadAllPlugins(ctx, configs); err != nil {
		mpi.logger.Error("failed_to_load_plugins", "error", err)
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	// Register plugins as services in the MCpeg registry
	if err := mpi.registerPluginsAsServices(); err != nil {
		mpi.logger.Error("failed_to_register_plugin_services", "error", err)
		return fmt.Errorf("failed to register plugin services: %w", err)
	}

	mpi.metrics.Inc("plugin_integration_initializations_total")
	mpi.logger.Info("mcpeg_plugins_initialized_successfully")

	return nil
}

// registerPluginsAsServices registers each plugin as a service in the MCpeg service registry
func (mpi *MCpegPluginIntegration) registerPluginsAsServices() error {
	services := mpi.loader.CreateRegisteredServices()

	for _, service := range services {
		// Update service timestamps
		service.RegisteredAt = time.Now()
		service.LastSeen = time.Now()

		// Create service registration request
		req := registry.ServiceRegistrationRequest{
			Name:        service.Name,
			Type:        service.Type,
			Version:     service.Version,
			Description: service.Description,
			Endpoint:    "plugin://internal",
			Protocol:    "plugin",
			Tools:       service.Tools,
			Resources:   service.Resources,
			Prompts:     service.Prompts,
			Metadata:    service.Metadata,
			Tags:        service.Tags,
		}

		// Register with the service registry
		if _, err := mpi.registry.RegisterService(context.Background(), req); err != nil {
			mpi.logger.Error("failed_to_register_plugin_service",
				"service_id", service.ID,
				"plugin", service.Name,
				"error", err)
			return fmt.Errorf("failed to register plugin service %s: %w", service.Name, err)
		}

		mpi.logger.Info("plugin_service_registered",
			"service_id", service.ID,
			"plugin", service.Name,
			"tools_count", len(service.Tools),
			"resources_count", len(service.Resources),
			"prompts_count", len(service.Prompts))
	}

	mpi.metrics.Set("registered_plugin_services_count", float64(len(services)))
	return nil
}

// HandlePluginToolCall handles tool calls for plugins
func (mpi *MCpegPluginIntegration) HandlePluginToolCall(ctx context.Context, toolName string, args []byte) (interface{}, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		mpi.metrics.Observe("plugin_tool_call_duration_ms", float64(duration.Milliseconds()), "tool", toolName)
	}()

	mpi.logger.Debug("handling_plugin_tool_call",
		"tool", toolName,
		"args_size", len(args))

	result, err := mpi.adapter.HandleToolCall(ctx, toolName, args)
	if err != nil {
		mpi.metrics.Inc("plugin_tool_call_errors_total", "tool", toolName)
		return nil, err
	}

	mpi.metrics.Inc("plugin_tool_call_successes_total", "tool", toolName)
	return result, nil
}

// HandlePluginResourceRequest handles resource requests for plugins
func (mpi *MCpegPluginIntegration) HandlePluginResourceRequest(ctx context.Context, pluginName, resourceURI string) (interface{}, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		mpi.metrics.Observe("plugin_resource_request_duration_ms", float64(duration.Milliseconds()),
			"plugin", pluginName, "resource", resourceURI)
	}()

	mpi.logger.Debug("handling_plugin_resource_request",
		"plugin", pluginName,
		"resource", resourceURI)

	result, err := mpi.adapter.HandleResourceRequest(ctx, pluginName, resourceURI)
	if err != nil {
		mpi.metrics.Inc("plugin_resource_request_errors_total", "plugin", pluginName, "resource", resourceURI)
		return nil, err
	}

	mpi.metrics.Inc("plugin_resource_request_successes_total", "plugin", pluginName, "resource", resourceURI)
	return result, nil
}

// HandlePluginPromptRequest handles prompt requests for plugins
func (mpi *MCpegPluginIntegration) HandlePluginPromptRequest(ctx context.Context, pluginName, promptName string, args []byte) (interface{}, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		mpi.metrics.Observe("plugin_prompt_request_duration_ms", float64(duration.Milliseconds()),
			"plugin", pluginName, "prompt", promptName)
	}()

	mpi.logger.Debug("handling_plugin_prompt_request",
		"plugin", pluginName,
		"prompt", promptName,
		"args_size", len(args))

	result, err := mpi.adapter.HandlePromptRequest(ctx, pluginName, promptName, args)
	if err != nil {
		mpi.metrics.Inc("plugin_prompt_request_errors_total", "plugin", pluginName, "prompt", promptName)
		return nil, err
	}

	mpi.metrics.Inc("plugin_prompt_request_successes_total", "plugin", pluginName, "prompt", promptName)
	return result, nil
}

// GetPluginInfo returns information about a specific plugin
func (mpi *MCpegPluginIntegration) GetPluginInfo(pluginName string) (map[string]interface{}, error) {
	return mpi.loader.GetPluginInfo(pluginName)
}

// GetAllPluginInfo returns information about all loaded plugins
func (mpi *MCpegPluginIntegration) GetAllPluginInfo() map[string]interface{} {
	return mpi.loader.GetAllPluginInfo()
}

// GetAllPluginTools returns all tools from all plugins
func (mpi *MCpegPluginIntegration) GetAllPluginTools() []registry.ToolDefinition {
	return mpi.loader.GetPluginManager().GetAllTools()
}

// GetAllPluginResources returns all resources from all plugins
func (mpi *MCpegPluginIntegration) GetAllPluginResources() []registry.ResourceDefinition {
	return mpi.loader.GetPluginManager().GetAllResources()
}

// HealthCheckPlugins checks the health of all plugins
func (mpi *MCpegPluginIntegration) HealthCheckPlugins(ctx context.Context) map[string]interface{} {
	results := mpi.loader.HealthCheckAllPlugins(ctx)

	healthStatus := make(map[string]interface{})
	healthyCount := 0
	totalCount := len(results)

	for pluginName, err := range results {
		if err != nil {
			healthStatus[pluginName] = map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			}
		} else {
			healthStatus[pluginName] = map[string]interface{}{
				"status": "healthy",
			}
			healthyCount++
		}
	}

	overallStatus := "healthy"
	if healthyCount < totalCount {
		if healthyCount == 0 {
			overallStatus = "unhealthy"
		} else {
			overallStatus = "degraded"
		}
	}

	return map[string]interface{}{
		"overall_status": overallStatus,
		"healthy_count":  healthyCount,
		"total_count":    totalCount,
		"plugins":        healthStatus,
	}
}

// GetPluginMetrics returns metrics for all plugins
func (mpi *MCpegPluginIntegration) GetPluginMetrics() map[string]interface{} {
	allStats := mpi.metrics.GetAllStats()
	pluginStats := make(map[string]interface{})

	// Filter metrics related to plugins
	for metricName, stats := range allStats {
		if len(metricName) > 6 && metricName[:6] == "plugin" {
			pluginStats[metricName] = map[string]interface{}{
				"count":      stats.Count,
				"sum":        stats.Sum,
				"average":    stats.Average,
				"last_value": stats.LastValue,
				"trend":      stats.Trend,
			}
		}
	}

	return map[string]interface{}{
		"plugin_metrics": pluginStats,
		"total_metrics":  len(pluginStats),
	}
}

// ShutdownPlugins shuts down all plugins
func (mpi *MCpegPluginIntegration) ShutdownPlugins(ctx context.Context) error {
	mpi.logger.Info("shutting_down_mcpeg_plugins")

	if err := mpi.loader.ShutdownAllPlugins(ctx); err != nil {
		mpi.logger.Error("failed_to_shutdown_plugins", "error", err)
		return err
	}

	mpi.metrics.Inc("plugin_integration_shutdowns_total")
	mpi.logger.Info("mcpeg_plugins_shutdown_successfully")
	return nil
}

// UpdatePluginConfiguration updates the configuration for a specific plugin
func (mpi *MCpegPluginIntegration) UpdatePluginConfiguration(ctx context.Context, pluginName string, config map[string]interface{}) error {
	// This would typically involve reinitializing the plugin with new config
	// For now, we'll log the request and return success
	mpi.logger.Info("plugin_configuration_update_requested",
		"plugin", pluginName,
		"config_keys", len(config))

	// In a full implementation, this would:
	// 1. Validate the new configuration
	// 2. Stop the current plugin instance
	// 3. Reinitialize with new configuration
	// 4. Update the service registry

	mpi.metrics.Inc("plugin_configuration_updates_total", "plugin", pluginName)
	return fmt.Errorf("plugin configuration updates not yet implemented")
}

// GetPluginConfiguration returns the current configuration for a plugin
func (mpi *MCpegPluginIntegration) GetPluginConfiguration(pluginName string) (map[string]interface{}, error) {
	configs := mpi.loader.GetDefaultPluginConfigs()

	if config, exists := configs[pluginName]; exists {
		return map[string]interface{}{
			"plugin":        pluginName,
			"configuration": config.Config,
		}, nil
	}

	return nil, fmt.Errorf("plugin %s not found", pluginName)
}

// ListPluginCapabilities returns a summary of all plugin capabilities
func (mpi *MCpegPluginIntegration) ListPluginCapabilities() map[string]interface{} {
	allInfo := mpi.loader.GetAllPluginInfo()

	totalTools := 0
	totalResources := 0
	totalPrompts := 0

	capabilities := make(map[string]interface{})

	if pluginsInfo, ok := allInfo["plugins"].(map[string]interface{}); ok {
		for pluginName, info := range pluginsInfo {
			if pluginInfo, ok := info.(map[string]interface{}); ok {
				if caps, ok := pluginInfo["capabilities"].(map[string]interface{}); ok {
					if toolsCount, ok := caps["tools_count"].(int); ok {
						totalTools += toolsCount
					}
					if resourcesCount, ok := caps["resources_count"].(int); ok {
						totalResources += resourcesCount
					}
					if promptsCount, ok := caps["prompts_count"].(int); ok {
						totalPrompts += promptsCount
					}
				}

				capabilities[pluginName] = pluginInfo
			}
		}
	}

	return map[string]interface{}{
		"summary": map[string]interface{}{
			"total_plugins":   allInfo["total_plugins"],
			"total_tools":     totalTools,
			"total_resources": totalResources,
			"total_prompts":   totalPrompts,
		},
		"plugins": capabilities,
	}
}

// GetPluginManager returns the underlying plugin manager
func (mpi *MCpegPluginIntegration) GetPluginManager() *plugins.PluginManager {
	return mpi.loader.GetPluginManager()
}
