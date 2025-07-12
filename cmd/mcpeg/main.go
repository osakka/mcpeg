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
	"github.com/osakka/mcpeg/pkg/codegen"
	"github.com/osakka/mcpeg/pkg/config"
	"github.com/osakka/mcpeg/pkg/health"
	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
	"github.com/osakka/mcpeg/pkg/paths"
	"github.com/osakka/mcpeg/pkg/process"
	"github.com/osakka/mcpeg/pkg/validation"
)

// Build information (set by build system)
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
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

	// Process management
	pidManager    *process.PIDManager
	daemonManager *process.DaemonManager

	// Command line flags
	configFile string
	devMode    bool
	daemon     bool
	pidFile    string
	logFile    string
}

// CodegenConfig represents codegen configuration
type CodegenConfig struct {
	// Input options
	SpecFile   string
	SpecURL    string
	SpecFormat string

	// Output options
	OutputDir   string
	PackageName string
	ModulePath  string

	// Generation options
	GenerateTypes      bool
	GenerateHandlers   bool
	GenerateClients    bool
	GenerateValidators bool
	GenerateTests      bool

	// Code style options
	UsePointers    bool
	JSONTags       bool
	ValidationTags bool

	// Validation options
	StrictValidation bool
	ValidateOnly     bool

	// Debug options
	Verbose bool
	Debug   bool
}

func main() {
	if len(os.Args) < 2 {
		showHelp()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "gateway", "server", "serve":
		runGateway(os.Args[2:])
	case "codegen", "generate", "gen":
		runCodegen(os.Args[2:])
	case "validate", "val":
		runValidate(os.Args[2:])
	case "version", "ver", "-v", "--version":
		showVersion()
	case "help", "-h", "--help":
		showHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		showHelp()
		os.Exit(1)
	}
}

// runGateway runs the gateway (this is the main daemon functionality)
func runGateway(args []string) {
	app := &GatewayApp{}

	// Parse command line flags
	if err := app.parseFlags(args); err != nil {
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
		// Clean up PID file on startup failure
		if app.pidManager != nil {
			app.pidManager.RemovePID()
		}
		os.Exit(1)
	}

	// Wait for context cancellation (shutdown signal)
	<-ctx.Done()

	// Cleanup PID file on shutdown
	if app.pidManager != nil {
		app.pidManager.RemovePID()
	}

	app.logger.Info("gateway_shutdown_complete")
}

// parseFlags parses command line flags
func (app *GatewayApp) parseFlags(args []string) error {
	// Create dedicated flag set for gateway subcommand
	flagSet := flag.NewFlagSet("gateway", flag.ExitOnError)

	// Configuration flags
	flagSet.StringVar(&app.configFile, "config", paths.GetDefaultConfigPath(), "Path to configuration file")
	flagSet.BoolVar(&app.devMode, "dev", false, "Enable development mode")

	// Daemon mode flags
	flagSet.BoolVar(&app.daemon, "daemon", false, "Run in daemon mode (background)")
	flagSet.StringVar(&app.pidFile, "pid-file", paths.GetDefaultPIDFile(), "Path to PID file")
	flagSet.StringVar(&app.logFile, "log-file", paths.GetDefaultLogFile(), "Path to log file")

	// Show help and version flags
	showHelp := flagSet.Bool("help", false, "Show help")
	showVersion := flagSet.Bool("version", false, "Show version")

	// Control flags
	stop := flagSet.Bool("stop", false, "Stop running daemon")
	restart := flagSet.Bool("restart", false, "Restart daemon")
	status := flagSet.Bool("status", false, "Show daemon status")
	logRotate := flagSet.Bool("log-rotate", false, "Signal daemon to rotate logs")

	flagSet.Usage = func() {
		fmt.Fprintf(os.Stderr, "MCpeg Gateway - Model Context Protocol Enablement Gateway\n")
		fmt.Fprintf(os.Stderr, "Pronounced \"MC peg\" • The Peg That Connects Model Contexts\n\n")
		fmt.Fprintf(os.Stderr, "A high-performance gateway for routing MCP requests to backend services.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  mcpeg gateway [options]\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  # Start with default settings\n")
		fmt.Fprintf(os.Stderr, "  mcpeg gateway\n\n")
		fmt.Fprintf(os.Stderr, "  # Start with custom configuration\n")
		fmt.Fprintf(os.Stderr, "  mcpeg gateway -config config.yaml\n\n")
		fmt.Fprintf(os.Stderr, "  # Start in development mode\n")
		fmt.Fprintf(os.Stderr, "  mcpeg gateway -dev\n\n")
		fmt.Fprintf(os.Stderr, "  # Start as daemon\n")
		fmt.Fprintf(os.Stderr, "  mcpeg gateway -daemon\n\n")
		fmt.Fprintf(os.Stderr, "  # Control daemon\n")
		fmt.Fprintf(os.Stderr, "  mcpeg gateway -stop\n")
		fmt.Fprintf(os.Stderr, "  mcpeg gateway -restart\n")
		fmt.Fprintf(os.Stderr, "  mcpeg gateway -status\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flagSet.PrintDefaults()
	}

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	if *showHelp {
		flagSet.Usage()
		os.Exit(0)
	}

	if *showVersion {
		app.showVersion()
		os.Exit(0)
	}

	// Handle control commands
	if *stop || *restart || *status || *logRotate {
		return app.handleControlCommand(*stop, *restart, *status, *logRotate)
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

	// Disable TLS for development mode
	app.gatewayConfig.Server.TLS.Enabled = false
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
	app.healthMgr = health.NewHealthManager(app.logger, app.metrics, Version)

	// Initialize process management
	app.pidManager = process.NewPIDManager(app.pidFile, app.logger)

	// Initialize daemon manager
	daemonConfig := process.DaemonConfig{
		PIDFile:    app.pidFile,
		LogFile:    app.logFile,
		WorkingDir: "",
		Background: app.daemon,
	}

	app.daemonManager = process.NewDaemonManager(daemonConfig, app.logger)

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
		"components_initialized", []string{"logger", "metrics", "validator", "health_manager", "server", "pid_manager", "daemon_manager"})

	return nil
}

// createLogger creates a logger based on configuration
func (app *GatewayApp) createLogger() logging.Logger {
	// If running as daemon and log file is specified, use file logger
	if process.IsDaemon() && app.logFile != "" {
		// Create file logger configuration
		fileConfig := logging.FileLoggerConfig{
			FilePath:    app.logFile,
			MaxSize:     10 * 1024 * 1024, // 10MB
			MaxBackups:  5,
			MaxAge:      7, // 7 days
			Compress:    true,
			BufferSize:  4096,
			SyncOnWrite: false,
		}

		// Create production logger with both console and file output
		if prodLogger, err := logging.NewProductionLogger(app.gatewayConfig.Logging.Level, fileConfig); err == nil {
			return prodLogger
		}
	}

	// Fallback to simple console logger
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
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGUSR1)

	go func() {
		for {
			sig := <-sigChan
			app.logger.Info("signal_received", "signal", sig.String())

			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				// Graceful shutdown
				app.logger.Info("initiating_graceful_shutdown")
				cancel()
				return
			case syscall.SIGHUP:
				// Config reload signal
				app.logger.Info("config_reload_signal_received")
				// TODO: Implement config reload
			case syscall.SIGUSR1:
				// Log rotation signal
				app.logger.Info("log_rotation_signal_received")
				// TODO: Implement log rotation
			}
		}
	}()
}

// start starts the gateway server
func (app *GatewayApp) start(ctx context.Context) error {
	app.logger.Info("gateway_starting",
		"address", fmt.Sprintf("%s:%d", app.gatewayConfig.Server.Address, app.gatewayConfig.Server.Port),
		"tls_enabled", app.gatewayConfig.Server.TLS.Enabled,
		"dev_mode", app.gatewayConfig.Development.Enabled,
		"daemon_mode", app.daemon)

	// Handle daemon mode
	if app.daemon {
		if err := app.daemonManager.Daemonize(); err != nil {
			return fmt.Errorf("failed to daemonize: %w", err)
		}
		// If we reach here, we're in the daemon child process
		// Detach from terminal and setup daemon environment
		if err := app.daemonManager.DetachFromTerminal(); err != nil {
			app.logger.Error("failed_to_detach_from_terminal", "error", err)
		}
	}

	// Write PID file
	if err := app.pidManager.WritePID(); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	// Setup cleanup on exit
	app.pidManager.SetupCleanupOnExit()

	// Print startup banner (only in non-daemon mode)
	if !app.daemon {
		app.printBanner()
	}

	// Start the server
	if err := app.server.Start(ctx); err != nil {
		app.logger.Error("gateway_start_failed", "error", err)
		// Clean up PID file on server start failure
		app.pidManager.RemovePID()
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
Version: %s • Commit: %s • Built: %s
API-First • Production-Ready • MCP-Compliant
`

	fmt.Printf(banner, Version, Commit, BuildTime)
	fmt.Printf("Starting on %s:%d (TLS: %v, Dev: %v, Daemon: %v)\n",
		app.gatewayConfig.Server.Address,
		app.gatewayConfig.Server.Port,
		app.gatewayConfig.Server.TLS.Enabled,
		app.gatewayConfig.Development.Enabled,
		app.daemon)

	if app.daemon {
		fmt.Printf("PID File: %s\n", app.pidFile)
		fmt.Printf("Log File: %s\n", app.logFile)
	}

	fmt.Println()
}

// showVersion shows version information
func (app *GatewayApp) showVersion() {
	fmt.Printf("MCpeg Gateway - Model Context Protocol Enablement Gateway\n")
	fmt.Printf("Version: %s\n", Version)
	fmt.Printf("Commit:  %s\n", Commit)
	fmt.Printf("Built:   %s\n", BuildTime)
	fmt.Printf("Go:      %s\n", "go1.24.2")
}

// runCodegen runs the code generation tool
func runCodegen(args []string) {
	fs := flag.NewFlagSet("codegen", flag.ExitOnError)

	var config CodegenConfig

	// Input options
	fs.StringVar(&config.SpecFile, "spec-file", "", "Path to OpenAPI specification file")
	fs.StringVar(&config.SpecURL, "spec-url", "", "URL to OpenAPI specification")
	fs.StringVar(&config.SpecFormat, "format", "", "Specification format (json|yaml), auto-detected if not specified")

	// Output options
	fs.StringVar(&config.OutputDir, "output", "build/generated", "Output directory for generated code")
	fs.StringVar(&config.PackageName, "package", "generated", "Go package name for generated code")
	fs.StringVar(&config.ModulePath, "module", "github.com/osakka/mcpeg", "Go module path")

	// Generation options
	fs.BoolVar(&config.GenerateTypes, "types", true, "Generate type definitions")
	fs.BoolVar(&config.GenerateHandlers, "handlers", true, "Generate HTTP handlers")
	fs.BoolVar(&config.GenerateClients, "clients", true, "Generate client code")
	fs.BoolVar(&config.GenerateValidators, "validators", true, "Generate validation functions")
	fs.BoolVar(&config.GenerateTests, "tests", false, "Generate test code")

	// Code style options
	fs.BoolVar(&config.UsePointers, "pointers", true, "Use pointers for optional fields")
	fs.BoolVar(&config.JSONTags, "json-tags", true, "Add JSON struct tags")
	fs.BoolVar(&config.ValidationTags, "validation-tags", true, "Add validation struct tags")

	// Validation options
	fs.BoolVar(&config.StrictValidation, "strict", false, "Enable strict validation")
	fs.BoolVar(&config.ValidateOnly, "validate-only", false, "Only validate specification without generating code")

	// Debug options
	fs.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")
	fs.BoolVar(&config.Debug, "debug", false, "Enable debug logging")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "MCpeg Code Generator\n\n")
		fmt.Fprintf(os.Stderr, "Generates Go code from OpenAPI specifications following API-first principles.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  mcpeg codegen [options]\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  mcpeg codegen -spec-file api/openapi/mcp-gateway.yaml\n")
		fmt.Fprintf(os.Stderr, "  mcpeg codegen -spec-url https://api.example.com/openapi.yaml\n")
		fmt.Fprintf(os.Stderr, "  mcpeg codegen -spec-file api.yaml -validate-only\n")
		fmt.Fprintf(os.Stderr, "  mcpeg codegen -spec-file api.yaml -output internal/generated\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	// Validate command line arguments
	if config.SpecFile == "" && config.SpecURL == "" {
		fmt.Fprintf(os.Stderr, "Error: must specify either -spec-file or -spec-url\n\n")
		fs.Usage()
		os.Exit(1)
	}

	if config.SpecFile != "" && config.SpecURL != "" {
		fmt.Fprintf(os.Stderr, "Error: cannot specify both -spec-file and -spec-url\n\n")
		fs.Usage()
		os.Exit(1)
	}

	// Execute code generation
	if err := executeCodegen(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runValidate validates OpenAPI specifications
func runValidate(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)

	var specFile string
	var specURL string
	var strict bool
	var verbose bool

	fs.StringVar(&specFile, "spec-file", "", "Path to OpenAPI specification file")
	fs.StringVar(&specURL, "spec-url", "", "URL to OpenAPI specification")
	fs.BoolVar(&strict, "strict", false, "Enable strict validation")
	fs.BoolVar(&verbose, "verbose", false, "Enable verbose output")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "MCpeg OpenAPI Validator\n\n")
		fmt.Fprintf(os.Stderr, "Validates OpenAPI specifications for compliance and best practices.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  mcpeg validate [options]\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  mcpeg validate -spec-file api/openapi/mcp-gateway.yaml\n")
		fmt.Fprintf(os.Stderr, "  mcpeg validate -spec-url https://api.example.com/openapi.yaml\n")
		fmt.Fprintf(os.Stderr, "  mcpeg validate -spec-file api.yaml -strict\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	// Validate arguments
	if specFile == "" && specURL == "" {
		fmt.Fprintf(os.Stderr, "Error: must specify either -spec-file or -spec-url\n\n")
		fs.Usage()
		os.Exit(1)
	}

	// Execute validation
	config := CodegenConfig{
		SpecFile:         specFile,
		SpecURL:          specURL,
		StrictValidation: strict,
		ValidateOnly:     true,
		Verbose:          verbose,
	}

	if err := executeCodegen(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func executeCodegen(config CodegenConfig) error {
	// Set up logging
	logger := setupCodegenLogging(config)

	logger.Info("mcpeg_codegen_starting",
		"spec_file", config.SpecFile,
		"spec_url", config.SpecURL,
		"output_dir", config.OutputDir,
		"package", config.PackageName,
		"validate_only", config.ValidateOnly)

	// Set up metrics (no-op for codegen)
	metrics := &noOpMetrics{}

	// Set up validator
	validator := validation.NewValidator(logger, metrics)

	// Set up parser
	parser := codegen.NewOpenAPIParser(logger, validator)

	ctx := context.Background()

	// Parse OpenAPI specification
	var parseResult *codegen.ParseResult
	var err error

	if config.SpecFile != "" {
		parseResult, err = parser.ParseFromFile(ctx, config.SpecFile)
	} else {
		parseResult, err = parser.ParseFromURL(ctx, config.SpecURL)
	}

	if err != nil {
		return fmt.Errorf("failed to parse specification: %w", err)
	}

	// Report parsing results
	reportParseResults(parseResult)

	// If validation failed and strict mode, exit
	if !parseResult.Valid && config.StrictValidation {
		return fmt.Errorf("specification validation failed in strict mode")
	}

	// If validate-only mode, exit here
	if config.ValidateOnly {
		if parseResult.Valid {
			logger.Info("specification_validation_passed")
			fmt.Println("✅ OpenAPI specification is valid")
		} else {
			logger.Warn("specification_validation_failed_but_continuing")
			fmt.Println("⚠️  OpenAPI specification has validation errors but is parseable")
		}
		return nil
	}

	// Set up code generator
	generator := codegen.NewCodeGenerator(logger, metrics)

	// Generate code
	logger.Info("starting_code_generation")
	fmt.Println("🔧 Generating Go code from OpenAPI specification...")

	generated, err := generator.GenerateFromSpec(ctx, parseResult.Spec)
	if err != nil {
		return fmt.Errorf("code generation failed: %w", err)
	}

	// Write generated code to files
	logger.Info("writing_generated_code", "output_dir", config.OutputDir)
	fmt.Printf("📁 Writing generated code to %s...\n", config.OutputDir)

	if err := generator.WriteCode(ctx, generated); err != nil {
		return fmt.Errorf("failed to write generated code: %w", err)
	}

	// Report generation results
	reportGenerationResults(generated)

	logger.Info("mcpeg_codegen_completed")
	fmt.Println("✅ Code generation completed successfully!")

	return nil
}

func setupCodegenLogging(config CodegenConfig) logging.Logger {
	level := "info"
	if config.Debug {
		level = "debug"
	} else if config.Verbose {
		level = "info"
	} else {
		level = "warn"
	}

	return &consoleLogger{level: level}
}

func reportParseResults(result *codegen.ParseResult) {
	fmt.Printf("📋 Parsed OpenAPI specification:\n")
	fmt.Printf("   Title: %s\n", result.Spec.Info.Title)
	fmt.Printf("   Version: %s\n", result.Spec.Info.Version)
	fmt.Printf("   Format: %s\n", result.Metadata.Format)
	fmt.Printf("   Size: %d bytes\n", result.Metadata.Size)
	fmt.Printf("   Parse time: %v\n", result.Metadata.ParseDuration)
	fmt.Printf("   Paths: %d\n", len(result.Spec.Paths))
	fmt.Printf("   Schemas: %d\n", len(result.Spec.Components.Schemas))

	if len(result.Errors) > 0 {
		fmt.Printf("\n❌ Validation errors (%d):\n", len(result.Errors))
		for _, err := range result.Errors {
			fmt.Printf("   • %s: %s (%s)\n", err.Path, err.Message, err.Code)
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Printf("\n⚠️  Validation warnings (%d):\n", len(result.Warnings))
		for _, warn := range result.Warnings {
			fmt.Printf("   • %s: %s (%s)\n", warn.Path, warn.Message, warn.Code)
		}
	}

	if result.Valid {
		fmt.Println("\n✅ Specification is valid")
	} else {
		fmt.Println("\n❌ Specification has validation errors")
	}

	fmt.Println()
}

func reportGenerationResults(generated *codegen.GeneratedCode) {
	fmt.Printf("📦 Generated code summary:\n")
	fmt.Printf("   Package: %s\n", generated.Package)
	fmt.Printf("   Types: %d\n", len(generated.Types))
	fmt.Printf("   Functions: %d\n", len(generated.Functions))
	fmt.Printf("   Constants: %d\n", len(generated.Constants))
	fmt.Printf("   Imports: %d\n", len(generated.Imports))

	if len(generated.Types) > 0 {
		fmt.Printf("\n📋 Generated types:\n")
		for _, t := range generated.Types {
			fmt.Printf("   • %s (%s)\n", t.Name, t.Type)
		}
	}

	if len(generated.Functions) > 0 {
		fmt.Printf("\n🔧 Generated functions:\n")
		for _, f := range generated.Functions {
			paramCount := len(f.Parameters)
			returnCount := len(f.Returns)
			fmt.Printf("   • %s (%d params, %d returns)\n", f.Name, paramCount, returnCount)
		}
	}

	fmt.Println()
}

func showVersion() {
	fmt.Printf("MCpeg - Model Context Protocol Enablement Gateway\n")
	fmt.Printf("Pronounced \"MC peg\" • The Peg That Connects Model Contexts\n\n")
	fmt.Printf("Version: %s\n", Version)
	fmt.Printf("Commit:  %s\n", Commit)
	fmt.Printf("Built:   %s\n", BuildTime)
	fmt.Printf("Go:      %s\n", "go1.24.2")
}

func showHelp() {
	fmt.Printf("MCpeg - Model Context Protocol Enablement Gateway\n")
	fmt.Printf("Pronounced \"MC peg\" • The Peg That Connects Model Contexts\n\n")
	fmt.Printf("A unified tool for MCP gateway operations and development.\n\n")
	fmt.Printf("Usage:\n")
	fmt.Printf("  mcpeg <command> [options]\n\n")
	fmt.Printf("Commands:\n")
	fmt.Printf("  gateway    Start the MCP gateway server\n")
	fmt.Printf("  codegen    Generate Go code from OpenAPI specifications\n")
	fmt.Printf("  validate   Validate OpenAPI specifications\n")
	fmt.Printf("  version    Show version information\n")
	fmt.Printf("  help       Show this help message\n\n")
	fmt.Printf("Examples:\n")
	fmt.Printf("  mcpeg gateway -dev                                    # Start development server\n")
	fmt.Printf("  mcpeg gateway -daemon                                 # Start as daemon\n")
	fmt.Printf("  mcpeg gateway -config production.yaml                # Custom configuration\n")
	fmt.Printf("  mcpeg gateway -stop                                   # Stop daemon\n")
	fmt.Printf("  mcpeg gateway -status                                 # Check daemon status\n")
	fmt.Printf("  mcpeg codegen -spec-file api/openapi/mcp-gateway.yaml # Generate code\n")
	fmt.Printf("  mcpeg validate -spec-file api/openapi/mcp-gateway.yaml # Validate spec\n")
	fmt.Printf("  mcpeg version                                         # Show version\n\n")
	fmt.Printf("Use 'mcpeg <command> -h' for more information about a command.\n")
}

// Simple logger implementation for configuration loading
type simpleLogger struct{}

func (l *simpleLogger) WithComponent(component string) logging.Logger  { return l }
func (l *simpleLogger) WithContext(ctx context.Context) logging.Logger { return l }
func (l *simpleLogger) WithTraceID(traceID string) logging.Logger      { return l }
func (l *simpleLogger) WithSpanID(spanID string) logging.Logger        { return l }

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

// Simple logger implementations (reused from existing code)
type consoleLogger struct {
	level string
}

func (l *consoleLogger) WithComponent(component string) logging.Logger {
	return l
}

func (l *consoleLogger) WithContext(ctx context.Context) logging.Logger {
	return l
}

func (l *consoleLogger) WithTraceID(traceID string) logging.Logger {
	return l
}

func (l *consoleLogger) WithSpanID(spanID string) logging.Logger {
	return l
}

func (l *consoleLogger) Trace(msg string, fields ...interface{}) {
	if l.level == "trace" {
		fmt.Printf("[TRACE] %s %v\n", msg, fields)
	}
}

func (l *consoleLogger) Debug(msg string, fields ...interface{}) {
	if l.level == "debug" || l.level == "trace" {
		fmt.Printf("[DEBUG] %s %v\n", msg, fields)
	}
}

func (l *consoleLogger) Info(msg string, fields ...interface{}) {
	if l.level == "debug" || l.level == "info" || l.level == "trace" {
		fmt.Printf("[INFO] %s %v\n", msg, fields)
	}
}

func (l *consoleLogger) Warn(msg string, fields ...interface{}) {
	fmt.Printf("[WARN] %s %v\n", msg, fields)
}

func (l *consoleLogger) Error(msg string, fields ...interface{}) {
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

type noOpMetrics struct{}

func (m *noOpMetrics) Inc(name string, labels ...string)                    {}
func (m *noOpMetrics) Add(name string, value float64, labels ...string)     {}
func (m *noOpMetrics) Set(name string, value float64, labels ...string)     {}
func (m *noOpMetrics) Observe(name string, value float64, labels ...string) {}
func (m *noOpMetrics) Time(name string, labels ...string) metrics.Timer     { return &noOpTimer{} }
func (m *noOpMetrics) WithLabels(labels map[string]string) metrics.Metrics  { return m }
func (m *noOpMetrics) WithPrefix(prefix string) metrics.Metrics             { return m }
func (m *noOpMetrics) GetStats(name string) metrics.MetricStats             { return metrics.MetricStats{} }
func (m *noOpMetrics) GetAllStats() map[string]metrics.MetricStats {
	return make(map[string]metrics.MetricStats)
}

type noOpTimer struct{}

func (t *noOpTimer) Duration() time.Duration { return 0 }
func (t *noOpTimer) Stop() time.Duration     { return 0 }

// handleControlCommand handles daemon control commands
func (app *GatewayApp) handleControlCommand(stop, restart, status, logRotate bool) error {
	// Create temporary PID manager for control operations
	pidManager := process.NewPIDManager(app.pidFile, &simpleLogger{})

	switch {
	case status:
		return app.handleStatusCommand(pidManager)
	case stop:
		return app.handleStopCommand(pidManager)
	case restart:
		return app.handleRestartCommand(pidManager)
	case logRotate:
		return app.handleLogRotateCommand(pidManager)
	default:
		return fmt.Errorf("unknown control command")
	}
}

// handleStatusCommand handles the status command
func (app *GatewayApp) handleStatusCommand(pidManager *process.PIDManager) error {
	status := pidManager.GetProcessStatus()

	fmt.Printf("MCpeg Gateway Status:\n")
	fmt.Printf("  Status: %s\n", func() string {
		if status.Running {
			return "Running"
		}
		return "Stopped"
	}())

	if status.Running {
		fmt.Printf("  PID: %d\n", status.PID)
		fmt.Printf("  Start Time: %s\n", status.StartTime)
		fmt.Printf("  Uptime: %s\n", status.Uptime)
	}

	fmt.Printf("  PID File: %s\n", status.PIDFile)

	return nil
}

// handleStopCommand handles the stop command
func (app *GatewayApp) handleStopCommand(pidManager *process.PIDManager) error {
	isRunning, pid, err := pidManager.IsRunning()
	if err != nil {
		return fmt.Errorf("failed to check if daemon is running: %w", err)
	}

	if !isRunning {
		fmt.Println("MCpeg Gateway is not running")
		return nil
	}

	fmt.Printf("Stopping MCpeg Gateway (PID: %d)...\n", pid)

	if err := pidManager.StopProcess(false); err != nil {
		return fmt.Errorf("failed to stop daemon: %w", err)
	}

	fmt.Println("MCpeg Gateway stopped successfully")
	return nil
}

// handleRestartCommand handles the restart command
func (app *GatewayApp) handleRestartCommand(pidManager *process.PIDManager) error {
	isRunning, pid, err := pidManager.IsRunning()
	if err != nil {
		return fmt.Errorf("failed to check if daemon is running: %w", err)
	}

	if isRunning {
		fmt.Printf("Stopping MCpeg Gateway (PID: %d)...\n", pid)
		if err := pidManager.StopProcess(false); err != nil {
			return fmt.Errorf("failed to stop daemon: %w", err)
		}
		fmt.Println("MCpeg Gateway stopped")
	}

	// Start new daemon process
	fmt.Println("Starting MCpeg Gateway...")

	// Re-exec with daemon flag
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	args := []string{"gateway", "--daemon"}
	if app.configFile != "" {
		args = append(args, "--config", app.configFile)
	}
	if app.pidFile != "" {
		args = append(args, "--pid-file", app.pidFile)
	}
	if app.logFile != "" {
		args = append(args, "--log-file", app.logFile)
	}

	if err := syscall.Exec(execPath, append([]string{execPath}, args...), os.Environ()); err != nil {
		return fmt.Errorf("failed to restart daemon: %w", err)
	}

	return nil
}

// handleLogRotateCommand handles the log rotate command
func (app *GatewayApp) handleLogRotateCommand(pidManager *process.PIDManager) error {
	isRunning, pid, err := pidManager.IsRunning()
	if err != nil {
		return fmt.Errorf("failed to check if daemon is running: %w", err)
	}

	if !isRunning {
		fmt.Println("MCpeg Gateway is not running")
		return nil
	}

	fmt.Printf("Sending log rotation signal to MCpeg Gateway (PID: %d)...\n", pid)

	if err := pidManager.LogRotationSignal(); err != nil {
		return fmt.Errorf("failed to send log rotation signal: %w", err)
	}

	fmt.Println("Log rotation signal sent successfully")
	return nil
}
