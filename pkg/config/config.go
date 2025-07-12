package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

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
	// Use reflection to apply environment variable overrides
	configValue := reflect.ValueOf(config)
	if configValue.Kind() != reflect.Ptr || configValue.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("config must be a pointer to a struct")
	}

	configStruct := configValue.Elem()
	configType := configStruct.Type()

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
		value := parts[1]

		// Apply the override using reflection
		if err := l.applyEnvOverride(configStruct, configType, key, value); err != nil {
			l.logger.Warn("config_env_override_failed",
				"key", key,
				"env_var", parts[0],
				"error", err)
			continue
		}

		l.logger.Debug("config_env_override_applied",
			"key", key,
			"env_var", parts[0],
			"value", value)

		overrideCount++
	}

	if overrideCount > 0 {
		l.logger.Info("config_env_overrides_applied",
			"overrides_count", overrideCount,
			"env_prefix", envPrefix)
	}

	return nil
}

// applyEnvOverride applies a single environment variable override using reflection
func (l *Loader) applyEnvOverride(configStruct reflect.Value, configType reflect.Type, key string, value string) error {
	// Convert environment variable key to struct field name
	// Convert UPPER_SNAKE_CASE to PascalCase
	fieldName := l.envKeyToFieldName(key)

	// Find the field
	field := configStruct.FieldByName(fieldName)
	if !field.IsValid() {
		return fmt.Errorf("field %s not found", fieldName)
	}

	if !field.CanSet() {
		return fmt.Errorf("field %s cannot be set", fieldName)
	}

	// Convert the string value to the appropriate type
	return l.setFieldValue(field, value, fieldName)
}

// envKeyToFieldName converts environment variable key to struct field name
func (l *Loader) envKeyToFieldName(key string) string {
	// Convert UPPER_SNAKE_CASE to PascalCase
	parts := strings.Split(strings.ToLower(key), "_")
	result := ""
	for _, part := range parts {
		if len(part) > 0 {
			result += strings.Title(part)
		}
	}
	return result
}

// setFieldValue sets a field value with proper type conversion
func (l *Loader) setFieldValue(field reflect.Value, value string, fieldName string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer value for %s: %w", fieldName, err)
		}
		field.SetInt(intVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer value for %s: %w", fieldName, err)
		}
		field.SetUint(uintVal)
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid float value for %s: %w", fieldName, err)
		}
		field.SetFloat(floatVal)
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value for %s: %w", fieldName, err)
		}
		field.SetBool(boolVal)
	case reflect.Slice:
		// Handle slices (e.g., []string)
		if field.Type().Elem().Kind() == reflect.String {
			// Split comma-separated values
			values := strings.Split(value, ",")
			slice := reflect.MakeSlice(field.Type(), len(values), len(values))
			for i, v := range values {
				slice.Index(i).SetString(strings.TrimSpace(v))
			}
			field.Set(slice)
		} else {
			return fmt.Errorf("unsupported slice type for %s", fieldName)
		}
	case reflect.Struct:
		// Handle time.Duration
		if field.Type().String() == "time.Duration" {
			duration, err := time.ParseDuration(value)
			if err != nil {
				return fmt.Errorf("invalid duration value for %s: %w", fieldName, err)
			}
			field.Set(reflect.ValueOf(duration))
		} else {
			return fmt.Errorf("unsupported struct type for %s: %s", fieldName, field.Type())
		}
	default:
		return fmt.Errorf("unsupported field type for %s: %s", fieldName, field.Kind())
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

// Note: GetDefaultConfigPath has been moved to pkg/paths/paths.go to centralize path management

// GenerateExampleConfig generates an example configuration file
func GenerateExampleConfig(filePath string, exampleConfig interface{}) error {
	loader := NewLoader(&noOpLogger{})
	return loader.SaveToFile(filePath, exampleConfig)
}

// noOpLogger is a simple logger for cases where we don't have a real logger available
type noOpLogger struct{}

func (l *noOpLogger) WithComponent(component string) logging.Logger  { return l }
func (l *noOpLogger) WithContext(ctx context.Context) logging.Logger { return l }
func (l *noOpLogger) WithTraceID(traceID string) logging.Logger      { return l }
func (l *noOpLogger) WithSpanID(spanID string) logging.Logger        { return l }
func (l *noOpLogger) Trace(msg string, fields ...interface{})        {}
func (l *noOpLogger) Debug(msg string, fields ...interface{})        {}
func (l *noOpLogger) Info(msg string, fields ...interface{})         {}
func (l *noOpLogger) Warn(msg string, fields ...interface{})         {}
func (l *noOpLogger) Error(msg string, fields ...interface{})        {}
