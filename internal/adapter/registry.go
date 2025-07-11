package adapter

import (
	"fmt"
	"sync"

	"github.com/osakka/mcpeg/pkg/logging"
)

// Factory creates a new instance of a service adapter
type Factory func(logger logging.Logger) (ServiceAdapter, error)

// Registry manages service adapter factories
type Registry struct {
	mu        sync.RWMutex
	factories map[string]Factory
	logger    logging.Logger
}

// DefaultRegistry is the global adapter registry
var DefaultRegistry = NewRegistry(logging.New("adapter.registry"))

// NewRegistry creates a new adapter registry
func NewRegistry(logger logging.Logger) *Registry {
	return &Registry{
		factories: make(map[string]Factory),
		logger:    logger,
	}
}

// Register registers a new adapter factory
func (r *Registry) Register(name string, factory Factory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[name]; exists {
		return fmt.Errorf("adapter %s already registered", name)
	}

	r.factories[name] = factory
	r.logger.Info("adapter_registered",
		"name", name,
		"total_adapters", len(r.factories))

	return nil
}

// Create creates a new adapter instance
func (r *Registry) Create(name string, logger logging.Logger) (ServiceAdapter, error) {
	r.mu.RLock()
	factory, exists := r.factories[name]
	r.mu.RUnlock()

	if !exists {
		available := r.ListAvailable()
		r.logger.Error("adapter_not_found",
			"requested", name,
			"available", available,
			"suggested_actions", []string{
				"Check adapter name spelling",
				"Verify adapter is registered",
				"Use one of the available adapters",
			})
		return nil, fmt.Errorf("adapter %s not found", name)
	}

	adapter, err := factory(logger)
	if err != nil {
		r.logger.Error("adapter_creation_failed",
			"name", name,
			"error", err,
			"suggested_actions", []string{
				"Check adapter dependencies",
				"Verify configuration",
				"Review adapter logs",
			})
		return nil, fmt.Errorf("failed to create adapter %s: %w", name, err)
	}

	r.logger.Debug("adapter_created", "name", name)
	return adapter, nil
}

// ListAvailable returns a list of available adapter names
func (r *Registry) ListAvailable() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// MustRegister registers an adapter factory and panics on error
func (r *Registry) MustRegister(name string, factory Factory) {
	if err := r.Register(name, factory); err != nil {
		panic(fmt.Sprintf("failed to register adapter %s: %v", name, err))
	}
}

// Global registration functions

// Register registers an adapter factory with the default registry
func Register(name string, factory Factory) error {
	return DefaultRegistry.Register(name, factory)
}

// MustRegister registers an adapter factory with the default registry and panics on error
func MustRegister(name string, factory Factory) {
	DefaultRegistry.MustRegister(name, factory)
}

// Create creates an adapter instance using the default registry
func Create(name string, logger logging.Logger) (ServiceAdapter, error) {
	return DefaultRegistry.Create(name, logger)
}

// ListAvailable returns available adapters from the default registry
func ListAvailable() []string {
	return DefaultRegistry.ListAvailable()
}

// AdapterInfo provides information about a registered adapter
type AdapterInfo struct {
	Name        string
	Type        string
	Description string
	Tools       []string
	Resources   []string
}

// GetAdapterInfo retrieves information about a registered adapter
func (r *Registry) GetAdapterInfo(name string) (*AdapterInfo, error) {
	// Create a temporary instance to get metadata
	adapter, err := r.Create(name, r.logger.WithComponent("info"))
	if err != nil {
		return nil, err
	}

	// Extract tool and resource names
	tools := adapter.GetTools()
	toolNames := make([]string, len(tools))
	for i, tool := range tools {
		toolNames[i] = tool.Name
	}

	resources := adapter.GetResources()
	resourceNames := make([]string, len(resources))
	for i, resource := range resources {
		resourceNames[i] = resource.URI
	}

	info := &AdapterInfo{
		Name:        adapter.Name(),
		Type:        adapter.Type(),
		Description: adapter.Description(),
		Tools:       toolNames,
		Resources:   resourceNames,
	}

	// Clean up temporary instance
	if err := adapter.Stop(nil); err != nil {
		r.logger.Warn("failed_to_stop_temp_adapter",
			"name", name,
			"error", err)
	}

	return info, nil
}

// GetAllAdapterInfo returns information about all registered adapters
func (r *Registry) GetAllAdapterInfo() ([]*AdapterInfo, error) {
	names := r.ListAvailable()
	infos := make([]*AdapterInfo, 0, len(names))

	for _, name := range names {
		info, err := r.GetAdapterInfo(name)
		if err != nil {
			r.logger.Warn("failed_to_get_adapter_info",
				"name", name,
				"error", err)
			continue
		}
		infos = append(infos, info)
	}

	return infos, nil
}