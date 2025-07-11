package context

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
)

// ContextKey type for context keys to avoid collisions
type ContextKey string

const (
	// Request context keys
	RequestIDKey     ContextKey = "request_id"
	CorrelationIDKey ContextKey = "correlation_id"
	TraceIDKey       ContextKey = "trace_id"
	SpanIDKey        ContextKey = "span_id"
	UserIDKey        ContextKey = "user_id"
	SessionIDKey     ContextKey = "session_id"
	ClientInfoKey    ContextKey = "client_info"
	
	// Service context keys
	ServiceNameKey    ContextKey = "service_name"
	ServiceVersionKey ContextKey = "service_version"
	OperationKey      ContextKey = "operation"
	ComponentKey      ContextKey = "component"
	
	// Performance context keys
	StartTimeKey     ContextKey = "start_time"
	DeadlineKey      ContextKey = "deadline"
	TimeoutKey       ContextKey = "timeout"
	
	// Security context keys
	AuthMethodKey    ContextKey = "auth_method"
	PermissionsKey   ContextKey = "permissions"
	SecurityLevelKey ContextKey = "security_level"
	
	// Feature flags
	FeatureFlagsKey ContextKey = "feature_flags"
	
	// LLM debugging context
	LLMContextKey ContextKey = "llm_debug_context"
)

// RequestContext contains distributed request context information
type RequestContext struct {
	RequestID     string                 `json:"request_id"`
	CorrelationID string                 `json:"correlation_id"`
	TraceID       string                 `json:"trace_id"`
	SpanID        string                 `json:"span_id"`
	UserID        string                 `json:"user_id,omitempty"`
	SessionID     string                 `json:"session_id,omitempty"`
	ClientInfo    *ClientInfo            `json:"client_info,omitempty"`
	StartTime     time.Time              `json:"start_time"`
	Timeout       time.Duration          `json:"timeout,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ClientInfo contains information about the requesting client
type ClientInfo struct {
	Name      string            `json:"name"`
	Version   string            `json:"version"`
	UserAgent string            `json:"user_agent,omitempty"`
	IPAddress string            `json:"ip_address,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
}

// ServiceContext contains service-specific context information
type ServiceContext struct {
	ServiceName    string                 `json:"service_name"`
	ServiceVersion string                 `json:"service_version"`
	Operation      string                 `json:"operation"`
	Component      string                 `json:"component"`
	InstanceID     string                 `json:"instance_id"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// SecurityContext contains security-related context information
type SecurityContext struct {
	AuthMethod    string   `json:"auth_method,omitempty"`
	Permissions   []string `json:"permissions,omitempty"`
	SecurityLevel string   `json:"security_level,omitempty"`
	Authenticated bool     `json:"authenticated"`
	UserRoles     []string `json:"user_roles,omitempty"`
}

// LLMDebugContext contains information optimized for LLM debugging
type LLMDebugContext struct {
	OperationPath    []string               `json:"operation_path"`
	CallStack        []CallFrame            `json:"call_stack"`
	ErrorChain       []string               `json:"error_chain,omitempty"`
	PerformanceData  map[string]interface{} `json:"performance_data"`
	ResourceUsage    map[string]interface{} `json:"resource_usage"`
	Suggestions      []string               `json:"suggestions,omitempty"`
	TroubleshootingInfo map[string]interface{} `json:"troubleshooting_info"`
}

// CallFrame represents a stack frame for debugging
type CallFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Package  string `json:"package"`
}

// ContextManager manages distributed context propagation
type ContextManager struct {
	logger  logging.Logger
	metrics metrics.Metrics
	config  ContextConfig
}

// ContextConfig configures context management behavior
type ContextConfig struct {
	// Propagation settings
	PropagateHeaders     []string      `yaml:"propagate_headers"`
	DefaultTimeout       time.Duration `yaml:"default_timeout"`
	MaxContextSize       int           `yaml:"max_context_size"`
	
	// Debugging settings
	IncludeCallStack     bool `yaml:"include_call_stack"`
	IncludeResourceUsage bool `yaml:"include_resource_usage"`
	MaxCallStackDepth    int  `yaml:"max_call_stack_depth"`
	
	// Performance settings
	EnableMetrics        bool `yaml:"enable_metrics"`
	SampleRate          float64 `yaml:"sample_rate"`
}

// NewContextManager creates a new context manager
func NewContextManager(logger logging.Logger, metrics metrics.Metrics) *ContextManager {
	return &ContextManager{
		logger:  logger.WithComponent("context_manager"),
		metrics: metrics,
		config:  defaultContextConfig(),
	}
}

// WithRequestContext adds request context to the context
func (cm *ContextManager) WithRequestContext(ctx context.Context, reqCtx *RequestContext) context.Context {
	// Add individual fields for easy access
	ctx = context.WithValue(ctx, RequestIDKey, reqCtx.RequestID)
	ctx = context.WithValue(ctx, CorrelationIDKey, reqCtx.CorrelationID)
	ctx = context.WithValue(ctx, TraceIDKey, reqCtx.TraceID)
	ctx = context.WithValue(ctx, SpanIDKey, reqCtx.SpanID)
	ctx = context.WithValue(ctx, StartTimeKey, reqCtx.StartTime)
	
	if reqCtx.UserID != "" {
		ctx = context.WithValue(ctx, UserIDKey, reqCtx.UserID)
	}
	
	if reqCtx.SessionID != "" {
		ctx = context.WithValue(ctx, SessionIDKey, reqCtx.SessionID)
	}
	
	if reqCtx.ClientInfo != nil {
		ctx = context.WithValue(ctx, ClientInfoKey, reqCtx.ClientInfo)
	}
	
	// Set timeout if specified
	if reqCtx.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, reqCtx.Timeout)
		// Store cancel function in context for cleanup
		ctx = context.WithValue(ctx, "cancel_func", cancel)
	}
	
	// Add LLM debug context
	if cm.config.IncludeCallStack {
		llmCtx := cm.buildLLMContext(ctx, reqCtx.Operation)
		ctx = context.WithValue(ctx, LLMContextKey, llmCtx)
	}
	
	cm.recordContextMetrics(ctx, "request_context_added")
	
	return ctx
}

// WithServiceContext adds service context to the context
func (cm *ContextManager) WithServiceContext(ctx context.Context, svcCtx *ServiceContext) context.Context {
	ctx = context.WithValue(ctx, ServiceNameKey, svcCtx.ServiceName)
	ctx = context.WithValue(ctx, ServiceVersionKey, svcCtx.ServiceVersion)
	ctx = context.WithValue(ctx, OperationKey, svcCtx.Operation)
	ctx = context.WithValue(ctx, ComponentKey, svcCtx.Component)
	
	// Update LLM context with service information
	if llmCtx, ok := ctx.Value(LLMContextKey).(*LLMDebugContext); ok {
		llmCtx.OperationPath = append(llmCtx.OperationPath, svcCtx.Operation)
		llmCtx.TroubleshootingInfo["service"] = map[string]interface{}{
			"name":      svcCtx.ServiceName,
			"version":   svcCtx.ServiceVersion,
			"component": svcCtx.Component,
			"instance":  svcCtx.InstanceID,
		}
		ctx = context.WithValue(ctx, LLMContextKey, llmCtx)
	}
	
	cm.recordContextMetrics(ctx, "service_context_added")
	
	return ctx
}

// WithSecurityContext adds security context to the context
func (cm *ContextManager) WithSecurityContext(ctx context.Context, secCtx *SecurityContext) context.Context {
	ctx = context.WithValue(ctx, AuthMethodKey, secCtx.AuthMethod)
	ctx = context.WithValue(ctx, PermissionsKey, secCtx.Permissions)
	ctx = context.WithValue(ctx, SecurityLevelKey, secCtx.SecurityLevel)
	
	// Update LLM context with security information
	if llmCtx, ok := ctx.Value(LLMContextKey).(*LLMDebugContext); ok {
		llmCtx.TroubleshootingInfo["security"] = map[string]interface{}{
			"authenticated":   secCtx.Authenticated,
			"auth_method":     secCtx.AuthMethod,
			"security_level":  secCtx.SecurityLevel,
			"permissions":     secCtx.Permissions,
			"user_roles":      secCtx.UserRoles,
		}
		ctx = context.WithValue(ctx, LLMContextKey, llmCtx)
	}
	
	cm.recordContextMetrics(ctx, "security_context_added")
	
	return ctx
}

// WithOperation adds operation context for tracking
func (cm *ContextManager) WithOperation(ctx context.Context, operation string) context.Context {
	ctx = context.WithValue(ctx, OperationKey, operation)
	
	// Update LLM context operation path
	if llmCtx, ok := ctx.Value(LLMContextKey).(*LLMDebugContext); ok {
		llmCtx.OperationPath = append(llmCtx.OperationPath, operation)
		ctx = context.WithValue(ctx, LLMContextKey, llmCtx)
	}
	
	return ctx
}

// WithTimeout adds a timeout to the context
func (cm *ContextManager) WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx = context.WithValue(ctx, TimeoutKey, timeout)
	return context.WithTimeout(ctx, timeout)
}

// GetRequestContext retrieves request context from context
func (cm *ContextManager) GetRequestContext(ctx context.Context) *RequestContext {
	reqCtx := &RequestContext{}
	
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		reqCtx.RequestID = requestID
	}
	
	if correlationID, ok := ctx.Value(CorrelationIDKey).(string); ok {
		reqCtx.CorrelationID = correlationID
	}
	
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		reqCtx.TraceID = traceID
	}
	
	if spanID, ok := ctx.Value(SpanIDKey).(string); ok {
		reqCtx.SpanID = spanID
	}
	
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		reqCtx.UserID = userID
	}
	
	if sessionID, ok := ctx.Value(SessionIDKey).(string); ok {
		reqCtx.SessionID = sessionID
	}
	
	if clientInfo, ok := ctx.Value(ClientInfoKey).(*ClientInfo); ok {
		reqCtx.ClientInfo = clientInfo
	}
	
	if startTime, ok := ctx.Value(StartTimeKey).(time.Time); ok {
		reqCtx.StartTime = startTime
	}
	
	if timeout, ok := ctx.Value(TimeoutKey).(time.Duration); ok {
		reqCtx.Timeout = timeout
	}
	
	return reqCtx
}

// GetServiceContext retrieves service context from context
func (cm *ContextManager) GetServiceContext(ctx context.Context) *ServiceContext {
	svcCtx := &ServiceContext{}
	
	if serviceName, ok := ctx.Value(ServiceNameKey).(string); ok {
		svcCtx.ServiceName = serviceName
	}
	
	if serviceVersion, ok := ctx.Value(ServiceVersionKey).(string); ok {
		svcCtx.ServiceVersion = serviceVersion
	}
	
	if operation, ok := ctx.Value(OperationKey).(string); ok {
		svcCtx.Operation = operation
	}
	
	if component, ok := ctx.Value(ComponentKey).(string); ok {
		svcCtx.Component = component
	}
	
	return svcCtx
}

// GetLLMContext retrieves LLM debug context from context
func (cm *ContextManager) GetLLMContext(ctx context.Context) *LLMDebugContext {
	if llmCtx, ok := ctx.Value(LLMContextKey).(*LLMDebugContext); ok {
		return llmCtx
	}
	return nil
}

// PropagateContext creates a new context with propagated information
func (cm *ContextManager) PropagateContext(parent context.Context, operation string) context.Context {
	// Create new context with timeout from parent
	ctx := context.Background()
	
	// Propagate request context
	if reqCtx := cm.GetRequestContext(parent); reqCtx.RequestID != "" {
		ctx = cm.WithRequestContext(ctx, reqCtx)
	}
	
	// Propagate service context
	if svcCtx := cm.GetServiceContext(parent); svcCtx.ServiceName != "" {
		ctx = cm.WithServiceContext(ctx, svcCtx)
	}
	
	// Add new operation
	if operation != "" {
		ctx = cm.WithOperation(ctx, operation)
	}
	
	// Copy deadline from parent
	if deadline, ok := parent.Deadline(); ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, deadline)
		ctx = context.WithValue(ctx, "cancel_func", cancel)
	}
	
	return ctx
}

// buildLLMContext builds comprehensive context for LLM debugging
func (cm *ContextManager) buildLLMContext(ctx context.Context, operation string) *LLMDebugContext {
	llmCtx := &LLMDebugContext{
		OperationPath:       []string{operation},
		CallStack:           cm.captureCallStack(),
		PerformanceData:     make(map[string]interface{}),
		ResourceUsage:       make(map[string]interface{}),
		TroubleshootingInfo: make(map[string]interface{}),
	}
	
	// Add performance data
	llmCtx.PerformanceData["start_time"] = time.Now()
	if deadline, ok := ctx.Deadline(); ok {
		llmCtx.PerformanceData["deadline"] = deadline
		llmCtx.PerformanceData["timeout"] = time.Until(deadline)
	}
	
	// Add resource usage if enabled
	if cm.config.IncludeResourceUsage {
		llmCtx.ResourceUsage = cm.captureResourceUsage()
	}
	
	// Add troubleshooting context
	llmCtx.TroubleshootingInfo["go_version"] = runtime.Version()
	llmCtx.TroubleshootingInfo["num_goroutines"] = runtime.NumGoroutine()
	llmCtx.TroubleshootingInfo["num_cpus"] = runtime.NumCPU()
	
	return llmCtx
}

// captureCallStack captures the current call stack for debugging
func (cm *ContextManager) captureCallStack() []CallFrame {
	if !cm.config.IncludeCallStack {
		return nil
	}
	
	frames := make([]CallFrame, 0, cm.config.MaxCallStackDepth)
	
	// Start from caller of this function (skip 2: this function and buildLLMContext)
	for i := 2; i < cm.config.MaxCallStackDepth+2; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		
		// Extract package name
		parts := strings.Split(fn.Name(), "/")
		pkg := ""
		if len(parts) > 1 {
			pkg = parts[len(parts)-2]
		}
		
		frames = append(frames, CallFrame{
			Function: fn.Name(),
			File:     file,
			Line:     line,
			Package:  pkg,
		})
	}
	
	return frames
}

// captureResourceUsage captures current resource usage
func (cm *ContextManager) captureResourceUsage() map[string]interface{} {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	return map[string]interface{}{
		"memory": map[string]interface{}{
			"alloc_mb":      float64(memStats.Alloc) / 1024 / 1024,
			"sys_mb":        float64(memStats.Sys) / 1024 / 1024,
			"heap_objects":  memStats.HeapObjects,
			"gc_cycles":     memStats.NumGC,
		},
		"goroutines": runtime.NumGoroutine(),
		"timestamp":  time.Now(),
	}
}

// recordContextMetrics records context-related metrics
func (cm *ContextManager) recordContextMetrics(ctx context.Context, event string) {
	if !cm.config.EnableMetrics {
		return
	}
	
	labels := []string{
		"event", event,
	}
	
	if reqCtx := cm.GetRequestContext(ctx); reqCtx.RequestID != "" {
		labels = append(labels, "has_request_id", "true")
	} else {
		labels = append(labels, "has_request_id", "false")
	}
	
	if svcCtx := cm.GetServiceContext(ctx); svcCtx.ServiceName != "" {
		labels = append(labels, "service", svcCtx.ServiceName)
	}
	
	cm.metrics.Inc("context_operations_total", labels...)
}

// Helper functions for common context operations
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

func GetCorrelationID(ctx context.Context) string {
	if correlationID, ok := ctx.Value(CorrelationIDKey).(string); ok {
		return correlationID
	}
	return ""
}

func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

func GetOperation(ctx context.Context) string {
	if operation, ok := ctx.Value(OperationKey).(string); ok {
		return operation
	}
	return ""
}

func GetServiceName(ctx context.Context) string {
	if serviceName, ok := ctx.Value(ServiceNameKey).(string); ok {
		return serviceName
	}
	return ""
}

func defaultContextConfig() ContextConfig {
	return ContextConfig{
		PropagateHeaders: []string{
			"X-Request-ID",
			"X-Correlation-ID", 
			"X-Trace-ID",
			"X-User-ID",
		},
		DefaultTimeout:       30 * time.Second,
		MaxContextSize:       1024 * 1024, // 1MB
		IncludeCallStack:     true,
		IncludeResourceUsage: true,
		MaxCallStackDepth:    10,
		EnableMetrics:        true,
		SampleRate:          1.0,
	}
}

import "strings"