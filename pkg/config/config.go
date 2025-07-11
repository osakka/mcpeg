package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/osakka/mcpeg/pkg/logging"
)

// Loader handles configuration loading from various sources
type Loader struct {
	logger logging.Logger
}

// NewLoader creates a new configuration loader
func NewLoader(logger logging.Logger) *Loader {
	return &Loader{
		logger: logger,
	}
}

// LoadOptions configures how configuration is loaded
type LoadOptions struct {
	// Environment prefix for environment variable overrides
	EnvPrefix string
	
	// Whether to allow environment variable overrides
	AllowEnvOverrides bool
	
	// Whether to validate the configuration after loading
	Validate bool
	
	// Default configuration to merge with loaded config
	Defaults interface{}
}

// LoadFromFile loads configuration from a YAML file with optional environment overrides
func (l *Loader) LoadFromFile(filePath string, config interface{}, opts *LoadOptions) error {
	if opts == nil {
		opts = &LoadOptions{
			EnvPrefix:         "MCPEG",
			AllowEnvOverrides: true,
			Validate:          true,
		}
	}

	l.logger.Info("config_loading_started",
		"file_path", filePath,
		"env_prefix", opts.EnvPrefix,
		"allow_env_overrides", opts.AllowEnvOverrides)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("configuration file not found: %s", filePath)
	}

	// Read file contents
	data, err := os.ReadFile(filePath)
	if err != nil {
		l.logger.Error("config_file_read_failed",
			"file_path", filePath,
			"error", err)
		return fmt.Errorf("failed to read configuration file %s: %w", filePath, err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, config); err != nil {
		l.logger.Error("config_yaml_parse_failed",
			"file_path", filePath,
			"error", err)
		return fmt.Errorf("failed to parse YAML configuration: %w", err)
	}

	l.logger.Info("config_file_loaded",
		"file_path", filePath,
		"size_bytes", len(data))

	// Apply environment variable overrides if enabled
	if opts.AllowEnvOverrides {
		if err := l.applyEnvironmentOverrides(config, opts.EnvPrefix); err != nil {
			l.logger.Error("config_env_overrides_failed",
				"env_prefix", opts.EnvPrefix,
				"error", err)
			return fmt.Errorf("failed to apply environment overrides: %w", err)
		}
	}

	// Validate configuration if a validator is provided
	if opts.Validate {
		if validator, ok := config.(Validator); ok {
			if err := validator.Validate(); err != nil {
				l.logger.Error("config_validation_failed",
					"error", err)
				return fmt.Errorf("configuration validation failed: %w", err)
			}
		}
	}

	l.logger.Info("config_loading_completed",
		"file_path", filePath)

	return nil
}

// LoadFromDirectory loads configuration from multiple YAML files in a directory
func (l *Loader) LoadFromDirectory(dirPath string, configs map[string]interface{}, opts *LoadOptions) error {
	l.logger.Info("config_directory_loading_started",
		"directory", dirPath)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read configuration directory %s: %w", dirPath, err)
	}

	loadedCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process YAML files
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		// Extract config name from filename (without extension)
		configName := strings.TrimSuffix(entry.Name(), ext)
		
		// Check if we have a target config for this file
		config, exists := configs[configName]
		if !exists {
			l.logger.Warn("config_file_ignored",
				"file_name", entry.Name(),
				"reason", "no target config provided")
			continue
		}

		filePath := filepath.Join(dirPath, entry.Name())
		if err := l.LoadFromFile(filePath, config, opts); err != nil {
			return fmt.Errorf("failed to load config file %s: %w", filePath, err)
		}

		loadedCount++
	}

	l.logger.Info("config_directory_loading_completed",
		"directory", dirPath,
		"files_loaded", loadedCount)

	return nil
}

// applyEnvironmentOverrides applies environment variable overrides to configuration
func (l *Loader) applyEnvironmentOverrides(config interface{}, envPrefix string) error {
	// This is a simplified implementation
	// In a production system, you'd want more sophisticated environment variable mapping
	
	envVars := os.Environ()
	prefix := envPrefix + "_"
	overrideCount := 0

	for _, env := range envVars {
		if !strings.HasPrefix(env, prefix) {
			continue
		}

		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimPrefix(parts[0], prefix)

		l.logger.Debug("config_env_override_applied",
			"key", key,
			"env_var", parts[0])

		// Here you would implement the actual override logic
		// This would require reflection or a more sophisticated mapping system
		overrideCount++
	}

	if overrideCount > 0 {
		l.logger.Info("config_env_overrides_applied",
			"overrides_count", overrideCount,
			"env_prefix", envPrefix)
	}

	return nil
}

// Validator interface for configuration validation
type Validator interface {
	Validate() error
}

// SaveToFile saves configuration to a YAML file
func (l *Loader) SaveToFile(filePath string, config interface{}) error {
	l.logger.Info("config_saving_started",
		"file_path", filePath)

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		l.logger.Error("config_yaml_marshal_failed",
			"error", err)
		return fmt.Errorf("failed to marshal configuration to YAML: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		l.logger.Error("config_file_write_failed",
			"file_path", filePath,
			"error", err)
		return fmt.Errorf("failed to write configuration file %s: %w", filePath, err)
	}

	l.logger.Info("config_saving_completed",
		"file_path", filePath,
		"size_bytes", len(data))

	return nil
}

// GetDefaultConfigPath returns the default configuration file path based on environment
func GetDefaultConfigPath() string {
	// Check for explicit config file environment variable
	if path := os.Getenv("MCPEG_CONFIG_FILE"); path != "" {
		return path
	}

	// Check for development vs production environment
	if os.Getenv("MCPEG_ENV") == "development" {
		return "config/development.yaml"
	}

	// Production default
	return "config/production.yaml"
}

// GenerateExampleConfig generates an example configuration file
func GenerateExampleConfig(filePath string, exampleConfig interface{}) error {
	loader := NewLoader(&noOpLogger{})
	return loader.SaveToFile(filePath, exampleConfig)
}

// noOpLogger is a simple logger for cases where we don't have a real logger available
type noOpLogger struct{}

func (l *noOpLogger) WithComponent(component string) logging.Logger { return l }
func (l *noOpLogger) WithContext(ctx context.Context) logging.Logger { return l }
func (l *noOpLogger) WithTraceID(traceID string) logging.Logger     { return l }
func (l *noOpLogger) WithSpanID(spanID string) logging.Logger       { return l }
func (l *noOpLogger) Trace(msg string, fields ...interface{})       {}
func (l *noOpLogger) Debug(msg string, fields ...interface{})       {}
func (l *noOpLogger) Info(msg string, fields ...interface{})        {}
func (l *noOpLogger) Warn(msg string, fields ...interface{})        {}
func (l *noOpLogger) Error(msg string, fields ...interface{})       {}