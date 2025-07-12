package adapter

import (
	"context"
	"time"
)

// ServiceAdapter is the interface all service adapters must implement
type ServiceAdapter interface {
	// Metadata
	Name() string        // Unique adapter name (e.g., "mysql", "vault")
	Type() string        // Service type (e.g., "database", "secrets")
	Description() string // Human-readable description

	// Lifecycle
	Initialize(config ServiceConfig) error // One-time initialization
	Start(ctx context.Context) error       // Start the adapter
	Stop(ctx context.Context) error        // Graceful shutdown

	// MCP Protocol Implementation
	GetTools() []ToolDefinition         // MCP tools this adapter provides
	GetResources() []ResourceDefinition // MCP resources this adapter provides
	GetPrompts() []PromptDefinition     // MCP prompts this adapter provides

	// Execution
	ExecuteTool(ctx context.Context, tool string, params map[string]interface{}) (interface{}, error)
	GetResource(ctx context.Context, uri string) (interface{}, error)

	// Health and Monitoring
	HealthCheck(ctx context.Context) error
	GetMetrics() AdapterMetrics
	GetStatus() AdapterStatus
}

// ServiceConfig contains configuration for a service adapter
type ServiceConfig struct {
	// Basic configuration
	Enabled bool   `yaml:"enabled"`
	Type    string `yaml:"type"`
	Driver  string `yaml:"driver"`

	// Resource limits
	MaxConnections int           `yaml:"max_connections"`
	Timeout        time.Duration `yaml:"timeout"`
	MemoryLimitMB  int           `yaml:"memory_limit_mb"`

	// Circuit breaker configuration
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker"`

	// Custom configuration for specific adapter
	Custom map[string]interface{} `yaml:"config"`
}

// CircuitBreakerConfig configures the circuit breaker for an adapter
type CircuitBreakerConfig struct {
	Enabled          bool          `yaml:"enabled"`
	FailureThreshold int           `yaml:"failure_threshold"`
	ResetTimeout     time.Duration `yaml:"reset_timeout"`
	HalfOpenMax      int           `yaml:"half_open_max"`
}

// ToolDefinition defines an MCP tool
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// ResourceDefinition defines an MCP resource
type ResourceDefinition struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mime_type"`
}

// PromptDefinition defines an MCP prompt
type PromptDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Arguments   map[string]interface{} `json:"arguments"`
}

// AdapterMetrics contains runtime metrics for an adapter
type AdapterMetrics struct {
	// Request metrics
	TotalRequests   uint64        `json:"total_requests"`
	SuccessRequests uint64        `json:"success_requests"`
	FailedRequests  uint64        `json:"failed_requests"`
	AverageLatency  time.Duration `json:"average_latency_ms"`

	// Resource metrics
	ActiveConnections int `json:"active_connections"`
	MemoryUsageMB     int `json:"memory_usage_mb"`

	// Circuit breaker metrics
	CircuitBreakerState string `json:"circuit_breaker_state"`
	ConsecutiveFailures int    `json:"consecutive_failures"`

	// Timestamp
	LastUpdated time.Time `json:"last_updated"`
}

// AdapterStatus represents the current status of an adapter
type AdapterStatus struct {
	State           AdapterState  `json:"state"`
	Message         string        `json:"message"`
	LastHealthCheck time.Time     `json:"last_health_check"`
	Uptime          time.Duration `json:"uptime"`
}

// AdapterState represents the state of an adapter
type AdapterState string

const (
	StateUninitialized AdapterState = "uninitialized"
	StateInitialized   AdapterState = "initialized"
	StateStarting      AdapterState = "starting"
	StateRunning       AdapterState = "running"
	StateStopping      AdapterState = "stopping"
	StateStopped       AdapterState = "stopped"
	StateError         AdapterState = "error"
)

// BaseAdapter provides common functionality for all adapters
type BaseAdapter struct {
	name    string
	typ     string
	state   AdapterState
	config  ServiceConfig
	metrics AdapterMetrics
	started time.Time
}

// NewBaseAdapter creates a new base adapter
func NewBaseAdapter(name, typ string) *BaseAdapter {
	return &BaseAdapter{
		name:  name,
		typ:   typ,
		state: StateUninitialized,
	}
}

// Name returns the adapter name
func (b *BaseAdapter) Name() string {
	return b.name
}

// Type returns the adapter type
func (b *BaseAdapter) Type() string {
	return b.typ
}

// GetStatus returns the adapter status
func (b *BaseAdapter) GetStatus() AdapterStatus {
	uptime := time.Duration(0)
	if !b.started.IsZero() {
		uptime = time.Since(b.started)
	}

	return AdapterStatus{
		State:  b.state,
		Uptime: uptime,
	}
}

// GetMetrics returns the adapter metrics
func (b *BaseAdapter) GetMetrics() AdapterMetrics {
	b.metrics.LastUpdated = time.Now()
	return b.metrics
}
