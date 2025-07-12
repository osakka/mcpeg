package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/osakka/mcpeg/internal/registry"
	"github.com/osakka/mcpeg/pkg/paths"
)

// MemoryService provides persistent key-value storage across sessions
type MemoryService struct {
	*BasePlugin
	storage    map[string]interface{}
	storageMux sync.RWMutex
	dataFile   string
	autoSave   bool
}

// NewMemoryService creates a new memory service plugin
func NewMemoryService() *MemoryService {
	return &MemoryService{
		BasePlugin: NewBasePlugin(
			"memory",
			"1.0.0",
			"Persistent key-value storage service for maintaining context across sessions",
		),
		storage:  make(map[string]interface{}),
		autoSave: true,
	}
}

// Initialize initializes the memory service
func (ms *MemoryService) Initialize(ctx context.Context, config PluginConfig) error {
	if err := ms.BasePlugin.Initialize(ctx, config); err != nil {
		return err
	}
	
	// Set up data file path using centralized path configuration
	pathConfig := paths.DefaultPaths()
	dataDir := pathConfig.GetDataDir()
	if configDataDir, ok := config.Config["data_dir"].(string); ok {
		dataDir = configDataDir
	}
	
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}
	
	ms.dataFile = filepath.Join(dataDir, "memory_storage.json")
	
	// Load existing data
	if err := ms.loadData(); err != nil {
		ms.logger.Warn("failed_to_load_existing_data", "error", err)
		// Continue with empty storage
	}
	
	ms.logger.Info("memory_service_initialized",
		"data_file", ms.dataFile,
		"existing_keys", len(ms.storage))
	
	return nil
}

// GetTools returns the tools provided by the memory service
func (ms *MemoryService) GetTools() []registry.ToolDefinition {
	return []registry.ToolDefinition{
		{
			Name:        "memory_store",
			Description: "Store a value in memory with a given key",
			Category:    "memory",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"key": map[string]interface{}{
						"type":        "string",
						"description": "The key to store the value under",
					},
					"value": map[string]interface{}{
						"description": "The value to store (can be any JSON-serializable data)",
					},
					"ttl": map[string]interface{}{
						"type":        "integer",
						"description": "Optional time-to-live in seconds",
						"minimum":     1,
					},
				},
				"required": []string{"key", "value"},
			},
			Examples: []registry.ToolExample{
				{
					Description: "Store a simple string value",
					Input: map[string]interface{}{
						"key":   "user_name",
						"value": "John Doe",
					},
				},
				{
					Description: "Store complex data with TTL",
					Input: map[string]interface{}{
						"key": "session_data",
						"value": map[string]interface{}{
							"user_id": 123,
							"preferences": map[string]interface{}{
								"theme": "dark",
								"language": "en",
							},
						},
						"ttl": 3600,
					},
				},
			},
		},
		{
			Name:        "memory_retrieve",
			Description: "Retrieve a value from memory by key",
			Category:    "memory",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"key": map[string]interface{}{
						"type":        "string",
						"description": "The key to retrieve the value for",
					},
				},
				"required": []string{"key"},
			},
			Examples: []registry.ToolExample{
				{
					Description: "Retrieve a stored value",
					Input: map[string]interface{}{
						"key": "user_name",
					},
				},
			},
		},
		{
			Name:        "memory_list",
			Description: "List all keys in memory or search by pattern",
			Category:    "memory",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "Optional pattern to filter keys (supports wildcards)",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of keys to return",
						"minimum":     1,
						"maximum":     1000,
						"default":     100,
					},
				},
			},
			Examples: []registry.ToolExample{
				{
					Description: "List all keys",
					Input:       map[string]interface{}{},
				},
				{
					Description: "Search for keys matching pattern",
					Input: map[string]interface{}{
						"pattern": "user_*",
						"limit":   50,
					},
				},
			},
		},
		{
			Name:        "memory_delete",
			Description: "Delete a value from memory by key",
			Category:    "memory",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"key": map[string]interface{}{
						"type":        "string",
						"description": "The key to delete",
					},
				},
				"required": []string{"key"},
			},
			Examples: []registry.ToolExample{
				{
					Description: "Delete a stored value",
					Input: map[string]interface{}{
						"key": "user_name",
					},
				},
			},
		},
		{
			Name:        "memory_clear",
			Description: "Clear all data from memory",
			Category:    "memory",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"confirm": map[string]interface{}{
						"type":        "boolean",
						"description": "Must be true to confirm clearing all data",
					},
				},
				"required": []string{"confirm"},
			},
			Examples: []registry.ToolExample{
				{
					Description: "Clear all memory data",
					Input: map[string]interface{}{
						"confirm": true,
					},
				},
			},
		},
	}
}

// GetResources returns resources provided by the memory service
func (ms *MemoryService) GetResources() []registry.ResourceDefinition {
	return []registry.ResourceDefinition{
		{
			Name:        "memory_stats",
			Type:        "application/json",
			Description: "Memory service statistics and usage information",
		},
		{
			Name:        "memory_dump",
			Type:        "application/json", 
			Description: "Complete memory dump (for debugging)",
		},
	}
}

// GetPrompts returns prompts provided by the memory service
func (ms *MemoryService) GetPrompts() []registry.PromptDefinition {
	return []registry.PromptDefinition{
		{
			Name:        "memory_search",
			Description: "Search memory for relevant information",
			Category:    "search",
		},
		{
			Name:        "memory_context",
			Description: "Get contextual information from memory",
			Category:    "context",
		},
	}
}

// CallTool executes a memory service tool
func (ms *MemoryService) CallTool(ctx context.Context, name string, args json.RawMessage) (interface{}, error) {
	start := time.Now()
	defer func() {
		ms.LogToolCall(name, time.Since(start), nil)
	}()
	
	switch name {
	case "memory_store":
		return ms.handleStore(args)
	case "memory_retrieve":
		return ms.handleRetrieve(args)
	case "memory_list":
		return ms.handleList(args)
	case "memory_delete":
		return ms.handleDelete(args)
	case "memory_clear":
		return ms.handleClear(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// ReadResource reads a memory service resource
func (ms *MemoryService) ReadResource(ctx context.Context, uri string) (interface{}, error) {
	start := time.Now()
	defer func() {
		ms.LogResourceAccess(uri, time.Since(start), nil)
	}()
	
	switch uri {
	case "memory_stats":
		return ms.getStats(), nil
	case "memory_dump":
		return ms.getDump(), nil
	default:
		return nil, fmt.Errorf("unknown resource: %s", uri)
	}
}

// ListResources lists available resources
func (ms *MemoryService) ListResources(ctx context.Context) ([]registry.ResourceDefinition, error) {
	return ms.GetResources(), nil
}

// GetPrompt returns a prompt
func (ms *MemoryService) GetPrompt(ctx context.Context, name string, args json.RawMessage) (interface{}, error) {
	switch name {
	case "memory_search":
		return ms.handleSearchPrompt(args)
	case "memory_context":
		return ms.handleContextPrompt(args)
	default:
		return nil, fmt.Errorf("unknown prompt: %s", name)
	}
}

// Tool handlers

func (ms *MemoryService) handleStore(args json.RawMessage) (interface{}, error) {
	var req struct {
		Key   string      `json:"key"`
		Value interface{} `json:"value"`
		TTL   *int        `json:"ttl,omitempty"`
	}
	
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	
	if req.Key == "" {
		return nil, fmt.Errorf("key cannot be empty")
	}
	
	ms.storageMux.Lock()
	defer ms.storageMux.Unlock()
	
	// Store the value
	ms.storage[req.Key] = req.Value
	
	// Handle TTL if specified
	if req.TTL != nil && *req.TTL > 0 {
		go ms.scheduleExpiration(req.Key, time.Duration(*req.TTL)*time.Second)
	}
	
	// Save to disk if auto-save is enabled
	if ms.autoSave {
		if err := ms.saveData(); err != nil {
			ms.logger.Warn("failed_to_save_data", "error", err)
		}
	}
	
	ms.metrics.Inc("memory_store_operations_total")
	ms.metrics.Set("memory_keys_count", float64(len(ms.storage)))
	
	return map[string]interface{}{
		"success": true,
		"key":     req.Key,
		"message": "Value stored successfully",
	}, nil
}

func (ms *MemoryService) handleRetrieve(args json.RawMessage) (interface{}, error) {
	var req struct {
		Key string `json:"key"`
	}
	
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	
	ms.storageMux.RLock()
	defer ms.storageMux.RUnlock()
	
	value, exists := ms.storage[req.Key]
	if !exists {
		return nil, fmt.Errorf("key '%s' not found", req.Key)
	}
	
	ms.metrics.Inc("memory_retrieve_operations_total")
	
	return map[string]interface{}{
		"key":   req.Key,
		"value": value,
	}, nil
}

func (ms *MemoryService) handleList(args json.RawMessage) (interface{}, error) {
	var req struct {
		Pattern *string `json:"pattern,omitempty"`
		Limit   *int    `json:"limit,omitempty"`
	}
	
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	
	limit := 100
	if req.Limit != nil {
		limit = *req.Limit
	}
	
	ms.storageMux.RLock()
	defer ms.storageMux.RUnlock()
	
	var keys []string
	for key := range ms.storage {
		if req.Pattern == nil || ms.matchesPattern(key, *req.Pattern) {
			keys = append(keys, key)
			if len(keys) >= limit {
				break
			}
		}
	}
	
	ms.metrics.Inc("memory_list_operations_total")
	
	return map[string]interface{}{
		"keys":  keys,
		"count": len(keys),
		"total": len(ms.storage),
	}, nil
}

func (ms *MemoryService) handleDelete(args json.RawMessage) (interface{}, error) {
	var req struct {
		Key string `json:"key"`
	}
	
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	
	ms.storageMux.Lock()
	defer ms.storageMux.Unlock()
	
	_, exists := ms.storage[req.Key]
	if !exists {
		return nil, fmt.Errorf("key '%s' not found", req.Key)
	}
	
	delete(ms.storage, req.Key)
	
	// Save to disk if auto-save is enabled
	if ms.autoSave {
		if err := ms.saveData(); err != nil {
			ms.logger.Warn("failed_to_save_data", "error", err)
		}
	}
	
	ms.metrics.Inc("memory_delete_operations_total")
	ms.metrics.Set("memory_keys_count", float64(len(ms.storage)))
	
	return map[string]interface{}{
		"success": true,
		"key":     req.Key,
		"message": "Key deleted successfully",
	}, nil
}

func (ms *MemoryService) handleClear(args json.RawMessage) (interface{}, error) {
	var req struct {
		Confirm bool `json:"confirm"`
	}
	
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	
	if !req.Confirm {
		return nil, fmt.Errorf("must confirm to clear all data")
	}
	
	ms.storageMux.Lock()
	defer ms.storageMux.Unlock()
	
	oldCount := len(ms.storage)
	ms.storage = make(map[string]interface{})
	
	// Save to disk if auto-save is enabled
	if ms.autoSave {
		if err := ms.saveData(); err != nil {
			ms.logger.Warn("failed_to_save_data", "error", err)
		}
	}
	
	ms.metrics.Inc("memory_clear_operations_total")
	ms.metrics.Set("memory_keys_count", 0)
	
	return map[string]interface{}{
		"success":     true,
		"keys_cleared": oldCount,
		"message":     fmt.Sprintf("Cleared %d keys from memory", oldCount),
	}, nil
}

// Helper methods

func (ms *MemoryService) scheduleExpiration(key string, ttl time.Duration) {
	time.Sleep(ttl)
	
	ms.storageMux.Lock()
	defer ms.storageMux.Unlock()
	
	delete(ms.storage, key)
	
	ms.logger.Debug("memory_key_expired", "key", key)
	
	if ms.autoSave {
		if err := ms.saveData(); err != nil {
			ms.logger.Warn("failed_to_save_data_after_expiration", "error", err)
		}
	}
}

func (ms *MemoryService) matchesPattern(str, pattern string) bool {
	// Simple wildcard matching - could be enhanced with regex
	if pattern == "*" {
		return true
	}
	
	// For now, just check if pattern is contained in string
	// In production, you'd want proper wildcard/regex matching
	return len(str) >= len(pattern) && 
		(str == pattern || 
		 (len(pattern) > 0 && pattern[len(pattern)-1] == '*' && 
		  len(str) >= len(pattern)-1 && 
		  str[:len(pattern)-1] == pattern[:len(pattern)-1]))
}

func (ms *MemoryService) getStats() map[string]interface{} {
	ms.storageMux.RLock()
	defer ms.storageMux.RUnlock()
	
	return map[string]interface{}{
		"total_keys":   len(ms.storage),
		"data_file":    ms.dataFile,
		"auto_save":    ms.autoSave,
		"plugin_name":  ms.Name(),
		"plugin_version": ms.Version(),
	}
}

func (ms *MemoryService) getDump() map[string]interface{} {
	ms.storageMux.RLock()
	defer ms.storageMux.RUnlock()
	
	// Create a copy to avoid race conditions
	dump := make(map[string]interface{})
	for key, value := range ms.storage {
		dump[key] = value
	}
	
	return dump
}

func (ms *MemoryService) handleSearchPrompt(args json.RawMessage) (interface{}, error) {
	// Implementation for search prompt
	return map[string]interface{}{
		"prompt": "Search memory for relevant information",
		"context": "Use memory_list to find relevant keys, then memory_retrieve to get values",
	}, nil
}

func (ms *MemoryService) handleContextPrompt(args json.RawMessage) (interface{}, error) {
	// Implementation for context prompt
	return map[string]interface{}{
		"prompt": "Get contextual information from memory",
		"context": "Use memory_retrieve to get specific context or memory_list to browse available context",
	}, nil
}

// Data persistence

func (ms *MemoryService) saveData() error {
	data, err := json.MarshalIndent(ms.storage, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	
	if err := os.WriteFile(ms.dataFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write data file: %w", err)
	}
	
	return nil
}

func (ms *MemoryService) loadData() error {
	data, err := os.ReadFile(ms.dataFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, start with empty storage
		}
		return fmt.Errorf("failed to read data file: %w", err)
	}
	
	if err := json.Unmarshal(data, &ms.storage); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}
	
	return nil
}

// Shutdown saves data and cleans up
func (ms *MemoryService) Shutdown(ctx context.Context) error {
	// Save data before shutdown
	ms.storageMux.Lock()
	defer ms.storageMux.Unlock()
	
	if err := ms.saveData(); err != nil {
		ms.logger.Error("failed_to_save_data_on_shutdown", "error", err)
	}
	
	return ms.BasePlugin.Shutdown(ctx)
}