package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/osakka/mcpeg/internal/registry"
	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
	"github.com/osakka/mcpeg/pkg/plugins"
	"github.com/osakka/mcpeg/pkg/rbac"
)

// PluginHandlerImpl implements the PluginHandler interface
type PluginHandlerImpl struct {
	pluginManager *plugins.PluginManager
	logger        logging.Logger
	metrics       metrics.Metrics
	config        PluginHandlerConfig
}

// PluginHandlerConfig configures the plugin handler
type PluginHandlerConfig struct {
	DefaultTimeout time.Duration `yaml:"default_timeout"`
	MaxRetries     int           `yaml:"max_retries"`
	RetryBackoff   time.Duration `yaml:"retry_backoff"`
	CacheEnabled   bool          `yaml:"cache_enabled"`
	CacheTTL       time.Duration `yaml:"cache_ttl"`
}

// NewPluginHandler creates a new plugin handler instance
func NewPluginHandler(pluginManager *plugins.PluginManager, config PluginHandlerConfig, logger logging.Logger, metrics metrics.Metrics) *PluginHandlerImpl {
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryBackoff == 0 {
		config.RetryBackoff = time.Second
	}
	if config.CacheTTL == 0 {
		config.CacheTTL = 5 * time.Minute
	}

	return &PluginHandlerImpl{
		pluginManager: pluginManager,
		logger:        logger,
		metrics:       metrics,
		config:        config,
	}
}

// InvokePlugin executes a plugin tool with the given parameters
func (ph *PluginHandlerImpl) InvokePlugin(ctx context.Context, pluginName, toolName string, params map[string]interface{}, capabilities *rbac.ProcessedCapabilities) (*ToolResult, error) {
	timer := ph.metrics.Time("plugin_invocation_duration", "plugin", pluginName, "tool", toolName)
	defer timer.Stop()

	// Check plugin access permissions
	if !ph.hasPluginAccess(pluginName, capabilities) {
		ph.metrics.Inc("plugin_access_denied", "plugin", pluginName, "user", capabilities.UserID)
		ph.logger.Warn("plugin_access_denied",
			"plugin", pluginName,
			"user", capabilities.UserID,
			"roles", capabilities.Roles)
		return nil, fmt.Errorf("access denied to plugin: %s", pluginName)
	}

	// Get plugin instance
	plugin, exists := ph.pluginManager.GetPlugin(pluginName)
	if !exists {
		ph.metrics.Inc("plugin_not_found", "plugin", pluginName)
		return nil, fmt.Errorf("plugin not found: %s", pluginName)
	}

	// Validate tool exists and check permissions
	if !ph.hasToolAccess(pluginName, toolName, capabilities) {
		ph.metrics.Inc("tool_access_denied", "plugin", pluginName, "tool", toolName)
		ph.logger.Warn("tool_access_denied",
			"plugin", pluginName,
			"tool", toolName,
			"user", capabilities.UserID)
		return nil, fmt.Errorf("access denied to tool: %s.%s", pluginName, toolName)
	}

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, ph.config.DefaultTimeout)
	defer cancel()

	// Execute plugin tool with metrics and logging
	ph.metrics.Inc("plugin_tool_calls", "plugin", pluginName, "tool", toolName)
	ph.logger.Info("plugin_tool_invocation_started",
		"plugin", pluginName,
		"tool", toolName,
		"user", capabilities.UserID,
		"session", capabilities.SessionID)

	result, err := ph.executeWithRetry(timeoutCtx, plugin, toolName, params)
	if err != nil {
		ph.metrics.Inc("plugin_tool_errors", "plugin", pluginName, "tool", toolName)
		ph.logger.Error("plugin_tool_execution_failed",
			"plugin", pluginName,
			"tool", toolName,
			"user", capabilities.UserID,
			"error", err)

		return &ToolResult{
			Content: []Content{
				TextContent{
					Type: "text",
					Text: fmt.Sprintf("Tool execution failed: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	ph.metrics.Inc("plugin_tool_success", "plugin", pluginName, "tool", toolName)
	ph.logger.Info("plugin_tool_execution_completed",
		"plugin", pluginName,
		"tool", toolName,
		"user", capabilities.UserID)

	// Convert plugin result to MCP tool result
	return ph.convertToToolResult(result), nil
}

// GetPluginCapabilities returns the capabilities of a specific plugin
func (ph *PluginHandlerImpl) GetPluginCapabilities(pluginName string, capabilities *rbac.ProcessedCapabilities) (*PluginCapabilities, error) {
	if !ph.hasPluginAccess(pluginName, capabilities) {
		return nil, fmt.Errorf("access denied to plugin: %s", pluginName)
	}

	plugin, exists := ph.pluginManager.GetPlugin(pluginName)
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", pluginName)
	}

	// Get filtered tools, resources, and prompts
	tools, _ := ph.GetPluginTools(pluginName, capabilities)
	resources, _ := ph.GetPluginResources(pluginName, capabilities)
	prompts, _ := ph.GetPluginPrompts(pluginName, capabilities)

	// Get user's permissions for this plugin
	permissions := capabilities.Plugins[pluginName]
	if wildcard, hasWildcard := capabilities.Plugins["*"]; hasWildcard && len(capabilities.Plugins) == 1 {
		permissions = wildcard
	}

	return &PluginCapabilities{
		Name:        plugin.Name(),
		Version:     plugin.Version(),
		Description: plugin.Description(),
		Tools:       tools,
		Resources:   resources,
		Prompts:     prompts,
		Permissions: permissions,
	}, nil
}

// ListAvailablePlugins returns a list of plugins the user has access to
func (ph *PluginHandlerImpl) ListAvailablePlugins(capabilities *rbac.ProcessedCapabilities) []string {
	allPlugins := ph.pluginManager.ListPlugins()
	accessible := make([]string, 0, len(allPlugins))

	for pluginName := range allPlugins {
		if ph.hasPluginAccess(pluginName, capabilities) {
			accessible = append(accessible, pluginName)
		}
	}

	ph.logger.Debug("plugin_access_list_generated",
		"user", capabilities.UserID,
		"total_plugins", len(allPlugins),
		"accessible_plugins", len(accessible))

	return accessible
}

// GetPluginTools returns the tools available for a plugin
func (ph *PluginHandlerImpl) GetPluginTools(pluginName string, capabilities *rbac.ProcessedCapabilities) ([]Tool, error) {
	if !ph.hasPluginAccess(pluginName, capabilities) {
		return nil, fmt.Errorf("access denied to plugin: %s", pluginName)
	}

	plugin, exists := ph.pluginManager.GetPlugin(pluginName)
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", pluginName)
	}

	pluginTools := plugin.GetTools()
	mcpTools := make([]Tool, 0, len(pluginTools))

	for _, tool := range pluginTools {
		if ph.isToolAllowedForPermission(tool, capabilities.Plugins[pluginName]) {
			mcpTool := Tool{
				Name:        fmt.Sprintf("%s.%s", pluginName, tool.Name),
				Description: fmt.Sprintf("[%s] %s", pluginName, tool.Description),
				InputSchema: tool.InputSchema,
			}

			mcpTools = append(mcpTools, mcpTool)
		}
	}

	return mcpTools, nil
}

// GetPluginResources returns the resources available for a plugin
func (ph *PluginHandlerImpl) GetPluginResources(pluginName string, capabilities *rbac.ProcessedCapabilities) ([]Resource, error) {
	if !ph.hasPluginAccess(pluginName, capabilities) {
		return nil, fmt.Errorf("access denied to plugin: %s", pluginName)
	}

	plugin, exists := ph.pluginManager.GetPlugin(pluginName)
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", pluginName)
	}

	pluginResources := plugin.GetResources()
	mcpResources := make([]Resource, 0, len(pluginResources))

	permission := capabilities.Plugins[pluginName]
	if !permission.CanRead {
		return mcpResources, nil // No read access, return empty
	}

	for _, resource := range pluginResources {
		mcpResource := Resource{
			URI:         fmt.Sprintf("plugin://%s/%s", pluginName, resource.URI),
			Name:        resource.Name,
			Description: resource.Description,
			MimeType:    resource.MimeType,
		}
		mcpResources = append(mcpResources, mcpResource)
	}

	return mcpResources, nil
}

// GetPluginPrompts returns the prompts available for a plugin
func (ph *PluginHandlerImpl) GetPluginPrompts(pluginName string, capabilities *rbac.ProcessedCapabilities) ([]Prompt, error) {
	if !ph.hasPluginAccess(pluginName, capabilities) {
		return nil, fmt.Errorf("access denied to plugin: %s", pluginName)
	}

	plugin, exists := ph.pluginManager.GetPlugin(pluginName)
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", pluginName)
	}

	pluginPrompts := plugin.GetPrompts()
	mcpPrompts := make([]Prompt, 0, len(pluginPrompts))

	permission := capabilities.Plugins[pluginName]
	if !permission.CanRead {
		return mcpPrompts, nil // No read access, return empty
	}

	for _, prompt := range pluginPrompts {
		mcpPrompt := Prompt{
			Name:        fmt.Sprintf("%s.%s", pluginName, prompt.Name),
			Description: prompt.Description,
		}

		mcpPrompts = append(mcpPrompts, mcpPrompt)
	}

	return mcpPrompts, nil
}

// HealthCheck checks if a plugin is healthy and accessible
func (ph *PluginHandlerImpl) HealthCheck(pluginName string) (*PluginHealth, error) {
	plugin, exists := ph.pluginManager.GetPlugin(pluginName)
	if !exists {
		return &PluginHealth{
			Name:      pluginName,
			Healthy:   false,
			Status:    "not_found",
			LastCheck: time.Now(),
			Error:     "Plugin not found",
		}, nil
	}

	// Perform health check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := plugin.HealthCheck(ctx)
	health := &PluginHealth{
		Name:      pluginName,
		Healthy:   err == nil,
		LastCheck: time.Now(),
	}

	if err != nil {
		health.Status = "unhealthy"
		health.Error = err.Error()
	} else {
		health.Status = "healthy"
	}

	return health, nil
}

// Helper methods

func (ph *PluginHandlerImpl) hasPluginAccess(pluginName string, capabilities *rbac.ProcessedCapabilities) bool {
	return capabilities.HasPermission(pluginName, "execute")
}

func (ph *PluginHandlerImpl) hasToolAccess(pluginName, toolName string, capabilities *rbac.ProcessedCapabilities) bool {
	// Check if user has execute permission on the plugin
	if !capabilities.HasPermission(pluginName, "execute") {
		return false
	}

	// Additional tool-level permission checks based on tool type
	permission := capabilities.Plugins[pluginName]
	if wildcardPerm, hasWildcard := capabilities.Plugins["*"]; hasWildcard && len(capabilities.Plugins) == 1 {
		permission = wildcardPerm
	}

	// Determine required permission based on tool name patterns
	if ph.isDestructiveTool(toolName) {
		return permission.CanWrite && permission.CanAdmin
	} else if ph.isWriteTool(toolName) {
		return permission.CanWrite
	} else {
		return permission.CanRead
	}
}

func (ph *PluginHandlerImpl) isDestructiveTool(toolName string) bool {
	destructivePatterns := []string{"delete", "clear", "remove", "drop", "destroy"}
	toolLower := strings.ToLower(toolName)
	for _, pattern := range destructivePatterns {
		if strings.Contains(toolLower, pattern) {
			return true
		}
	}
	return false
}

func (ph *PluginHandlerImpl) isWriteTool(toolName string) bool {
	writePatterns := []string{"store", "save", "create", "update", "set", "put", "post", "write"}
	toolLower := strings.ToLower(toolName)
	for _, pattern := range writePatterns {
		if strings.Contains(toolLower, pattern) {
			return true
		}
	}
	return false
}

func (ph *PluginHandlerImpl) isToolAllowedForPermission(tool registry.ToolDefinition, permission rbac.PluginPermission) bool {
	toolName := strings.ToLower(tool.Name)

	if ph.isDestructiveTool(toolName) {
		return permission.CanWrite && permission.CanAdmin
	} else if ph.isWriteTool(toolName) {
		return permission.CanWrite
	} else {
		return permission.CanRead
	}
}

func (ph *PluginHandlerImpl) executeWithRetry(ctx context.Context, plugin plugins.Plugin, toolName string, params map[string]interface{}) (interface{}, error) {
	var lastErr error

	// Convert params to JSON
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	for attempt := 0; attempt <= ph.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Apply backoff delay
			select {
			case <-time.After(ph.config.RetryBackoff * time.Duration(attempt)):
			case <-ctx.Done():
				return nil, ctx.Err()
			}

			ph.logger.Debug("plugin_tool_retry_attempt",
				"tool", toolName,
				"attempt", attempt,
				"max_retries", ph.config.MaxRetries)
		}

		result, err := plugin.CallTool(ctx, toolName, paramsJSON)
		if err == nil {
			if attempt > 0 {
				ph.metrics.Inc("plugin_tool_retry_success", "tool", toolName, "attempts", fmt.Sprintf("%d", attempt))
			}
			return result, nil
		}

		lastErr = err
		ph.metrics.Inc("plugin_tool_retry", "tool", toolName, "attempt", fmt.Sprintf("%d", attempt))

		// Don't retry on context cancellation or permission errors
		if ctx.Err() != nil || strings.Contains(err.Error(), "access denied") {
			break
		}
	}

	ph.metrics.Inc("plugin_tool_retry_exhausted", "tool", toolName)
	return nil, fmt.Errorf("tool execution failed after %d retries: %w", ph.config.MaxRetries, lastErr)
}

func (ph *PluginHandlerImpl) convertToToolResult(result interface{}) *ToolResult {
	switch v := result.(type) {
	case string:
		return &ToolResult{
			Content: []Content{
				TextContent{
					Type: "text",
					Text: v,
				},
			},
			IsError: false,
		}
	case map[string]interface{}:
		// Handle structured results
		if text, ok := v["text"].(string); ok {
			return &ToolResult{
				Content: []Content{
					TextContent{
						Type: "text",
						Text: text,
					},
				},
				IsError: false,
			}
		}
		// Convert to JSON string for complex types
		return &ToolResult{
			Content: []Content{
				TextContent{
					Type: "text",
					Text: fmt.Sprintf("%+v", v),
				},
			},
			IsError: false,
		}
	default:
		// Convert to JSON string for complex types
		return &ToolResult{
			Content: []Content{
				TextContent{
					Type: "text",
					Text: fmt.Sprintf("%+v", result),
				},
			},
			IsError: false,
		}
	}
}
