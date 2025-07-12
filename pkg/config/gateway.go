package config

import (
	"fmt"
	"time"

	"github.com/osakka/mcpeg/internal/server"
)

// GatewayConfig represents the complete gateway configuration
type GatewayConfig struct {
	// Server configuration
	Server ServerConfig `yaml:"server"`

	// Logging configuration
	Logging LoggingConfig `yaml:"logging"`

	// Metrics configuration
	Metrics MetricsConfig `yaml:"metrics"`

	// Service registry configuration
	Registry RegistryConfig `yaml:"registry"`

	// Security configuration
	Security SecurityConfig `yaml:"security"`

	// Development mode settings
	Development DevelopmentConfig `yaml:"development"`
}

// ServerConfig configures the HTTP server
type ServerConfig struct {
	// Basic server settings
	Address string `yaml:"address"`
	Port    int    `yaml:"port"`

	// Timeout settings
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`

	// TLS configuration
	TLS TLSConfig `yaml:"tls"`

	// CORS configuration
	CORS CORSConfig `yaml:"cors"`

	// Middleware settings
	Middleware MiddlewareConfig `yaml:"middleware"`

	// Health check settings
	HealthCheck HealthCheckConfig `yaml:"health_check"`
}

// TLSConfig configures TLS/SSL settings
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`

	// Advanced TLS settings
	MinVersion string   `yaml:"min_version"` // "1.2" or "1.3"
	Ciphers    []string `yaml:"ciphers"`
}

// CORSConfig configures Cross-Origin Resource Sharing
type CORSConfig struct {
	Enabled          bool     `yaml:"enabled"`
	AllowOrigins     []string `yaml:"allow_origins"`
	AllowMethods     []string `yaml:"allow_methods"`
	AllowHeaders     []string `yaml:"allow_headers"`
	ExposeHeaders    []string `yaml:"expose_headers"`
	AllowCredentials bool     `yaml:"allow_credentials"`
	MaxAge           int      `yaml:"max_age"`
}

// MiddlewareConfig configures HTTP middleware
type MiddlewareConfig struct {
	// Compression settings
	Compression CompressionConfig `yaml:"compression"`

	// Rate limiting settings
	RateLimit RateLimitConfig `yaml:"rate_limit"`

	// Request logging settings
	RequestLogging RequestLoggingConfig `yaml:"request_logging"`
}

// CompressionConfig configures response compression
type CompressionConfig struct {
	Enabled bool     `yaml:"enabled"`
	Level   int      `yaml:"level"` // 1-9, higher = better compression
	Types   []string `yaml:"types"` // MIME types to compress
}

// RateLimitConfig configures request rate limiting
type RateLimitConfig struct {
	Enabled    bool          `yaml:"enabled"`
	RPS        int           `yaml:"rps"`         // Requests per second
	Burst      int           `yaml:"burst"`       // Burst capacity
	WindowSize time.Duration `yaml:"window_size"` // Time window for rate limiting
}

// RequestLoggingConfig configures request/response logging
type RequestLoggingConfig struct {
	Enabled        bool     `yaml:"enabled"`
	IncludeBody    bool     `yaml:"include_body"`
	ExcludePaths   []string `yaml:"exclude_paths"`
	IncludeHeaders []string `yaml:"include_headers"`
}

// HealthCheckConfig configures health check endpoints
type HealthCheckConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Endpoint string `yaml:"endpoint"`
	Detailed bool   `yaml:"detailed"` // Include detailed health information
}

// LoggingConfig configures application logging
type LoggingConfig struct {
	Level  string `yaml:"level"`  // trace, debug, info, warn, error
	Format string `yaml:"format"` // json, text

	// Output configuration
	Output OutputConfig `yaml:"output"`

	// Structured logging settings
	Structured StructuredLoggingConfig `yaml:"structured"`
}

// OutputConfig configures log output destinations
type OutputConfig struct {
	Console ConsoleOutputConfig `yaml:"console"`
	File    FileOutputConfig    `yaml:"file"`
}

// ConsoleOutputConfig configures console output
type ConsoleOutputConfig struct {
	Enabled   bool `yaml:"enabled"`
	Colorized bool `yaml:"colorized"`
}

// FileOutputConfig configures file output
type FileOutputConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Path       string `yaml:"path"`
	MaxSize    int    `yaml:"max_size"` // MB
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"` // days
	Compress   bool   `yaml:"compress"`
}

// StructuredLoggingConfig configures structured logging features
type StructuredLoggingConfig struct {
	IncludeTraceID bool `yaml:"include_trace_id"`
	IncludeSpanID  bool `yaml:"include_span_id"`
	IncludeCaller  bool `yaml:"include_caller"`
	IncludeStack   bool `yaml:"include_stack"`
}

// MetricsConfig configures metrics collection and export
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Address string `yaml:"address"`
	Port    int    `yaml:"port"`

	// Prometheus settings
	Prometheus PrometheusConfig `yaml:"prometheus"`

	// Collection settings
	Collection MetricsCollectionConfig `yaml:"collection"`
}

// PrometheusConfig configures Prometheus metrics export
type PrometheusConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Endpoint  string `yaml:"endpoint"`
	Namespace string `yaml:"namespace"`
	Subsystem string `yaml:"subsystem"`
}

// MetricsCollectionConfig configures what metrics to collect
type MetricsCollectionConfig struct {
	HTTP     bool `yaml:"http"`     // HTTP request metrics
	System   bool `yaml:"system"`   // System resource metrics
	Business bool `yaml:"business"` // Business logic metrics

	// Collection intervals
	SystemInterval time.Duration `yaml:"system_interval"`
}

// RegistryConfig configures the service registry
type RegistryConfig struct {
	// Service discovery settings
	Discovery DiscoveryConfig `yaml:"discovery"`

	// Load balancing settings
	LoadBalancer LoadBalancerConfig `yaml:"load_balancer"`

	// Health checking settings
	HealthChecks HealthChecksConfig `yaml:"health_checks"`
}

// DiscoveryConfig configures service discovery mechanisms
type DiscoveryConfig struct {
	// Static service configuration
	Static StaticDiscoveryConfig `yaml:"static"`

	// Consul discovery
	Consul ConsulDiscoveryConfig `yaml:"consul"`

	// Kubernetes discovery
	Kubernetes KubernetesDiscoveryConfig `yaml:"kubernetes"`

	// File-based discovery
	File FileDiscoveryConfig `yaml:"file"`
}

// StaticDiscoveryConfig configures static service definitions
type StaticDiscoveryConfig struct {
	Enabled  bool                  `yaml:"enabled"`
	Services []StaticServiceConfig `yaml:"services"`
}

// StaticServiceConfig defines a static service
type StaticServiceConfig struct {
	Name      string            `yaml:"name"`
	Type      string            `yaml:"type"`
	Endpoints []EndpointConfig  `yaml:"endpoints"`
	Metadata  map[string]string `yaml:"metadata"`
}

// EndpointConfig defines a service endpoint
type EndpointConfig struct {
	Address string   `yaml:"address"`
	Port    int      `yaml:"port"`
	Weight  int      `yaml:"weight"`
	Tags    []string `yaml:"tags"`
}

// ConsulDiscoveryConfig configures Consul service discovery
type ConsulDiscoveryConfig struct {
	Enabled bool   `yaml:"enabled"`
	Address string `yaml:"address"`
	Token   string `yaml:"token"`

	// Service filtering
	ServicePrefix string   `yaml:"service_prefix"`
	Tags          []string `yaml:"tags"`
}

// KubernetesDiscoveryConfig configures Kubernetes service discovery
type KubernetesDiscoveryConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Namespace string `yaml:"namespace"`

	// Service selection
	LabelSelector string `yaml:"label_selector"`
}

// FileDiscoveryConfig configures file-based service discovery
type FileDiscoveryConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`

	// File watching
	WatchEnabled  bool          `yaml:"watch_enabled"`
	WatchInterval time.Duration `yaml:"watch_interval"`
}

// LoadBalancerConfig configures load balancing behavior
type LoadBalancerConfig struct {
	Strategy string `yaml:"strategy"` // round_robin, least_connections, weighted, hash, random

	// Health-based routing
	HealthAware bool `yaml:"health_aware"`

	// Circuit breaker settings
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker"`
}

// CircuitBreakerConfig configures circuit breaker behavior
type CircuitBreakerConfig struct {
	Enabled             bool          `yaml:"enabled"`
	FailureThreshold    int           `yaml:"failure_threshold"`
	RecoveryTimeout     time.Duration `yaml:"recovery_timeout"`
	HalfOpenMaxRequests int           `yaml:"half_open_max_requests"`
}

// HealthChecksConfig configures service health checking
type HealthChecksConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`

	// Health check types
	HTTP HealthCheckHTTPConfig `yaml:"http"`
	TCP  HealthCheckTCPConfig  `yaml:"tcp"`
}

// HealthCheckHTTPConfig configures HTTP health checks
type HealthCheckHTTPConfig struct {
	Enabled        bool              `yaml:"enabled"`
	Path           string            `yaml:"path"`
	Method         string            `yaml:"method"`
	Headers        map[string]string `yaml:"headers"`
	ExpectedStatus []int             `yaml:"expected_status"`
}

// HealthCheckTCPConfig configures TCP health checks
type HealthCheckTCPConfig struct {
	Enabled bool `yaml:"enabled"`
}

// SecurityConfig configures security settings
type SecurityConfig struct {
	// API key authentication
	APIKey APIKeyConfig `yaml:"api_key"`

	// JWT authentication
	JWT JWTConfig `yaml:"jwt"`

	// Request validation
	Validation ValidationConfig `yaml:"validation"`
}

// APIKeyConfig configures API key authentication
type APIKeyConfig struct {
	Enabled bool     `yaml:"enabled"`
	Header  string   `yaml:"header"`
	Keys    []string `yaml:"keys"`
}

// JWTConfig configures JWT authentication
type JWTConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Secret    string `yaml:"secret"`
	Algorithm string `yaml:"algorithm"`

	// Token validation
	ValidateExpiry   bool `yaml:"validate_expiry"`
	ValidateIssuer   bool `yaml:"validate_issuer"`
	ValidateAudience bool `yaml:"validate_audience"`
}

// ValidationConfig configures request validation
type ValidationConfig struct {
	Enabled      bool `yaml:"enabled"`
	StrictMode   bool `yaml:"strict_mode"`
	ValidateBody bool `yaml:"validate_body"`
}

// DevelopmentConfig configures development-specific settings
type DevelopmentConfig struct {
	Enabled bool `yaml:"enabled"`

	// Development server settings
	HotReload    bool `yaml:"hot_reload"`
	DebugMode    bool `yaml:"debug_mode"`
	ProfilerPort int  `yaml:"profiler_port"`

	// Admin endpoints
	AdminEndpoints AdminEndpointsConfig `yaml:"admin_endpoints"`
}

// AdminEndpointsConfig configures administrative endpoints
type AdminEndpointsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Prefix  string `yaml:"prefix"`

	// Available admin functions
	ConfigReload     bool `yaml:"config_reload"`
	ServiceDiscovery bool `yaml:"service_discovery"`
	HealthChecks     bool `yaml:"health_checks"`
}

// Validate validates the gateway configuration
func (c *GatewayConfig) Validate() error {
	// Server validation
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535, got %d", c.Server.Port)
	}

	if c.Server.TLS.Enabled {
		if c.Server.TLS.CertFile == "" {
			return fmt.Errorf("TLS cert file is required when TLS is enabled")
		}
		if c.Server.TLS.KeyFile == "" {
			return fmt.Errorf("TLS key file is required when TLS is enabled")
		}
	}

	// Metrics validation
	if c.Metrics.Enabled {
		if c.Metrics.Port <= 0 || c.Metrics.Port > 65535 {
			return fmt.Errorf("metrics port must be between 1 and 65535, got %d", c.Metrics.Port)
		}
	}

	// Load balancer strategy validation
	validStrategies := []string{"round_robin", "least_connections", "weighted", "hash", "random"}
	strategy := c.Registry.LoadBalancer.Strategy
	if strategy == "" {
		strategy = "round_robin" // default
	}

	valid := false
	for _, validStrategy := range validStrategies {
		if strategy == validStrategy {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid load balancer strategy: %s, must be one of %v", strategy, validStrategies)
	}

	return nil
}

// ToServerConfig converts GatewayConfig to server.ServerConfig
func (c *GatewayConfig) ToServerConfig() server.ServerConfig {
	return server.ServerConfig{
		Address:               c.Server.Address,
		Port:                  c.Server.Port,
		ReadTimeout:           c.Server.ReadTimeout,
		WriteTimeout:          c.Server.WriteTimeout,
		IdleTimeout:           c.Server.IdleTimeout,
		ShutdownTimeout:       c.Server.ShutdownTimeout,
		TLSEnabled:            c.Server.TLS.Enabled,
		TLSCertFile:           c.Server.TLS.CertFile,
		TLSKeyFile:            c.Server.TLS.KeyFile,
		CORSEnabled:           c.Server.CORS.Enabled,
		CORSAllowOrigins:      c.Server.CORS.AllowOrigins,
		CORSAllowMethods:      c.Server.CORS.AllowMethods,
		CORSAllowHeaders:      c.Server.CORS.AllowHeaders,
		EnableCompression:     c.Server.Middleware.Compression.Enabled,
		EnableRateLimit:       c.Server.Middleware.RateLimit.Enabled,
		RateLimitRPS:          c.Server.Middleware.RateLimit.RPS,
		EnableHealthEndpoints: c.Server.HealthCheck.Enabled,
		EnableMetricsEndpoint: c.Metrics.Enabled,
		EnableAdminEndpoints:  c.Development.AdminEndpoints.Enabled,
	}
}

// GetDefaults returns a configuration with sensible defaults
func GetDefaults() *GatewayConfig {
	return &GatewayConfig{
		Server: ServerConfig{
			Address:         "0.0.0.0",
			Port:            8080,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			IdleTimeout:     60 * time.Second,
			ShutdownTimeout: 30 * time.Second,
			TLS: TLSConfig{
				Enabled:    false,
				MinVersion: "1.2",
			},
			CORS: CORSConfig{
				Enabled:      true,
				AllowOrigins: []string{"*"},
				AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowHeaders: []string{"Content-Type", "Authorization", "X-Client-ID", "X-Session-ID"},
			},
			Middleware: MiddlewareConfig{
				Compression: CompressionConfig{
					Enabled: true,
					Level:   6,
					Types:   []string{"application/json", "text/html", "text/css", "application/javascript"},
				},
				RateLimit: RateLimitConfig{
					Enabled:    false,
					RPS:        1000,
					Burst:      2000,
					WindowSize: time.Minute,
				},
				RequestLogging: RequestLoggingConfig{
					Enabled:      true,
					IncludeBody:  false,
					ExcludePaths: []string{"/health", "/metrics"},
				},
			},
			HealthCheck: HealthCheckConfig{
				Enabled:  true,
				Endpoint: "/health",
				Detailed: false,
			},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: OutputConfig{
				Console: ConsoleOutputConfig{
					Enabled:   true,
					Colorized: false,
				},
				File: FileOutputConfig{
					Enabled: false,
				},
			},
			Structured: StructuredLoggingConfig{
				IncludeTraceID: true,
				IncludeSpanID:  true,
				IncludeCaller:  false,
				IncludeStack:   false,
			},
		},
		Metrics: MetricsConfig{
			Enabled: true,
			Address: "0.0.0.0",
			Port:    9090,
			Prometheus: PrometheusConfig{
				Enabled:   true,
				Endpoint:  "/metrics",
				Namespace: "mcpeg",
				Subsystem: "gateway",
			},
			Collection: MetricsCollectionConfig{
				HTTP:           true,
				System:         true,
				Business:       true,
				SystemInterval: 15 * time.Second,
			},
		},
		Registry: RegistryConfig{
			Discovery: DiscoveryConfig{
				Static: StaticDiscoveryConfig{
					Enabled:  true,
					Services: []StaticServiceConfig{},
				},
				Consul: ConsulDiscoveryConfig{
					Enabled: false,
					Address: "localhost:8500",
				},
				Kubernetes: KubernetesDiscoveryConfig{
					Enabled:   false,
					Namespace: "default",
				},
				File: FileDiscoveryConfig{
					Enabled:       false,
					WatchEnabled:  true,
					WatchInterval: 30 * time.Second,
				},
			},
			LoadBalancer: LoadBalancerConfig{
				Strategy:    "round_robin",
				HealthAware: true,
				CircuitBreaker: CircuitBreakerConfig{
					Enabled:             true,
					FailureThreshold:    5,
					RecoveryTimeout:     30 * time.Second,
					HalfOpenMaxRequests: 3,
				},
			},
			HealthChecks: HealthChecksConfig{
				Enabled:  true,
				Interval: 30 * time.Second,
				Timeout:  5 * time.Second,
				HTTP: HealthCheckHTTPConfig{
					Enabled:        true,
					Path:           "/health",
					Method:         "GET",
					ExpectedStatus: []int{200},
				},
				TCP: HealthCheckTCPConfig{
					Enabled: false,
				},
			},
		},
		Security: SecurityConfig{
			APIKey: APIKeyConfig{
				Enabled: false,
				Header:  "X-API-Key",
			},
			JWT: JWTConfig{
				Enabled:          false,
				Algorithm:        "HS256",
				ValidateExpiry:   true,
				ValidateIssuer:   false,
				ValidateAudience: false,
			},
			Validation: ValidationConfig{
				Enabled:      true,
				StrictMode:   false,
				ValidateBody: true,
			},
		},
		Development: DevelopmentConfig{
			Enabled:      false,
			HotReload:    false,
			DebugMode:    false,
			ProfilerPort: 6060,
			AdminEndpoints: AdminEndpointsConfig{
				Enabled:          false,
				Prefix:           "/admin",
				ConfigReload:     true,
				ServiceDiscovery: true,
				HealthChecks:     true,
			},
		},
	}
}
