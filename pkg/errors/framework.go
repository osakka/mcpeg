package errors

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
)

// ErrorCategory classifies errors for better handling
type ErrorCategory string

const (
	CategoryValidation     ErrorCategory = "validation"
	CategoryAuthentication ErrorCategory = "authentication"
	CategoryAuthorization  ErrorCategory = "authorization"
	CategoryRateLimit      ErrorCategory = "rate_limit"
	CategoryTimeout        ErrorCategory = "timeout"
	CategoryUnavailable    ErrorCategory = "unavailable"
	CategoryInternal       ErrorCategory = "internal"
	CategoryNetwork        ErrorCategory = "network"
	CategoryConfiguration  ErrorCategory = "configuration"
	CategoryResource       ErrorCategory = "resource"
	CategoryBusiness       ErrorCategory = "business"
)

// ErrorSeverity indicates the impact level
type ErrorSeverity string

const (
	SeverityLow      ErrorSeverity = "low"
	SeverityMedium   ErrorSeverity = "medium"
	SeverityHigh     ErrorSeverity = "high"
	SeverityCritical ErrorSeverity = "critical"
)

// MCPError is our enhanced error type with full context
type MCPError struct {
	// Core error information
	Code      int           `json:"code"`
	Message   string        `json:"message"`
	Category  ErrorCategory `json:"category"`
	Severity  ErrorSeverity `json:"severity"`
	Service   string        `json:"service"`
	Operation string        `json:"operation"`

	// Context and debugging
	Context     map[string]interface{} `json:"context"`
	Suggestions []string               `json:"suggestions"`
	Actions     []RecoveryAction       `json:"recovery_actions"`

	// Behavior flags
	Retryable bool `json:"retryable"`
	UserError bool `json:"user_error"`
	Temporary bool `json:"temporary"`

	// Tracing and correlation
	TraceID     string `json:"trace_id"`
	SpanID      string `json:"span_id"`
	Correlation string `json:"correlation_id"`

	// Error chain
	Cause error       `json:"cause,omitempty"`
	Chain []*MCPError `json:"error_chain,omitempty"`

	// Metadata
	Timestamp   time.Time   `json:"timestamp"`
	Source      ErrorSource `json:"source"`
	Fingerprint string      `json:"fingerprint"`

	// Recovery information
	RecoveryAttempted bool       `json:"recovery_attempted"`
	RecoveryResult    string     `json:"recovery_result,omitempty"`
	RetryCount        int        `json:"retry_count"`
	NextRetryAt       *time.Time `json:"next_retry_at,omitempty"`
}

// ErrorSource provides source location information
type ErrorSource struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Package  string `json:"package"`
}

// RecoveryAction describes automated recovery actions
type RecoveryAction struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Automated   bool                   `json:"automated"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Success     bool                   `json:"success"`
	ExecutedAt  *time.Time             `json:"executed_at,omitempty"`
}

// Error implements the error interface
func (e *MCPError) Error() string {
	return fmt.Sprintf("[%s:%s] %s: %s", e.Service, e.Category, e.Operation, e.Message)
}

// Unwrap implements error unwrapping
func (e *MCPError) Unwrap() error {
	return e.Cause
}

// ErrorHandler manages error processing and recovery
type ErrorHandler struct {
	logger    logging.Logger
	metrics   metrics.Metrics
	recovery  RecoveryManager
	policies  []ErrorPolicy
	reporters []ErrorReporter
}

// ErrorPolicy defines how to handle specific error types
type ErrorPolicy struct {
	Matches     func(*MCPError) bool
	MaxRetries  int
	BackoffFunc func(attempt int) time.Duration
	Recovery    []string // Recovery action types to attempt
	Escalation  []string // Escalation targets
	Severity    ErrorSeverity
}

// NewErrorHandler creates a comprehensive error handler
func NewErrorHandler(logger logging.Logger, metrics metrics.Metrics) *ErrorHandler {
	eh := &ErrorHandler{
		logger:    logger.WithComponent("error_handler"),
		metrics:   metrics,
		recovery:  NewRecoveryManager(logger, metrics),
		policies:  defaultErrorPolicies(),
		reporters: []ErrorReporter{},
	}

	return eh
}

// Handle processes an error with full context and recovery
func (eh *ErrorHandler) Handle(ctx context.Context, err error, operation string, context map[string]interface{}) *MCPError {
	// Convert to MCPError if not already
	mcpErr := eh.toMCPError(err, operation, context)

	// Add trace information
	mcpErr.TraceID = getTraceID(ctx)
	mcpErr.SpanID = getSpanID(ctx)
	mcpErr.Correlation = getCorrelationID(ctx)

	// Apply error policies
	policy := eh.findPolicy(mcpErr)
	if policy != nil {
		mcpErr.Severity = policy.Severity
		mcpErr.Retryable = policy.MaxRetries > 0
	}

	// Generate fingerprint for deduplication
	mcpErr.Fingerprint = eh.generateFingerprint(mcpErr)

	// Attempt recovery if configured
	if policy != nil && len(policy.Recovery) > 0 {
		mcpErr.RecoveryAttempted = true
		mcpErr.RecoveryResult = eh.recovery.AttemptRecovery(ctx, mcpErr, policy.Recovery)
	}

	// Record metrics
	eh.recordMetrics(mcpErr)

	// Log with full context
	eh.logError(mcpErr)

	// Report to external systems
	eh.reportError(mcpErr)

	return mcpErr
}

// toMCPError converts any error to MCPError with enhanced context
func (eh *ErrorHandler) toMCPError(err error, operation string, context map[string]interface{}) *MCPError {
	if mcpErr, ok := err.(*MCPError); ok {
		// Already an MCPError, add context
		for k, v := range context {
			mcpErr.Context[k] = v
		}
		return mcpErr
	}

	// Create new MCPError
	mcpErr := &MCPError{
		Code:      -32603, // Default internal error
		Message:   err.Error(),
		Category:  eh.categorizeError(err),
		Severity:  eh.assessSeverity(err),
		Operation: operation,
		Context:   context,
		Timestamp: time.Now(),
		Source:    eh.getErrorSource(),
		Cause:     err,
	}

	// Add suggestions based on error type
	mcpErr.Suggestions = eh.generateSuggestions(mcpErr)

	return mcpErr
}

// categorizeError automatically categorizes errors
func (eh *ErrorHandler) categorizeError(err error) ErrorCategory {
	errStr := strings.ToLower(err.Error())

	switch {
	case contains(errStr, "timeout", "deadline"):
		return CategoryTimeout
	case contains(errStr, "connection", "network", "unreachable"):
		return CategoryNetwork
	case contains(errStr, "permission", "forbidden", "unauthorized"):
		return CategoryAuthorization
	case contains(errStr, "invalid", "malformed", "parse"):
		return CategoryValidation
	case contains(errStr, "rate limit", "too many"):
		return CategoryRateLimit
	case contains(errStr, "unavailable", "service down", "maintenance"):
		return CategoryUnavailable
	case contains(errStr, "config", "setting", "property"):
		return CategoryConfiguration
	case contains(errStr, "memory", "disk", "resource"):
		return CategoryResource
	default:
		return CategoryInternal
	}
}

// assessSeverity determines error severity
func (eh *ErrorHandler) assessSeverity(err error) ErrorSeverity {
	errStr := strings.ToLower(err.Error())

	switch {
	case contains(errStr, "panic", "fatal", "critical", "system"):
		return SeverityCritical
	case contains(errStr, "error", "failed", "exception"):
		return SeverityHigh
	case contains(errStr, "warning", "deprecated", "slow"):
		return SeverityMedium
	default:
		return SeverityLow
	}
}

// generateSuggestions creates actionable suggestions
func (eh *ErrorHandler) generateSuggestions(mcpErr *MCPError) []string {
	suggestions := []string{}

	switch mcpErr.Category {
	case CategoryTimeout:
		suggestions = append(suggestions,
			"Increase timeout configuration",
			"Check backend service health",
			"Verify network connectivity",
			"Consider implementing circuit breaker")

	case CategoryAuthentication:
		suggestions = append(suggestions,
			"Verify authentication credentials",
			"Check token expiration",
			"Refresh authentication if applicable",
			"Review authentication configuration")

	case CategoryRateLimit:
		suggestions = append(suggestions,
			"Implement exponential backoff",
			"Reduce request frequency",
			"Contact administrator for limit increase",
			"Use request queuing")

	case CategoryValidation:
		suggestions = append(suggestions,
			"Check request format and parameters",
			"Verify required fields are present",
			"Validate data types and formats",
			"Review API documentation")

	case CategoryNetwork:
		suggestions = append(suggestions,
			"Check network connectivity",
			"Verify DNS resolution",
			"Check firewall rules",
			"Test with alternative endpoints")

	case CategoryUnavailable:
		suggestions = append(suggestions,
			"Retry operation after delay",
			"Check service status page",
			"Switch to backup service if available",
			"Implement graceful degradation")
	}

	return suggestions
}

// getErrorSource captures source location
func (eh *ErrorHandler) getErrorSource() ErrorSource {
	pc, file, line, _ := runtime.Caller(3)
	fn := runtime.FuncForPC(pc)

	parts := strings.Split(fn.Name(), "/")
	pkg := ""
	if len(parts) > 1 {
		pkg = parts[len(parts)-2]
	}

	return ErrorSource{
		Function: fn.Name(),
		File:     file,
		Line:     line,
		Package:  pkg,
	}
}

// recordMetrics records error metrics
func (eh *ErrorHandler) recordMetrics(mcpErr *MCPError) {
	labels := map[string]string{
		"category":  string(mcpErr.Category),
		"severity":  string(mcpErr.Severity),
		"service":   mcpErr.Service,
		"operation": mcpErr.Operation,
		"retryable": fmt.Sprintf("%t", mcpErr.Retryable),
	}

	eh.metrics.Inc("errors_total", labelsToSlice(labels)...)
	eh.metrics.Inc(fmt.Sprintf("errors_%s_total", mcpErr.Category), labelsToSlice(labels)...)

	if mcpErr.RecoveryAttempted {
		eh.metrics.Inc("error_recovery_attempts_total", labelsToSlice(labels)...)
		if mcpErr.RecoveryResult == "success" {
			eh.metrics.Inc("error_recovery_success_total", labelsToSlice(labels)...)
		}
	}
}

// logError logs with full LLM-optimized context
func (eh *ErrorHandler) logError(mcpErr *MCPError) {
	logLevel := "error"
	if mcpErr.Severity == SeverityLow {
		logLevel = "warn"
	} else if mcpErr.Severity == SeverityCritical {
		logLevel = "error"
	}

	fields := []interface{}{
		"error_category", mcpErr.Category,
		"error_severity", mcpErr.Severity,
		"error_code", mcpErr.Code,
		"service", mcpErr.Service,
		"operation", mcpErr.Operation,
		"retryable", mcpErr.Retryable,
		"user_error", mcpErr.UserError,
		"temporary", mcpErr.Temporary,
		"trace_id", mcpErr.TraceID,
		"span_id", mcpErr.SpanID,
		"correlation_id", mcpErr.Correlation,
		"fingerprint", mcpErr.Fingerprint,
		"suggestions", mcpErr.Suggestions,
		"recovery_attempted", mcpErr.RecoveryAttempted,
		"recovery_result", mcpErr.RecoveryResult,
		"retry_count", mcpErr.RetryCount,
		"source_function", mcpErr.Source.Function,
		"source_file", mcpErr.Source.File,
		"source_line", mcpErr.Source.Line,
		"context", mcpErr.Context,
	}

	switch logLevel {
	case "warn":
		eh.logger.Warn("error_occurred", fields...)
	default:
		eh.logger.Error("error_occurred", fields...)
	}
}

// Standard error constructors with enhanced context
func ValidationError(service, operation, message string, context map[string]interface{}) *MCPError {
	return &MCPError{
		Code:      -32602,
		Message:   message,
		Category:  CategoryValidation,
		Severity:  SeverityMedium,
		Service:   service,
		Operation: operation,
		Context:   context,
		UserError: true,
		Retryable: false,
		Timestamp: time.Now(),
		Suggestions: []string{
			"Check request parameters",
			"Verify data formats",
			"Review API documentation",
		},
	}
}

func TimeoutError(service, operation string, timeout time.Duration, context map[string]interface{}) *MCPError {
	if context == nil {
		context = make(map[string]interface{})
	}
	context["timeout_duration"] = timeout.String()

	return &MCPError{
		Code:      -32001,
		Message:   fmt.Sprintf("Operation timed out after %v", timeout),
		Category:  CategoryTimeout,
		Severity:  SeverityHigh,
		Service:   service,
		Operation: operation,
		Context:   context,
		Retryable: true,
		Temporary: true,
		Timestamp: time.Now(),
		Suggestions: []string{
			"Increase timeout configuration",
			"Check backend service health",
			"Verify network connectivity",
			"Consider implementing circuit breaker",
		},
	}
}

func UnavailableError(service, operation string, cause error, context map[string]interface{}) *MCPError {
	return &MCPError{
		Code:      -32004,
		Message:   fmt.Sprintf("Service %s is unavailable", service),
		Category:  CategoryUnavailable,
		Severity:  SeverityCritical,
		Service:   service,
		Operation: operation,
		Context:   context,
		Cause:     cause,
		Retryable: true,
		Temporary: true,
		Timestamp: time.Now(),
		Suggestions: []string{
			"Retry operation after delay",
			"Check service status",
			"Switch to backup service",
			"Implement graceful degradation",
		},
	}
}

// Helper functions
func contains(s string, substrings ...string) bool {
	for _, substr := range substrings {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

func labelsToSlice(labels map[string]string) []string {
	slice := make([]string, 0, len(labels)*2)
	for k, v := range labels {
		slice = append(slice, k, v)
	}
	return slice
}

func getTraceID(ctx context.Context) string {
	if v := ctx.Value("trace_id"); v != nil {
		return v.(string)
	}
	return ""
}

func getSpanID(ctx context.Context) string {
	if v := ctx.Value("span_id"); v != nil {
		return v.(string)
	}
	return ""
}

func getCorrelationID(ctx context.Context) string {
	if v := ctx.Value("correlation_id"); v != nil {
		return v.(string)
	}
	return ""
}

func (eh *ErrorHandler) generateFingerprint(mcpErr *MCPError) string {
	// Create unique fingerprint for error deduplication
	data := fmt.Sprintf("%s:%s:%s:%s",
		mcpErr.Service,
		mcpErr.Category,
		mcpErr.Operation,
		mcpErr.Message)
	// In production, use proper hash function
	return fmt.Sprintf("%x", len(data))
}

func (eh *ErrorHandler) findPolicy(mcpErr *MCPError) *ErrorPolicy {
	for _, policy := range eh.policies {
		if policy.Matches(mcpErr) {
			return &policy
		}
	}
	return nil
}

func (eh *ErrorHandler) reportError(mcpErr *MCPError) {
	for _, reporter := range eh.reporters {
		go reporter.Report(mcpErr)
	}
}

func defaultErrorPolicies() []ErrorPolicy {
	return []ErrorPolicy{
		{
			Matches:    func(e *MCPError) bool { return e.Category == CategoryTimeout },
			MaxRetries: 3,
			BackoffFunc: func(attempt int) time.Duration {
				return time.Duration(attempt*attempt) * time.Second
			},
			Recovery: []string{"circuit_breaker", "fallback"},
			Severity: SeverityHigh,
		},
		{
			Matches:    func(e *MCPError) bool { return e.Category == CategoryRateLimit },
			MaxRetries: 5,
			BackoffFunc: func(attempt int) time.Duration {
				return time.Duration(1<<attempt) * time.Second
			},
			Recovery: []string{"exponential_backoff"},
			Severity: SeverityMedium,
		},
	}
}

// ErrorReporter interface for external error reporting
type ErrorReporter interface {
	Report(mcpErr *MCPError) error
}

// RecoveryManager handles automated error recovery
type RecoveryManager struct {
	logger  logging.Logger
	metrics metrics.Metrics
	actions map[string]RecoveryActionFunc
}

type RecoveryActionFunc func(ctx context.Context, mcpErr *MCPError) error

func NewRecoveryManager(logger logging.Logger, metrics metrics.Metrics) RecoveryManager {
	return RecoveryManager{
		logger:  logger.WithComponent("recovery_manager"),
		metrics: metrics,
		actions: map[string]RecoveryActionFunc{
			"circuit_breaker":     circuitBreakerRecovery,
			"fallback":            fallbackRecovery,
			"exponential_backoff": exponentialBackoffRecovery,
		},
	}
}

func (rm RecoveryManager) AttemptRecovery(ctx context.Context, mcpErr *MCPError, actions []string) string {
	for _, actionType := range actions {
		if action, exists := rm.actions[actionType]; exists {
			if err := action(ctx, mcpErr); err == nil {
				rm.logger.Info("recovery_successful",
					"action_type", actionType,
					"error_fingerprint", mcpErr.Fingerprint)
				return "success"
			}
		}
	}
	return "failed"
}

// Recovery action implementations
func circuitBreakerRecovery(ctx context.Context, mcpErr *MCPError) error {
	// Implementation would trigger circuit breaker
	return nil
}

func fallbackRecovery(ctx context.Context, mcpErr *MCPError) error {
	// Implementation would switch to fallback service
	return nil
}

func exponentialBackoffRecovery(ctx context.Context, mcpErr *MCPError) error {
	// Implementation would schedule retry with backoff
	return nil
}

// Error type checking functions
func IsValidationError(err error) bool {
	if mcpErr, ok := err.(*MCPError); ok {
		return mcpErr.Category == CategoryValidation
	}
	return false
}

func IsUnavailableError(err error) bool {
	if mcpErr, ok := err.(*MCPError); ok {
		return mcpErr.Category == CategoryUnavailable
	}
	return false
}

func IsTimeoutError(err error) bool {
	if mcpErr, ok := err.(*MCPError); ok {
		return mcpErr.Category == CategoryTimeout
	}
	return false
}
