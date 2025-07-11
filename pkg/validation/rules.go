package validation

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

// RequiredFieldRule validates that required fields are present
type RequiredFieldRule struct{}

func (r *RequiredFieldRule) Name() string {
	return "required_field"
}

func (r *RequiredFieldRule) Description() string {
	return "Validates that required fields are present and not empty"
}

func (r *RequiredFieldRule) Category() string {
	return "general"
}

func (r *RequiredFieldRule) Severity() ErrorSeverity {
	return SeverityError
}

func (r *RequiredFieldRule) Validate(ctx context.Context, value interface{}, field string) ValidationResult {
	result := ValidationResult{
		Valid:       true,
		Errors:      make([]ValidationError, 0),
		Warnings:    make([]ValidationWarning, 0),
		Suggestions: make([]string, 0),
		Performance: ValidationPerformance{FieldsChecked: 1},
	}
	
	if isEmpty(value) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    field,
			Message:  "Required field is missing or empty",
			Code:     "REQUIRED_FIELD_MISSING",
			Value:    value,
			Severity: r.Severity(),
			Suggestions: []string{
				fmt.Sprintf("Provide a value for field '%s'", field),
				"Check if this field should be optional",
			},
		})
	}
	
	return result
}

// TypeValidationRule validates data types
type TypeValidationRule struct{}

func (r *TypeValidationRule) Name() string {
	return "type_validation"
}

func (r *TypeValidationRule) Description() string {
	return "Validates that values match expected data types"
}

func (r *TypeValidationRule) Category() string {
	return "general"
}

func (r *TypeValidationRule) Severity() ErrorSeverity {
	return SeverityError
}

func (r *TypeValidationRule) Validate(ctx context.Context, value interface{}, field string) ValidationResult {
	result := ValidationResult{
		Valid:       true,
		Errors:      make([]ValidationError, 0),
		Warnings:    make([]ValidationWarning, 0),
		Suggestions: make([]string, 0),
		Performance: ValidationPerformance{FieldsChecked: 1},
	}
	
	// This is a basic type validation
	// In practice, you would specify expected types for validation
	if value != nil {
		valueType := reflect.TypeOf(value).String()
		result.Context = map[string]interface{}{
			"detected_type": valueType,
		}
	}
	
	return result
}

// RangeValidationRule validates numeric ranges
type RangeValidationRule struct{}

func (r *RangeValidationRule) Name() string {
	return "range_validation"
}

func (r *RangeValidationRule) Description() string {
	return "Validates that numeric values are within acceptable ranges"
}

func (r *RangeValidationRule) Category() string {
	return "general"
}

func (r *RangeValidationRule) Severity() ErrorSeverity {
	return SeverityError
}

func (r *RangeValidationRule) Validate(ctx context.Context, value interface{}, field string) ValidationResult {
	result := ValidationResult{
		Valid:       true,
		Errors:      make([]ValidationError, 0),
		Warnings:    make([]ValidationWarning, 0),
		Suggestions: make([]string, 0),
		Performance: ValidationPerformance{FieldsChecked: 1},
	}
	
	// Check if value is numeric
	rv := reflect.ValueOf(value)
	if rv.Kind() >= reflect.Int && rv.Kind() <= reflect.Float64 {
		// This is a placeholder for range validation
		// Actual implementation would check against configured ranges
		result.Context = map[string]interface{}{
			"is_numeric": true,
			"value":      value,
		}
	}
	
	return result
}

// FormatValidationRule validates string formats
type FormatValidationRule struct{}

func (r *FormatValidationRule) Name() string {
	return "format_validation"
}

func (r *FormatValidationRule) Description() string {
	return "Validates that string values match expected formats"
}

func (r *FormatValidationRule) Category() string {
	return "general"
}

func (r *FormatValidationRule) Severity() ErrorSeverity {
	return SeverityWarning
}

func (r *FormatValidationRule) Validate(ctx context.Context, value interface{}, field string) ValidationResult {
	result := ValidationResult{
		Valid:       true,
		Errors:      make([]ValidationError, 0),
		Warnings:    make([]ValidationWarning, 0),
		Suggestions: make([]string, 0),
		Performance: ValidationPerformance{FieldsChecked: 1},
	}
	
	if str, ok := value.(string); ok {
		// Basic format checks could be added here
		result.Context = map[string]interface{}{
			"string_length": len(str),
			"is_empty":      str == "",
		}
	}
	
	return result
}

// MCPMethodRule validates MCP method names
type MCPMethodRule struct{}

func (r *MCPMethodRule) Name() string {
	return "mcp_method"
}

func (r *MCPMethodRule) Description() string {
	return "Validates MCP method names according to protocol specification"
}

func (r *MCPMethodRule) Category() string {
	return "mcp"
}

func (r *MCPMethodRule) Severity() ErrorSeverity {
	return SeverityError
}

func (r *MCPMethodRule) Validate(ctx context.Context, value interface{}, field string) ValidationResult {
	result := ValidationResult{
		Valid:       true,
		Errors:      make([]ValidationError, 0),
		Warnings:    make([]ValidationWarning, 0),
		Suggestions: make([]string, 0),
		Performance: ValidationPerformance{FieldsChecked: 1},
	}
	
	method, ok := value.(string)
	if !ok {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    field,
			Message:  "Method must be a string",
			Code:     "METHOD_NOT_STRING",
			Value:    value,
			Severity: r.Severity(),
		})
		return result
	}
	
	if method == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    field,
			Message:  "Method name cannot be empty",
			Code:     "METHOD_EMPTY",
			Value:    value,
			Severity: r.Severity(),
		})
		return result
	}
	
	// Validate method naming convention
	if !isValidMCPMethodName(method) {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   field,
			Message: "Method name doesn't follow MCP naming conventions",
			Code:    "METHOD_NAMING_CONVENTION",
			Value:   method,
			Suggestions: []string{
				"Use lowercase letters with forward slashes",
				"Follow pattern: category/action (e.g., tools/list)",
			},
		})
	}
	
	return result
}

// MCPVersionRule validates MCP protocol versions
type MCPVersionRule struct{}

func (r *MCPVersionRule) Name() string {
	return "mcp_version"
}

func (r *MCPVersionRule) Description() string {
	return "Validates MCP protocol version strings"
}

func (r *MCPVersionRule) Category() string {
	return "mcp"
}

func (r *MCPVersionRule) Severity() ErrorSeverity {
	return SeverityError
}

func (r *MCPVersionRule) Validate(ctx context.Context, value interface{}, field string) ValidationResult {
	result := ValidationResult{
		Valid:       true,
		Errors:      make([]ValidationError, 0),
		Warnings:    make([]ValidationWarning, 0),
		Suggestions: make([]string, 0),
		Performance: ValidationPerformance{FieldsChecked: 1},
	}
	
	version, ok := value.(string)
	if !ok {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    field,
			Message:  "Protocol version must be a string",
			Code:     "VERSION_NOT_STRING",
			Value:    value,
			Severity: r.Severity(),
		})
		return result
	}
	
	// Validate against known MCP versions
	validVersions := map[string]bool{
		"2025-03-26": true,
	}
	
	if !validVersions[version] {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   field,
			Message: fmt.Sprintf("Unknown MCP protocol version: %s", version),
			Code:    "UNKNOWN_PROTOCOL_VERSION",
			Value:   version,
			Suggestions: []string{
				"Use a supported MCP protocol version",
				"Current supported version: 2025-03-26",
			},
		})
	}
	
	return result
}

// MCPParameterRule validates MCP method parameters
type MCPParameterRule struct{}

func (r *MCPParameterRule) Name() string {
	return "mcp_parameter"
}

func (r *MCPParameterRule) Description() string {
	return "Validates MCP method parameters according to method requirements"
}

func (r *MCPParameterRule) Category() string {
	return "mcp"
}

func (r *MCPParameterRule) Severity() ErrorSeverity {
	return SeverityError
}

func (r *MCPParameterRule) Validate(ctx context.Context, value interface{}, field string) ValidationResult {
	result := ValidationResult{
		Valid:       true,
		Errors:      make([]ValidationError, 0),
		Warnings:    make([]ValidationWarning, 0),
		Suggestions: make([]string, 0),
		Performance: ValidationPerformance{FieldsChecked: 1},
	}
	
	// Basic parameter validation
	// Specific validation would depend on the method context
	
	if value == nil {
		result.Context = map[string]interface{}{
			"parameters_present": false,
		}
	} else {
		result.Context = map[string]interface{}{
			"parameters_present": true,
			"parameter_type":     reflect.TypeOf(value).String(),
		}
	}
	
	return result
}

// Helper functions
func isEmpty(value interface{}) bool {
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
	case reflect.Interface:
		return rv.IsNil()
	default:
		return false
	}
}

func isValidMCPMethodName(method string) bool {
	// MCP method names should:
	// - Be lowercase
	// - Use forward slashes as separators
	// - Not start or end with slashes
	// - Not contain spaces
	
	if method != strings.ToLower(method) {
		return false
	}
	
	if strings.Contains(method, " ") {
		return false
	}
	
	if strings.HasPrefix(method, "/") || strings.HasSuffix(method, "/") {
		return false
	}
	
	if strings.Contains(method, "//") {
		return false
	}
	
	return true
}