package validation

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// MemoryCache provides in-memory caching for validation results
type MemoryCache struct {
	cache   map[string]*CacheEntry
	mutex   sync.RWMutex
	maxSize int
}

// CacheEntry represents a cached validation result
type CacheEntry struct {
	Result      *ValidationResult
	ExpiresAt   time.Time
	AccessCount int
	LastAccess  time.Time
}

// NewMemoryCache creates a new in-memory validation cache
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		cache:   make(map[string]*CacheEntry),
		maxSize: 1000, // Default max size
	}
}

// Get retrieves a validation result from cache
func (c *MemoryCache) Get(key string) (*ValidationResult, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		// Remove expired entry (deferred cleanup)
		go c.removeExpired(key)
		return nil, false
	}

	// Update access statistics
	entry.AccessCount++
	entry.LastAccess = time.Now()

	return entry.Result, true
}

// Set stores a validation result in cache
func (c *MemoryCache) Set(key string, result *ValidationResult, expiry time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check cache size and evict if necessary
	if len(c.cache) >= c.maxSize {
		c.evictLRU()
	}

	c.cache[key] = &CacheEntry{
		Result:      result,
		ExpiresAt:   time.Now().Add(expiry),
		AccessCount: 1,
		LastAccess:  time.Now(),
	}
}

// Clear removes all entries from cache
func (c *MemoryCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache = make(map[string]*CacheEntry)
}

// removeExpired removes an expired entry
func (c *MemoryCache) removeExpired(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.cache, key)
}

// evictLRU removes the least recently used entry
func (c *MemoryCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time = time.Now()

	for key, entry := range c.cache {
		if entry.LastAccess.Before(oldestTime) {
			oldestTime = entry.LastAccess
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(c.cache, oldestKey)
	}
}

// getCachedResult generates cache key and retrieves cached result
func (v *Validator) getCachedResult(value interface{}, category string) *ValidationResult {
	key := v.generateCacheKey(value, category)
	result, found := v.cache.Get(key)
	if found {
		return result
	}
	return nil
}

// setCachedResult generates cache key and stores result
func (v *Validator) setCachedResult(value interface{}, category string, result *ValidationResult) {
	key := v.generateCacheKey(value, category)
	v.cache.Set(key, result, v.config.CacheExpiry)
}

// generateCacheKey creates a unique cache key for the validation
func (v *Validator) generateCacheKey(value interface{}, category string) string {
	// Serialize value to create consistent key
	data, err := json.Marshal(map[string]interface{}{
		"value":    value,
		"category": category,
		"config":   v.config,
	})
	if err != nil {
		// Fallback to string representation
		return fmt.Sprintf("%s:%v", category, value)
	}

	// Create MD5 hash for consistent key length
	hash := md5.Sum(data)
	return fmt.Sprintf("validation:%x", hash)
}

// validateAgainstSchema validates data against a JSON schema (simplified implementation)
func (v *Validator) validateAgainstSchema(ctx context.Context, data interface{}, schema interface{}) ValidationResult {
	result := ValidationResult{
		Valid:       true,
		Errors:      make([]ValidationError, 0),
		Warnings:    make([]ValidationWarning, 0),
		Context:     make(map[string]interface{}),
		Suggestions: make([]string, 0),
	}

	// This is a simplified schema validation implementation
	// In a full implementation, you would use a proper JSON Schema validator

	result.Context["schema_validation"] = map[string]interface{}{
		"schema_type": fmt.Sprintf("%T", schema),
		"data_type":   fmt.Sprintf("%T", data),
		"note":        "Simplified schema validation implementation",
	}

	return result
}

// recordValidationMetrics records validation performance metrics
func (v *Validator) recordValidationMetrics(result ValidationResult, category string) {
	labels := []string{
		"category", category,
		"valid", fmt.Sprintf("%t", result.Valid),
	}

	v.metrics.Inc("validation_requests_total", labels...)
	v.metrics.Set("validation_duration_seconds", result.Performance.Duration.Seconds(), labels...)
	v.metrics.Set("validation_rules_evaluated", float64(result.Performance.RulesEvaluated), labels...)
	v.metrics.Set("validation_fields_checked", float64(result.Performance.FieldsChecked), labels...)
	v.metrics.Set("validation_cache_hits", float64(result.Performance.CacheHits), labels...)
	v.metrics.Set("validation_cache_misses", float64(result.Performance.CacheMisses), labels...)

	if !result.Valid {
		v.metrics.Inc("validation_failures_total", labels...)
		v.metrics.Set("validation_error_count", float64(len(result.Errors)), labels...)
	}

	if len(result.Warnings) > 0 {
		v.metrics.Set("validation_warning_count", float64(len(result.Warnings)), labels...)
	}
}

// logValidationResult logs validation results with LLM context
func (v *Validator) logValidationResult(result ValidationResult, category string, value interface{}) {
	logLevel := "info"
	message := "validation_completed"

	if !result.Valid {
		logLevel = "error"
		message = "validation_failed"
	} else if len(result.Warnings) > 0 {
		logLevel = "warn"
		message = "validation_completed_with_warnings"
	}

	fields := []interface{}{
		"category", category,
		"valid", result.Valid,
		"error_count", len(result.Errors),
		"warning_count", len(result.Warnings),
		"duration", result.Performance.Duration,
		"rules_evaluated", result.Performance.RulesEvaluated,
		"fields_checked", result.Performance.FieldsChecked,
		"cache_hits", result.Performance.CacheHits,
		"cache_misses", result.Performance.CacheMisses,
		"suggestions", result.Suggestions,
		"context", result.Context,
	}

	// Add detailed error information for failures
	if !result.Valid {
		errorDetails := make([]map[string]interface{}, 0, len(result.Errors))
		for _, err := range result.Errors {
			errorDetails = append(errorDetails, map[string]interface{}{
				"field":       err.Field,
				"message":     err.Message,
				"code":        err.Code,
				"value":       err.Value,
				"expected":    err.Expected,
				"severity":    err.Severity,
				"suggestions": err.Suggestions,
				"context":     err.Context,
			})
		}
		fields = append(fields, "errors", errorDetails)
	}

	// Add warning information
	if len(result.Warnings) > 0 {
		warningDetails := make([]map[string]interface{}, 0, len(result.Warnings))
		for _, warn := range result.Warnings {
			warningDetails = append(warningDetails, map[string]interface{}{
				"field":       warn.Field,
				"message":     warn.Message,
				"code":        warn.Code,
				"value":       warn.Value,
				"suggestions": warn.Suggestions,
			})
		}
		fields = append(fields, "warnings", warningDetails)
	}

	switch logLevel {
	case "error":
		v.logger.Error(message, fields...)
	case "warn":
		v.logger.Warn(message, fields...)
	default:
		v.logger.Info(message, fields...)
	}
}

// registerBuiltInRules registers the built-in validation rules
func (v *Validator) registerBuiltInRules() {
	// Register basic validation rules
	v.RegisterRule("general", &RequiredFieldRule{})
	v.RegisterRule("general", &TypeValidationRule{})
	v.RegisterRule("general", &RangeValidationRule{})
	v.RegisterRule("general", &FormatValidationRule{})

	// Register MCP-specific rules
	v.RegisterRule("mcp", &MCPMethodRule{})
	v.RegisterRule("mcp", &MCPVersionRule{})
	v.RegisterRule("mcp", &MCPParameterRule{})

	v.logger.Info("built_in_validation_rules_registered",
		"total_categories", len(v.rules),
		"total_rules", v.getTotalRuleCount())
}

// getTotalRuleCount returns the total number of registered rules
func (v *Validator) getTotalRuleCount() int {
	total := 0
	for _, rules := range v.rules {
		total += len(rules)
	}
	return total
}
