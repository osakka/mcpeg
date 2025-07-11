package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/osakka/mcpeg/pkg/codegen"
	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
	"github.com/osakka/mcpeg/pkg/validation"
)

// CodegenCommand represents the codegen CLI command
type CodegenCommand struct {
	// Input options
	SpecFile     string
	SpecURL      string
	SpecFormat   string
	
	// Output options
	OutputDir    string
	PackageName  string
	ModulePath   string
	
	// Generation options
	GenerateTypes     bool
	GenerateHandlers  bool
	GenerateClients   bool
	GenerateValidators bool
	GenerateTests     bool
	
	// Code style options
	UsePointers      bool
	JSONTags         bool
	ValidationTags   bool
	
	// Validation options
	StrictValidation bool
	ValidateOnly     bool
	
	// Debug options
	Verbose          bool
	Debug            bool
}

func main() {
	cmd := &CodegenCommand{}
	
	// Define command line flags
	flag.StringVar(&cmd.SpecFile, "spec-file", "", "Path to OpenAPI specification file")
	flag.StringVar(&cmd.SpecURL, "spec-url", "", "URL to OpenAPI specification")
	flag.StringVar(&cmd.SpecFormat, "format", "", "Specification format (json|yaml), auto-detected if not specified")
	
	flag.StringVar(&cmd.OutputDir, "output", "generated", "Output directory for generated code")
	flag.StringVar(&cmd.PackageName, "package", "generated", "Go package name for generated code")
	flag.StringVar(&cmd.ModulePath, "module", "github.com/osakka/mcpeg", "Go module path")
	
	flag.BoolVar(&cmd.GenerateTypes, "types", true, "Generate type definitions")
	flag.BoolVar(&cmd.GenerateHandlers, "handlers", true, "Generate HTTP handlers")
	flag.BoolVar(&cmd.GenerateClients, "clients", true, "Generate client code")
	flag.BoolVar(&cmd.GenerateValidators, "validators", true, "Generate validation functions")
	flag.BoolVar(&cmd.GenerateTests, "tests", false, "Generate test code")
	
	flag.BoolVar(&cmd.UsePointers, "pointers", true, "Use pointers for optional fields")
	flag.BoolVar(&cmd.JSONTags, "json-tags", true, "Add JSON struct tags")
	flag.BoolVar(&cmd.ValidationTags, "validation-tags", true, "Add validation struct tags")
	
	flag.BoolVar(&cmd.StrictValidation, "strict", false, "Enable strict validation")
	flag.BoolVar(&cmd.ValidateOnly, "validate-only", false, "Only validate specification without generating code")
	
	flag.BoolVar(&cmd.Verbose, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&cmd.Debug, "debug", false, "Enable debug logging")
	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "MCPEG Code Generator\n\n")
		fmt.Fprintf(os.Stderr, "Generates Go code from OpenAPI specifications following API-first principles.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  # Generate from local file\n")
		fmt.Fprintf(os.Stderr, "  %s -spec-file api/openapi/mcp-gateway.yaml -output internal/generated\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Generate from URL\n")
		fmt.Fprintf(os.Stderr, "  %s -spec-url https://api.example.com/openapi.yaml -package api\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Validate only\n")
		fmt.Fprintf(os.Stderr, "  %s -spec-file api/openapi/mcp-gateway.yaml -validate-only\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
	
	flag.Parse()
	
	// Validate command line arguments
	if err := cmd.validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		flag.Usage()
		os.Exit(1)
	}
	
	// Run the command
	if err := cmd.run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// validate validates command line arguments
func (cmd *CodegenCommand) validate() error {
	// Must specify either spec file or URL
	if cmd.SpecFile == "" && cmd.SpecURL == "" {
		return fmt.Errorf("must specify either -spec-file or -spec-url")
	}
	
	if cmd.SpecFile != "" && cmd.SpecURL != "" {
		return fmt.Errorf("cannot specify both -spec-file and -spec-url")
	}
	
	// Validate spec file exists if specified
	if cmd.SpecFile != "" {
		if _, err := os.Stat(cmd.SpecFile); err != nil {
			return fmt.Errorf("spec file not found: %s", cmd.SpecFile)
		}
	}
	
	// Validate package name
	if cmd.PackageName == "" {
		return fmt.Errorf("package name cannot be empty")
	}
	
	// Validate output directory
	if cmd.OutputDir == "" {
		return fmt.Errorf("output directory cannot be empty")
	}
	
	return nil
}

// run executes the code generation
func (cmd *CodegenCommand) run() error {
	// Set up logging
	logger := cmd.setupLogging()
	
	logger.Info("mcpeg_codegen_starting",
		"spec_file", cmd.SpecFile,
		"spec_url", cmd.SpecURL,
		"output_dir", cmd.OutputDir,
		"package", cmd.PackageName,
		"validate_only", cmd.ValidateOnly)
	
	// Set up metrics (no-op for CLI)
	metrics := &noOpMetrics{}
	
	// Set up validator
	validator := validation.NewValidator(logger, metrics)
	
	// Set up parser
	parser := codegen.NewOpenAPIParser(logger, validator)
	
	ctx := context.Background()
	
	// Parse OpenAPI specification
	var parseResult *codegen.ParseResult
	var err error
	
	if cmd.SpecFile != "" {
		parseResult, err = parser.ParseFromFile(ctx, cmd.SpecFile)
	} else {
		parseResult, err = parser.ParseFromURL(ctx, cmd.SpecURL)
	}
	
	if err != nil {
		return fmt.Errorf("failed to parse specification: %w", err)
	}
	
	// Report parsing results
	cmd.reportParseResults(parseResult)
	
	// If validation failed and strict mode, exit
	if !parseResult.Valid && cmd.StrictValidation {
		return fmt.Errorf("specification validation failed in strict mode")
	}
	
	// If validate-only mode, exit here
	if cmd.ValidateOnly {
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
	
	// Configure generator
	if err := cmd.configureGenerator(generator); err != nil {
		return fmt.Errorf("failed to configure generator: %w", err)
	}
	
	// Generate code
	logger.Info("starting_code_generation")
	fmt.Println("🔧 Generating Go code from OpenAPI specification...")
	
	generated, err := generator.GenerateFromSpec(ctx, parseResult.Spec)
	if err != nil {
		return fmt.Errorf("code generation failed: %w", err)
	}
	
	// Write generated code to files
	logger.Info("writing_generated_code", "output_dir", cmd.OutputDir)
	fmt.Printf("📁 Writing generated code to %s...\n", cmd.OutputDir)
	
	if err := generator.WriteCode(ctx, generated); err != nil {
		return fmt.Errorf("failed to write generated code: %w", err)
	}
	
	// Report generation results
	cmd.reportGenerationResults(generated)
	
	logger.Info("mcpeg_codegen_completed")
	fmt.Println("✅ Code generation completed successfully!")
	
	return nil
}

// setupLogging configures logging based on command line flags
func (cmd *CodegenCommand) setupLogging() logging.Logger {
	level := "info"
	if cmd.Debug {
		level = "debug"
	} else if cmd.Verbose {
		level = "info"
	} else {
		level = "warn"
	}
	
	// Create simple console logger for CLI
	return &consoleLogger{level: level}
}

// configureGenerator configures the code generator
func (cmd *CodegenCommand) configureGenerator(generator *codegen.CodeGenerator) error {
	// This would configure the generator with command line options
	// For now, we'll use a simple approach
	return nil
}

// reportParseResults reports the results of parsing
func (cmd *CodegenCommand) reportParseResults(result *codegen.ParseResult) {
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

// reportGenerationResults reports the results of code generation
func (cmd *CodegenCommand) reportGenerationResults(generated *codegen.GeneratedCode) {
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

// consoleLogger implements a simple console logger for the CLI
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

// noOpMetrics implements a no-op metrics collector for the CLI
type noOpMetrics struct{}

func (m *noOpMetrics) Inc(name string, labels ...string) {}
func (m *noOpMetrics) Add(name string, value float64, labels ...string) {}
func (m *noOpMetrics) Set(name string, value float64, labels ...string) {}
func (m *noOpMetrics) Observe(name string, value float64, labels ...string) {}
func (m *noOpMetrics) Time(name string, labels ...string) metrics.Timer { return &noOpTimer{} }
func (m *noOpMetrics) WithLabels(labels map[string]string) metrics.Metrics { return m }
func (m *noOpMetrics) WithPrefix(prefix string) metrics.Metrics { return m }
func (m *noOpMetrics) GetStats(name string) metrics.MetricStats { return metrics.MetricStats{} }
func (m *noOpMetrics) GetAllStats() map[string]metrics.MetricStats { return make(map[string]metrics.MetricStats) }

// noOpTimer implements a no-op timer for the CLI
type noOpTimer struct{}
func (t *noOpTimer) Duration() time.Duration { return 0 }
func (t *noOpTimer) Stop() time.Duration { return 0 }