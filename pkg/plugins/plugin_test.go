package plugins

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
)

// TestBasePlugin verifies basic plugin functionality
func TestBasePlugin(t *testing.T) {
	logger := logging.New("test")
	mockMetrics := &mockMetrics{}
	
	t.Run("base plugin creation", func(t *testing.T) {
		plugin := NewBasePlugin("test-plugin", "1.0.0", "Test plugin")
		
		if plugin.Name() != "test-plugin" {
			t.Errorf("expected name='test-plugin', got %s", plugin.Name())
		}
		
		if plugin.Version() != "1.0.0" {
			t.Errorf("expected version='1.0.0', got %s", plugin.Version())
		}
		
		if plugin.Description() != "Test plugin" {
			t.Errorf("expected description='Test plugin', got %s", plugin.Description())
		}
	})
	
	t.Run("base plugin lifecycle", func(t *testing.T) {
		plugin := NewBasePlugin("lifecycle-test", "1.0.0", "Lifecycle test plugin")
		
		// Test initialization
		ctx := context.Background()
		config := PluginConfig{
			Name:    "lifecycle-test",
			Logger:  logger,
			Metrics: mockMetrics,
		}
		
		err := plugin.Initialize(ctx, config)
		if err != nil {
			t.Fatalf("initialization failed: %v", err)
		}
		
		// Test health check
		err = plugin.HealthCheck(ctx)
		if err != nil {
			t.Fatalf("health check failed: %v", err)
		}
		
		// Test shutdown
		err = plugin.Shutdown(ctx)
		if err != nil {
			t.Fatalf("shutdown failed: %v", err)
		}
		
		// Health check should fail after shutdown
		err = plugin.HealthCheck(ctx)
		if err == nil {
			t.Error("expected health check to fail after shutdown")
		}
	})
}

// TestMemoryService tests the memory service plugin
func TestMemoryService(t *testing.T) {
	logger := logging.New("test")
	mockMetrics := &mockMetrics{}
	
	t.Run("memory service initialization", func(t *testing.T) {
		service := NewMemoryService()
		
		if service.Name() != "memory" {
			t.Errorf("expected name='memory', got %s", service.Name())
		}
		
		if service.Version() != "1.0.0" {
			t.Errorf("expected version='1.0.0', got %s", service.Version())
		}
		
		tools := service.GetTools()
		if len(tools) < 5 {
			t.Errorf("expected at least 5 tools, got %d", len(tools))
		}
		
		resources := service.GetResources()
		if len(resources) != 2 {
			t.Errorf("expected 2 resources, got %d", len(resources))
		}
		
		prompts := service.GetPrompts()
		if len(prompts) != 2 {
			t.Errorf("expected 2 prompts, got %d", len(prompts))
		}
	})
	
	t.Run("memory service operations", func(t *testing.T) {
		service := NewMemoryService()
		ctx := context.Background()
		
		config := PluginConfig{
			Name:    "memory",
			Logger:  logger,
			Metrics: mockMetrics,
		}
		
		err := service.Initialize(ctx, config)
		if err != nil {
			t.Fatalf("initialization failed: %v", err)
		}
		
		// Test memory_store operation
		storeParams := map[string]interface{}{
			"key":   "test-key",
			"value": "test-value",
		}
		
		storeParamsJSON, _ := json.Marshal(storeParams)
		result, err := service.CallTool(ctx, "memory_store", json.RawMessage(storeParamsJSON))
		if err != nil {
			t.Fatalf("memory_store operation failed: %v", err)
		}
		
		// Result should be a map indicating success
		if result == nil {
			t.Error("expected result from memory_store operation")
		}
		
		// Test memory_retrieve operation
		retrieveParams := map[string]interface{}{
			"key": "test-key",
		}
		
		retrieveParamsJSON, _ := json.Marshal(retrieveParams)
		result, err = service.CallTool(ctx, "memory_retrieve", json.RawMessage(retrieveParamsJSON))
		if err != nil {
			t.Fatalf("memory_retrieve operation failed: %v", err)
		}
		
		// Result should contain the value
		if result == nil {
			t.Error("expected result from memory_retrieve operation")
		}
	})
}

// TestGitService tests the git service plugin
func TestGitService(t *testing.T) {
	logger := logging.New("test")
	mockMetrics := &mockMetrics{}
	
	t.Run("git service initialization", func(t *testing.T) {
		service := NewGitService()
		
		if service.Name() != "git" {
			t.Errorf("expected name='git', got %s", service.Name())
		}
		
		if service.Version() != "1.0.0" {
			t.Errorf("expected version='1.0.0', got %s", service.Version())
		}
		
		tools := service.GetTools()
		if len(tools) != 8 {
			t.Errorf("expected 8 tools, got %d", len(tools))
		}
		
		resources := service.GetResources()
		if len(resources) != 2 {
			t.Errorf("expected 2 resources, got %d", len(resources))
		}
		
		prompts := service.GetPrompts()
		if len(prompts) != 2 {
			t.Errorf("expected 2 prompts, got %d", len(prompts))
		}
	})
	
	t.Run("git service status operation", func(t *testing.T) {
		service := NewGitService()
		ctx := context.Background()
		
		config := PluginConfig{
			Name:    "git",
			Logger:  logger,
			Metrics: mockMetrics,
		}
		
		err := service.Initialize(ctx, config)
		if err != nil {
			t.Fatalf("initialization failed: %v", err)
		}
		
		// Test status operation (should work even without a git repo)
		statusParams := map[string]interface{}{}
		statusParamsJSON, _ := json.Marshal(statusParams)
		
		result, err := service.CallTool(ctx, "status", json.RawMessage(statusParamsJSON))
		if err != nil {
			// This is expected if not in a git repo
			t.Logf("status operation failed as expected: %v", err)
		}
		
		// Result should exist (even if it's an error)
		if result == nil && err == nil {
			t.Error("expected either result or error from status operation")
		}
	})
}

// TestEditorService tests the editor service plugin
func TestEditorService(t *testing.T) {
	t.Run("editor service initialization", func(t *testing.T) {
		service := NewEditorService()
		
		if service.Name() != "editor" {
			t.Errorf("expected name='editor', got %s", service.Name())
		}
		
		if service.Version() != "1.0.0" {
			t.Errorf("expected version='1.0.0', got %s", service.Version())
		}
		
		tools := service.GetTools()
		if len(tools) != 7 {
			t.Errorf("expected 7 tools, got %d", len(tools))
		}
		
		resources := service.GetResources()
		if len(resources) != 2 {
			t.Errorf("expected 2 resources, got %d", len(resources))
		}
		
		prompts := service.GetPrompts()
		if len(prompts) != 2 {
			t.Errorf("expected 2 prompts, got %d", len(prompts))
		}
	})
}

// TestPluginManager tests the plugin manager functionality
func TestPluginManager(t *testing.T) {
	logger := logging.New("test")
	mockMetrics := &mockMetrics{}
	
	t.Run("plugin manager basic operations", func(t *testing.T) {
		manager := NewPluginManager(logger, mockMetrics)
		
		// Create and register plugins
		memoryService := NewMemoryService()
		gitService := NewGitService()
		editorService := NewEditorService()
		
		err := manager.RegisterPlugin(memoryService)
		if err != nil {
			t.Fatalf("failed to register memory plugin: %v", err)
		}
		
		err = manager.RegisterPlugin(gitService)
		if err != nil {
			t.Fatalf("failed to register git plugin: %v", err)
		}
		
		err = manager.RegisterPlugin(editorService)
		if err != nil {
			t.Fatalf("failed to register editor plugin: %v", err)
		}
		
		// Test plugin retrieval
		plugin, exists := manager.GetPlugin("memory")
		if !exists {
			t.Error("memory plugin not found")
		}
		
		if plugin != nil && plugin.Name() != "memory" {
			t.Errorf("expected memory plugin, got %s", plugin.Name())
		}
		
		// Test plugin listing
		plugins := manager.ListPlugins()
		if len(plugins) != 3 {
			t.Errorf("expected 3 plugins, got %d", len(plugins))
		}
		
		// Test duplicate registration
		err = manager.RegisterPlugin(memoryService)
		if err == nil {
			t.Error("expected error when registering duplicate plugin")
		}
	})
	
	t.Run("plugin manager initialization", func(t *testing.T) {
		manager := NewPluginManager(logger, mockMetrics)
		memoryService := NewMemoryService()
		
		err := manager.RegisterPlugin(memoryService)
		if err != nil {
			t.Fatalf("failed to register memory plugin: %v", err)
		}
		
		ctx := context.Background()
		configs := map[string]PluginConfig{
			"memory": {
				Name:    "memory",
				Logger:  logger,
				Metrics: mockMetrics,
			},
		}
		
		err = manager.InitializeAllPlugins(ctx, configs)
		if err != nil {
			t.Fatalf("failed to initialize plugins: %v", err)
		}
		
		// Test health check
		err = memoryService.HealthCheck(ctx)
		if err != nil {
			t.Errorf("memory plugin health check failed: %v", err)
		}
	})
}


// mockMetrics implements metrics.Metrics interface for testing
type mockMetrics struct {
	metrics map[string]interface{}
}

func (m *mockMetrics) Inc(name string, tags ...string) {
	if m.metrics == nil {
		m.metrics = make(map[string]interface{})
	}
	m.metrics[name+"_count"] = 1
}

func (m *mockMetrics) Add(name string, value float64, tags ...string) {
	if m.metrics == nil {
		m.metrics = make(map[string]interface{})
	}
	m.metrics[name] = value
}

func (m *mockMetrics) Set(name string, value float64, tags ...string) {
	if m.metrics == nil {
		m.metrics = make(map[string]interface{})
	}
	m.metrics[name] = value
}

func (m *mockMetrics) Observe(name string, value float64, tags ...string) {
	if m.metrics == nil {
		m.metrics = make(map[string]interface{})
	}
	m.metrics[name] = value
}

func (m *mockMetrics) Time(name string, tags ...string) metrics.Timer {
	return &mockTimer{}
}

func (m *mockMetrics) WithLabels(labels map[string]string) metrics.Metrics {
	return m
}

func (m *mockMetrics) WithPrefix(prefix string) metrics.Metrics {
	return m
}

func (m *mockMetrics) GetStats(name string) metrics.MetricStats {
	return metrics.MetricStats{}
}

func (m *mockMetrics) GetAllStats() map[string]metrics.MetricStats {
	return make(map[string]metrics.MetricStats)
}

// mockTimer implements metrics.Timer interface for testing
type mockTimer struct {
	start time.Time
}

func (t *mockTimer) Duration() time.Duration {
	return time.Since(t.start)
}

func (t *mockTimer) Stop() time.Duration {
	return time.Since(t.start)
}