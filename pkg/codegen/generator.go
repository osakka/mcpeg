package codegen

import (
	"bytes"
	"context"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
)

// CodeGenerator generates Go code from OpenAPI specifications
type CodeGenerator struct {
	logger    logging.Logger
	metrics   metrics.Metrics
	config    GeneratorConfig
	templates map[string]*template.Template
}

// GeneratorConfig configures code generation behavior
type GeneratorConfig struct {
	// Output settings
	OutputDir        string `yaml:"output_dir"`
	PackageName      string `yaml:"package_name"`
	ModulePath       string `yaml:"module_path"`
	
	// Generation options
	GenerateTypes    bool   `yaml:"generate_types"`
	GenerateHandlers bool   `yaml:"generate_handlers"`
	GenerateClients  bool   `yaml:"generate_clients"`
	GenerateValidators bool `yaml:"generate_validators"`
	GenerateTests    bool   `yaml:"generate_tests"`
	
	// Code style options
	UsePointers      bool   `yaml:"use_pointers"`
	JSONTags         bool   `yaml:"json_tags"`
	ValidationTags   bool   `yaml:"validation_tags"`
	
	// Template options
	TemplateDir      string `yaml:"template_dir"`
	CustomTemplates  map[string]string `yaml:"custom_templates"`
}

// OpenAPISpec represents a parsed OpenAPI specification
type OpenAPISpec struct {
	OpenAPI    string                 `json:"openapi"`
	Info       APIInfo                `json:"info"`
	Servers    []Server               `json:"servers"`
	Paths      map[string]PathItem    `json:"paths"`
	Components Components             `json:"components"`
	Security   []SecurityRequirement  `json:"security"`
	Tags       []Tag                  `json:"tags"`
}

// APIInfo represents API information
type APIInfo struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Version     string  `json:"version"`
	Contact     Contact `json:"contact"`
	License     License `json:"license"`
}

// Contact represents contact information
type Contact struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// License represents license information
type License struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Server represents a server configuration
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

// PathItem represents a path and its operations
type PathItem struct {
	GET    *Operation `json:"get,omitempty"`
	POST   *Operation `json:"post,omitempty"`
	PUT    *Operation `json:"put,omitempty"`
	DELETE *Operation `json:"delete,omitempty"`
	PATCH  *Operation `json:"patch,omitempty"`
}

// Operation represents an API operation
type Operation struct {
	OperationID string                `json:"operationId"`
	Summary     string                `json:"summary"`
	Description string                `json:"description"`
	Tags        []string              `json:"tags"`
	Parameters  []Parameter           `json:"parameters"`
	RequestBody *RequestBody          `json:"requestBody,omitempty"`
	Responses   map[string]Response   `json:"responses"`
	Security    []SecurityRequirement `json:"security"`
}

// Parameter represents an operation parameter
type Parameter struct {
	Name        string      `json:"name"`
	In          string      `json:"in"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Schema      Schema      `json:"schema"`
	Example     interface{} `json:"example"`
}

// RequestBody represents a request body
type RequestBody struct {
	Description string               `json:"description"`
	Required    bool                 `json:"required"`
	Content     map[string]MediaType `json:"content"`
}

// Response represents an API response
type Response struct {
	Description string               `json:"description"`
	Headers     map[string]Header    `json:"headers"`
	Content     map[string]MediaType `json:"content"`
}

// Header represents a response header
type Header struct {
	Description string      `json:"description"`
	Schema      Schema      `json:"schema"`
	Example     interface{} `json:"example"`
}

// MediaType represents a media type
type MediaType struct {
	Schema   Schema                 `json:"schema"`
	Example  interface{}            `json:"example"`
	Examples map[string]Example     `json:"examples"`
}

// Example represents an example value
type Example struct {
	Summary     string      `json:"summary"`
	Description string      `json:"description"`
	Value       interface{} `json:"value"`
}

// Components represents reusable components
type Components struct {
	Schemas         map[string]Schema         `json:"schemas"`
	Responses       map[string]Response       `json:"responses"`
	Parameters      map[string]Parameter      `json:"parameters"`
	RequestBodies   map[string]RequestBody    `json:"requestBodies"`
	Headers         map[string]Header         `json:"headers"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes"`
}

// Schema represents a JSON schema
type Schema struct {
	Type                 string            `json:"type,omitempty"`
	Format               string            `json:"format,omitempty"`
	Title                string            `json:"title,omitempty"`
	Description          string            `json:"description,omitempty"`
	Default              interface{}       `json:"default,omitempty"`
	Example              interface{}       `json:"example,omitempty"`
	Enum                 []interface{}     `json:"enum,omitempty"`
	Required             []string          `json:"required,omitempty"`
	Properties           map[string]Schema `json:"properties,omitempty"`
	Items                *Schema           `json:"items,omitempty"`
	AdditionalProperties interface{}       `json:"additionalProperties,omitempty"`
	AllOf                []Schema          `json:"allOf,omitempty"`
	OneOf                []Schema          `json:"oneOf,omitempty"`
	AnyOf                []Schema          `json:"anyOf,omitempty"`
	Ref                  string            `json:"$ref,omitempty"`
	Minimum              *float64          `json:"minimum,omitempty"`
	Maximum              *float64          `json:"maximum,omitempty"`
	MinLength            *int              `json:"minLength,omitempty"`
	MaxLength            *int              `json:"maxLength,omitempty"`
	Pattern              string            `json:"pattern,omitempty"`
}

// SecurityScheme represents a security scheme
type SecurityScheme struct {
	Type         string `json:"type"`
	Description  string `json:"description"`
	Name         string `json:"name"`
	In           string `json:"in"`
	Scheme       string `json:"scheme"`
	BearerFormat string `json:"bearerFormat"`
}

// SecurityRequirement represents a security requirement
type SecurityRequirement map[string][]string

// Tag represents an operation tag
type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// GeneratedCode represents generated code output
type GeneratedCode struct {
	Package   string
	Imports   []string
	Types     []TypeDefinition
	Functions []FunctionDefinition
	Constants []ConstantDefinition
	Variables []VariableDefinition
}

// TypeDefinition represents a generated type
type TypeDefinition struct {
	Name        string
	Type        string
	Comment     string
	Fields      []FieldDefinition
	Methods     []MethodDefinition
	Tags        map[string]string
}

// FieldDefinition represents a struct field
type FieldDefinition struct {
	Name     string
	Type     string
	Comment  string
	Tags     map[string]string
	Optional bool
}

// MethodDefinition represents a method
type MethodDefinition struct {
	Name       string
	Receiver   string
	Parameters []ParameterDefinition
	Returns    []ParameterDefinition
	Body       string
	Comment    string
}

// FunctionDefinition represents a function
type FunctionDefinition struct {
	Name       string
	Parameters []ParameterDefinition
	Returns    []ParameterDefinition
	Body       string
	Comment    string
}

// ParameterDefinition represents a function/method parameter
type ParameterDefinition struct {
	Name string
	Type string
}

// ConstantDefinition represents a constant
type ConstantDefinition struct {
	Name    string
	Type    string
	Value   string
	Comment string
}

// VariableDefinition represents a variable
type VariableDefinition struct {
	Name    string
	Type    string
	Value   string
	Comment string
}

// NewCodeGenerator creates a new code generator
func NewCodeGenerator(logger logging.Logger, metrics metrics.Metrics) *CodeGenerator {
	cg := &CodeGenerator{
		logger:    logger.WithComponent("code_generator"),
		metrics:   metrics,
		config:    defaultGeneratorConfig(),
		templates: make(map[string]*template.Template),
	}
	
	// Load default templates
	if err := cg.loadTemplates(); err != nil {
		logger.Error("failed_to_load_templates", "error", err)
	}
	
	return cg
}

// GenerateFromSpec generates Go code from an OpenAPI specification
func (cg *CodeGenerator) GenerateFromSpec(ctx context.Context, spec *OpenAPISpec) (*GeneratedCode, error) {
	start := time.Now()
	
	cg.logger.Info("starting_code_generation",
		"spec_title", spec.Info.Title,
		"spec_version", spec.Info.Version,
		"paths_count", len(spec.Paths),
		"schemas_count", len(spec.Components.Schemas))
	
	generated := &GeneratedCode{
		Package: cg.config.PackageName,
		Imports: cg.generateImports(spec),
	}
	
	// Generate types from schemas
	if cg.config.GenerateTypes {
		types, err := cg.generateTypes(ctx, spec)
		if err != nil {
			return nil, fmt.Errorf("failed to generate types: %w", err)
		}
		generated.Types = types
	}
	
	// Generate handlers from paths
	if cg.config.GenerateHandlers {
		functions, err := cg.generateHandlers(ctx, spec)
		if err != nil {
			return nil, fmt.Errorf("failed to generate handlers: %w", err)
		}
		generated.Functions = append(generated.Functions, functions...)
	}
	
	// Generate client code
	if cg.config.GenerateClients {
		clientTypes, clientFunctions, err := cg.generateClients(ctx, spec)
		if err != nil {
			return nil, fmt.Errorf("failed to generate clients: %w", err)
		}
		generated.Types = append(generated.Types, clientTypes...)
		generated.Functions = append(generated.Functions, clientFunctions...)
	}
	
	// Generate validators
	if cg.config.GenerateValidators {
		validators, err := cg.generateValidators(ctx, spec)
		if err != nil {
			return nil, fmt.Errorf("failed to generate validators: %w", err)
		}
		generated.Functions = append(generated.Functions, validators...)
	}
	
	// Generate constants
	constants := cg.generateConstants(spec)
	generated.Constants = constants
	
	duration := time.Since(start)
	
	cg.logger.Info("code_generation_completed",
		"duration", duration,
		"types_generated", len(generated.Types),
		"functions_generated", len(generated.Functions),
		"constants_generated", len(generated.Constants))
	
	// Record metrics
	cg.recordGenerationMetrics(spec, generated, duration)
	
	return generated, nil
}

// WriteCode writes generated code to files
func (cg *CodeGenerator) WriteCode(ctx context.Context, generated *GeneratedCode) error {
	if err := os.MkdirAll(cg.config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Generate main types file
	if len(generated.Types) > 0 {
		if err := cg.writeTypesFile(generated); err != nil {
			return fmt.Errorf("failed to write types file: %w", err)
		}
	}
	
	// Generate handlers file
	if len(generated.Functions) > 0 {
		if err := cg.writeFunctionsFile(generated); err != nil {
			return fmt.Errorf("failed to write functions file: %w", err)
		}
	}
	
	// Generate constants file
	if len(generated.Constants) > 0 {
		if err := cg.writeConstantsFile(generated); err != nil {
			return fmt.Errorf("failed to write constants file: %w", err)
		}
	}
	
	cg.logger.Info("code_files_written",
		"output_dir", cg.config.OutputDir,
		"package", generated.Package)
	
	return nil
}

// generateTypes generates Go types from OpenAPI schemas
func (cg *CodeGenerator) generateTypes(ctx context.Context, spec *OpenAPISpec) ([]TypeDefinition, error) {
	var types []TypeDefinition
	
	for name, schema := range spec.Components.Schemas {
		typeDef, err := cg.schemaToType(name, schema, spec)
		if err != nil {
			return nil, fmt.Errorf("failed to convert schema %s: %w", name, err)
		}
		types = append(types, *typeDef)
	}
	
	return types, nil
}

// generateHandlers generates HTTP handlers from OpenAPI paths
func (cg *CodeGenerator) generateHandlers(ctx context.Context, spec *OpenAPISpec) ([]FunctionDefinition, error) {
	var functions []FunctionDefinition
	
	for path, pathItem := range spec.Paths {
		if pathItem.GET != nil {
			fn := cg.operationToHandler("GET", path, pathItem.GET, spec)
			functions = append(functions, fn)
		}
		if pathItem.POST != nil {
			fn := cg.operationToHandler("POST", path, pathItem.POST, spec)
			functions = append(functions, fn)
		}
		if pathItem.PUT != nil {
			fn := cg.operationToHandler("PUT", path, pathItem.PUT, spec)
			functions = append(functions, fn)
		}
		if pathItem.DELETE != nil {
			fn := cg.operationToHandler("DELETE", path, pathItem.DELETE, spec)
			functions = append(functions, fn)
		}
	}
	
	return functions, nil
}

// generateClients generates client code from OpenAPI paths
func (cg *CodeGenerator) generateClients(ctx context.Context, spec *OpenAPISpec) ([]TypeDefinition, []FunctionDefinition, error) {
	var types []TypeDefinition
	var functions []FunctionDefinition
	
	// Generate client struct
	clientType := TypeDefinition{
		Name:    "Client",
		Type:    "struct",
		Comment: fmt.Sprintf("Client provides access to the %s API", spec.Info.Title),
		Fields: []FieldDefinition{
			{
				Name: "baseURL",
				Type: "string",
				Tags: map[string]string{"json": "base_url"},
			},
			{
				Name: "httpClient",
				Type: "*http.Client",
				Tags: map[string]string{"json": "-"},
			},
			{
				Name: "apiKey",
				Type: "string",
				Tags: map[string]string{"json": "-"},
			},
		},
	}
	types = append(types, clientType)
	
	// Generate constructor
	constructor := FunctionDefinition{
		Name: "NewClient",
		Parameters: []ParameterDefinition{
			{Name: "baseURL", Type: "string"},
			{Name: "apiKey", Type: "string"},
		},
		Returns: []ParameterDefinition{
			{Name: "", Type: "*Client"},
		},
		Body: cg.generateClientConstructor(),
		Comment: "NewClient creates a new API client",
	}
	functions = append(functions, constructor)
	
	// Generate client methods for each operation
	for path, pathItem := range spec.Paths {
		if pathItem.GET != nil {
			method := cg.operationToClientMethod("GET", path, pathItem.GET, spec)
			functions = append(functions, method)
		}
		if pathItem.POST != nil {
			method := cg.operationToClientMethod("POST", path, pathItem.POST, spec)
			functions = append(functions, method)
		}
		// Add other HTTP methods as needed
	}
	
	return types, functions, nil
}

// generateValidators generates validation functions
func (cg *CodeGenerator) generateValidators(ctx context.Context, spec *OpenAPISpec) ([]FunctionDefinition, error) {
	var functions []FunctionDefinition
	
	for name, schema := range spec.Components.Schemas {
		validator := cg.schemaToValidator(name, schema)
		functions = append(functions, validator)
	}
	
	return functions, nil
}

// generateConstants generates constants from the specification
func (cg *CodeGenerator) generateConstants(spec *OpenAPISpec) []ConstantDefinition {
	var constants []ConstantDefinition
	
	// API version constant
	constants = append(constants, ConstantDefinition{
		Name:    "APIVersion",
		Type:    "string",
		Value:   fmt.Sprintf(`"%s"`, spec.Info.Version),
		Comment: "APIVersion is the version of the API",
	})
	
	// Server URLs
	for i, server := range spec.Servers {
		name := fmt.Sprintf("ServerURL%d", i)
		if server.Description != "" {
			name = toCamelCase(server.Description) + "URL"
		}
		constants = append(constants, ConstantDefinition{
			Name:    name,
			Type:    "string",
			Value:   fmt.Sprintf(`"%s"`, server.URL),
			Comment: fmt.Sprintf("%s: %s", name, server.Description),
		})
	}
	
	return constants
}

// generateImports generates import statements
func (cg *CodeGenerator) generateImports(spec *OpenAPISpec) []string {
	imports := []string{
		"context",
		"encoding/json",
		"fmt",
		"net/http",
		"time",
	}
	
	// Add validation imports if generating validators
	if cg.config.GenerateValidators {
		imports = append(imports, "github.com/osakka/mcpeg/pkg/validation")
	}
	
	return imports
}

// schemaToType converts an OpenAPI schema to a Go type definition
func (cg *CodeGenerator) schemaToType(name string, schema Schema, spec *OpenAPISpec) (*TypeDefinition, error) {
	typeDef := &TypeDefinition{
		Name:    toPascalCase(name),
		Comment: schema.Description,
		Tags:    make(map[string]string),
	}
	
	if schema.Type == "object" || len(schema.Properties) > 0 {
		typeDef.Type = "struct"
		
		for propName, propSchema := range schema.Properties {
			field := FieldDefinition{
				Name:    toPascalCase(propName),
				Comment: propSchema.Description,
				Tags:    make(map[string]string),
			}
			
			// Determine if field is optional
			isRequired := false
			for _, required := range schema.Required {
				if required == propName {
					isRequired = true
					break
				}
			}
			field.Optional = !isRequired
			
			// Generate Go type for field
			goType, err := cg.schemaToGoType(propSchema, spec)
			if err != nil {
				return nil, fmt.Errorf("failed to convert property %s: %w", propName, err)
			}
			
			if field.Optional && !strings.HasPrefix(goType, "*") && !strings.HasPrefix(goType, "[]") && !strings.HasPrefix(goType, "map[") {
				goType = "*" + goType
			}
			
			field.Type = goType
			
			// Add JSON tags
			if cg.config.JSONTags {
				jsonTag := propName
				if field.Optional {
					jsonTag += ",omitempty"
				}
				field.Tags["json"] = jsonTag
			}
			
			// Add validation tags
			if cg.config.ValidationTags {
				validationTag := cg.generateValidationTag(propSchema, isRequired)
				if validationTag != "" {
					field.Tags["validate"] = validationTag
				}
			}
			
			typeDef.Fields = append(typeDef.Fields, field)
		}
	} else {
		// Handle non-object types (aliases)
		goType, err := cg.schemaToGoType(schema, spec)
		if err != nil {
			return nil, err
		}
		typeDef.Type = goType
	}
	
	return typeDef, nil
}

// schemaToGoType converts an OpenAPI schema to a Go type string
func (cg *CodeGenerator) schemaToGoType(schema Schema, spec *OpenAPISpec) (string, error) {
	// Handle references
	if schema.Ref != "" {
		refName := extractRefName(schema.Ref)
		return toPascalCase(refName), nil
	}
	
	switch schema.Type {
	case "string":
		if len(schema.Enum) > 0 {
			// For enum types, generate a string type for now
		// In a full implementation, we'd generate custom enum types
		return "string", nil
		}
		return "string", nil
	case "integer":
		if schema.Format == "int64" {
			return "int64", nil
		}
		return "int", nil
	case "number":
		if schema.Format == "float" {
			return "float32", nil
		}
		return "float64", nil
	case "boolean":
		return "bool", nil
	case "array":
		if schema.Items == nil {
			return "[]interface{}", nil
		}
		itemType, err := cg.schemaToGoType(*schema.Items, spec)
		if err != nil {
			return "", err
		}
		return "[]" + itemType, nil
	case "object":
		if schema.AdditionalProperties != nil {
			// Handle typed additional properties
			if propSchema, ok := schema.AdditionalProperties.(Schema); ok {
				propType, err := cg.schemaToGoType(propSchema, spec)
				if err != nil {
					return "map[string]interface{}", nil
				}
				return "map[string]" + propType, nil
			}
			return "map[string]interface{}", nil
		}
		return "map[string]interface{}", nil
	default:
		return "interface{}", nil
	}
}

// operationToHandler converts an OpenAPI operation to a handler function
func (cg *CodeGenerator) operationToHandler(method, path string, op *Operation, spec *OpenAPISpec) FunctionDefinition {
	funcName := toPascalCase(op.OperationID)
	if funcName == "" {
		funcName = toPascalCase(method + strings.ReplaceAll(path, "/", "_"))
	}
	
	return FunctionDefinition{
		Name: funcName,
		Parameters: []ParameterDefinition{
			{Name: "w", Type: "http.ResponseWriter"},
			{Name: "r", Type: "*http.Request"},
		},
		Body:    cg.generateHandlerBody(method, path, op, spec),
		Comment: fmt.Sprintf("%s handles %s %s - %s", funcName, method, path, op.Summary),
	}
}

// operationToClientMethod converts an OpenAPI operation to a client method
func (cg *CodeGenerator) operationToClientMethod(method, path string, op *Operation, spec *OpenAPISpec) FunctionDefinition {
	funcName := toPascalCase(op.OperationID)
	if funcName == "" {
		funcName = toPascalCase(method + strings.ReplaceAll(path, "/", "_"))
	}
	
	params := []ParameterDefinition{
		{Name: "c", Type: "*Client"},
		{Name: "ctx", Type: "context.Context"},
	}
	
	// Add parameters from operation
	for _, param := range op.Parameters {
		goType, _ := cg.schemaToGoType(param.Schema, spec)
		params = append(params, ParameterDefinition{
			Name: toCamelCase(param.Name),
			Type: goType,
		})
	}
	
	// Add request body parameter if present
	if op.RequestBody != nil {
		bodyType := "interface{}"
		if op.RequestBody.Content != nil {
			for contentType, mediaType := range op.RequestBody.Content {
				if contentType == "application/json" && mediaType.Schema.Type != "" {
					if generatedType, err := cg.schemaToGoType(mediaType.Schema, spec); err == nil {
						bodyType = generatedType
					}
				}
			}
		}
		params = append(params, ParameterDefinition{
			Name: "body",
			Type: bodyType,
		})
	}
	
	// Generate return type from response schemas
	responseType := "interface{}"
	if op.Responses != nil {
		for statusCode, response := range op.Responses {
			if statusCode == "200" || statusCode == "201" {
				if response.Content != nil {
					for contentType, mediaType := range response.Content {
						if contentType == "application/json" && mediaType.Schema.Type != "" {
							if generatedType, err := cg.schemaToGoType(mediaType.Schema, spec); err == nil {
								responseType = generatedType
							}
						}
					}
				}
			}
		}
	}
	
	returns := []ParameterDefinition{
		{Name: "", Type: responseType},
		{Name: "", Type: "error"},
	}
	
	return FunctionDefinition{
		Name:       funcName,
		Parameters: params,
		Returns:    returns,
		Body:       cg.generateClientMethodBody(method, path, op, spec),
		Comment:    fmt.Sprintf("%s calls %s %s - %s", funcName, method, path, op.Summary),
	}
}

// schemaToValidator generates a validation function for a schema
func (cg *CodeGenerator) schemaToValidator(name string, schema Schema) FunctionDefinition {
	funcName := fmt.Sprintf("Validate%s", toPascalCase(name))
	
	return FunctionDefinition{
		Name: funcName,
		Parameters: []ParameterDefinition{
			{Name: "ctx", Type: "context.Context"},
			{Name: "value", Type: toPascalCase(name)},
		},
		Returns: []ParameterDefinition{
			{Name: "", Type: "validation.ValidationResult"},
		},
		Body:    cg.generateValidatorBody(name, schema),
		Comment: fmt.Sprintf("%s validates a %s instance", funcName, name),
	}
}

// generateHandlerBody generates the body of a handler function
func (cg *CodeGenerator) generateHandlerBody(method, path string, op *Operation, spec *OpenAPISpec) string {
	tmpl := `	// TODO: Implement handler for {{.Method}} {{.Path}}
	// Summary: {{.Summary}}
	// Description: {{.Description}}
	
	w.Header().Set("Content-Type", "application/json")
	
	// Extract parameters
	{{range .Parameters}}
	// {{.Name}} ({{.In}}): {{.Description}}
	{{end}}
	
	// Process request
	response := map[string]interface{}{
		"message": "Handler not implemented",
		"operation": "{{.OperationID}}",
	}
	
	json.NewEncoder(w).Encode(response)`
	
	t := template.Must(template.New("handler").Parse(tmpl))
	var buf bytes.Buffer
	t.Execute(&buf, map[string]interface{}{
		"Method":      method,
		"Path":        path,
		"Summary":     op.Summary,
		"Description": op.Description,
		"OperationID": op.OperationID,
		"Parameters":  op.Parameters,
	})
	
	return buf.String()
}

// generateClientMethodBody generates the body of a client method
func (cg *CodeGenerator) generateClientMethodBody(method, path string, op *Operation, spec *OpenAPISpec) string {
	return fmt.Sprintf(`	// HTTP %s request to %s
	url := c.baseURL + "%s"
	
	// Create request
	req, err := http.NewRequest("%s", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %%w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer " + c.apiKey)
	}
	
	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %%w", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %%d: request failed", resp.StatusCode)
	}
	
	// Parse response
	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %%w", err)
	}
	
	return result, nil`, method, path, path, strings.ToUpper(method))
}

// generateClientConstructor generates the client constructor body
func (cg *CodeGenerator) generateClientConstructor() string {
	return `	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     apiKey,
	}`
}

// generateValidatorBody generates the body of a validator function
func (cg *CodeGenerator) generateValidatorBody(name string, schema Schema) string {
	return fmt.Sprintf(`	// Validate %s against schema
	if input == nil {
		return validation.ValidationResult{Valid: false, Error: "input is nil"}
	}
	
	// Basic validation - in production, use JSON schema validation
	result := validation.ValidationResult{Valid: true}
	
	// Type validation
	if schema.Type != "" {
		// Add type checking logic here
	}
	
	return result`, name)
}

// generateValidationTag generates validation tags for a schema
func (cg *CodeGenerator) generateValidationTag(schema Schema, required bool) string {
	var tags []string
	
	if required {
		tags = append(tags, "required")
	}
	
	if schema.MinLength != nil {
		tags = append(tags, fmt.Sprintf("min=%d", *schema.MinLength))
	}
	
	if schema.MaxLength != nil {
		tags = append(tags, fmt.Sprintf("max=%d", *schema.MaxLength))
	}
	
	if schema.Pattern != "" {
		tags = append(tags, fmt.Sprintf("regex=%s", schema.Pattern))
	}
	
	return strings.Join(tags, ",")
}

// Helper functions for code generation

func toPascalCase(s string) string {
	if s == "" {
		return ""
	}
	
	words := strings.FieldsFunc(s, func(c rune) bool {
		return c == '_' || c == '-' || c == ' ' || c == '.'
	})
	
	for i, word := range words {
		words[i] = strings.Title(strings.ToLower(word))
	}
	
	return strings.Join(words, "")
}

func toCamelCase(s string) string {
	pascal := toPascalCase(s)
	if len(pascal) == 0 {
		return ""
	}
	
	return strings.ToLower(pascal[:1]) + pascal[1:]
}

func extractRefName(ref string) string {
	parts := strings.Split(ref, "/")
	return parts[len(parts)-1]
}

func defaultGeneratorConfig() GeneratorConfig {
	return GeneratorConfig{
		OutputDir:        "generated",
		PackageName:      "generated",
		ModulePath:       "github.com/osakka/mcpeg",
		GenerateTypes:    true,
		GenerateHandlers: true,
		GenerateClients:  true,
		GenerateValidators: true,
		GenerateTests:    false,
		UsePointers:      true,
		JSONTags:         true,
		ValidationTags:   true,
		TemplateDir:      "templates",
		CustomTemplates:  make(map[string]string),
	}
}

// File writing methods

func (cg *CodeGenerator) writeTypesFile(generated *GeneratedCode) error {
	filename := filepath.Join(cg.config.OutputDir, "types.go")
	content := cg.renderTypesFile(generated)
	
	formatted, err := format.Source([]byte(content))
	if err != nil {
		cg.logger.Warn("failed_to_format_types_file", "error", err)
		formatted = []byte(content)
	}
	
	return os.WriteFile(filename, formatted, 0644)
}

func (cg *CodeGenerator) writeFunctionsFile(generated *GeneratedCode) error {
	filename := filepath.Join(cg.config.OutputDir, "handlers.go")
	content := cg.renderFunctionsFile(generated)
	
	formatted, err := format.Source([]byte(content))
	if err != nil {
		cg.logger.Warn("failed_to_format_functions_file", "error", err)
		formatted = []byte(content)
	}
	
	return os.WriteFile(filename, formatted, 0644)
}

func (cg *CodeGenerator) writeConstantsFile(generated *GeneratedCode) error {
	filename := filepath.Join(cg.config.OutputDir, "constants.go")
	content := cg.renderConstantsFile(generated)
	
	formatted, err := format.Source([]byte(content))
	if err != nil {
		cg.logger.Warn("failed_to_format_constants_file", "error", err)
		formatted = []byte(content)
	}
	
	return os.WriteFile(filename, formatted, 0644)
}

func (cg *CodeGenerator) renderTypesFile(generated *GeneratedCode) string {
	var buf bytes.Buffer
	
	// Package declaration
	fmt.Fprintf(&buf, "package %s\n\n", generated.Package)
	
	// Imports
	if len(generated.Imports) > 0 {
		buf.WriteString("import (\n")
		for _, imp := range generated.Imports {
			fmt.Fprintf(&buf, "\t\"%s\"\n", imp)
		}
		buf.WriteString(")\n\n")
	}
	
	// Types
	for _, typeDef := range generated.Types {
		if typeDef.Comment != "" {
			fmt.Fprintf(&buf, "// %s %s\n", typeDef.Name, typeDef.Comment)
		}
		
		if typeDef.Type == "struct" {
			fmt.Fprintf(&buf, "type %s struct {\n", typeDef.Name)
			for _, field := range typeDef.Fields {
				if field.Comment != "" {
					fmt.Fprintf(&buf, "\t// %s\n", field.Comment)
				}
				
				tagStr := ""
				if len(field.Tags) > 0 {
					var tags []string
					for k, v := range field.Tags {
						tags = append(tags, fmt.Sprintf(`%s:"%s"`, k, v))
					}
					tagStr = fmt.Sprintf(" `%s`", strings.Join(tags, " "))
				}
				
				fmt.Fprintf(&buf, "\t%s %s%s\n", field.Name, field.Type, tagStr)
			}
			buf.WriteString("}\n\n")
		} else {
			fmt.Fprintf(&buf, "type %s %s\n\n", typeDef.Name, typeDef.Type)
		}
	}
	
	return buf.String()
}

func (cg *CodeGenerator) renderFunctionsFile(generated *GeneratedCode) string {
	var buf bytes.Buffer
	
	// Package declaration
	fmt.Fprintf(&buf, "package %s\n\n", generated.Package)
	
	// Imports
	if len(generated.Imports) > 0 {
		buf.WriteString("import (\n")
		for _, imp := range generated.Imports {
			fmt.Fprintf(&buf, "\t\"%s\"\n", imp)
		}
		buf.WriteString(")\n\n")
	}
	
	// Functions
	for _, funcDef := range generated.Functions {
		if funcDef.Comment != "" {
			fmt.Fprintf(&buf, "// %s\n", funcDef.Comment)
		}
		
		// Function signature
		buf.WriteString("func ")
		if len(funcDef.Parameters) > 0 && funcDef.Parameters[0].Type == "*Client" {
			// Method
			fmt.Fprintf(&buf, "(%s %s) %s(", funcDef.Parameters[0].Name, funcDef.Parameters[0].Type, funcDef.Name)
			for i, param := range funcDef.Parameters[1:] {
				if i > 0 {
					buf.WriteString(", ")
				}
				fmt.Fprintf(&buf, "%s %s", param.Name, param.Type)
			}
		} else {
			// Function
			fmt.Fprintf(&buf, "%s(", funcDef.Name)
			for i, param := range funcDef.Parameters {
				if i > 0 {
					buf.WriteString(", ")
				}
				fmt.Fprintf(&buf, "%s %s", param.Name, param.Type)
			}
		}
		buf.WriteString(")")
		
		// Return types
		if len(funcDef.Returns) > 0 {
			if len(funcDef.Returns) == 1 {
				fmt.Fprintf(&buf, " %s", funcDef.Returns[0].Type)
			} else {
				buf.WriteString(" (")
				for i, ret := range funcDef.Returns {
					if i > 0 {
						buf.WriteString(", ")
					}
					buf.WriteString(ret.Type)
				}
				buf.WriteString(")")
			}
		}
		
		buf.WriteString(" {\n")
		buf.WriteString(funcDef.Body)
		buf.WriteString("\n}\n\n")
	}
	
	return buf.String()
}

func (cg *CodeGenerator) renderConstantsFile(generated *GeneratedCode) string {
	var buf bytes.Buffer
	
	// Package declaration
	fmt.Fprintf(&buf, "package %s\n\n", generated.Package)
	
	// Constants
	if len(generated.Constants) > 0 {
		buf.WriteString("const (\n")
		for _, constDef := range generated.Constants {
			if constDef.Comment != "" {
				fmt.Fprintf(&buf, "\t// %s\n", constDef.Comment)
			}
			fmt.Fprintf(&buf, "\t%s = %s\n", constDef.Name, constDef.Value)
		}
		buf.WriteString(")\n")
	}
	
	return buf.String()
}

func (cg *CodeGenerator) loadTemplates() error {
	// Load default templates - in a real implementation,
	// this would load templates from files
	return nil
}

func (cg *CodeGenerator) recordGenerationMetrics(spec *OpenAPISpec, generated *GeneratedCode, duration time.Duration) {
	labels := []string{
		"spec_title", spec.Info.Title,
		"spec_version", spec.Info.Version,
	}
	
	cg.metrics.Set("codegen_duration_seconds", duration.Seconds(), labels...)
	cg.metrics.Set("codegen_types_generated", float64(len(generated.Types)), labels...)
	cg.metrics.Set("codegen_functions_generated", float64(len(generated.Functions)), labels...)
	cg.metrics.Set("codegen_constants_generated", float64(len(generated.Constants)), labels...)
	cg.metrics.Inc("codegen_runs_total", labels...)
}