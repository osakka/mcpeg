package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/osakka/mcpeg/internal/server"
	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
	"github.com/osakka/mcpeg/pkg/validation"
	"github.com/osakka/mcpeg/pkg/health"
)

// GatewayApp represents the main gateway application
type GatewayApp struct {
	config    AppConfig
	logger    logging.Logger
	metrics   metrics.Metrics
	validator *validation.Validator
	healthMgr *health.HealthManager
	server    *server.GatewayServer
}

// AppConfig configures the gateway application
type AppConfig struct {
	// Server configuration
	Server server.ServerConfig `yaml:"server"`
	
	// Logging configuration
	LogLevel  string `yaml:"log_level"`
	LogFormat string `yaml:"log_format"`
	
	// Metrics configuration
	MetricsEnabled bool   `yaml:"metrics_enabled"`
	MetricsAddress string `yaml:"metrics_address"`
	MetricsPort    int    `yaml:"metrics_port"`
	
	// Configuration file
	ConfigFile string `yaml:"-"`
	
	// Development mode
	DevMode bool `yaml:"dev_mode"`
}

func main() {
	app := &GatewayApp{}
	
	// Parse command line flags
	if err := app.parseFlags(); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}
	
	// Load configuration
	if err := app.loadConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	
	// Initialize components
	if err := app.initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing application: %v\n", err)
		os.Exit(1)
	}
	
	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	app.setupSignalHandling(cancel)
	
	// Start the gateway
	if err := app.start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting gateway: %v\n", err)
		os.Exit(1)
	}
}

// parseFlags parses command line flags
func (app *GatewayApp) parseFlags() error {
	flag.StringVar(&app.config.ConfigFile, "config", "", "Path to configuration file")
	flag.StringVar(&app.config.LogLevel, "log-level", "info", "Log level (trace, debug, info, warn, error)")
	flag.StringVar(&app.config.LogFormat, "log-format", "json", "Log format (json, text)")
	flag.BoolVar(&app.config.MetricsEnabled, "metrics", true, "Enable metrics collection")
	flag.StringVar(&app.config.MetricsAddress, "metrics-address", "0.0.0.0", "Metrics server address")
	flag.IntVar(&app.config.MetricsPort, "metrics-port", 9090, "Metrics server port")
	flag.StringVar(&app.config.Server.Address, "address", "0.0.0.0", "Server listen address")
	flag.IntVar(&app.config.Server.Port, "port", 8080, "Server listen port")
	flag.BoolVar(&app.config.DevMode, "dev", false, "Enable development mode")
	flag.BoolVar(&app.config.Server.TLSEnabled, "tls", false, "Enable TLS")
	flag.StringVar(&app.config.Server.TLSCertFile, "tls-cert", "", "TLS certificate file")
	flag.StringVar(&app.config.Server.TLSKeyFile, "tls-key", "", "TLS key file")
	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "MCPEG - Model Context Protocol Enablement Gateway\\n\\n")
		fmt.Fprintf(os.Stderr, "A high-performance gateway for routing MCP requests to backend services.\\n\\n")
		fmt.Fprintf(os.Stderr, "Usage:\\n")
		fmt.Fprintf(os.Stderr, "  %s [options]\\n\\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Examples:\\n")
		fmt.Fprintf(os.Stderr, "  # Start with default settings\\n")
		fmt.Fprintf(os.Stderr, "  %s\\n\\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Start with custom configuration\\n")
		fmt.Fprintf(os.Stderr, "  %s -config config.yaml\\n\\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Start with TLS enabled\\n")
		fmt.Fprintf(os.Stderr, "  %s -tls -tls-cert cert.pem -tls-key key.pem\\n\\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Start in development mode\\n")
		fmt.Fprintf(os.Stderr, "  %s -dev -log-level debug\\n\\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\\n")
		flag.PrintDefaults()
	}
	
	flag.Parse()
	
	// Validate required flags
	if app.config.Server.TLSEnabled {
		if app.config.Server.TLSCertFile == "" || app.config.Server.TLSKeyFile == "" {
			return fmt.Errorf("TLS certificate and key files are required when TLS is enabled")
		}
	}
	
	return nil
}

// loadConfig loads configuration from file if specified
func (app *GatewayApp) loadConfig() error {
	// Set defaults
	app.config.Server = app.getDefaultServerConfig()
	
	// Load from configuration file if specified
	if app.config.ConfigFile != "" {
		// TODO: Implement YAML configuration loading
		fmt.Printf("Loading configuration from: %s (not yet implemented)\\n", app.config.ConfigFile)
	}
	
	// Override with development mode settings
	if app.config.DevMode {
		app.config.LogLevel = "debug"
		app.config.Server.EnableHealthEndpoints = true
		app.config.Server.EnableMetricsEndpoint = true
		app.config.Server.EnableAdminEndpoints = true
	}
	
	return nil
}

// initialize initializes all application components
func (app *GatewayApp) initialize() error {
	// Initialize logger
	app.logger = logging.New("mcpeg_gateway")
	
	// Initialize metrics
	if app.config.MetricsEnabled {
		app.metrics = &consoleMetrics{logger: app.logger}
	} else {
		app.metrics = &noOpMetrics{}
	}
	
	// Initialize validator
	app.validator = validation.NewValidator(app.logger, app.metrics)
	
	// Initialize health manager
	app.healthMgr = health.NewHealthManager(app.logger, app.metrics, "1.0.0")
	
	// Initialize gateway server
	app.server = server.NewGatewayServer(
		app.config.Server,
		app.logger,
		app.metrics,
		app.validator,
		app.healthMgr,
	)
	
	app.logger.Info("application_initialized",
		"version", "1.0.0",
		"address", fmt.Sprintf("%s:%d", app.config.Server.Address, app.config.Server.Port),
		"tls_enabled", app.config.Server.TLSEnabled,
		"dev_mode", app.config.DevMode)
	
	return nil
}

// setupSignalHandling sets up graceful shutdown on signals
func (app *GatewayApp) setupSignalHandling(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		sig := <-sigChan
		app.logger.Info("signal_received", "signal", sig.String())
		cancel()
	}()
}

// start starts the gateway server
func (app *GatewayApp) start(ctx context.Context) error {
	app.logger.Info("mcpeg_gateway_starting",
		"address", fmt.Sprintf("%s:%d", app.config.Server.Address, app.config.Server.Port),
		"tls_enabled", app.config.Server.TLSEnabled,
		"metrics_enabled", app.config.MetricsEnabled)
	
	// Print startup banner
	app.printBanner()
	
	// Start the server
	if err := app.server.Start(ctx); err != nil {
		app.logger.Error("gateway_start_failed", "error", err)
		return err
	}
	
	return nil
}

// printBanner prints the startup banner
func (app *GatewayApp) printBanner() {
	banner := `
 __  __  ____  ____   ______ _____ 
|  \/  |/ ___||  _ \ |  ____/ ____|
| |\/| | |    | |_) || |__ | |  __ 
| |  | | |    |  __/ |  __|| | |_ |
| |  | | |____| |    | |___| |__| |
|_|  |_|\_____|_|    |______\_____|

Model Context Protocol Enablement Gateway
Version: 1.0.0
API-First • Production-Ready • MCP-Compliant
`
	
	fmt.Print(banner)
	fmt.Printf("Starting on %s:%d (TLS: %v)\\n\\n",
		app.config.Server.Address,
		app.config.Server.Port,
		app.config.Server.TLSEnabled)
}

// getDefaultServerConfig returns default server configuration
func (app *GatewayApp) getDefaultServerConfig() server.ServerConfig {
	return server.ServerConfig{
		Address:         "0.0.0.0",
		Port:            8080,
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownTimeout: 30 * time.Second,
		TLSEnabled:      false,
		CORSEnabled:     true,
		CORSAllowOrigins: []string{"*"},
		CORSAllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		CORSAllowHeaders: []string{"Content-Type", "Authorization", "X-Client-ID", "X-Session-ID"},
		EnableCompression:     true,
		EnableRateLimit:       false,
		RateLimitRPS:         1000,
		EnableHealthEndpoints: true,
		EnableMetricsEndpoint: true,
		EnableAdminEndpoints:  false,
	}
}

// Simple console metrics implementation for the CLI
type consoleMetrics struct {
	logger logging.Logger
}

func (m *consoleMetrics) Inc(name string, labels ...string) {
	m.logger.Debug("metric_inc", "name", name, "labels", labels)
}

func (m *consoleMetrics) Add(name string, value float64, labels ...string) {
	m.logger.Debug("metric_add", "name", name, "value", value, "labels", labels)
}

func (m *consoleMetrics) Set(name string, value float64, labels ...string) {
	m.logger.Debug("metric_set", "name", name, "value", value, "labels", labels)
}

func (m *consoleMetrics) Observe(name string, value float64, labels ...string) {
	m.logger.Debug("metric_observe", "name", name, "value", value, "labels", labels)
}

func (m *consoleMetrics) Time(name string, labels ...string) metrics.Timer {
	return &consoleTimer{start: time.Now(), name: name, metrics: m}
}

func (m *consoleMetrics) WithLabels(labels map[string]string) metrics.Metrics {
	return m
}

func (m *consoleMetrics) WithPrefix(prefix string) metrics.Metrics {
	return m
}

func (m *consoleMetrics) GetStats(name string) metrics.MetricStats {
	return metrics.MetricStats{}
}

func (m *consoleMetrics) GetAllStats() map[string]metrics.MetricStats {
	return make(map[string]metrics.MetricStats)
}

type consoleTimer struct {
	start   time.Time
	name    string
	metrics *consoleMetrics
}

func (t *consoleTimer) Duration() time.Duration {
	return time.Since(t.start)
}

func (t *consoleTimer) Stop() time.Duration {
	duration := time.Since(t.start)
	t.metrics.logger.Debug("timer_stopped", "name", t.name, "duration", duration)
	return duration
}

// No-op metrics implementation
type noOpMetrics struct{}

func (m *noOpMetrics) Inc(name string, labels ...string)                                {}
func (m *noOpMetrics) Add(name string, value float64, labels ...string)                {}
func (m *noOpMetrics) Set(name string, value float64, labels ...string)                {}
func (m *noOpMetrics) Observe(name string, value float64, labels ...string)            {}
func (m *noOpMetrics) Time(name string, labels ...string) metrics.Timer                { return &noOpTimer{} }
func (m *noOpMetrics) WithLabels(labels map[string]string) metrics.Metrics             { return m }
func (m *noOpMetrics) WithPrefix(prefix string) metrics.Metrics                        { return m }
func (m *noOpMetrics) GetStats(name string) metrics.MetricStats                        { return metrics.MetricStats{} }
func (m *noOpMetrics) GetAllStats() map[string]metrics.MetricStats                     { return make(map[string]metrics.MetricStats) }

type noOpTimer struct{}

func (t *noOpTimer) Duration() time.Duration { return 0 }
func (t *noOpTimer) Stop() time.Duration     { return 0 }