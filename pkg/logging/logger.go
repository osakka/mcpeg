package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"runtime/debug"
	"time"
)

// LogLevel represents the severity of a log entry
type LogLevel string

const (
	LevelTrace LogLevel = "TRACE"
	LevelDebug LogLevel = "DEBUG"
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARN"
	LevelError LogLevel = "ERROR"
)

// Logger is the interface for LLM-optimized logging
type Logger interface {
	Trace(operation string, fields ...interface{})
	Debug(operation string, fields ...interface{})
	Info(operation string, fields ...interface{})
	Warn(operation string, fields ...interface{})
	Error(operation string, fields ...interface{})
	WithContext(ctx context.Context) Logger
	WithComponent(component string) Logger
	WithTraceID(traceID string) Logger
	WithSpanID(spanID string) Logger
}

// Entry represents a structured log entry optimized for LLM consumption
type Entry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       LogLevel               `json:"level"`
	Component   string                 `json:"component"`
	Operation   string                 `json:"operation"`
	TraceID     string                 `json:"trace_id,omitempty"`
	SpanID      string                 `json:"span_id,omitempty"`
	ParentSpanID string                `json:"parent_span_id,omitempty"`
	Message     string                 `json:"message"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Context     *ExecutionContext      `json:"context,omitempty"`
	Breadcrumbs []Breadcrumb          `json:"breadcrumbs,omitempty"`
	Suggestions []string              `json:"suggestions,omitempty"`
}

// ExecutionContext provides runtime context for debugging
type ExecutionContext struct {
	Goroutine   int    `json:"goroutine"`
	Function    string `json:"function"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	MemoryUsage int64  `json:"memory_bytes"`
}

// Breadcrumb represents a step in the execution path
type Breadcrumb struct {
	Timestamp time.Time              `json:"timestamp"`
	Operation string                 `json:"operation"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// ErrorContext provides comprehensive error information
type ErrorContext struct {
	Type            string                 `json:"type"`
	Message         string                 `json:"message"`
	StackTrace      string                 `json:"stack_trace,omitempty"`
	Cause           *ErrorContext          `json:"cause,omitempty"`
	RecoveryAttempted bool                 `json:"recovery_attempted"`
	RecoveryResult  string                 `json:"recovery_result,omitempty"`
	SuggestedFixes  []string              `json:"suggested_fixes,omitempty"`
	RelatedData     map[string]interface{} `json:"related_data,omitempty"`
}

// llmLogger implements the Logger interface
type llmLogger struct {
	component    string
	traceID      string
	spanID       string
	parentSpanID string
	breadcrumbs  []Breadcrumb
	output       func(entry Entry)
}

// New creates a new LLM-optimized logger
func New(component string) Logger {
	return &llmLogger{
		component: component,
		output:    defaultOutput,
	}
}

// defaultOutput writes JSON to stdout
func defaultOutput(entry Entry) {
	data, _ := json.Marshal(entry)
	fmt.Println(string(data))
}

func (l *llmLogger) log(level LogLevel, operation string, fields []interface{}) {
	entry := Entry{
		Timestamp:    time.Now().UTC(),
		Level:        level,
		Component:    l.component,
		Operation:    operation,
		TraceID:      l.traceID,
		SpanID:       l.spanID,
		ParentSpanID: l.parentSpanID,
		Message:      formatMessage(operation, level),
		Data:         parseFields(fields),
		Context:      getExecutionContext(),
		Breadcrumbs:  l.breadcrumbs,
	}

	// Add suggestions for errors
	if level == LevelError || level == LevelWarn {
		entry.Suggestions = generateSuggestions(operation, entry.Data)
	}

	l.output(entry)
}

func (l *llmLogger) Trace(operation string, fields ...interface{}) {
	l.log(LevelTrace, operation, fields)
}

func (l *llmLogger) Debug(operation string, fields ...interface{}) {
	l.log(LevelDebug, operation, fields)
}

func (l *llmLogger) Info(operation string, fields ...interface{}) {
	l.log(LevelInfo, operation, fields)
}

func (l *llmLogger) Warn(operation string, fields ...interface{}) {
	l.log(LevelWarn, operation, fields)
}

func (l *llmLogger) Error(operation string, fields ...interface{}) {
	l.log(LevelError, operation, fields)
}

func (l *llmLogger) WithContext(ctx context.Context) Logger {
	newLogger := *l
	// Extract trace information from context
	if traceID := ctx.Value("trace_id"); traceID != nil {
		newLogger.traceID = traceID.(string)
	}
	if spanID := ctx.Value("span_id"); spanID != nil {
		newLogger.spanID = spanID.(string)
	}
	return &newLogger
}

func (l *llmLogger) WithComponent(component string) Logger {
	newLogger := *l
	newLogger.component = component
	return &newLogger
}

func (l *llmLogger) WithTraceID(traceID string) Logger {
	newLogger := *l
	newLogger.traceID = traceID
	return &newLogger
}

func (l *llmLogger) WithSpanID(spanID string) Logger {
	newLogger := *l
	newLogger.spanID = spanID
	return &newLogger
}

// Helper functions

func parseFields(fields []interface{}) map[string]interface{} {
	data := make(map[string]interface{})
	for i := 0; i < len(fields)-1; i += 2 {
		key, ok := fields[i].(string)
		if ok {
			data[key] = fields[i+1]
		}
	}
	return data
}

func formatMessage(operation string, level LogLevel) string {
	switch level {
	case LevelError:
		return fmt.Sprintf("Error during %s", operation)
	case LevelWarn:
		return fmt.Sprintf("Warning during %s", operation)
	default:
		return fmt.Sprintf("Operation %s", operation)
	}
}

func getExecutionContext() *ExecutionContext {
	pc, file, line, _ := runtime.Caller(4)
	fn := runtime.FuncForPC(pc)
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return &ExecutionContext{
		Goroutine:   runtime.NumGoroutine(),
		Function:    fn.Name(),
		File:        file,
		Line:        line,
		MemoryUsage: int64(m.Alloc),
	}
}

func generateSuggestions(operation string, data map[string]interface{}) []string {
	suggestions := []string{}
	
	// Add context-specific suggestions based on operation and error type
	if errorType, ok := data["error_type"].(string); ok {
		switch errorType {
		case "timeout":
			suggestions = append(suggestions, 
				"Increase timeout duration",
				"Check backend service health",
				"Enable circuit breaker",
				"Implement request queuing")
		case "connection_refused":
			suggestions = append(suggestions,
				"Verify backend service is running",
				"Check network connectivity",
				"Verify firewall rules",
				"Check service discovery configuration")
		case "rate_limit":
			suggestions = append(suggestions,
				"Implement exponential backoff",
				"Use request queuing",
				"Increase rate limit if possible",
				"Distribute load across time")
		}
	}
	
	return suggestions
}

// LogError provides comprehensive error logging
func LogError(logger Logger, err error, operation string, context map[string]interface{}) {
	fields := make([]interface{}, 0, len(context)*2+10)
	
	// Add error details
	fields = append(fields, 
		"error_type", fmt.Sprintf("%T", err),
		"error_message", err.Error(),
		"stack_trace", string(debug.Stack()))
	
	// Add context
	for k, v := range context {
		fields = append(fields, k, v)
	}
	
	// Build error chain if wrapped
	if cause := buildErrorChain(err); cause != nil {
		fields = append(fields, "error_chain", cause)
	}
	
	logger.Error(operation, fields...)
}

func buildErrorChain(err error) *ErrorContext {
	if err == nil {
		return nil
	}
	
	ctx := &ErrorContext{
		Type:    fmt.Sprintf("%T", err),
		Message: err.Error(),
	}
	
	// Check if error is wrapped
	if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
		if cause := unwrapper.Unwrap(); cause != nil {
			ctx.Cause = buildErrorChain(cause)
		}
	}
	
	return ctx
}