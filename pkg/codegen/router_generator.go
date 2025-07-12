package codegen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

// RouterGenerator generates Go router code from MCP schema
type RouterGenerator struct {
	Schema    MCPSchema    `json:"schema"`
	Config    RouterConfig `json:"config"`
	Templates map[string]*template.Template
}

// MCPSchema represents the complete MCP API schema
type MCPSchema struct {
	Version   string                   `json:"version"`
	Methods   map[string]MethodSchema  `json:"methods"`
	Types     map[string]TypeSchema    `json:"types"`
	Services  map[string]ServiceSchema `json:"services"`
	Generated time.Time                `json:"generated"`
}

// MethodSchema defines an MCP method
type MethodSchema struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Params      TypeSchema  `json:"params"`
	Result      TypeSchema  `json:"result"`
	Errors      []ErrorCode `json:"errors"`
	Auth        AuthConfig  `json:"auth"`
	RateLimit   RateLimit   `json:"rate_limit"`
}

// TypeSchema defines a data type
type TypeSchema struct {
	Type       string                    `json:"type"`
	Properties map[string]PropertySchema `json:"properties"`
	Required   []string                  `json:"required"`
	OneOf      []TypeSchema              `json:"oneOf"`
	Items      *TypeSchema               `json:"items"`
	Enum       []interface{}             `json:"enum"`
}

// PropertySchema defines a property within a type
type PropertySchema struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Format      string      `json:"format"`
	Default     interface{} `json:"default"`
	Minimum     *float64    `json:"minimum"`
	Maximum     *float64    `json:"maximum"`
	Pattern     string      `json:"pattern"`
}

// ServiceSchema defines service-specific schema
type ServiceSchema struct {
	Name      string                    `json:"name"`
	Type      string                    `json:"type"`
	Tools     map[string]ToolSchema     `json:"tools"`
	Resources map[string]ResourceSchema `json:"resources"`
	Prompts   map[string]PromptSchema   `json:"prompts"`
}

// ToolSchema defines a service tool
type ToolSchema struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	InputSchema TypeSchema `json:"input_schema"`
	OutputType  string     `json:"output_type"`
	Auth        AuthConfig `json:"auth"`
}

// ResourceSchema defines a service resource
type ResourceSchema struct {
	URIPattern  string     `json:"uri_pattern"`
	Description string     `json:"description"`
	MimeType    string     `json:"mime_type"`
	Auth        AuthConfig `json:"auth"`
}

// PromptSchema defines a service prompt
type PromptSchema struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Arguments   map[string]TypeSchema `json:"arguments"`
	Template    string                `json:"template"`
}

// RouterConfig configures router generation
type RouterConfig struct {
	PackageName       string            `json:"package_name"`
	OutputPath        string            `json:"output_path"`
	IncludeMetrics    bool              `json:"include_metrics"`
	IncludeLogging    bool              `json:"include_logging"`
	IncludeAuth       bool              `json:"include_auth"`
	IncludeValidation bool              `json:"include_validation"`
	CustomTypes       map[string]string `json:"custom_types"`
}

// AuthConfig defines authentication requirements
type AuthConfig struct {
	Required    bool     `json:"required"`
	Permissions []string `json:"permissions"`
	Scopes      []string `json:"scopes"`
}

// RateLimit defines rate limiting for methods
type RateLimit struct {
	Enabled        bool `json:"enabled"`
	RequestsPerMin int  `json:"requests_per_minute"`
	Burst          int  `json:"burst"`
}

// ErrorCode defines possible error responses
type ErrorCode struct {
	Code        int    `json:"code"`
	Message     string `json:"message"`
	Description string `json:"description"`
}

// NewRouterGenerator creates a new router generator
func NewRouterGenerator(schema MCPSchema, config RouterConfig) *RouterGenerator {
	rg := &RouterGenerator{
		Schema:    schema,
		Config:    config,
		Templates: make(map[string]*template.Template),
	}

	rg.loadTemplates()
	return rg
}

// GenerateRouter generates the complete router code
func (rg *RouterGenerator) GenerateRouter() (string, error) {
	var buf bytes.Buffer

	// Generate package header
	if err := rg.Templates["header"].Execute(&buf, rg); err != nil {
		return "", fmt.Errorf("failed to generate header: %w", err)
	}

	// Generate imports
	if err := rg.Templates["imports"].Execute(&buf, rg); err != nil {
		return "", fmt.Errorf("failed to generate imports: %w", err)
	}

	// Generate router function
	if err := rg.Templates["router"].Execute(&buf, rg); err != nil {
		return "", fmt.Errorf("failed to generate router: %w", err)
	}

	// Generate method handlers
	for methodName, method := range rg.Schema.Methods {
		handlerData := struct {
			Generator *RouterGenerator
			Method    MethodSchema
			Name      string
		}{
			Generator: rg,
			Method:    method,
			Name:      methodName,
		}

		if err := rg.Templates["handler"].Execute(&buf, handlerData); err != nil {
			return "", fmt.Errorf("failed to generate handler for %s: %w", methodName, err)
		}
	}

	// Generate validation functions
	if rg.Config.IncludeValidation {
		if err := rg.Templates["validation"].Execute(&buf, rg); err != nil {
			return "", fmt.Errorf("failed to generate validation: %w", err)
		}
	}

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("failed to format generated code: %w", err)
	}

	return string(formatted), nil
}

// loadTemplates loads all code generation templates
func (rg *RouterGenerator) loadTemplates() {
	rg.Templates["header"] = template.Must(template.New("header").Parse(headerTemplate))
	rg.Templates["imports"] = template.Must(template.New("imports").Parse(importsTemplate))
	rg.Templates["router"] = template.Must(template.New("router").Parse(routerTemplate))
	rg.Templates["handler"] = template.Must(template.New("handler").Parse(handlerTemplate))
	rg.Templates["validation"] = template.Must(template.New("validation").Parse(validationTemplate))
}

// Template definitions
const headerTemplate = `// Code generated by MCPEG router generator. DO NOT EDIT.
// Generated at: {{.Schema.Generated.Format "2006-01-02 15:04:05"}}
// Schema version: {{.Schema.Version}}

package {{.Config.PackageName}}
`

const importsTemplate = `
import (
	"context"
	"encoding/json"
	"net/http"
	"time"
	
	"github.com/gorilla/mux"
	"github.com/osakka/mcpeg/internal/adapter"
	"github.com/osakka/mcpeg/internal/mcp/types"
	{{if .Config.IncludeLogging}}"github.com/osakka/mcpeg/pkg/logging"{{end}}
	{{if .Config.IncludeMetrics}}"github.com/osakka/mcpeg/pkg/metrics"{{end}}
)
`

const routerTemplate = `
// GeneratedRouter creates the MCP router with all method handlers
func GeneratedRouter(
	adapters map[string]adapter.ServiceAdapter,
	{{if .Config.IncludeLogging}}logger logging.Logger,{{end}}
	{{if .Config.IncludeMetrics}}metrics metrics.Metrics,{{end}}
) *mux.Router {
	r := mux.NewRouter()
	
	// Add middleware
	{{if .Config.IncludeMetrics}}r.Use(metricsMiddleware(metrics)){{end}}
	{{if .Config.IncludeLogging}}r.Use(loggingMiddleware(logger)){{end}}
	{{if .Config.IncludeAuth}}r.Use(authMiddleware()){{end}}
	{{if .Config.IncludeValidation}}r.Use(validationMiddleware()){{end}}
	
	// MCP protocol routes
	{{range $name, $method := .Schema.Methods}}
	r.HandleFunc("/mcp/v1/{{$name}}", handle{{pascalCase $name}}(adapters{{if $.Config.IncludeLogging}}, logger{{end}}{{if $.Config.IncludeMetrics}}, metrics{{end}})).Methods("POST")
	{{end}}
	
	// Health and management routes
	r.HandleFunc("/health", handleHealth(adapters)).Methods("GET")
	r.HandleFunc("/health/live", handleLiveness()).Methods("GET")
	r.HandleFunc("/health/ready", handleReadiness(adapters)).Methods("GET")
	{{if .Config.IncludeMetrics}}r.HandleFunc("/metrics", handleMetrics(metrics)).Methods("GET"){{end}}
	
	return r
}
`

const handlerTemplate = `
// handle{{pascalCase .Name}} handles the {{.Name}} MCP method
func handle{{pascalCase .Name}}(
	adapters map[string]adapter.ServiceAdapter,
	{{if .Generator.Config.IncludeLogging}}logger logging.Logger,{{end}}
	{{if .Generator.Config.IncludeMetrics}}metrics metrics.Metrics,{{end}}
) http.HandlerFunc {
	{{if .Generator.Config.IncludeMetrics}}
	componentMetrics := metrics.NewComponentMetrics("mcp.{{.Name}}", metrics, logger)
	{{end}}
	
	return func(w http.ResponseWriter, r *http.Request) {
		{{if .Generator.Config.IncludeMetrics}}
		done := componentMetrics.StartOperation("{{.Name}}")
		defer done(nil)
		{{end}}
		
		// Parse JSON-RPC request
		var req types.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			{{if .Generator.Config.IncludeLogging}}
			logger.Error("invalid_json_rpc_request",
				"method", "{{.Name}}",
				"error", err,
				"suggested_actions", []string{
					"Check request format",
					"Verify Content-Type header",
					"Validate JSON syntax",
				})
			{{end}}
			writeErrorResponse(w, types.ErrorCodeParseError, "Invalid JSON-RPC request", nil)
			return
		}
		
		// Validate method name
		if req.Method != "{{.Name}}" {
			{{if .Generator.Config.IncludeLogging}}
			logger.Error("method_mismatch",
				"expected", "{{.Name}}",
				"received", req.Method,
				"suggested_actions", []string{
					"Check method name in request",
					"Verify endpoint URL",
				})
			{{end}}
			writeErrorResponse(w, types.ErrorCodeMethodNotFound, "Method not found", nil)
			return
		}
		
		{{if .Generator.Config.IncludeValidation}}
		// Validate parameters
		if err := validate{{pascalCase .Name}}Params(req.Params); err != nil {
			{{if .Generator.Config.IncludeLogging}}
			logger.Error("parameter_validation_failed",
				"method", "{{.Name}}",
				"error", err,
				"suggested_actions", []string{
					"Check parameter types",
					"Verify required parameters",
					"Review parameter format",
				})
			{{end}}
			writeErrorResponse(w, types.ErrorCodeInvalidParams, err.Error(), nil)
			return
		}
		{{end}}
		
		// Parse method-specific parameters
		var params {{.Method.Params.Type | goType}}
		if req.Params != nil {
			if err := json.Unmarshal(req.Params, &params); err != nil {
				{{if .Generator.Config.IncludeLogging}}
				logger.Error("parameter_parsing_failed",
					"method", "{{.Name}}",
					"error", err,
					"raw_params", string(req.Params))
				{{end}}
				writeErrorResponse(w, types.ErrorCodeInvalidParams, "Invalid parameters", nil)
				return
			}
		}
		
		// Execute method-specific logic
		result, err := execute{{pascalCase .Name}}(r.Context(), adapters, params{{if .Generator.Config.IncludeLogging}}, logger{{end}})
		if err != nil {
			{{if .Generator.Config.IncludeLogging}}
			logger.Error("method_execution_failed",
				"method", "{{.Name}}",
				"error", err,
				"params", params)
			{{end}}
			
			// Convert error to MCP error format
			mcpErr := convertToMCPError(err)
			writeErrorResponse(w, mcpErr.Code, mcpErr.Message, mcpErr.Data)
			return
		}
		
		// Write successful response
		response := types.Response{
			JSONRPC: "2.0",
			Result:  result,
			ID:      req.ID,
		}
		
		{{if .Generator.Config.IncludeLogging}}
		logger.Info("method_completed",
			"method", "{{.Name}}",
			"success", true)
		{{end}}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// execute{{pascalCase .Name}} implements the business logic for {{.Name}}
func execute{{pascalCase .Name}}(
	ctx context.Context,
	adapters map[string]adapter.ServiceAdapter,
	params {{.Method.Params.Type | goType}},
	{{if .Generator.Config.IncludeLogging}}logger logging.Logger,{{end}}
) (interface{}, error) {
	// TODO: Implement {{.Name}} business logic
	// This is where the actual MCP method implementation goes
	return nil, fmt.Errorf("{{.Name}} not yet implemented")
}
`

const validationTemplate = `
{{if .Config.IncludeValidation}}
{{range $name, $method := .Schema.Methods}}
// validate{{pascalCase $name}}Params validates parameters for {{$name}}
func validate{{pascalCase $name}}Params(params json.RawMessage) error {
	if params == nil {
		{{if hasRequiredParams $method.Params}}
		return fmt.Errorf("missing required parameters")
		{{else}}
		return nil
		{{end}}
	}
	
	// TODO: Implement JSON schema validation for {{$name}}
	// Validate against: {{$method.Params | json}}
	return nil
}
{{end}}
{{end}}
`

// Helper functions for templates
func init() {
	// Add custom template functions
	funcMap := template.FuncMap{
		"pascalCase":        pascalCase,
		"goType":            goType,
		"json":              toJSON,
		"hasRequiredParams": hasRequiredParams,
	}

	// Apply function map to all templates
	template.Must(template.New("").Funcs(funcMap).Parse(""))
}

func pascalCase(s string) string {
	if len(s) == 0 {
		return s
	}

	// Convert snake_case or kebab-case to PascalCase
	result := ""
	capitalize := true

	for _, r := range s {
		if r == '_' || r == '-' || r == '/' {
			capitalize = true
		} else if capitalize {
			result += string(r - 'a' + 'A')
			capitalize = false
		} else {
			result += string(r)
		}
	}

	return result
}

func goType(typeName string) string {
	switch typeName {
	case "string":
		return "string"
	case "integer":
		return "int"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	case "array":
		return "[]interface{}"
	case "object":
		return "map[string]interface{}"
	default:
		return "interface{}"
	}
}

func toJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func hasRequiredParams(params TypeSchema) bool {
	return len(params.Required) > 0
}

// GenerateFromConfig generates router from configuration
func GenerateFromConfig(configPath string) error {
	// Load schema from configuration file
	schema, err := LoadSchemaFromMCP(configPath)
	if err != nil {
		return fmt.Errorf("failed to load schema from config: %w", err)
	}

	// Generate router code
	config := RouterConfig{
		PackageName:       "generated",
		IncludeLogging:    true,
		IncludeMetrics:    true,
		IncludeValidation: true,
	}
	generator := NewRouterGenerator(schema, config)
	code, err := generator.GenerateRouter()
	if err != nil {
		return fmt.Errorf("failed to generate router code: %w", err)
	}

	// Write to output path (for now, just return the code)
	fmt.Printf("Generated router code:\n%s\n", code)

	return nil
}

// LoadSchemaFromMCP loads schema from MCP specification
func LoadSchemaFromMCP(specPath string) (MCPSchema, error) {
	// Load the specification file
	data, err := os.ReadFile(specPath)
	if err != nil {
		return MCPSchema{}, fmt.Errorf("failed to read spec file: %w", err)
	}

	// Parse as JSON or YAML
	var spec map[string]interface{}
	if strings.HasSuffix(specPath, ".yaml") || strings.HasSuffix(specPath, ".yml") {
		err = yaml.Unmarshal(data, &spec)
	} else {
		err = json.Unmarshal(data, &spec)
	}
	if err != nil {
		return MCPSchema{}, fmt.Errorf("failed to parse spec: %w", err)
	}

	// Extract method definitions from the spec
	schema := MCPSchema{
		Version: "1.0.0",
		Methods: make(map[string]MethodSchema),
	}

	// For now, return a basic schema with common MCP methods
	// In production, this would parse the actual MCP specification
	schema.Methods["initialize"] = MethodSchema{
		Name:        "initialize",
		Description: "Initialize the MCP connection",
		Params: TypeSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"protocolVersion": {Type: "string"},
				"capabilities":    {Type: "object"},
			},
			Required: []string{"protocolVersion"},
		},
		Result: TypeSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"protocolVersion": {Type: "string"},
				"capabilities":    {Type: "object"},
			},
		},
	}

	return schema, nil
}
