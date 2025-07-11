package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/osakka/mcpeg/internal/mcp/types"
	"github.com/osakka/mcpeg/internal/registry"
	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
	"github.com/osakka/mcpeg/pkg/validation"
	"github.com/osakka/mcpeg/pkg/errors"
)

// MCPRouter handles routing of MCP requests to appropriate service adapters
type MCPRouter struct {
	registry   *registry.ServiceRegistry
	logger     logging.Logger
	metrics    metrics.Metrics
	validator  *validation.Validator
	config     RouterConfig
}

// RouterConfig configures the MCP router
type RouterConfig struct {
	// Request routing
	DefaultTimeout      time.Duration `yaml:"default_timeout"`
	MaxRequestSize      int64         `yaml:"max_request_size"`
	EnableMethodRouting bool          `yaml:"enable_method_routing"`
	
	// Load balancing
	LoadBalancingEnabled bool   `yaml:"load_balancing_enabled"`
	LoadBalancingStrategy string `yaml:"load_balancing_strategy"`
	
	// Validation
	ValidateRequests  bool `yaml:"validate_requests"`
	ValidateResponses bool `yaml:"validate_responses"`
	
	// Error handling
	RetryEnabled      bool          `yaml:"retry_enabled"`
	RetryAttempts     int           `yaml:"retry_attempts"`
	RetryBackoff      time.Duration `yaml:"retry_backoff"`
	
	// Monitoring
	EnableMetrics bool `yaml:"enable_metrics"`
	EnableTracing bool `yaml:"enable_tracing"`
}

// RequestContext provides context for request routing
type RequestContext struct {
	RequestID    string
	TraceID      string
	SpanID       string
	ClientID     string
	UserID       string
	SessionID    string
	StartTime    time.Time
	Method       string
	ServiceType  string
	Preferences  map[string]interface{}
}

// NewMCPRouter creates a new MCP router
func NewMCPRouter(
	registry *registry.ServiceRegistry,
	logger logging.Logger,
	metrics metrics.Metrics,
	validator *validation.Validator,
) *MCPRouter {
	return &MCPRouter{
		registry:  registry,
		logger:    logger.WithComponent("mcp_router"),
		metrics:   metrics,
		validator: validator,
		config:    defaultRouterConfig(),
	}
}

// SetupRoutes configures HTTP routes for the MCP router
func (mr *MCPRouter) SetupRoutes(router *mux.Router) {
	// MCP JSON-RPC endpoint
	router.HandleFunc("/mcp", mr.handleMCPRequest).Methods("POST")
	
	// MCP method-specific endpoints
	if mr.config.EnableMethodRouting {
		router.HandleFunc("/mcp/tools/list", mr.handleToolsList).Methods("POST")
		router.HandleFunc("/mcp/tools/call", mr.handleToolsCall).Methods("POST")
		router.HandleFunc("/mcp/resources/list", mr.handleResourcesList).Methods("POST")
		router.HandleFunc("/mcp/resources/read", mr.handleResourcesRead).Methods("POST")
		router.HandleFunc("/mcp/resources/subscribe", mr.handleResourcesSubscribe).Methods("POST")
		router.HandleFunc("/mcp/prompts/list", mr.handlePromptsList).Methods("POST")
		router.HandleFunc("/mcp/prompts/get", mr.handlePromptsGet).Methods("POST")
		router.HandleFunc("/mcp/completion/complete", mr.handleCompletionComplete).Methods("POST")
		router.HandleFunc("/mcp/logging/setLevel", mr.handleLoggingSetLevel).Methods("POST")
		router.HandleFunc("/mcp/sampling/createMessage", mr.handleSamplingCreateMessage).Methods("POST")
		router.HandleFunc("/mcp/roots/list", mr.handleRootsList).Methods("POST")
	}
}

// handleMCPRequest handles generic MCP JSON-RPC requests
func (mr *MCPRouter) handleMCPRequest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	// Create request context
	reqCtx := mr.createRequestContext(r)
	
	mr.logger.Info("mcp_request_started",
		"request_id", reqCtx.RequestID,
		"method", reqCtx.Method,
		"client_ip", r.RemoteAddr)
	
	// Parse JSON-RPC request
	var mcpReq types.Request
	if err := mr.parseRequest(r, &mcpReq); err != nil {
		mr.writeErrorResponse(w, reqCtx, types.ErrorCodeParseError, "Invalid JSON-RPC request", err)
		return
	}
	
	reqCtx.Method = mcpReq.Method
	
	// Validate request
	if mr.config.ValidateRequests {
		if err := mr.validateRequest(&mcpReq); err != nil {
			mr.writeErrorResponse(w, reqCtx, types.ErrorCodeInvalidParams, "Request validation failed", err)
			return
		}
	}
	
	// Route request to appropriate service
	result, err := mr.routeRequest(r.Context(), reqCtx, &mcpReq)
	if err != nil {
		mr.handleRoutingError(w, reqCtx, err)
		return
	}
	
	// Validate response
	if mr.config.ValidateResponses {
		if err := mr.validateResponse(result); err != nil {
			mr.logger.Warn("response_validation_failed",
				"request_id", reqCtx.RequestID,
				"error", err)
		}
	}
	
	// Write successful response
	response := types.Response{
		JSONRPC: "2.0",
		Result:  result,
		ID:      mcpReq.ID,
	}
	
	mr.writeJSONResponse(w, response)
	
	// Record metrics
	duration := time.Since(start)
	mr.recordRequestMetrics(reqCtx, duration, nil)
	
	mr.logger.Info("mcp_request_completed",
		"request_id", reqCtx.RequestID,
		"method", reqCtx.Method,
		"duration", duration,
		"success", true)
}

// routeRequest routes an MCP request to the appropriate service
func (mr *MCPRouter) routeRequest(ctx context.Context, reqCtx *RequestContext, mcpReq *types.Request) (interface{}, error) {
	// Determine target service type based on method
	serviceType := mr.determineServiceType(mcpReq.Method)
	if serviceType == "" {
		return nil, errors.ValidationError("mcp_router", "route_request",
			fmt.Sprintf("Unknown MCP method: %s", mcpReq.Method), map[string]interface{}{
				"method": mcpReq.Method,
				"request_id": reqCtx.RequestID,
			})
	}
	
	reqCtx.ServiceType = serviceType
	
	// Create selection criteria
	criteria := registry.SelectionCriteria{
		LoadBalancing: mr.config.LoadBalancingStrategy,
		Metadata:      reqCtx.Preferences,
	}
	
	// Select service instance
	service, err := mr.registry.SelectService(serviceType, criteria)
	if err != nil {
		return nil, errors.UnavailableError("mcp_router", "route_request", err, map[string]interface{}{
			"service_type": serviceType,
			"method": mcpReq.Method,
			"request_id": reqCtx.RequestID,
		})
	}
	
	mr.logger.Debug("service_selected_for_request",
		"request_id", reqCtx.RequestID,
		"service_id", service.ID,
		"service_type", serviceType,
		"method", mcpReq.Method)
	
	// Execute request with retry logic
	var result interface{}
	var lastErr error
	
	attempts := 1
	if mr.config.RetryEnabled {
		attempts = mr.config.RetryAttempts
	}
	
	for attempt := 1; attempt <= attempts; attempt++ {
		startTime := time.Now()
		
		result, lastErr = mr.executeRequest(ctx, service, mcpReq)
		
		duration := time.Since(startTime)
		
		if lastErr == nil {
			// Success - record metrics and return
			mr.registry.GetLoadBalancer().RecordSuccess(service, duration)
			return result, nil
		}
		
		// Record failure
		mr.registry.GetLoadBalancer().RecordFailure(service, lastErr)
		
		mr.logger.Warn("service_request_failed",
			"request_id", reqCtx.RequestID,
			"service_id", service.ID,
			"attempt", attempt,
			"max_attempts", attempts,
			"error", lastErr,
			"duration", duration)
		
		// If not the last attempt, wait before retrying
		if attempt < attempts {
			backoff := mr.config.RetryBackoff * time.Duration(attempt)
			time.Sleep(backoff)
			
			// Try to select a different service instance for retry
			if newService, err := mr.registry.SelectService(serviceType, criteria); err == nil {
				service = newService
				mr.logger.Debug("retrying_with_different_service",
					"request_id", reqCtx.RequestID,
					"new_service_id", service.ID,
					"attempt", attempt+1)
			}
		}
	}
	
	return nil, lastErr
}

// executeRequest executes an MCP request against a specific service
func (mr *MCPRouter) executeRequest(ctx context.Context, service *registry.RegisteredService, mcpReq *types.Request) (interface{}, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: mr.config.DefaultTimeout,
	}
	
	// Prepare request body
	reqBody, err := json.Marshal(mcpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", service.Endpoint, strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	
	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "MCPEG/1.0")
	
	// Execute request
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}
	
	// Parse response
	var mcpResp types.Response
	if err := json.NewDecoder(resp.Body).Decode(&mcpResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	// Check for JSON-RPC error
	if mcpResp.Error != nil {
		return nil, fmt.Errorf("MCP error %d: %s", mcpResp.Error.Code, mcpResp.Error.Message)
	}
	
	return mcpResp.Result, nil
}

// determineServiceType determines the appropriate service type for an MCP method
func (mr *MCPRouter) determineServiceType(method string) string {
	// Map MCP methods to service types
	methodServiceMap := map[string]string{
		"tools/list":           "tool_provider",
		"tools/call":           "tool_provider",
		"resources/list":       "resource_provider",
		"resources/read":       "resource_provider",
		"resources/subscribe":  "resource_provider",
		"prompts/list":         "prompt_provider",
		"prompts/get":          "prompt_provider",
		"completion/complete":  "completion_provider",
		"logging/setLevel":     "logging_provider",
		"sampling/createMessage": "sampling_provider",
		"roots/list":           "root_provider",
	}
	
	if serviceType, exists := methodServiceMap[method]; exists {
		return serviceType
	}
	
	// Try to extract service type from method prefix
	parts := strings.Split(method, "/")
	if len(parts) >= 2 {
		return parts[0] + "_provider"
	}
	
	// Default to generic adapter
	return "generic_adapter"
}

// Method-specific handlers (simplified implementations)

func (mr *MCPRouter) handleToolsList(w http.ResponseWriter, r *http.Request) {
	mr.handleMethodRequest(w, r, "tools/list")
}

func (mr *MCPRouter) handleToolsCall(w http.ResponseWriter, r *http.Request) {
	mr.handleMethodRequest(w, r, "tools/call")
}

func (mr *MCPRouter) handleResourcesList(w http.ResponseWriter, r *http.Request) {
	mr.handleMethodRequest(w, r, "resources/list")
}

func (mr *MCPRouter) handleResourcesRead(w http.ResponseWriter, r *http.Request) {
	mr.handleMethodRequest(w, r, "resources/read")
}

func (mr *MCPRouter) handleResourcesSubscribe(w http.ResponseWriter, r *http.Request) {
	mr.handleMethodRequest(w, r, "resources/subscribe")
}

func (mr *MCPRouter) handlePromptsList(w http.ResponseWriter, r *http.Request) {
	mr.handleMethodRequest(w, r, "prompts/list")
}

func (mr *MCPRouter) handlePromptsGet(w http.ResponseWriter, r *http.Request) {
	mr.handleMethodRequest(w, r, "prompts/get")
}

func (mr *MCPRouter) handleCompletionComplete(w http.ResponseWriter, r *http.Request) {
	mr.handleMethodRequest(w, r, "completion/complete")
}

func (mr *MCPRouter) handleLoggingSetLevel(w http.ResponseWriter, r *http.Request) {
	mr.handleMethodRequest(w, r, "logging/setLevel")
}

func (mr *MCPRouter) handleSamplingCreateMessage(w http.ResponseWriter, r *http.Request) {
	mr.handleMethodRequest(w, r, "sampling/createMessage")
}

func (mr *MCPRouter) handleRootsList(w http.ResponseWriter, r *http.Request) {
	mr.handleMethodRequest(w, r, "roots/list")
}

// handleMethodRequest handles method-specific requests
func (mr *MCPRouter) handleMethodRequest(w http.ResponseWriter, r *http.Request, method string) {
	// Parse request body as parameters
	var params json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		mr.writeErrorResponse(w, nil, types.ErrorCodeParseError, "Invalid request body", err)
		return
	}
	
	// Create MCP request and route through standard handler
	mr.handleMCPRequest(w, r)
}

// Helper methods

func (mr *MCPRouter) createRequestContext(r *http.Request) *RequestContext {
	return &RequestContext{
		RequestID:   generateRequestID(),
		TraceID:     r.Header.Get("X-Trace-ID"),
		SpanID:      r.Header.Get("X-Span-ID"),
		ClientID:    r.Header.Get("X-Client-ID"),
		UserID:      r.Header.Get("X-User-ID"),
		SessionID:   r.Header.Get("X-Session-ID"),
		StartTime:   time.Now(),
		Preferences: make(map[string]interface{}),
	}
}

func (mr *MCPRouter) parseRequest(r *http.Request, mcpReq *types.Request) error {
	if r.ContentLength > mr.config.MaxRequestSize {
		return fmt.Errorf("request too large: %d bytes", r.ContentLength)
	}
	
	return json.NewDecoder(r.Body).Decode(mcpReq)
}

func (mr *MCPRouter) validateRequest(mcpReq *types.Request) error {
	if mcpReq.JSONRPC != "2.0" {
		return fmt.Errorf("invalid JSON-RPC version: %s", mcpReq.JSONRPC)
	}
	
	if mcpReq.Method == "" {
		return fmt.Errorf("missing method")
	}
	
	return nil
}

func (mr *MCPRouter) validateResponse(result interface{}) error {
	mr.logger.Debug("mcp_response_validation_started", "result_type", fmt.Sprintf("%T", result))
	
	if result == nil {
		return fmt.Errorf("response result cannot be nil")
	}

	// Validate based on result type - comprehensive MCP schema validation
	switch v := result.(type) {
	case *types.InitializeResult:
		return mr.validateInitializeResult(v)
	case *types.ListResourcesResult:
		return mr.validateListResourcesResult(v)
	case *types.ReadResourceResult:
		return mr.validateReadResourceResult(v)
	case *types.SubscribeResult:
		return mr.validateSubscribeResult(v)
	case *types.UnsubscribeResult:
		return mr.validateUnsubscribeResult(v)
	case *types.ListPromptsResult:
		return mr.validateListPromptsResult(v)
	case *types.GetPromptResult:
		return mr.validateGetPromptResult(v)
	case *types.ListToolsResult:
		return mr.validateListToolsResult(v)
	case *types.CallToolResult:
		return mr.validateCallToolResult(v)
	case *types.LoggingLevelResult:
		return mr.validateLoggingLevelResult(v)
	case *types.CompleteResult:
		return mr.validateCompleteResult(v)
	case map[string]interface{}:
		// Generic validation for unknown response types
		return mr.validateGenericResponse(v)
	default:
		mr.logger.Warn("mcp_response_validation_unknown_type", 
			"type", fmt.Sprintf("%T", result))
		// Allow unknown types but log warning
		return nil
	}
}

// validateInitializeResult validates initialize response
func (mr *MCPRouter) validateInitializeResult(result *types.InitializeResult) error {
	if result == nil {
		return fmt.Errorf("initialize result cannot be nil")
	}

	// Validate protocol version
	if result.ProtocolVersion == "" {
		return fmt.Errorf("protocol version is required")
	}

	// Validate version format (semantic versioning)
	if !isValidSemanticVersion(result.ProtocolVersion) {
		return fmt.Errorf("invalid protocol version format: %s", result.ProtocolVersion)
	}

	// Validate capabilities structure
	if err := mr.validateCapabilities(&result.Capabilities); err != nil {
		return fmt.Errorf("invalid capabilities: %w", err)
	}

	// Validate server info if present
	if err := mr.validateServerInfo(&result.ServerInfo); err != nil {
		return fmt.Errorf("invalid server info: %w", err)
	}

	mr.logger.Debug("mcp_initialize_result_validated",
		"protocol_version", result.ProtocolVersion,
		"server_name", getServerName(&result.ServerInfo))

	return nil
}

// validateListResourcesResult validates list resources response
func (mr *MCPRouter) validateListResourcesResult(result *types.ListResourcesResult) error {
	if result == nil {
		return fmt.Errorf("list resources result cannot be nil")
	}

	if result.Resources == nil {
		return fmt.Errorf("resources array cannot be nil")
	}

	// Validate each resource
	for i, resource := range result.Resources {
		if err := mr.validateResource(&resource, fmt.Sprintf("resource[%d]", i)); err != nil {
			return err
		}
	}

	// Validate pagination if present
	if result.NextCursor == "" {
		// Empty string is valid - means no more pages
	}

	mr.logger.Debug("mcp_list_resources_result_validated",
		"resource_count", len(result.Resources),
		"has_next_cursor", result.NextCursor != "")

	return nil
}

// validateReadResourceResult validates read resource response
func (mr *MCPRouter) validateReadResourceResult(result *types.ReadResourceResult) error {
	if result == nil {
		return fmt.Errorf("read resource result cannot be nil")
	}

	if result.Contents == nil {
		return fmt.Errorf("contents array cannot be nil")
	}

	// Validate each content item
	for i, content := range result.Contents {
		if err := mr.validateResourceContent(&content, fmt.Sprintf("content[%d]", i)); err != nil {
			return err
		}
	}

	mr.logger.Debug("mcp_read_resource_result_validated",
		"content_count", len(result.Contents))

	return nil
}

// validateListToolsResult validates list tools response
func (mr *MCPRouter) validateListToolsResult(result *types.ListToolsResult) error {
	if result == nil {
		return fmt.Errorf("list tools result cannot be nil")
	}

	if result.Tools == nil {
		return fmt.Errorf("tools array cannot be nil")
	}

	// Validate each tool
	for i, tool := range result.Tools {
		if err := mr.validateTool(&tool, fmt.Sprintf("tool[%d]", i)); err != nil {
			return err
		}
	}

	// Validate pagination
	if result.NextCursor == "" {
		// Empty string is valid - means no more pages
	}

	mr.logger.Debug("mcp_list_tools_result_validated",
		"tool_count", len(result.Tools),
		"has_next_cursor", result.NextCursor != "")

	return nil
}

// validateCallToolResult validates call tool response
func (mr *MCPRouter) validateCallToolResult(result *types.CallToolResult) error {
	if result == nil {
		return fmt.Errorf("call tool result cannot be nil")
	}

	if result.Content == nil {
		return fmt.Errorf("content array cannot be nil")
	}

	// Validate each content item
	for i, content := range result.Content {
		// Convert ToolContent to Content for validation
		genericContent := &types.Content{
			Type: content.Type,
			Text: content.Text,
			Data: content.Data,
		}
		if err := mr.validateToolContent(genericContent, fmt.Sprintf("content[%d]", i)); err != nil {
			return err
		}
	}

	// Validate that we have at least one content item
	if len(result.Content) == 0 {
		return fmt.Errorf("tool result must contain at least one content item")
	}

	mr.logger.Debug("mcp_call_tool_result_validated",
		"content_count", len(result.Content),
		"is_error", result.IsError)

	return nil
}

// validateListPromptsResult validates list prompts response
func (mr *MCPRouter) validateListPromptsResult(result *types.ListPromptsResult) error {
	if result == nil {
		return fmt.Errorf("list prompts result cannot be nil")
	}

	if result.Prompts == nil {
		return fmt.Errorf("prompts array cannot be nil")
	}

	// Validate each prompt
	for i, prompt := range result.Prompts {
		if err := mr.validatePrompt(&prompt, fmt.Sprintf("prompt[%d]", i)); err != nil {
			return err
		}
	}

	mr.logger.Debug("mcp_list_prompts_result_validated",
		"prompt_count", len(result.Prompts))

	return nil
}

// validateGetPromptResult validates get prompt response
func (mr *MCPRouter) validateGetPromptResult(result *types.GetPromptResult) error {
	if result == nil {
		return fmt.Errorf("get prompt result cannot be nil")
	}

	if result.Messages == nil {
		return fmt.Errorf("messages array cannot be nil")
	}

	if len(result.Messages) == 0 {
		return fmt.Errorf("prompt must contain at least one message")
	}

	// Validate each message
	for i, message := range result.Messages {
		if err := mr.validatePromptMessage(&message, fmt.Sprintf("message[%d]", i)); err != nil {
			return err
		}
	}

	mr.logger.Debug("mcp_get_prompt_result_validated",
		"message_count", len(result.Messages))

	return nil
}

// validateCompleteResult validates completion response
func (mr *MCPRouter) validateCompleteResult(result *types.CompleteResult) error {
	if result == nil {
		return fmt.Errorf("complete result cannot be nil")
	}

	// Validate completion values
	if result.Completion.Values == nil {
		return fmt.Errorf("completion values cannot be nil")
	}

	mr.logger.Debug("mcp_complete_result_validated",
		"has_completion", true,
		"model", result.Completion.Model)

	return nil
}

// Helper validation functions for sub-objects

func (mr *MCPRouter) validateCapabilities(capabilities *types.ServerCapabilities) error {
	// Validate logging capability
	if capabilities.Logging != nil && capabilities.Logging.Level == "" {
		return fmt.Errorf("logging level cannot be empty if logging capability is specified")
	}

	// Validate prompts capability
	if capabilities.Prompts != nil {
		if capabilities.Prompts.ListChanged {
			// If list_changed is true, validate it's properly supported
			mr.logger.Debug("prompts_list_changed_capability_enabled")
		}
	}

	// Validate resources capability
	if capabilities.Resources != nil {
		if capabilities.Resources.Subscribe {
			mr.logger.Debug("resource_subscription_capability_enabled")
		}
		if capabilities.Resources.ListChanged {
			mr.logger.Debug("resource_list_changed_capability_enabled")
		}
	}

	// Validate tools capability
	if capabilities.Tools != nil {
		if capabilities.Tools.ListChanged {
			mr.logger.Debug("tools_list_changed_capability_enabled")
		}
	}

	return nil
}

func (mr *MCPRouter) validateServerInfo(serverInfo *types.ServerInfo) error {
	if serverInfo.Name == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	if serverInfo.Version == "" {
		return fmt.Errorf("server version cannot be empty")
	}

	return nil
}

func (mr *MCPRouter) validateResource(resource *types.Resource, context string) error {
	if resource.URI == "" {
		return fmt.Errorf("%s: resource URI cannot be empty", context)
	}

	if !isValidURI(resource.URI) {
		return fmt.Errorf("%s: invalid resource URI format: %s", context, resource.URI)
	}

	if resource.Name == "" {
		return fmt.Errorf("%s: resource name cannot be empty", context)
	}

	return nil
}

func (mr *MCPRouter) validateResourceContent(content *types.ResourceContent, context string) error {
	if content.URI == "" {
		return fmt.Errorf("%s: content URI cannot be empty", context)
	}

	if !isValidURI(content.URI) {
		return fmt.Errorf("%s: invalid content URI format: %s", context, content.URI)
	}

	// Validate that content has either text or blob data
	hasText := content.Text != ""
	hasBlob := content.Blob != ""
	
	if !hasText && !hasBlob {
		return fmt.Errorf("%s: content must have either text or blob data", context)
	}

	if hasText && hasBlob {
		return fmt.Errorf("%s: content cannot have both text and blob data", context)
	}

	return nil
}

func (mr *MCPRouter) validateTool(tool *types.Tool, context string) error {
	if tool.Name == "" {
		return fmt.Errorf("%s: tool name cannot be empty", context)
	}

	if tool.Description == "" {
		return fmt.Errorf("%s: tool description cannot be empty", context)
	}

	// Validate tool name format (alphanumeric, underscores, hyphens)
	if !isValidToolName(tool.Name) {
		return fmt.Errorf("%s: invalid tool name format: %s", context, tool.Name)
	}

	// Validate input schema if present
	if tool.InputSchema != nil {
		if err := mr.validateJSONSchema(tool.InputSchema, context+".input_schema"); err != nil {
			return err
		}
	}

	return nil
}

func (mr *MCPRouter) validateToolContent(content *types.Content, context string) error {
	if content.Type == "" {
		return fmt.Errorf("%s: content type cannot be empty", context)
	}

	// Validate content type
	validTypes := []string{"text", "image", "resource"}
	if !contains(validTypes, content.Type) {
		return fmt.Errorf("%s: invalid content type: %s, must be one of %v", context, content.Type, validTypes)
	}

	// Type-specific validation
	switch content.Type {
	case "text":
		if content.Text == "" {
			return fmt.Errorf("%s: text content cannot be empty", context)
		}
	case "image":
		if content.Data == nil {
			return fmt.Errorf("%s: image content data cannot be empty", context)
		}
	case "resource":
		if content.Data == nil {
			return fmt.Errorf("%s: resource content must have data", context)
		}
	}

	return nil
}

func (mr *MCPRouter) validatePrompt(prompt *types.Prompt, context string) error {
	if prompt.Name == "" {
		return fmt.Errorf("%s: prompt name cannot be empty", context)
	}

	if prompt.Description == "" {
		return fmt.Errorf("%s: prompt description cannot be empty", context)
	}

	// Validate prompt name format
	if !isValidPromptName(prompt.Name) {
		return fmt.Errorf("%s: invalid prompt name format: %s", context, prompt.Name)
	}

	return nil
}

func (mr *MCPRouter) validatePromptMessage(message *types.PromptMessage, context string) error {
	if message.Role == "" {
		return fmt.Errorf("%s: message role cannot be empty", context)
	}

	// Validate role
	validRoles := []string{"user", "assistant", "system"}
	if !contains(validRoles, message.Role) {
		return fmt.Errorf("%s: invalid message role: %s, must be one of %v", context, message.Role, validRoles)
	}

	// Validate content fields
	if message.Content.Type == "" {
		return fmt.Errorf("%s: message content type cannot be empty", context)
	}
	
	if message.Content.Text == "" {
		return fmt.Errorf("%s: message content text cannot be empty", context)
	}

	return nil
}

func (mr *MCPRouter) validatePromptContentItem(item interface{}, context string) error {
	contentMap, ok := item.(map[string]interface{})
	if !ok {
		return fmt.Errorf("%s: content item must be an object", context)
	}

	contentType, ok := contentMap["type"].(string)
	if !ok || contentType == "" {
		return fmt.Errorf("%s: content item must have type", context)
	}

	validTypes := []string{"text", "image", "resource"}
	if !contains(validTypes, contentType) {
		return fmt.Errorf("%s: invalid content type: %s", context, contentType)
	}

	return nil
}

func (mr *MCPRouter) validateJSONSchema(schema interface{}, context string) error {
	schemaMap, ok := schema.(map[string]interface{})
	if !ok {
		return fmt.Errorf("%s: schema must be an object", context)
	}

	// Basic JSON Schema validation
	if schemaType, exists := schemaMap["type"]; exists {
		if typeStr, ok := schemaType.(string); ok {
			validTypes := []string{"object", "array", "string", "number", "integer", "boolean", "null"}
			if !contains(validTypes, typeStr) {
				return fmt.Errorf("%s: invalid schema type: %s", context, typeStr)
			}
		}
	}

	return nil
}

// validateGenericResponse validates unknown response types
func (mr *MCPRouter) validateGenericResponse(result map[string]interface{}) error {
	// Basic validation for generic responses
	if len(result) == 0 {
		return fmt.Errorf("response result cannot be empty")
	}

	// Log unknown response structure for debugging
	mr.logger.Debug("mcp_generic_response_validated", "keys", getMapKeys(result))

	return nil
}

// Validation helper functions
func (mr *MCPRouter) validateSubscribeResult(result *types.SubscribeResult) error {
	// Subscribe result can be empty
	mr.logger.Debug("mcp_subscribe_result_validated")
	return nil
}

func (mr *MCPRouter) validateUnsubscribeResult(result *types.UnsubscribeResult) error {
	// Unsubscribe result can be empty
	mr.logger.Debug("mcp_unsubscribe_result_validated")
	return nil
}

func (mr *MCPRouter) validateLoggingLevelResult(result *types.LoggingLevelResult) error {
	// Logging level result can be empty
	mr.logger.Debug("mcp_logging_level_result_validated")
	return nil
}

// Utility functions for validation
func isValidSemanticVersion(version string) bool {
	// Simple semantic version validation (major.minor.patch)
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false
	}
	
	for _, part := range parts {
		if part == "" {
			return false
		}
		// Check if part is numeric
		for _, char := range part {
			if char < '0' || char > '9' {
				return false
			}
		}
	}
	
	return true
}

func isValidURI(uri string) bool {
	// Basic URI validation - non-empty and contains scheme
	return uri != "" && strings.Contains(uri, ":")
}

func isValidToolName(name string) bool {
	// Tool names should be alphanumeric with underscores and hyphens
	if name == "" {
		return false
	}
	
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') || 
			 (char >= 'A' && char <= 'Z') || 
			 (char >= '0' && char <= '9') || 
			 char == '_' || char == '-') {
			return false
		}
	}
	
	return true
}

func isValidPromptName(name string) bool {
	// Same validation as tool names
	return isValidToolName(name)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func getServerName(serverInfo *types.ServerInfo) string {
	if serverInfo != nil {
		return serverInfo.Name
	}
	return "unknown"
}

func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func (mr *MCPRouter) writeErrorResponse(w http.ResponseWriter, reqCtx *RequestContext, code int, message string, err error) {
	errorResp := types.Response{
		JSONRPC: "2.0",
		Error: &types.Error{
			Code:    code,
			Message: message,
			Data:    err.Error(),
		},
		ID: nil,
	}
	
	if reqCtx != nil {
		mr.recordRequestMetrics(reqCtx, time.Since(reqCtx.StartTime), err)
		mr.logger.Error("mcp_request_failed",
			"request_id", reqCtx.RequestID,
			"error_code", code,
			"error_message", message,
			"error", err)
	}
	
	mr.writeJSONResponse(w, errorResp)
}

func (mr *MCPRouter) writeJSONResponse(w http.ResponseWriter, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (mr *MCPRouter) handleRoutingError(w http.ResponseWriter, reqCtx *RequestContext, err error) {
	// Determine appropriate error code based on error type
	var code int
	var message string
	
	switch {
	case errors.IsValidationError(err):
		code = types.ErrorCodeInvalidParams
		message = "Invalid parameters"
	case errors.IsUnavailableError(err):
		code = types.ErrorCodeInternalError
		message = "Service unavailable"
	case errors.IsTimeoutError(err):
		code = types.ErrorCodeInternalError
		message = "Request timeout"
	default:
		code = types.ErrorCodeInternalError
		message = "Internal error"
	}
	
	mr.writeErrorResponse(w, reqCtx, code, message, err)
}

func (mr *MCPRouter) recordRequestMetrics(reqCtx *RequestContext, duration time.Duration, err error) {
	if !mr.config.EnableMetrics {
		return
	}
	
	labels := []string{
		"method", reqCtx.Method,
		"service_type", reqCtx.ServiceType,
	}
	
	if err != nil {
		labels = append(labels, "status", "error")
		mr.metrics.Inc("mcp_requests_total", labels...)
		mr.metrics.Inc("mcp_requests_failed_total", labels...)
	} else {
		labels = append(labels, "status", "success")
		mr.metrics.Inc("mcp_requests_total", labels...)
	}
	
	mr.metrics.Observe("mcp_request_duration_seconds", duration.Seconds(), labels...)
}

func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

func defaultRouterConfig() RouterConfig {
	return RouterConfig{
		DefaultTimeout:        30 * time.Second,
		MaxRequestSize:        10 * 1024 * 1024, // 10MB
		EnableMethodRouting:   true,
		LoadBalancingEnabled:  true,
		LoadBalancingStrategy: "round_robin",
		ValidateRequests:      true,
		ValidateResponses:     false,
		RetryEnabled:          true,
		RetryAttempts:         3,
		RetryBackoff:          1 * time.Second,
		EnableMetrics:         true,
		EnableTracing:         true,
	}
}