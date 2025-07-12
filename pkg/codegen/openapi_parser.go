package codegen

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/validation"
	"gopkg.in/yaml.v3"
)

// OpenAPIParser handles parsing and validation of OpenAPI specifications
type OpenAPIParser struct {
	logger    logging.Logger
	validator *validation.Validator
	config    ParserConfig
}

// ParserConfig configures OpenAPI parsing behavior
type ParserConfig struct {
	// Validation settings
	StrictValidation  bool `yaml:"strict_validation"`
	AllowExtensions   bool `yaml:"allow_extensions"`
	ResolveReferences bool `yaml:"resolve_references"`

	// Network settings for remote specs
	HTTPTimeout    time.Duration `yaml:"http_timeout"`
	MaxFileSize    int64         `yaml:"max_file_size"`
	AllowedSchemes []string      `yaml:"allowed_schemes"`

	// Caching settings
	EnableCaching bool          `yaml:"enable_caching"`
	CacheDir      string        `yaml:"cache_dir"`
	CacheExpiry   time.Duration `yaml:"cache_expiry"`
}

// ParseResult contains the result of parsing an OpenAPI specification
type ParseResult struct {
	Spec     *OpenAPISpec        `json:"spec"`
	Valid    bool                `json:"valid"`
	Errors   []ValidationError   `json:"errors,omitempty"`
	Warnings []ValidationWarning `json:"warnings,omitempty"`
	Metadata ParseMetadata       `json:"metadata"`
}

// ParseMetadata contains metadata about the parsing process
type ParseMetadata struct {
	Source             string        `json:"source"`
	Format             string        `json:"format"`
	Size               int64         `json:"size"`
	ParseDuration      time.Duration `json:"parse_duration"`
	ValidationTime     time.Duration `json:"validation_time"`
	ReferencesResolved int           `json:"references_resolved"`
	CacheHit           bool          `json:"cache_hit"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Path    string `json:"path"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// NewOpenAPIParser creates a new OpenAPI parser
func NewOpenAPIParser(logger logging.Logger, validator *validation.Validator) *OpenAPIParser {
	return &OpenAPIParser{
		logger:    logger.WithComponent("openapi_parser"),
		validator: validator,
		config:    defaultParserConfig(),
	}
}

// ParseFromFile parses an OpenAPI specification from a file
func (p *OpenAPIParser) ParseFromFile(ctx context.Context, filePath string) (*ParseResult, error) {
	start := time.Now()

	p.logger.Info("parsing_openapi_file",
		"file_path", filePath)

	// Check file existence and size
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if info.Size() > p.config.MaxFileSize {
		return nil, fmt.Errorf("file size %d exceeds maximum %d", info.Size(), p.config.MaxFileSize)
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Determine format
	format := p.detectFormat(filePath, content)

	result := &ParseResult{
		Metadata: ParseMetadata{
			Source: filePath,
			Format: format,
			Size:   info.Size(),
		},
	}

	// Parse content
	spec, err := p.parseContent(content, format)
	if err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Path:    "root",
			Message: err.Error(),
			Code:    "PARSE_ERROR",
		})
		result.Valid = false
		return result, nil
	}

	result.Spec = spec
	result.Metadata.ParseDuration = time.Since(start)

	// Validate specification
	if err := p.validateSpec(ctx, result); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Resolve references if enabled
	if p.config.ResolveReferences {
		baseDir := filepath.Dir(filePath)
		if err := p.resolveReferences(ctx, result, baseDir); err != nil {
			p.logger.Warn("reference_resolution_failed", "error", err)
			result.Warnings = append(result.Warnings, ValidationWarning{
				Path:    "references",
				Message: fmt.Sprintf("Failed to resolve references: %v", err),
				Code:    "REFERENCE_RESOLUTION_FAILED",
			})
		}
	}

	p.logger.Info("openapi_file_parsed",
		"file_path", filePath,
		"format", format,
		"valid", result.Valid,
		"errors", len(result.Errors),
		"warnings", len(result.Warnings),
		"parse_duration", result.Metadata.ParseDuration)

	return result, nil
}

// ParseFromURL parses an OpenAPI specification from a URL
func (p *OpenAPIParser) ParseFromURL(ctx context.Context, specURL string) (*ParseResult, error) {
	start := time.Now()

	p.logger.Info("parsing_openapi_url",
		"url", specURL)

	// Validate URL
	parsedURL, err := url.Parse(specURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if !p.isAllowedScheme(parsedURL.Scheme) {
		return nil, fmt.Errorf("scheme %s not allowed", parsedURL.Scheme)
	}

	// Check cache first
	if p.config.EnableCaching {
		if cached, err := p.getCachedSpec(specURL); err == nil && cached != nil {
			cached.Metadata.CacheHit = true
			p.logger.Info("openapi_spec_cache_hit", "url", specURL)
			return cached, nil
		}
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: p.config.HTTPTimeout,
	}

	// Make request
	req, err := http.NewRequestWithContext(ctx, "GET", specURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json, application/yaml, text/yaml")
	req.Header.Set("User-Agent", "MCPEG-OpenAPI-Parser/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Check content length
	if resp.ContentLength > p.config.MaxFileSize {
		return nil, fmt.Errorf("content length %d exceeds maximum %d", resp.ContentLength, p.config.MaxFileSize)
	}

	// Read response body with size limit
	limitedReader := io.LimitReader(resp.Body, p.config.MaxFileSize)
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Determine format from content type or content
	format := p.detectFormatFromResponse(resp, content)

	result := &ParseResult{
		Metadata: ParseMetadata{
			Source: specURL,
			Format: format,
			Size:   int64(len(content)),
		},
	}

	// Parse content
	spec, err := p.parseContent(content, format)
	if err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Path:    "root",
			Message: err.Error(),
			Code:    "PARSE_ERROR",
		})
		result.Valid = false
		return result, nil
	}

	result.Spec = spec
	result.Metadata.ParseDuration = time.Since(start)

	// Validate specification
	if err := p.validateSpec(ctx, result); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Cache result if successful
	if p.config.EnableCaching && result.Valid {
		if err := p.cacheSpec(specURL, result); err != nil {
			p.logger.Warn("failed_to_cache_spec", "url", specURL, "error", err)
		}
	}

	p.logger.Info("openapi_url_parsed",
		"url", specURL,
		"format", format,
		"valid", result.Valid,
		"errors", len(result.Errors),
		"warnings", len(result.Warnings),
		"parse_duration", result.Metadata.ParseDuration)

	return result, nil
}

// ParseFromString parses an OpenAPI specification from a string
func (p *OpenAPIParser) ParseFromString(ctx context.Context, content string, format string) (*ParseResult, error) {
	start := time.Now()

	contentBytes := []byte(content)

	// Auto-detect format if not specified
	if format == "" {
		format = p.detectFormat("", contentBytes)
	}

	result := &ParseResult{
		Metadata: ParseMetadata{
			Source: "string",
			Format: format,
			Size:   int64(len(contentBytes)),
		},
	}

	// Parse content
	spec, err := p.parseContent(contentBytes, format)
	if err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Path:    "root",
			Message: err.Error(),
			Code:    "PARSE_ERROR",
		})
		result.Valid = false
		return result, nil
	}

	result.Spec = spec
	result.Metadata.ParseDuration = time.Since(start)

	// Validate specification
	if err := p.validateSpec(ctx, result); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return result, nil
}

// parseContent parses content based on format
func (p *OpenAPIParser) parseContent(content []byte, format string) (*OpenAPISpec, error) {
	var spec OpenAPISpec

	switch format {
	case "json":
		if err := json.Unmarshal(content, &spec); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	case "yaml":
		if err := yaml.Unmarshal(content, &spec); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	return &spec, nil
}

// detectFormat detects the format of OpenAPI content
func (p *OpenAPIParser) detectFormat(filePath string, content []byte) string {
	// Check file extension first
	if filePath != "" {
		ext := strings.ToLower(filepath.Ext(filePath))
		switch ext {
		case ".json":
			return "json"
		case ".yaml", ".yml":
			return "yaml"
		}
	}

	// Try to detect from content
	trimmed := strings.TrimSpace(string(content))
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return "json"
	}

	// Default to YAML
	return "yaml"
}

// detectFormatFromResponse detects format from HTTP response
func (p *OpenAPIParser) detectFormatFromResponse(resp *http.Response, content []byte) string {
	// Check Content-Type header
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		return "json"
	}
	if strings.Contains(contentType, "yaml") || strings.Contains(contentType, "yml") {
		return "yaml"
	}

	// Fall back to content detection
	return p.detectFormat("", content)
}

// validateSpec validates an OpenAPI specification
func (p *OpenAPIParser) validateSpec(ctx context.Context, result *ParseResult) error {
	start := time.Now()
	defer func() {
		result.Metadata.ValidationTime = time.Since(start)
	}()

	if result.Spec == nil {
		return fmt.Errorf("no spec to validate")
	}

	spec := result.Spec
	result.Valid = true

	// Validate OpenAPI version
	if spec.OpenAPI == "" {
		result.Errors = append(result.Errors, ValidationError{
			Path:    "openapi",
			Message: "OpenAPI version is required",
			Code:    "MISSING_OPENAPI_VERSION",
		})
		result.Valid = false
	} else if !strings.HasPrefix(spec.OpenAPI, "3.0") && !strings.HasPrefix(spec.OpenAPI, "3.1") {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Path:    "openapi",
			Message: fmt.Sprintf("Unsupported OpenAPI version: %s", spec.OpenAPI),
			Code:    "UNSUPPORTED_VERSION",
		})
	}

	// Validate info section
	if spec.Info.Title == "" {
		result.Errors = append(result.Errors, ValidationError{
			Path:    "info.title",
			Message: "API title is required",
			Code:    "MISSING_TITLE",
		})
		result.Valid = false
	}

	if spec.Info.Version == "" {
		result.Errors = append(result.Errors, ValidationError{
			Path:    "info.version",
			Message: "API version is required",
			Code:    "MISSING_VERSION",
		})
		result.Valid = false
	}

	// Validate paths
	if len(spec.Paths) == 0 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Path:    "paths",
			Message: "No paths defined",
			Code:    "NO_PATHS",
		})
	}

	// Validate path structure
	for path, pathItem := range spec.Paths {
		if !strings.HasPrefix(path, "/") {
			result.Errors = append(result.Errors, ValidationError{
				Path:    fmt.Sprintf("paths.%s", path),
				Message: "Path must start with /",
				Code:    "INVALID_PATH",
			})
			result.Valid = false
		}

		// Validate operations
		p.validateOperation(fmt.Sprintf("paths.%s.get", path), pathItem.GET, result)
		p.validateOperation(fmt.Sprintf("paths.%s.post", path), pathItem.POST, result)
		p.validateOperation(fmt.Sprintf("paths.%s.put", path), pathItem.PUT, result)
		p.validateOperation(fmt.Sprintf("paths.%s.delete", path), pathItem.DELETE, result)
		p.validateOperation(fmt.Sprintf("paths.%s.patch", path), pathItem.PATCH, result)
	}

	// Validate components
	if spec.Components.Schemas != nil {
		for name, schema := range spec.Components.Schemas {
			p.validateSchema(fmt.Sprintf("components.schemas.%s", name), schema, result)
		}
	}

	// Use validation framework if available
	if p.validator != nil {
		validationResult := p.validator.Validate(ctx, spec, "openapi")
		if !validationResult.Valid {
			for _, err := range validationResult.Errors {
				result.Errors = append(result.Errors, ValidationError{
					Path:    err.Field,
					Message: err.Message,
					Code:    err.Code,
				})
			}
			result.Valid = false
		}

		for _, warn := range validationResult.Warnings {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Path:    warn.Field,
				Message: warn.Message,
				Code:    warn.Code,
			})
		}
	}

	return nil
}

// validateOperation validates an OpenAPI operation
func (p *OpenAPIParser) validateOperation(path string, op *Operation, result *ParseResult) {
	if op == nil {
		return
	}

	// Validate operation ID
	if op.OperationID == "" {
		if p.config.StrictValidation {
			result.Errors = append(result.Errors, ValidationError{
				Path:    path + ".operationId",
				Message: "Operation ID is required in strict mode",
				Code:    "MISSING_OPERATION_ID",
			})
			result.Valid = false
		} else {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Path:    path + ".operationId",
				Message: "Operation ID is recommended",
				Code:    "MISSING_OPERATION_ID",
			})
		}
	}

	// Validate responses
	if len(op.Responses) == 0 {
		result.Errors = append(result.Errors, ValidationError{
			Path:    path + ".responses",
			Message: "At least one response is required",
			Code:    "MISSING_RESPONSES",
		})
		result.Valid = false
	}

	// Check for success response
	hasSuccessResponse := false
	for code := range op.Responses {
		if strings.HasPrefix(code, "2") {
			hasSuccessResponse = true
			break
		}
	}

	if !hasSuccessResponse {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Path:    path + ".responses",
			Message: "No success response (2xx) defined",
			Code:    "NO_SUCCESS_RESPONSE",
		})
	}
}

// validateSchema validates a JSON schema
func (p *OpenAPIParser) validateSchema(path string, schema Schema, result *ParseResult) {
	// Validate type
	if schema.Type == "" && schema.Ref == "" && len(schema.AllOf) == 0 && len(schema.OneOf) == 0 && len(schema.AnyOf) == 0 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Path:    path + ".type",
			Message: "Schema type not specified",
			Code:    "MISSING_SCHEMA_TYPE",
		})
	}

	// Validate array schemas
	if schema.Type == "array" && schema.Items == nil {
		result.Errors = append(result.Errors, ValidationError{
			Path:    path + ".items",
			Message: "Array schema must define items",
			Code:    "MISSING_ARRAY_ITEMS",
		})
		result.Valid = false
	}

	// Validate object schemas
	if schema.Type == "object" && len(schema.Properties) == 0 && schema.AdditionalProperties == nil {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Path:    path,
			Message: "Object schema has no properties or additionalProperties",
			Code:    "EMPTY_OBJECT_SCHEMA",
		})
	}

	// Validate references
	if schema.Ref != "" {
		if !strings.HasPrefix(schema.Ref, "#/") {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Path:    path + ".$ref",
				Message: "External references not validated",
				Code:    "EXTERNAL_REFERENCE",
			})
		}
	}
}

// resolveReferences resolves $ref references in the specification
func (p *OpenAPIParser) resolveReferences(ctx context.Context, result *ParseResult, baseDir string) error {
	// This is a simplified implementation
	// In a full implementation, this would recursively resolve all references
	refCount := 0

	// Count and potentially resolve references
	for _, schema := range result.Spec.Components.Schemas {
		if schema.Ref != "" {
			refCount++
		}
	}

	result.Metadata.ReferencesResolved = refCount

	p.logger.Debug("references_processed",
		"count", refCount,
		"base_dir", baseDir)

	return nil
}

// cacheSpec caches a parsed specification
func (p *OpenAPIParser) cacheSpec(source string, result *ParseResult) error {
	if p.config.CacheDir == "" {
		return fmt.Errorf("cache directory not configured")
	}

	// Create cache directory
	if err := os.MkdirAll(p.config.CacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Generate cache key
	cacheKey := p.generateCacheKey(source)
	cachePath := filepath.Join(p.config.CacheDir, cacheKey+".json")

	// Serialize result
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	// Write to cache
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	p.logger.Debug("spec_cached",
		"source", source,
		"cache_key", cacheKey,
		"cache_path", cachePath)

	return nil
}

// getCachedSpec retrieves a cached specification
func (p *OpenAPIParser) getCachedSpec(source string) (*ParseResult, error) {
	if p.config.CacheDir == "" {
		return nil, fmt.Errorf("cache directory not configured")
	}

	cacheKey := p.generateCacheKey(source)
	cachePath := filepath.Join(p.config.CacheDir, cacheKey+".json")

	// Check if cache file exists and is not expired
	info, err := os.Stat(cachePath)
	if err != nil {
		return nil, err
	}

	if time.Since(info.ModTime()) > p.config.CacheExpiry {
		return nil, fmt.Errorf("cache expired")
	}

	// Read cache file
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	// Unmarshal result
	var result ParseResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached result: %w", err)
	}

	return &result, nil
}

// generateCacheKey generates a cache key for a source
func (p *OpenAPIParser) generateCacheKey(source string) string {
	// Simple hash-like key generation
	// In production, use proper hashing
	key := strings.ReplaceAll(source, "/", "_")
	key = strings.ReplaceAll(key, ":", "_")
	key = strings.ReplaceAll(key, "?", "_")
	key = strings.ReplaceAll(key, "&", "_")
	return key
}

// isAllowedScheme checks if a URL scheme is allowed
func (p *OpenAPIParser) isAllowedScheme(scheme string) bool {
	for _, allowed := range p.config.AllowedSchemes {
		if scheme == allowed {
			return true
		}
	}
	return false
}

// defaultParserConfig returns the default parser configuration
func defaultParserConfig() ParserConfig {
	return ParserConfig{
		StrictValidation:  false,
		AllowExtensions:   true,
		ResolveReferences: true,
		HTTPTimeout:       30 * time.Second,
		MaxFileSize:       10 * 1024 * 1024, // 10MB
		AllowedSchemes:    []string{"http", "https"},
		EnableCaching:     true,
		CacheDir:          "build/cache/openapi",
		CacheExpiry:       24 * time.Hour,
	}
}
