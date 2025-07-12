package validation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/osakka/mcpeg/internal/mcp/types"
	"github.com/osakka/mcpeg/pkg/logging"
)

// MCPValidator provides MCP protocol-specific validation
type MCPValidator struct {
	validator *Validator
	logger    logging.Logger
}

// NewMCPValidator creates a new MCP protocol validator
func NewMCPValidator(validator *Validator, logger logging.Logger) *MCPValidator {
	return &MCPValidator{
		validator: validator,
		logger:    logger.WithComponent("mcp_validator"),
	}
}

// ValidateRequest validates an MCP request
func (m *MCPValidator) ValidateRequest(ctx context.Context, request types.Request) ValidationResult {
	result := ValidationResult{
		Valid:       true,
		Errors:      make([]ValidationError, 0),
		Warnings:    make([]ValidationWarning, 0),
		Context:     make(map[string]interface{}),
		Suggestions: make([]string, 0),
	}

	start := time.Now()

	// Validate basic request structure
	result = m.mergeResults(result, m.validateRequestStructure(request))

	// Validate JSON-RPC 2.0 compliance
	result = m.mergeResults(result, m.validateJSONRPC(request))

	// Validate method-specific requirements
	result = m.mergeResults(result, m.validateMethod(request))

	// Validate parameters based on method
	result = m.mergeResults(result, m.validateParameters(request))

	// Add MCP-specific context
	result.Context["mcp_validation"] = map[string]interface{}{
		"method":           request.Method,
		"has_params":       request.Params != nil,
		"request_id":       request.ID,
		"validation_time":  time.Since(start),
		"protocol_version": "2025-03-26",
	}

	// Generate MCP-specific suggestions
	if !result.Valid {
		result.Suggestions = append(result.Suggestions, m.generateMCPSuggestions(request)...)
	}

	result.Performance.Duration = time.Since(start)

	return result
}

// ValidateResponse validates an MCP response
func (m *MCPValidator) ValidateResponse(ctx context.Context, response types.Response) ValidationResult {
	result := ValidationResult{
		Valid:       true,
		Errors:      make([]ValidationError, 0),
		Warnings:    make([]ValidationWarning, 0),
		Context:     make(map[string]interface{}),
		Suggestions: make([]string, 0),
	}

	start := time.Now()

	// Validate basic response structure
	result = m.mergeResults(result, m.validateResponseStructure(response))

	// Validate JSON-RPC 2.0 response compliance
	result = m.mergeResults(result, m.validateJSONRPCResponse(response))

	// Validate error structure if present
	if response.Error != nil {
		result = m.mergeResults(result, m.validateErrorStructure(*response.Error))
	}

	// Add response-specific context
	result.Context["mcp_response_validation"] = map[string]interface{}{
		"has_result":      response.Result != nil,
		"has_error":       response.Error != nil,
		"response_id":     response.ID,
		"validation_time": time.Since(start),
	}

	result.Performance.Duration = time.Since(start)

	return result
}

// validateRequestStructure validates the basic structure of an MCP request
func (m *MCPValidator) validateRequestStructure(request types.Request) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}

	// Validate method field
	if request.Method == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    "method",
			Message:  "Method field is required",
			Code:     "MISSING_METHOD",
			Severity: SeverityError,
			Suggestions: []string{
				"Provide a valid MCP method name",
				"Check MCP protocol documentation for available methods",
			},
		})
	}

	// Validate JSON-RPC version
	if request.JSONRPC != "2.0" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    "jsonrpc",
			Message:  "JSON-RPC version must be '2.0'",
			Code:     "INVALID_JSONRPC_VERSION",
			Value:    request.JSONRPC,
			Expected: "2.0",
			Severity: SeverityError,
			Suggestions: []string{
				"Set jsonrpc field to '2.0'",
				"Ensure compliance with JSON-RPC 2.0 specification",
			},
		})
	}

	// Validate ID field (optional for notifications)
	if request.ID != nil {
		// ID should be string, number, or null (we use interface{} so check type)
		switch request.ID.(type) {
		case string, int, int32, int64, float32, float64, nil:
			// Valid ID types
		default:
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   "id",
				Message: "ID should be string, number, or null",
				Code:    "UNUSUAL_ID_TYPE",
				Value:   request.ID,
				Suggestions: []string{
					"Use string or number for request ID",
					"Consider using UUID for unique identification",
				},
			})
		}
	}

	return result
}

// validateResponseStructure validates the basic structure of an MCP response
func (m *MCPValidator) validateResponseStructure(response types.Response) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}

	// Validate JSON-RPC version
	if response.JSONRPC != "2.0" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    "jsonrpc",
			Message:  "JSON-RPC version must be '2.0'",
			Code:     "INVALID_JSONRPC_VERSION",
			Value:    response.JSONRPC,
			Expected: "2.0",
			Severity: SeverityError,
		})
	}

	// Validate ID field is present
	if response.ID == nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    "id",
			Message:  "Response ID is required",
			Code:     "MISSING_RESPONSE_ID",
			Severity: SeverityError,
			Suggestions: []string{
				"Include the request ID in the response",
				"Match response ID to the original request ID",
			},
		})
	}

	// Validate either result or error is present (but not both)
	hasResult := response.Result != nil
	hasError := response.Error != nil

	if !hasResult && !hasError {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    "result/error",
			Message:  "Response must have either 'result' or 'error' field",
			Code:     "MISSING_RESULT_OR_ERROR",
			Severity: SeverityError,
			Suggestions: []string{
				"Include a 'result' field for successful responses",
				"Include an 'error' field for error responses",
			},
		})
	}

	if hasResult && hasError {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    "result/error",
			Message:  "Response cannot have both 'result' and 'error' fields",
			Code:     "BOTH_RESULT_AND_ERROR",
			Severity: SeverityError,
			Suggestions: []string{
				"Remove either 'result' or 'error' field",
				"Use 'result' for success, 'error' for failure",
			},
		})
	}

	return result
}

// validateJSONRPC validates JSON-RPC 2.0 compliance
func (m *MCPValidator) validateJSONRPC(request types.Request) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}

	// Additional JSON-RPC validations can be added here
	// For now, basic structure validation covers most requirements

	return result
}

// validateJSONRPCResponse validates JSON-RPC 2.0 response compliance
func (m *MCPValidator) validateJSONRPCResponse(response types.Response) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}

	// Additional JSON-RPC response validations can be added here

	return result
}

// validateMethod validates MCP method names and compliance
func (m *MCPValidator) validateMethod(request types.Request) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}

	// Define valid MCP methods
	validMethods := map[string]bool{
		"initialize":            true,
		"ping":                  true,
		"tools/list":            true,
		"tools/call":            true,
		"resources/list":        true,
		"resources/read":        true,
		"resources/subscribe":   true,
		"resources/unsubscribe": true,
		"prompts/list":          true,
		"prompts/get":           true,
		"logging/setLevel":      true,
		"completion/complete":   true,
	}

	if !validMethods[request.Method] {
		// Check if it's a notification method
		if strings.HasPrefix(request.Method, "notifications/") {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   "method",
				Message: fmt.Sprintf("Notification method: %s", request.Method),
				Code:    "NOTIFICATION_METHOD",
				Value:   request.Method,
				Suggestions: []string{
					"Ensure this is intended as a notification",
					"Notifications should not expect responses",
				},
			})
		} else {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   "method",
				Message: fmt.Sprintf("Unknown MCP method: %s", request.Method),
				Code:    "UNKNOWN_METHOD",
				Value:   request.Method,
				Suggestions: []string{
					"Verify the method name against MCP specification",
					"Check for typos in method name",
					"Ensure the method is supported by this server",
				},
			})
		}
	}

	// Validate method naming conventions
	if !m.isValidMethodName(request.Method) {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "method",
			Message: "Method name doesn't follow MCP naming conventions",
			Code:    "INVALID_METHOD_NAME",
			Value:   request.Method,
			Suggestions: []string{
				"Use lowercase with forward slashes as separators",
				"Follow pattern: category/action (e.g., tools/list)",
			},
		})
	}

	return result
}

// validateParameters validates method-specific parameters
func (m *MCPValidator) validateParameters(request types.Request) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}

	switch request.Method {
	case "initialize":
		result = m.mergeResults(result, m.validateInitializeParams(request.Params))
	case "tools/call":
		result = m.mergeResults(result, m.validateToolCallParams(request.Params))
	case "resources/read":
		result = m.mergeResults(result, m.validateResourceReadParams(request.Params))
	case "prompts/get":
		result = m.mergeResults(result, m.validatePromptGetParams(request.Params))
	}

	return result
}

// validateInitializeParams validates initialize method parameters
func (m *MCPValidator) validateInitializeParams(params interface{}) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}

	if params == nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    "params",
			Message:  "Initialize method requires parameters",
			Code:     "MISSING_INITIALIZE_PARAMS",
			Severity: SeverityError,
			Suggestions: []string{
				"Include protocolVersion and clientInfo in parameters",
				"Check MCP initialize method documentation",
			},
		})
		return result
	}

	// In a full implementation, we would parse params and validate specific fields
	// For now, we'll do basic structure validation

	return result
}

// validateToolCallParams validates tools/call method parameters
func (m *MCPValidator) validateToolCallParams(params interface{}) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}

	if params == nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    "params",
			Message:  "Tool call requires parameters",
			Code:     "MISSING_TOOL_CALL_PARAMS",
			Severity: SeverityError,
			Suggestions: []string{
				"Include 'name' parameter with tool name",
				"Include 'arguments' parameter if the tool requires them",
			},
		})
	}

	return result
}

// validateResourceReadParams validates resources/read method parameters
func (m *MCPValidator) validateResourceReadParams(params interface{}) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}

	if params == nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    "params",
			Message:  "Resource read requires parameters",
			Code:     "MISSING_RESOURCE_READ_PARAMS",
			Severity: SeverityError,
			Suggestions: []string{
				"Include 'uri' parameter with resource URI",
			},
		})
	}

	return result
}

// validatePromptGetParams validates prompts/get method parameters
func (m *MCPValidator) validatePromptGetParams(params interface{}) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}

	if params == nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    "params",
			Message:  "Prompt get requires parameters",
			Code:     "MISSING_PROMPT_GET_PARAMS",
			Severity: SeverityError,
			Suggestions: []string{
				"Include 'name' parameter with prompt name",
			},
		})
	}

	return result
}

// validateErrorStructure validates MCP error structure
func (m *MCPValidator) validateErrorStructure(error types.Error) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}

	// Validate error code
	if error.Code == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    "error.code",
			Message:  "Error code is required",
			Code:     "MISSING_ERROR_CODE",
			Severity: SeverityError,
			Suggestions: []string{
				"Include a valid JSON-RPC error code",
				"Use standard error codes: -32700 to -32099",
			},
		})
	}

	// Validate error message
	if error.Message == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    "error.message",
			Message:  "Error message is required",
			Code:     "MISSING_ERROR_MESSAGE",
			Severity: SeverityError,
			Suggestions: []string{
				"Provide a descriptive error message",
				"Include context about what went wrong",
			},
		})
	}

	// Validate standard error codes
	standardCodes := map[int]string{
		-32700: "Parse error",
		-32600: "Invalid Request",
		-32601: "Method not found",
		-32602: "Invalid params",
		-32603: "Internal error",
	}

	if msg, isStandard := standardCodes[error.Code]; isStandard {
		if error.Message != msg {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   "error.message",
				Message: fmt.Sprintf("Standard error code %d typically uses message: '%s'", error.Code, msg),
				Code:    "NON_STANDARD_ERROR_MESSAGE",
				Value:   error.Message,
				Suggestions: []string{
					fmt.Sprintf("Consider using standard message: '%s'", msg),
					"Ensure error message is consistent with error code",
				},
			})
		}
	}

	return result
}

// generateMCPSuggestions generates MCP-specific suggestions for failed validation
func (m *MCPValidator) generateMCPSuggestions(request types.Request) []string {
	suggestions := []string{}

	suggestions = append(suggestions,
		"Review MCP protocol specification (2025-03-26)",
		"Ensure JSON-RPC 2.0 compliance",
		"Validate method names against supported methods",
		"Check parameter structure for the specific method")

	// Method-specific suggestions
	switch request.Method {
	case "initialize":
		suggestions = append(suggestions,
			"Include protocolVersion, clientInfo, and capabilities",
			"Use semantic version format for protocolVersion")
	case "tools/call":
		suggestions = append(suggestions,
			"Include tool name and required arguments",
			"Validate argument types against tool schema")
	case "resources/read":
		suggestions = append(suggestions,
			"Include valid resource URI",
			"Ensure resource exists and is accessible")
	}

	return suggestions
}

// Helper methods
func (m *MCPValidator) isValidMethodName(method string) bool {
	// MCP method names should be lowercase with forward slashes
	return strings.ToLower(method) == method &&
		!strings.Contains(method, " ") &&
		!strings.HasPrefix(method, "/") &&
		!strings.HasSuffix(method, "/")
}

func (m *MCPValidator) mergeResults(base, additional ValidationResult) ValidationResult {
	result := base

	result.Errors = append(result.Errors, additional.Errors...)
	result.Warnings = append(result.Warnings, additional.Warnings...)
	result.Suggestions = append(result.Suggestions, additional.Suggestions...)

	if !additional.Valid {
		result.Valid = false
	}

	// Merge context
	for k, v := range additional.Context {
		result.Context[k] = v
	}

	// Merge performance metrics
	result.Performance.RulesEvaluated += additional.Performance.RulesEvaluated
	result.Performance.FieldsChecked += additional.Performance.FieldsChecked
	result.Performance.CacheHits += additional.Performance.CacheHits
	result.Performance.CacheMisses += additional.Performance.CacheMisses

	return result
}
