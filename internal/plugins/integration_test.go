package plugins

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/osakka/mcpeg/internal/registry"
	"github.com/osakka/mcpeg/pkg/health"
	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
	"github.com/osakka/mcpeg/pkg/plugins"
	"github.com/osakka/mcpeg/pkg/validation"
)

// TestPluginServiceRegistryIntegration tests the complete plugin integration with service registry
func TestPluginServiceRegistryIntegration(t *testing.T) {
	logger := logging.New("test")
	mockMetrics := &mockMetrics{}

	t.Run("plugin registration with service registry", func(t *testing.T) {
		ctx := context.Background()

		// Create service registry
		validator := validation.NewValidator(logger, mockMetrics)
		healthMgr := health.NewHealthManager(logger, mockMetrics, "test")

		serviceRegistry := registry.NewServiceRegistry(
			logger,
			mockMetrics,
			validator,
			healthMgr,
		)

		// Service registry is ready to use after creation

		// Create plugin integration
		integration := NewPluginIntegration(logger, mockMetrics, serviceRegistry)

		// Initialize plugin integration
		err = integration.Initialize(ctx)
		if err != nil {
			t.Fatalf("failed to initialize plugin integration: %v", err)
		}
		defer integration.Close()

		// Verify plugins are registered
		services, err := serviceRegistry.ListServices(ctx)
		if err != nil {
			t.Fatalf("failed to list services: %v", err)
		}

		// Should have 3 plugins registered
		pluginCount := 0
		for _, service := range services {
			if service.Type == "mcp_plugin" {
				pluginCount++
			}
		}

		if pluginCount != 3 {
			t.Errorf("expected 3 plugins registered, got %d", pluginCount)
		}

		// Verify specific plugins
		expectedPlugins := map[string]bool{
			"memory": false,
			"git":    false,
			"editor": false,
		}

		for _, service := range services {
			if service.Type == "mcp_plugin" {
				if _, exists := expectedPlugins[service.Name]; exists {
					expectedPlugins[service.Name] = true
				}
			}
		}

		for name, found := range expectedPlugins {
			if !found {
				t.Errorf("plugin %s not found in service registry", name)
			}
		}
	})

	t.Run("plugin health check bypass", func(t *testing.T) {
		ctx := context.Background()

		// Create service registry
		validator := validation.NewValidator(logger, mockMetrics)
		healthMgr := health.NewHealthManager(logger, mockMetrics, "test")

		serviceRegistry := registry.NewServiceRegistry(
			logger,
			mockMetrics,
			validator,
			healthMgr,
		)

		// Service registry is ready to use after creation

		// Create plugin integration
		integration := NewPluginIntegration(logger, mockMetrics, serviceRegistry)

		// Initialize plugin integration
		err = integration.Initialize(ctx)
		if err != nil {
			t.Fatalf("failed to initialize plugin integration: %v", err)
		}
		defer integration.Close()

		// Wait for health checks to run
		time.Sleep(100 * time.Millisecond)

		// Verify all plugins are healthy (health check should be bypassed)
		services, err := serviceRegistry.ListServices(ctx)
		if err != nil {
			t.Fatalf("failed to list services: %v", err)
		}

		for _, service := range services {
			if service.Type == "mcp_plugin" {
				if service.Health != registry.HealthHealthy {
					t.Errorf("plugin %s should be healthy, got %v", service.Name, service.Health)
				}

				if service.Status != registry.StatusActive {
					t.Errorf("plugin %s should be active, got %v", service.Name, service.Status)
				}
			}
		}
	})

	t.Run("plugin URL validation", func(t *testing.T) {
		ctx := context.Background()

		// Create service registry
		validator := validation.NewValidator(logger, mockMetrics)
		healthMgr := health.NewHealthManager(logger, mockMetrics, "test")

		serviceRegistry := registry.NewServiceRegistry(
			logger,
			mockMetrics,
			validator,
			healthMgr,
		)

		// Service registry is ready to use after creation

		// Test direct plugin URL registration
		req := &registry.ServiceRegistrationRequest{
			Name:      "test-plugin",
			Type:      "mcp_plugin",
			Version:   "1.0.0",
			Endpoint:  "plugin://internal",
			Tools:     []registry.ToolDefinition{},
			Resources: []registry.ResourceDefinition{},
			Prompts:   []registry.PromptDefinition{},
		}

		_, err = serviceRegistry.RegisterService(ctx, req)
		if err != nil {
			t.Errorf("plugin URL registration should succeed: %v", err)
		}

		// Test that HTTP URLs still work
		req.Name = "test-http"
		req.Endpoint = "http://localhost:8080"

		_, err = serviceRegistry.RegisterService(ctx, req)
		if err != nil {
			t.Errorf("HTTP URL registration should succeed: %v", err)
		}

		// Test that HTTPS URLs still work
		req.Name = "test-https"
		req.Endpoint = "https://localhost:8080"

		_, err = serviceRegistry.RegisterService(ctx, req)
		if err != nil {
			t.Errorf("HTTPS URL registration should succeed: %v", err)
		}

		// Test that invalid URLs are rejected
		req.Name = "test-invalid"
		req.Endpoint = "invalid-url"

		_, err = serviceRegistry.RegisterService(ctx, req)
		if err == nil {
			t.Error("invalid URL registration should fail")
		}
	})

	t.Run("plugin capabilities registration", func(t *testing.T) {
		ctx := context.Background()

		// Create service registry
		validator := validation.NewValidator(logger, mockMetrics)
		healthMgr := health.NewHealthManager(logger, mockMetrics, "test")

		serviceRegistry := registry.NewServiceRegistry(
			logger,
			mockMetrics,
			validator,
			healthMgr,
		)

		// Service registry is ready to use after creation

		// Create plugin integration
		integration := NewPluginIntegration(logger, mockMetrics, serviceRegistry)

		// Initialize plugin integration
		err = integration.Initialize(ctx)
		if err != nil {
			t.Fatalf("failed to initialize plugin integration: %v", err)
		}
		defer integration.Close()

		// Get services and verify capabilities
		services, err := serviceRegistry.ListServices(ctx)
		if err != nil {
			t.Fatalf("failed to list services: %v", err)
		}

		// Check memory plugin capabilities
		for _, service := range services {
			if service.Type == "mcp_plugin" && service.Name == "memory" {
				if len(service.Tools) != 5 {
					t.Errorf("memory plugin should have 5 tools, got %d", len(service.Tools))
				}

				if len(service.Resources) != 2 {
					t.Errorf("memory plugin should have 2 resources, got %d", len(service.Resources))
				}

				if len(service.Prompts) != 2 {
					t.Errorf("memory plugin should have 2 prompts, got %d", len(service.Prompts))
				}

				// Check specific tools
				toolNames := make([]string, len(service.Tools))
				for i, tool := range service.Tools {
					toolNames[i] = tool.Name
				}

				expectedTools := []string{"set", "get", "delete", "list", "clear"}
				for _, expectedTool := range expectedTools {
					found := false
					for _, toolName := range toolNames {
						if toolName == expectedTool {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("memory plugin missing tool: %s", expectedTool)
					}
				}

				break
			}
		}

		// Check git plugin capabilities
		for _, service := range services {
			if service.Type == "mcp_plugin" && service.Name == "git" {
				if len(service.Tools) != 8 {
					t.Errorf("git plugin should have 8 tools, got %d", len(service.Tools))
				}

				if len(service.Resources) != 2 {
					t.Errorf("git plugin should have 2 resources, got %d", len(service.Resources))
				}

				if len(service.Prompts) != 2 {
					t.Errorf("git plugin should have 2 prompts, got %d", len(service.Prompts))
				}

				break
			}
		}

		// Check editor plugin capabilities
		for _, service := range services {
			if service.Type == "mcp_plugin" && service.Name == "editor" {
				if len(service.Tools) != 7 {
					t.Errorf("editor plugin should have 7 tools, got %d", len(service.Tools))
				}

				if len(service.Resources) != 2 {
					t.Errorf("editor plugin should have 2 resources, got %d", len(service.Resources))
				}

				if len(service.Prompts) != 2 {
					t.Errorf("editor plugin should have 2 prompts, got %d", len(service.Prompts))
				}

				break
			}
		}
	})

	t.Run("plugin service discovery", func(t *testing.T) {
		ctx := context.Background()

		// Create service registry
		validator := validation.NewValidator(logger, mockMetrics)
		healthMgr := health.NewHealthManager(logger, mockMetrics, "test")

		serviceRegistry := registry.NewServiceRegistry(
			logger,
			mockMetrics,
			validator,
			healthMgr,
		)

		// Service registry is ready to use after creation

		// Create plugin integration
		integration := NewPluginIntegration(logger, mockMetrics, serviceRegistry)

		// Initialize plugin integration
		err = integration.Initialize(ctx)
		if err != nil {
			t.Fatalf("failed to initialize plugin integration: %v", err)
		}
		defer integration.Close()

		// Test service discovery by type
		services, err := serviceRegistry.GetServicesByType(ctx, "mcp_plugin")
		if err != nil {
			t.Fatalf("failed to get services by type: %v", err)
		}

		if len(services) != 3 {
			t.Errorf("expected 3 mcp_plugin services, got %d", len(services))
		}

		// Test service discovery by name
		service, err := serviceRegistry.GetService(ctx, "memory")
		if err != nil {
			t.Fatalf("failed to get memory service: %v", err)
		}

		if service.Name != "memory" {
			t.Errorf("expected service name 'memory', got %s", service.Name)
		}

		if service.Type != "mcp_plugin" {
			t.Errorf("expected service type 'mcp_plugin', got %s", service.Type)
		}

		if !strings.HasPrefix(service.Endpoint, "plugin://") {
			t.Errorf("expected plugin:// endpoint, got %s", service.Endpoint)
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

func (m *mockMetrics) Dec(name string, tags ...string) {
	if m.metrics == nil {
		m.metrics = make(map[string]interface{})
	}
	m.metrics[name+"_count"] = -1
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

func (m *mockMetrics) Add(name string, value float64, labels ...string) {
	if m.metrics == nil {
		m.metrics = make(map[string]interface{})
	}
	m.metrics[name+"_add"] = value
}

func (m *mockMetrics) Time(name string, labels ...string) metrics.Timer {
	return &mockTimer{}
}

func (m *mockMetrics) WithLabels(labels map[string]string) metrics.Metrics {
	return m
}

func (m *mockMetrics) WithPrefix(prefix string) metrics.Metrics {
	return m
}

func (m *mockMetrics) GetStats(name string) metrics.MetricStats {
	return metrics.MetricStats{
		Count:     1,
		Sum:       1.0,
		Average:   1.0,
		Min:       1.0,
		Max:       1.0,
		LastValue: 1.0,
		Trend:     "stable",
	}
}

func (m *mockMetrics) GetAllStats() map[string]metrics.MetricStats {
	return make(map[string]metrics.MetricStats)
}

func (m *mockMetrics) Close() error {
	return nil
}

type mockTimer struct{}

func (t *mockTimer) Stop() time.Duration {
	return time.Millisecond
}
