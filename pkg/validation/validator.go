package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
	"github.com/osakka/mcpeg/internal/mcp/types"
)

// ValidationResult represents the result of a validation operation
type ValidationResult struct {
	Valid       bool                   `json:"valid"`
	Errors      []ValidationError      `json:"errors,omitempty"`
	Warnings    []ValidationWarning    `json:"warnings,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Suggestions []string               `json:"suggestions,omitempty"`
	Performance ValidationPerformance  `json:"performance"`
}

// ValidationError represents a validation error with full context
type ValidationError struct {
	Field       string                 `json:"field"`
	Message     string                 `json:"message"`
	Code        string                 `json:"code"`
	Value       interface{}            `json:"value,omitempty"`
	Expected    interface{}            `json:"expected,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Suggestions []string               `json:"suggestions,omitempty"`
	Severity    ErrorSeverity          `json:"severity"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Field       string                 `json:"field"`
	Message     string                 `json:"message"`
	Code        string                 `json:"code"`
	Value       interface{}            `json:"value,omitempty"`
	Suggestions []string               `json:"suggestions,omitempty"`
}

// ValidationPerformance tracks validation performance metrics
type ValidationPerformance struct {
	Duration       time.Duration `json:"duration"`
	RulesEvaluated int           `json:"rules_evaluated"`
	FieldsChecked  int           `json:"fields_checked"`
	CacheHits      int           `json:"cache_hits"`
	CacheMisses    int           `json:"cache_misses"`
}

// ErrorSeverity defines the severity of validation errors
type ErrorSeverity string

const (
	SeverityInfo     ErrorSeverity = "info"
	SeverityWarning  ErrorSeverity = "warning"
	SeverityError    ErrorSeverity = "error"
	SeverityCritical ErrorSeverity = "critical"
)

// ValidationRule defines a validation rule interface
type ValidationRule interface {
	Name() string
	Description() string
	Validate(ctx context.Context, value interface{}, field string) ValidationResult
	Category() string
	Severity() ErrorSeverity
}

// Validator provides comprehensive validation with LLM-optimized error reporting
type Validator struct {
	rules      map[string][]ValidationRule
	logger     logging.Logger
	metrics    metrics.Metrics
	cache      ValidationCache
	config     ValidationConfig
	
	// Schema validation
	schemas    map[string]interface{}
	
	// MCP-specific validators
	mcpValidator *MCPValidator
}

// ValidationConfig configures validation behavior
type ValidationConfig struct {
	// Performance settings
	EnableCaching          bool          `yaml:"enable_caching"`
	CacheExpiry           time.Duration `yaml:"cache_expiry"`
	MaxCacheSize          int           `yaml:"max_cache_size"`
	
	// Validation settings
	StrictMode            bool          `yaml:"strict_mode"`
	FailFast              bool          `yaml:"fail_fast"`
	GenerateSuggestions   bool          `yaml:"generate_suggestions"`
	IncludePerformance    bool          `yaml:"include_performance"`
	
	// Error handling
	MaxErrors             int           `yaml:"max_errors"`
	TreatWarningsAsErrors bool          `yaml:"treat_warnings_as_errors"`
	
	// LLM optimization
	IncludeDetailedContext bool         `yaml:"include_detailed_context"`
	GenerateExamples       bool         `yaml:"generate_examples"`
}

// ValidationCache provides caching for validation results
type ValidationCache interface {
	Get(key string) (*ValidationResult, bool)
	Set(key string, result *ValidationResult, expiry time.Duration)
	Clear()
}

// NewValidator creates a comprehensive validation system
func NewValidator(logger logging.Logger, metrics metrics.Metrics) *Validator {
	v := &Validator{
		rules:   make(map[string][]ValidationRule),
		logger:  logger.WithComponent("validator"),
		metrics: metrics,
		cache:   NewMemoryCache(),
		config:  defaultValidationConfig(),
		schemas: make(map[string]interface{}),
	}
	
	// Initialize MCP-specific validator
	v.mcpValidator = NewMCPValidator(v, logger)
	
	// Register built-in validation rules
	v.registerBuiltInRules()
	
	return v
}

// RegisterRule registers a validation rule for a specific category
func (v *Validator) RegisterRule(category string, rule ValidationRule) {
	if v.rules[category] == nil {
		v.rules[category] = make([]ValidationRule, 0)
	}
	
	v.rules[category] = append(v.rules[category], rule)
	
	v.logger.Info("validation_rule_registered",
		"category", category,
		"rule_name", rule.Name(),
		"description", rule.Description(),
		"severity", rule.Severity())
}

// RegisterSchema registers a JSON schema for validation
func (v *Validator) RegisterSchema(name string, schema interface{}) error {
	v.schemas[name] = schema
	
	v.logger.Info("validation_schema_registered",
		"schema_name", name,
		"schema_type", reflect.TypeOf(schema).String())
	
	return nil
}

// Validate performs comprehensive validation on a value
func (v *Validator) Validate(ctx context.Context, value interface{}, category string) ValidationResult {
	start := time.Now()
	
	// Check cache first
	if v.config.EnableCaching {
		if cached := v.getCachedResult(value, category); cached != nil {
			cached.Performance.CacheHits++
			return *cached
		}
	}
	
	result := ValidationResult{
		Valid:       true,
		Errors:      make([]ValidationError, 0),
		Warnings:    make([]ValidationWarning, 0),
		Context:     make(map[string]interface{}),
		Suggestions: make([]string, 0),
		Performance: ValidationPerformance{
			CacheMisses: 1,
		},
	}
	
	// Add validation context
	if v.config.IncludeDetailedContext {
		result.Context = v.buildValidationContext(value, category)
	}
	
	// Get rules for category
	rules, exists := v.rules[category]
	if !exists {
		v.logger.Warn("no_validation_rules_found",
			"category", category,
			"available_categories", v.getAvailableCategories())
		
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "category",
			Message: fmt.Sprintf("No validation rules found for category: %s", category),
			Code:    "NO_RULES",
			Suggestions: []string{
				"Check if the validation category is correct",
				"Ensure validation rules are registered for this category",
				fmt.Sprintf("Available categories: %s", strings.Join(v.getAvailableCategories(), ", ")),
			},
		})
	}
	
	// Apply validation rules
	result.Performance.RulesEvaluated = len(rules)
	for _, rule := range rules {
		if v.config.FailFast && !result.Valid {
			break
		}
		
		if len(result.Errors) >= v.config.MaxErrors {
			break
		}
		
		ruleResult := rule.Validate(ctx, value, "")
		
		// Merge results
		result.Errors = append(result.Errors, ruleResult.Errors...)
		result.Warnings = append(result.Warnings, ruleResult.Warnings...)
		result.Suggestions = append(result.Suggestions, ruleResult.Suggestions...)
		
		if !ruleResult.Valid {
			result.Valid = false
		}
		
		// Add rule performance context
		result.Performance.FieldsChecked += ruleResult.Performance.FieldsChecked
	}
	
	// Handle warnings as errors if configured
	if v.config.TreatWarningsAsErrors && len(result.Warnings) > 0 {
		for _, warning := range result.Warnings {
			result.Errors = append(result.Errors, ValidationError{
				Field:       warning.Field,
				Message:     warning.Message,
				Code:        warning.Code,
				Value:       warning.Value,
				Suggestions: warning.Suggestions,
				Severity:    SeverityError,
			})
		}
		result.Valid = false
		result.Warnings = nil
	}
	
	// Generate suggestions if enabled
	if v.config.GenerateSuggestions {
		result.Suggestions = v.generateSuggestions(result, value, category)
	}
	
	// Record performance
	result.Performance.Duration = time.Since(start)
	
	// Cache result if enabled
	if v.config.EnableCaching && result.Valid {
		v.setCachedResult(value, category, &result)
	}
	
	// Record metrics
	v.recordValidationMetrics(result, category)
	
	// Log validation result
	v.logValidationResult(result, category, value)
	
	return result
}

// ValidateStruct validates a struct using field tags and registered rules
func (v *Validator) ValidateStruct(ctx context.Context, value interface{}) ValidationResult {
	return v.validateStructRecursive(ctx, value, "", 0)
}

// ValidateMCPRequest validates an MCP request
func (v *Validator) ValidateMCPRequest(ctx context.Context, request types.Request) ValidationResult {
	return v.mcpValidator.ValidateRequest(ctx, request)
}

// ValidateMCPResponse validates an MCP response
func (v *Validator) ValidateMCPResponse(ctx context.Context, response types.Response) ValidationResult {
	return v.mcpValidator.ValidateResponse(ctx, response)
}

// ValidateJSON validates JSON against a registered schema
func (v *Validator) ValidateJSON(ctx context.Context, data []byte, schemaName string) ValidationResult {
	schema, exists := v.schemas[schemaName]
	if !exists {
		return ValidationResult{
			Valid: false,
			Errors: []ValidationError{
				{
					Field:    "schema",
					Message:  fmt.Sprintf("Schema not found: %s", schemaName),
					Code:     "SCHEMA_NOT_FOUND",
					Severity: SeverityError,
					Suggestions: []string{
						"Verify the schema name is correct",
						"Ensure the schema is registered",
						fmt.Sprintf("Available schemas: %s", strings.Join(v.getAvailableSchemas(), ", ")),
					},
				},
			},
		}
	}
	
	// Parse JSON
	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return ValidationResult{
			Valid: false,
			Errors: []ValidationError{
				{
					Field:    "json",
					Message:  fmt.Sprintf("Invalid JSON: %s", err.Error()),
					Code:     "INVALID_JSON",
					Value:    string(data),
					Severity: SeverityError,
					Suggestions: []string{
						"Check JSON syntax",
						"Verify all quotes and brackets are properly closed",
						"Use a JSON validator to identify syntax errors",
					},
				},
			},
		}
	}
	
	// Validate against schema (simplified implementation)
	return v.validateAgainstSchema(ctx, parsed, schema)
}

// validateStructRecursive recursively validates struct fields
func (v *Validator) validateStructRecursive(ctx context.Context, value interface{}, fieldPath string, depth int) ValidationResult {
	const maxDepth = 10
	if depth > maxDepth {
		return ValidationResult{
			Valid: false,
			Errors: []ValidationError{
				{
					Field:    fieldPath,
					Message:  "Maximum validation depth exceeded",
					Code:     "MAX_DEPTH_EXCEEDED",
					Severity: SeverityError,
				},
			},
		}
	}
	
	result := ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}
	
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	
	if rv.Kind() != reflect.Struct {
		return result
	}
	
	rt := rv.Type()
	
	for i := 0; i < rv.NumField(); i++ {
		field := rt.Field(i)
		fieldValue := rv.Field(i)
		
		if !fieldValue.CanInterface() {
			continue
		}
		
		currentPath := field.Name
		if fieldPath != "" {
			currentPath = fieldPath + "." + field.Name
		}
		
		// Validate field using tags
		fieldResult := v.validateFieldByTags(ctx, fieldValue.Interface(), field, currentPath)
		
		// Merge results
		result.Errors = append(result.Errors, fieldResult.Errors...)
		result.Warnings = append(result.Warnings, fieldResult.Warnings...)
		if !fieldResult.Valid {
			result.Valid = false
		}
		
		// Recursively validate nested structs
		if fieldValue.Kind() == reflect.Struct || (fieldValue.Kind() == reflect.Ptr && fieldValue.Elem().Kind() == reflect.Struct) {
			nestedResult := v.validateStructRecursive(ctx, fieldValue.Interface(), currentPath, depth+1)
			result.Errors = append(result.Errors, nestedResult.Errors...)
			result.Warnings = append(result.Warnings, nestedResult.Warnings...)
			if !nestedResult.Valid {
				result.Valid = false
			}
		}
	}
	
	return result
}

// validateFieldByTags validates a field using struct tags
func (v *Validator) validateFieldByTags(ctx context.Context, value interface{}, field reflect.StructField, fieldPath string) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}
	
	// Check required tag
	if required := field.Tag.Get("required"); required == "true" {
		if v.isEmpty(value) {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:    fieldPath,
				Message:  "Field is required",
				Code:     "REQUIRED",
				Value:    value,
				Severity: SeverityError,
				Suggestions: []string{
					fmt.Sprintf("Provide a value for field '%s'", fieldPath),
					"Check if the field should be optional",
				},
			})
		}
	}
	
	// Check validation tag
	if validate := field.Tag.Get("validate"); validate != "" {
		tagResult := v.validateByTag(ctx, value, validate, fieldPath)
		result.Errors = append(result.Errors, tagResult.Errors...)
		result.Warnings = append(result.Warnings, tagResult.Warnings...)
		if !tagResult.Valid {
			result.Valid = false
		}
	}
	
	return result
}

// validateByTag validates using validation tags
func (v *Validator) validateByTag(ctx context.Context, value interface{}, tag, fieldPath string) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}
	
	// Parse validation tags (simplified implementation)
	rules := strings.Split(tag, ",")
	
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		parts := strings.Split(rule, "=")
		ruleName := parts[0]
		
		var ruleValue string
		if len(parts) > 1 {
			ruleValue = parts[1]
		}
		
		ruleResult := v.applyTagRule(ctx, value, ruleName, ruleValue, fieldPath)
		result.Errors = append(result.Errors, ruleResult.Errors...)
		result.Warnings = append(result.Warnings, ruleResult.Warnings...)
		if !ruleResult.Valid {
			result.Valid = false
		}
	}
	
	return result
}

// applyTagRule applies a specific validation tag rule
func (v *Validator) applyTagRule(ctx context.Context, value interface{}, ruleName, ruleValue, fieldPath string) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}
	
	switch ruleName {
	case "min":
		if minVal, err := strconv.Atoi(ruleValue); err == nil {
			if !v.validateMin(value, minVal) {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Field:    fieldPath,
					Message:  fmt.Sprintf("Value must be at least %d", minVal),
					Code:     "MIN_VALUE",
					Value:    value,
					Expected: minVal,
					Severity: SeverityError,
				})
			}
		}
		
	case "max":
		if maxVal, err := strconv.Atoi(ruleValue); err == nil {
			if !v.validateMax(value, maxVal) {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Field:    fieldPath,
					Message:  fmt.Sprintf("Value must be at most %d", maxVal),
					Code:     "MAX_VALUE",
					Value:    value,
					Expected: maxVal,
					Severity: SeverityError,
				})
			}
		}
		
	case "email":
		if !v.validateEmail(value) {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:    fieldPath,
				Message:  "Invalid email format",
				Code:     "INVALID_EMAIL",
				Value:    value,
				Severity: SeverityError,
				Suggestions: []string{
					"Ensure email follows format: user@domain.com",
					"Check for typos in email address",
				},
			})
		}
		
	case "url":
		if !v.validateURL(value) {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:    fieldPath,
				Message:  "Invalid URL format",
				Code:     "INVALID_URL",
				Value:    value,
				Severity: SeverityError,
				Suggestions: []string{
					"Ensure URL includes protocol (http:// or https://)",
					"Check for typos in URL",
				},
			})
		}
		
	case "regex":
		if !v.validateRegex(value, ruleValue) {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:    fieldPath,
				Message:  fmt.Sprintf("Value does not match pattern: %s", ruleValue),
				Code:     "REGEX_MISMATCH",
				Value:    value,
				Expected: ruleValue,
				Severity: SeverityError,
			})
		}
	}
	
	return result
}

// Helper validation methods
func (v *Validator) isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}
	
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.String:
		return rv.String() == ""
	case reflect.Slice, reflect.Map, reflect.Array:
		return rv.Len() == 0
	case reflect.Ptr:
		return rv.IsNil()
	default:
		return false
	}
}

func (v *Validator) validateMin(value interface{}, min int) bool {
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() >= int64(min)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint() >= uint64(min)
	case reflect.Float32, reflect.Float64:
		return rv.Float() >= float64(min)
	case reflect.String:
		return len(rv.String()) >= min
	case reflect.Slice, reflect.Map, reflect.Array:
		return rv.Len() >= min
	default:
		return false
	}
}

func (v *Validator) validateMax(value interface{}, max int) bool {
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() <= int64(max)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint() <= uint64(max)
	case reflect.Float32, reflect.Float64:
		return rv.Float() <= float64(max)
	case reflect.String:
		return len(rv.String()) <= max
	case reflect.Slice, reflect.Map, reflect.Array:
		return rv.Len() <= max
	default:
		return false
	}
}

func (v *Validator) validateEmail(value interface{}) bool {
	str, ok := value.(string)
	if !ok {
		return false
	}
	
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(str)
}

func (v *Validator) validateURL(value interface{}) bool {
	str, ok := value.(string)
	if !ok {
		return false
	}
	
	// Accept HTTP/HTTPS URLs and internal plugin URLs
	httpRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	pluginRegex := regexp.MustCompile(`^plugin://[^\s]*$`)
	
	return httpRegex.MatchString(str) || pluginRegex.MatchString(str)
}

func (v *Validator) validateRegex(value interface{}, pattern string) bool {
	str, ok := value.(string)
	if !ok {
		return false
	}
	
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	
	return regex.MatchString(str)
}

// Helper methods
func (v *Validator) buildValidationContext(value interface{}, category string) map[string]interface{} {
	return map[string]interface{}{
		"value_type":  reflect.TypeOf(value).String(),
		"category":    category,
		"timestamp":   time.Now(),
		"validator":   "mcpeg",
		"config":      v.config,
	}
}

func (v *Validator) generateSuggestions(result ValidationResult, value interface{}, category string) []string {
	suggestions := result.Suggestions
	
	if !result.Valid {
		suggestions = append(suggestions,
			"Review validation errors and fix the identified issues",
			"Check the documentation for the expected format",
			"Validate your data against the schema")
	}
	
	return deduplicateStrings(suggestions)
}

func (v *Validator) getAvailableCategories() []string {
	categories := make([]string, 0, len(v.rules))
	for category := range v.rules {
		categories = append(categories, category)
	}
	return categories
}

func (v *Validator) getAvailableSchemas() []string {
	schemas := make([]string, 0, len(v.schemas))
	for name := range v.schemas {
		schemas = append(schemas, name)
	}
	return schemas
}

func defaultValidationConfig() ValidationConfig {
	return ValidationConfig{
		EnableCaching:          true,
		CacheExpiry:           5 * time.Minute,
		MaxCacheSize:          1000,
		StrictMode:            false,
		FailFast:              false,
		GenerateSuggestions:   true,
		IncludePerformance:    true,
		MaxErrors:             50,
		TreatWarningsAsErrors: false,
		IncludeDetailedContext: true,
		GenerateExamples:       true,
	}
}

func deduplicateStrings(slice []string) []string {
	keys := make(map[string]bool)
	var result []string
	
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	
	return result
}