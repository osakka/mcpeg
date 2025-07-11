package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/osakka/mcpeg/internal/server"
	"github.com/osakka/mcpeg/pkg/config"
	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
	"github.com/osakka/mcpeg/pkg/validation"
	"github.com/osakka/mcpeg/pkg/health"
)

// GatewayApp represents the main gateway application
type GatewayApp struct {
	gatewayConfig *config.GatewayConfig
	configLoader  *config.Loader
	logger        logging.Logger
	metrics       metrics.Metrics
	validator     *validation.Validator
	healthMgr     *health.HealthManager
	server        *server.GatewayServer
	
	// Command line flags
	configFile string
	devMode    bool
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
	flag.StringVar(&app.configFile, "config", config.GetDefaultConfigPath(), "Path to configuration file")
	flag.BoolVar(&app.devMode, "dev", false, "Enable development mode")

	// Show help and version flags
	showHelp := flag.Bool("help", false, "Show help")
	showVersion := flag.Bool("version", false, "Show version")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "MCPEG Gateway - Model Context Protocol Enablement Gateway\n\n")
		fmt.Fprintf(os.Stderr, "A high-performance gateway for routing MCP requests to backend services.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  # Start with default settings\n")
		fmt.Fprintf(os.Stderr, "  %s\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Start with custom configuration\n")
		fmt.Fprintf(os.Stderr, "  %s -config config.yaml\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Start in development mode\n")
		fmt.Fprintf(os.Stderr, "  %s -dev\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if *showVersion {
		app.showVersion()
		os.Exit(0)
	}

	return nil
}

// loadConfig loads configuration from file and applies overrides
func (app *GatewayApp) loadConfig() error {
	// Initialize with defaults
	app.gatewayConfig = config.GetDefaults()
	
	// Create logger for config loading (using simple console logger initially)
	app.logger = &simpleLogger{}
	app.configLoader = config.NewLoader(app.logger)

	// Load configuration from file if it exists
	if _, err := os.Stat(app.configFile); err == nil {
		app.logger.Info("config_loading_from_file", "file_path", app.configFile)
		
		opts := &config.LoadOptions{
			EnvPrefix:         "MCPEG",
			AllowEnvOverrides: true,
			Validate:          true,
		}
		
		if err := app.configLoader.LoadFromFile(app.configFile, app.gatewayConfig, opts); err != nil {
			return fmt.Errorf("failed to load configuration from %s: %w", app.configFile, err)
		}
	} else {
		app.logger.Info("config_file_not_found_using_defaults", 
			"file_path", app.configFile,
			"using_defaults", true)
	}

	// Apply development mode overrides
	if app.devMode {
		app.logger.Info("config_applying_dev_mode_overrides")
		app.applyDevModeOverrides()
	}

	// Validate final configuration
	if err := app.gatewayConfig.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	app.logger.Info("config_loading_completed", 
		"server_address", fmt.Sprintf("%s:%d", app.gatewayConfig.Server.Address, app.gatewayConfig.Server.Port),
		"tls_enabled", app.gatewayConfig.Server.TLS.Enabled,
		"metrics_enabled", app.gatewayConfig.Metrics.Enabled,
		"development_mode", app.gatewayConfig.Development.Enabled)

	return nil
}

// applyDevModeOverrides applies development mode configuration overrides
func (app *GatewayApp) applyDevModeOverrides() {
	app.gatewayConfig.Development.Enabled = true
	app.gatewayConfig.Development.DebugMode = true
	app.gatewayConfig.Development.AdminEndpoints.Enabled = true
	app.gatewayConfig.Logging.Level = "debug"
	app.gatewayConfig.Server.HealthCheck.Detailed = true
	app.gatewayConfig.Metrics.Collection.SystemInterval = 5 * 1000000000 // 5 seconds in nanoseconds
}

// initialize initializes application components
func (app *GatewayApp) initialize() error {
	app.logger.Info("gateway_initialization_started")

	// Initialize proper logger based on configuration
	app.logger = app.createLogger()
	app.configLoader = config.NewLoader(app.logger)

	// Initialize metrics
	app.metrics = app.createMetrics()

	// Initialize validator
	app.validator = validation.NewValidator(app.logger, app.metrics)

	// Initialize health manager
	app.healthMgr = health.NewHealthManager(app.logger, app.metrics, "1.0.0")

	// Create and configure gateway server
	serverConfig := app.gatewayConfig.ToServerConfig()
	app.server = server.NewGatewayServer(
		serverConfig,
		app.logger,
		app.metrics,
		app.validator,
		app.healthMgr,
	)

	app.logger.Info("gateway_initialization_completed",
		"components_initialized", []string{"logger", "metrics", "validator", "health_manager", "server"})

	return nil
}

// createLogger creates a logger based on configuration
func (app *GatewayApp) createLogger() logging.Logger {
	// For now, return a simple console logger
	// In a complete implementation, this would create a proper logger based on config
	return &simpleLogger{}
}

// createMetrics creates a metrics collector based on configuration
func (app *GatewayApp) createMetrics() metrics.Metrics {
	if app.gatewayConfig.Metrics.Enabled {
		return &consoleMetrics{logger: app.logger}
	}
	return &noOpMetrics{}
}

// setupSignalHandling sets up graceful shutdown signal handling
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
	app.logger.Info("gateway_starting",
		"address", fmt.Sprintf("%s:%d", app.gatewayConfig.Server.Address, app.gatewayConfig.Server.Port),
		"tls_enabled", app.gatewayConfig.Server.TLS.Enabled,
		"dev_mode", app.gatewayConfig.Development.Enabled)

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
Version: dev • Built: unknown
API-First • Production-Ready • MCP-Compliant
`

	fmt.Print(banner)
	fmt.Printf("Starting on %s:%d (TLS: %v, Dev: %v)\n\n",
		app.gatewayConfig.Server.Address,
		app.gatewayConfig.Server.Port,
		app.gatewayConfig.Server.TLS.Enabled,
		app.gatewayConfig.Development.Enabled)
}

// showVersion shows version information
func (app *GatewayApp) showVersion() {
	fmt.Printf("MCPEG Gateway - Model Context Protocol Enablement Gateway\n")
	fmt.Printf("Version: %s\n", "dev")
	fmt.Printf("Commit:  %s\n", "unknown")
	fmt.Printf("Built:   %s\n", "unknown")
	fmt.Printf("Go:      %s\n", "go1.24.2")
}

// Simple logger implementation for configuration loading
type simpleLogger struct{}

func (l *simpleLogger) WithComponent(component string) logging.Logger { return l }
func (l *simpleLogger) WithContext(ctx interface{}) logging.Logger    { return l }
func (l *simpleLogger) WithTraceID(traceID string) logging.Logger     { return l }
func (l *simpleLogger) WithSpanID(spanID string) logging.Logger       { return l }

func (l *simpleLogger) Trace(msg string, fields ...interface{}) {
	fmt.Printf("[TRACE] %s %v\n", msg, fields)
}

func (l *simpleLogger) Debug(msg string, fields ...interface{}) {
	fmt.Printf("[DEBUG] %s %v\n", msg, fields)
}

func (l *simpleLogger) Info(msg string, fields ...interface{}) {
	fmt.Printf("[INFO] %s %v\n", msg, fields)
}

func (l *simpleLogger) Warn(msg string, fields ...interface{}) {
	fmt.Printf("[WARN] %s %v\n", msg, fields)
}

func (l *simpleLogger) Error(msg string, fields ...interface{}) {
	fmt.Printf("[ERROR] %s %v\n", msg, fields)
}

// Simple metrics implementations
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
	return &consoleTimer{start: 0, name: name, metrics: m}
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
	start   int64
	name    string
	metrics *consoleMetrics
}

func (t *consoleTimer) Duration() int64 {
	return 0
}

func (t *consoleTimer) Stop() int64 {
	duration := int64(0)
	t.metrics.logger.Debug("timer_stopped", "name", t.name, "duration", duration)
	return duration
}

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

func (t *noOpTimer) Duration() int64 { return 0 }
func (t *noOpTimer) Stop() int64     { return 0 }