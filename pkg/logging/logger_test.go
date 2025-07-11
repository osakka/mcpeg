package logging

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// testOutput captures log entries for testing
type testOutput struct {
	entries []Entry
}

func (t *testOutput) capture(entry Entry) {
	t.entries = append(t.entries, entry)
}

func TestLLMLogger(t *testing.T) {
	output := &testOutput{}
	logger := &llmLogger{
		component: "test.component",
		output:    output.capture,
	}

	t.Run("logs contain all required fields", func(t *testing.T) {
		logger.Info("test_operation", 
			"key1", "value1",
			"key2", 42,
			"nested", map[string]interface{}{
				"inner": "data",
			})

		if len(output.entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(output.entries))
		}

		entry := output.entries[0]
		
		// Verify required fields
		if entry.Level != LevelInfo {
			t.Errorf("expected level INFO, got %s", entry.Level)
		}
		if entry.Component != "test.component" {
			t.Errorf("expected component test.component, got %s", entry.Component)
		}
		if entry.Operation != "test_operation" {
			t.Errorf("expected operation test_operation, got %s", entry.Operation)
		}
		if entry.Timestamp.IsZero() {
			t.Error("timestamp should not be zero")
		}
		
		// Verify data fields
		if entry.Data["key1"] != "value1" {
			t.Errorf("expected key1=value1, got %v", entry.Data["key1"])
		}
		if entry.Data["key2"] != 42 {
			t.Errorf("expected key2=42, got %v", entry.Data["key2"])
		}
		
		// Verify execution context
		if entry.Context == nil {
			t.Error("execution context should not be nil")
		} else {
			if entry.Context.Goroutine <= 0 {
				t.Error("goroutine count should be positive")
			}
			if entry.Context.Function == "" {
				t.Error("function name should not be empty")
			}
		}
	})

	t.Run("error logs include suggestions", func(t *testing.T) {
		output.entries = nil
		logger.Error("api_call_failed",
			"error_type", "timeout",
			"endpoint", "https://api.example.com",
			"timeout_ms", 5000)

		if len(output.entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(output.entries))
		}

		entry := output.entries[0]
		if len(entry.Suggestions) == 0 {
			t.Error("error log should include suggestions")
		}
		
		// Verify timeout-specific suggestions
		foundTimeoutSuggestion := false
		for _, suggestion := range entry.Suggestions {
			if strings.Contains(suggestion, "timeout") {
				foundTimeoutSuggestion = true
				break
			}
		}
		if !foundTimeoutSuggestion {
			t.Error("should include timeout-specific suggestion")
		}
	})

	t.Run("context propagation", func(t *testing.T) {
		output.entries = nil
		
		ctx := context.WithValue(context.Background(), "trace_id", "trace-123")
		ctx = context.WithValue(ctx, "span_id", "span-456")
		
		contextLogger := logger.WithContext(ctx)
		contextLogger.Info("context_test")
		
		if len(output.entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(output.entries))
		}
		
		entry := output.entries[0]
		if entry.TraceID != "trace-123" {
			t.Errorf("expected trace_id=trace-123, got %s", entry.TraceID)
		}
		if entry.SpanID != "span-456" {
			t.Errorf("expected span_id=span-456, got %s", entry.SpanID)
		}
	})

	t.Run("JSON serialization", func(t *testing.T) {
		output.entries = nil
		logger.Info("json_test",
			"string", "value",
			"number", 123,
			"float", 123.45,
			"bool", true,
			"array", []string{"a", "b", "c"},
			"map", map[string]interface{}{"key": "value"})

		if len(output.entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(output.entries))
		}

		// Verify JSON marshaling works
		data, err := json.Marshal(output.entries[0])
		if err != nil {
			t.Fatalf("failed to marshal entry: %v", err)
		}

		// Verify it's valid JSON
		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		// Verify structure for LLM consumption
		if _, ok := parsed["timestamp"]; !ok {
			t.Error("JSON should contain timestamp")
		}
		if _, ok := parsed["data"]; !ok {
			t.Error("JSON should contain data")
		}
		if _, ok := parsed["context"]; !ok {
			t.Error("JSON should contain context")
		}
	})
}

func TestErrorChain(t *testing.T) {
	// Create nested errors
	rootErr := &testError{msg: "root cause"}
	wrappedErr := &wrappedError{msg: "wrapped error", cause: rootErr}
	
	chain := buildErrorChain(wrappedErr)
	
	if chain == nil {
		t.Fatal("error chain should not be nil")
	}
	
	if chain.Message != "wrapped error" {
		t.Errorf("expected message 'wrapped error', got %s", chain.Message)
	}
	
	if chain.Cause == nil {
		t.Fatal("cause should not be nil")
	}
	
	if chain.Cause.Message != "root cause" {
		t.Errorf("expected cause message 'root cause', got %s", chain.Cause.Message)
	}
}

// Test error types
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

type wrappedError struct {
	msg   string
	cause error
}

func (e *wrappedError) Error() string {
	return e.msg
}

func (e *wrappedError) Unwrap() error {
	return e.cause
}