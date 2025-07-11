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
	// TODO: Implement response validation based on MCP schema
	return nil
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