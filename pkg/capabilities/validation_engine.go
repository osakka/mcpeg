package capabilities

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
)

// ValidationEngine provides runtime capability validation and monitoring
type ValidationEngine struct {
	logger             logging.Logger
	metrics            metrics.Metrics
	aggregationEngine  *AggregationEngine
	analysisEngine     *AnalysisEngine

	// Validation state
	validationRules    map[string]*ValidationRule
	validationHistory  map[string][]*ValidationResult
	activeMonitors     map[string]*CapabilityMonitor
	
	// Runtime enforcement
	enforcementPolicies map[string]*EnforcementPolicy
	violations          map[string][]*PolicyViolation
	
	// Synchronization
	mutex sync.RWMutex

	// Configuration
	config ValidationConfig
}

// ValidationConfig configures the validation engine
type ValidationConfig struct {
	EnableRuntimeValidation   bool          `yaml:"enable_runtime_validation"`
	EnableCapabilityMonitoring bool         `yaml:"enable_capability_monitoring"`
	EnablePolicyEnforcement   bool          `yaml:"enable_policy_enforcement"`
	ValidationInterval        time.Duration `yaml:"validation_interval"`
	MonitoringInterval        time.Duration `yaml:"monitoring_interval"`
	ViolationThreshold        int           `yaml:"violation_threshold"`
	AutoRemediation           bool          `yaml:"auto_remediation"`
	ValidationTimeout         time.Duration `yaml:"validation_timeout"`
}

// ValidationRule defines how to validate a capability
type ValidationRule struct {
	RuleID          string                 `json:"rule_id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Category        SemanticCategory       `json:"category"`
	RuleType        ValidationRuleType     `json:"rule_type"`
	Severity        ValidationSeverity     `json:"severity"`
	Conditions      []ValidationCondition  `json:"conditions"`
	Actions         []ValidationAction     `json:"actions"`
	Enabled         bool                   `json:"enabled"`
	CreatedAt       time.Time              `json:"created_at"`
	LastModified    time.Time              `json:"last_modified"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// ValidationRuleType defines types of validation rules
type ValidationRuleType string

const (
	RuleTypeStructural    ValidationRuleType = "structural"    // Validate structure/schema
	RuleTypeBehavioral    ValidationRuleType = "behavioral"    // Validate behavior/response
	RuleTypePerformance   ValidationRuleType = "performance"   // Validate performance metrics
	RuleTypeSecurity      ValidationRuleType = "security"      // Validate security requirements
	RuleTypeCompliance    ValidationRuleType = "compliance"    // Validate compliance requirements
	RuleTypeIntegration   ValidationRuleType = "integration"   // Validate integration compatibility
)

// ValidationSeverity defines validation rule severity levels
type ValidationSeverity string

const (
	ValidationSeverityInfo     ValidationSeverity = "info"
	ValidationSeverityWarning  ValidationSeverity = "warning"
	ValidationSeverityError    ValidationSeverity = "error"
	ValidationSeverityCritical ValidationSeverity = "critical"
)

// ValidationCondition defines a condition to check
type ValidationCondition struct {
	Field       string                 `json:"field"`
	Operator    ConditionOperator      `json:"operator"`
	Value       interface{}            `json:"value"`
	Description string                 `json:"description"`
	Optional    bool                   `json:"optional"`
}

// ConditionOperator defines operators for validation conditions
type ConditionOperator string

const (
	OperatorEquals            ConditionOperator = "equals"
	OperatorNotEquals         ConditionOperator = "not_equals"
	OperatorGreaterThan       ConditionOperator = "greater_than"
	OperatorLessThan          ConditionOperator = "less_than"
	OperatorGreaterThanEqual  ConditionOperator = "greater_than_equal"
	OperatorLessThanEqual     ConditionOperator = "less_than_equal"
	OperatorContains          ConditionOperator = "contains"
	OperatorNotContains       ConditionOperator = "not_contains"
	OperatorStartsWith        ConditionOperator = "starts_with"
	OperatorEndsWith          ConditionOperator = "ends_with"
	OperatorMatches           ConditionOperator = "matches"
	OperatorExists            ConditionOperator = "exists"
	OperatorNotExists         ConditionOperator = "not_exists"
)

// ValidationAction defines actions to take when validation fails
type ValidationAction struct {
	ActionType  ValidationActionType   `json:"action_type"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Automatic   bool                   `json:"automatic"`
}

// ValidationActionType defines types of validation actions
type ValidationActionType string

const (
	ActionLog            ValidationActionType = "log"
	ActionAlert          ValidationActionType = "alert"
	ActionDisable        ValidationActionType = "disable"
	ActionQuarantine     ValidationActionType = "quarantine"
	ActionRemediate      ValidationActionType = "remediate"
	ActionEscalate       ValidationActionType = "escalate"
	ActionNotify         ValidationActionType = "notify"
)

// ValidationResult represents the result of a validation check
type ValidationResult struct {
	ResultID      string                 `json:"result_id"`
	RuleID        string                 `json:"rule_id"`
	PluginName    string                 `json:"plugin_name"`
	CapabilityName string                `json:"capability_name"`
	Status        ValidationStatus       `json:"status"`
	Score         float64                `json:"score"`
	Issues        []ValidationIssue      `json:"issues"`
	Recommendations []string             `json:"recommendations"`
	ExecutedAt    time.Time              `json:"executed_at"`
	Duration      time.Duration          `json:"duration"`
	Context       map[string]interface{} `json:"context"`
}

// ValidationStatus defines validation result statuses
type ValidationStatus string

const (
	StatusPassed    ValidationStatus = "passed"
	StatusWarning   ValidationStatus = "warning"
	StatusFailed    ValidationStatus = "failed"
	StatusError     ValidationStatus = "error"
	StatusSkipped   ValidationStatus = "skipped"
)

// ValidationIssue represents a specific validation issue
type ValidationIssue struct {
	IssueID     string                 `json:"issue_id"`
	Severity    ValidationSeverity     `json:"severity"`
	Message     string                 `json:"message"`
	Field       string                 `json:"field"`
	Expected    interface{}            `json:"expected"`
	Actual      interface{}            `json:"actual"`
	Suggestion  string                 `json:"suggestion"`
	Context     map[string]interface{} `json:"context"`
}

// CapabilityMonitor monitors a capability at runtime
type CapabilityMonitor struct {
	MonitorID       string                 `json:"monitor_id"`
	PluginName      string                 `json:"plugin_name"`
	CapabilityName  string                 `json:"capability_name"`
	MonitorType     MonitorType            `json:"monitor_type"`
	Configuration   MonitorConfiguration   `json:"configuration"`
	Status          MonitorStatus          `json:"status"`
	Metrics         RuntimeMetrics         `json:"metrics"`
	Alerts          []MonitorAlert         `json:"alerts"`
	CreatedAt       time.Time              `json:"created_at"`
	LastChecked     time.Time              `json:"last_checked"`
}

// MonitorType defines types of capability monitors
type MonitorType string

const (
	MonitorPerformance   MonitorType = "performance"
	MonitorAvailability  MonitorType = "availability"
	MonitorQuality       MonitorType = "quality"
	MonitorSecurity      MonitorType = "security"
	MonitorCompliance    MonitorType = "compliance"
	MonitorUsage         MonitorType = "usage"
)

// MonitorConfiguration configures a capability monitor
type MonitorConfiguration struct {
	CheckInterval     time.Duration          `json:"check_interval"`
	Thresholds        map[string]float64     `json:"thresholds"`
	AlertConditions   []AlertCondition       `json:"alert_conditions"`
	SamplingRate      float64                `json:"sampling_rate"`
	Enabled           bool                   `json:"enabled"`
	AutoRemediation   bool                   `json:"auto_remediation"`
}

// MonitorStatus defines monitor statuses
type MonitorStatus string

const (
	MonitorStatusActive    MonitorStatus = "active"
	MonitorStatusInactive  MonitorStatus = "inactive"
	MonitorStatusError     MonitorStatus = "error"
	MonitorStatusSuspended MonitorStatus = "suspended"
)

// RuntimeMetrics tracks runtime metrics for a capability
type RuntimeMetrics struct {
	CallCount         int64         `json:"call_count"`
	SuccessCount      int64         `json:"success_count"`
	ErrorCount        int64         `json:"error_count"`
	AverageLatency    time.Duration `json:"average_latency"`
	P95Latency        time.Duration `json:"p95_latency"`
	P99Latency        time.Duration `json:"p99_latency"`
	ThroughputRPS     float64       `json:"throughput_rps"`
	ErrorRate         float64       `json:"error_rate"`
	AvailabilityPct   float64       `json:"availability_pct"`
	LastSuccess       time.Time     `json:"last_success"`
	LastError         time.Time     `json:"last_error"`
	TrendDirection    string        `json:"trend_direction"`
}

// AlertCondition defines when to trigger alerts
type AlertCondition struct {
	Metric      string                 `json:"metric"`
	Operator    ConditionOperator      `json:"operator"`
	Threshold   float64                `json:"threshold"`
	Duration    time.Duration          `json:"duration"`
	Severity    ValidationSeverity     `json:"severity"`
	Description string                 `json:"description"`
}

// MonitorAlert represents an alert triggered by a monitor
type MonitorAlert struct {
	AlertID     string                 `json:"alert_id"`
	MonitorID   string                 `json:"monitor_id"`
	Condition   AlertCondition         `json:"condition"`
	Message     string                 `json:"message"`
	Severity    ValidationSeverity     `json:"severity"`
	Status      AlertStatus            `json:"status"`
	TriggeredAt time.Time              `json:"triggered_at"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	Context     map[string]interface{} `json:"context"`
}

// AlertStatus defines alert statuses
type AlertStatus string

const (
	AlertStatusActive    AlertStatus = "active"
	AlertStatusResolved  AlertStatus = "resolved"
	AlertStatusSuppressed AlertStatus = "suppressed"
)

// EnforcementPolicy defines how to enforce capability requirements
type EnforcementPolicy struct {
	PolicyID      string                 `json:"policy_id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Scope         PolicyScope            `json:"scope"`
	Requirements  []PolicyRequirement    `json:"requirements"`
	Actions       []EnforcementAction    `json:"actions"`
	Enabled       bool                   `json:"enabled"`
	Priority      int                    `json:"priority"`
	CreatedAt     time.Time              `json:"created_at"`
	LastModified  time.Time              `json:"last_modified"`
}

// PolicyScope defines the scope of an enforcement policy
type PolicyScope struct {
	Categories   []SemanticCategory `json:"categories"`
	Plugins      []string           `json:"plugins"`
	Capabilities []string           `json:"capabilities"`
	Global       bool               `json:"global"`
}

// PolicyRequirement defines a requirement that must be met
type PolicyRequirement struct {
	RequirementID   string                 `json:"requirement_id"`
	Type            RequirementType        `json:"type"`
	Description     string                 `json:"description"`
	Condition       ValidationCondition    `json:"condition"`
	Mandatory       bool                   `json:"mandatory"`
	GracePeriod     time.Duration          `json:"grace_period"`
}

// RequirementType defines types of policy requirements
type RequirementType string

const (
	RequirementQuality      RequirementType = "quality"
	RequirementPerformance  RequirementType = "performance"
	RequirementSecurity     RequirementType = "security"
	RequirementCompliance   RequirementType = "compliance"
	RequirementDocumentation RequirementType = "documentation"
	RequirementTesting      RequirementType = "testing"
)

// EnforcementAction defines actions to take when requirements are not met
type EnforcementAction struct {
	ActionType    EnforcementActionType  `json:"action_type"`
	Description   string                 `json:"description"`
	Parameters    map[string]interface{} `json:"parameters"`
	Automatic     bool                   `json:"automatic"`
	DelaySeconds  int                    `json:"delay_seconds"`
}

// EnforcementActionType defines types of enforcement actions
type EnforcementActionType string

const (
	EnforcementWarn       EnforcementActionType = "warn"
	EnforcementThrottle   EnforcementActionType = "throttle"
	EnforcementBlock      EnforcementActionType = "block"
	EnforcementQuarantine EnforcementActionType = "quarantine"
	EnforcementRevert     EnforcementActionType = "revert"
	EnforcementEscalate   EnforcementActionType = "escalate"
)

// PolicyViolation represents a violation of an enforcement policy
type PolicyViolation struct {
	ViolationID   string                 `json:"violation_id"`
	PolicyID      string                 `json:"policy_id"`
	RequirementID string                 `json:"requirement_id"`
	PluginName    string                 `json:"plugin_name"`
	CapabilityName string                `json:"capability_name"`
	Severity      ValidationSeverity     `json:"severity"`
	Message       string                 `json:"message"`
	Evidence      map[string]interface{} `json:"evidence"`
	Actions       []EnforcementAction    `json:"actions"`
	Status        ViolationStatus        `json:"status"`
	DetectedAt    time.Time              `json:"detected_at"`
	ResolvedAt    *time.Time             `json:"resolved_at,omitempty"`
}

// ViolationStatus defines violation statuses
type ViolationStatus string

const (
	ViolationStatusActive    ViolationStatus = "active"
	ViolationStatusResolved  ViolationStatus = "resolved"
	ViolationStatusSuppressed ViolationStatus = "suppressed"
	ViolationStatusEscalated ViolationStatus = "escalated"
)

// NewValidationEngine creates a new capability validation engine
func NewValidationEngine(
	logger logging.Logger,
	metrics metrics.Metrics,
	aggregationEngine *AggregationEngine,
	analysisEngine *AnalysisEngine,
	config ValidationConfig,
) *ValidationEngine {
	return &ValidationEngine{
		logger:              logger.WithComponent("capability-validation"),
		metrics:             metrics,
		aggregationEngine:   aggregationEngine,
		analysisEngine:      analysisEngine,
		validationRules:     make(map[string]*ValidationRule),
		validationHistory:   make(map[string][]*ValidationResult),
		activeMonitors:      make(map[string]*CapabilityMonitor),
		enforcementPolicies: make(map[string]*EnforcementPolicy),
		violations:          make(map[string][]*PolicyViolation),
		config:              config,
	}
}

// ValidateCapability performs comprehensive validation of a capability
func (ve *ValidationEngine) ValidateCapability(
	ctx context.Context,
	pluginName, capabilityName string,
) (*ValidationResult, error) {
	startTime := time.Now()
	defer func() {
		ve.metrics.Observe("capability_validation_duration_ms", float64(time.Since(startTime).Nanoseconds()/1e6))
	}()

	ve.logger.WithContext(ctx).Info("capability_validation_started",
		"plugin", pluginName,
		"capability", capabilityName,
	)

	// Get capability analysis
	analysis, exists := ve.analysisEngine.GetAnalysis(pluginName, capabilityName)
	if !exists {
		return nil, fmt.Errorf("capability analysis not found: %s:%s", pluginName, capabilityName)
	}

	result := &ValidationResult{
		ResultID:        fmt.Sprintf("validation_%d", time.Now().UnixNano()),
		PluginName:      pluginName,
		CapabilityName:  capabilityName,
		Status:          StatusPassed,
		Score:           1.0,
		Issues:          []ValidationIssue{},
		Recommendations: []string{},
		ExecutedAt:      time.Now(),
		Context:         make(map[string]interface{}),
	}

	var totalScore float64
	var ruleCount int

	// Apply validation rules
	for _, rule := range ve.getApplicableRules(analysis.SemanticCategory) {
		if !rule.Enabled {
			continue
		}

		ruleResult := ve.applyValidationRule(ctx, rule, analysis)
		result.RuleID = rule.RuleID

		// Accumulate issues
		result.Issues = append(result.Issues, ruleResult.Issues...)

		// Update overall score
		totalScore += ruleResult.Score
		ruleCount++

		// Update status based on worst issue severity
		if ruleResult.Status == StatusFailed && result.Status != StatusFailed {
			result.Status = StatusFailed
		} else if ruleResult.Status == StatusWarning && result.Status == StatusPassed {
			result.Status = StatusWarning
		}
	}

	// Calculate final score
	if ruleCount > 0 {
		result.Score = totalScore / float64(ruleCount)
	}

	// Generate recommendations based on issues
	result.Recommendations = ve.generateValidationRecommendations(result.Issues)

	result.Duration = time.Since(startTime)

	// Store validation history
	ve.mutex.Lock()
	key := fmt.Sprintf("%s:%s", pluginName, capabilityName)
	ve.validationHistory[key] = append(ve.validationHistory[key], result)
	
	// Keep only last 100 results per capability
	if len(ve.validationHistory[key]) > 100 {
		ve.validationHistory[key] = ve.validationHistory[key][1:]
	}
	ve.mutex.Unlock()

	ve.logger.WithContext(ctx).Info("capability_validation_completed",
		"plugin", pluginName,
		"capability", capabilityName,
		"status", result.Status,
		"score", result.Score,
		"issues", len(result.Issues),
	)

	ve.metrics.Inc("capability_validations_total")
	ve.metrics.Set("validation_score", result.Score)

	return result, nil
}

// applyValidationRule applies a single validation rule to a capability
func (ve *ValidationEngine) applyValidationRule(
	ctx context.Context,
	rule *ValidationRule,
	analysis *CapabilityAnalysis,
) *ValidationResult {
	result := &ValidationResult{
		RuleID:          rule.RuleID,
		Status:          StatusPassed,
		Score:           1.0,
		Issues:          []ValidationIssue{},
		Recommendations: []string{},
		ExecutedAt:      time.Now(),
		Context:         make(map[string]interface{}),
	}

	switch rule.RuleType {
	case RuleTypeStructural:
		ve.validateStructural(rule, analysis, result)
	case RuleTypeBehavioral:
		ve.validateBehavioral(rule, analysis, result)
	case RuleTypePerformance:
		ve.validatePerformance(rule, analysis, result)
	case RuleTypeSecurity:
		ve.validateSecurity(rule, analysis, result)
	case RuleTypeCompliance:
		ve.validateCompliance(rule, analysis, result)
	case RuleTypeIntegration:
		ve.validateIntegration(rule, analysis, result)
	}

	return result
}

// validateStructural validates the structural aspects of a capability
func (ve *ValidationEngine) validateStructural(
	rule *ValidationRule,
	analysis *CapabilityAnalysis,
	result *ValidationResult,
) {
	for _, condition := range rule.Conditions {
		issue := ve.checkCondition(condition, analysis, "structural")
		if issue != nil {
			result.Issues = append(result.Issues, *issue)
			result.Status = ve.getWorstStatus(result.Status, StatusWarning)
			result.Score *= 0.8
		}
	}
}

// validateBehavioral validates the behavioral aspects of a capability
func (ve *ValidationEngine) validateBehavioral(
	rule *ValidationRule,
	analysis *CapabilityAnalysis,
	result *ValidationResult,
) {
	// Check if capability has proper error handling
	if analysis.Quality.ErrorHandling < 0.7 {
		result.Issues = append(result.Issues, ValidationIssue{
			IssueID:    fmt.Sprintf("behavioral_error_handling_%d", time.Now().UnixNano()),
			Severity:   ValidationSeverityWarning,
			Message:    "Low error handling score",
			Field:      "error_handling",
			Expected:   0.7,
			Actual:     analysis.Quality.ErrorHandling,
			Suggestion: "Improve error handling and validation",
		})
		result.Status = StatusWarning
		result.Score *= 0.9
	}
}

// validatePerformance validates the performance aspects of a capability
func (ve *ValidationEngine) validatePerformance(
	rule *ValidationRule,
	analysis *CapabilityAnalysis,
	result *ValidationResult,
) {
	// Check performance score
	if analysis.Quality.PerformanceScore < 0.8 {
		result.Issues = append(result.Issues, ValidationIssue{
			IssueID:    fmt.Sprintf("performance_score_%d", time.Now().UnixNano()),
			Severity:   ValidationSeverityWarning,
			Message:    "Low performance score",
			Field:      "performance_score",
			Expected:   0.8,
			Actual:     analysis.Quality.PerformanceScore,
			Suggestion: "Optimize capability performance",
		})
		result.Status = StatusWarning
		result.Score *= 0.85
	}

	// Check average latency
	if analysis.Usage.AverageLatency > 5*time.Second {
		result.Issues = append(result.Issues, ValidationIssue{
			IssueID:    fmt.Sprintf("high_latency_%d", time.Now().UnixNano()),
			Severity:   ValidationSeverityError,
			Message:    "High average latency",
			Field:      "average_latency",
			Expected:   "< 5s",
			Actual:     analysis.Usage.AverageLatency.String(),
			Suggestion: "Optimize capability to reduce latency",
		})
		result.Status = StatusFailed
		result.Score *= 0.6
	}
}

// validateSecurity validates the security aspects of a capability
func (ve *ValidationEngine) validateSecurity(
	rule *ValidationRule,
	analysis *CapabilityAnalysis,
	result *ValidationResult,
) {
	// Check for authentication parameters
	hasAuth := false
	for _, param := range analysis.Parameters {
		if strings.Contains(strings.ToLower(param.Name), "auth") ||
		   strings.Contains(strings.ToLower(param.Name), "token") ||
		   strings.Contains(strings.ToLower(param.Name), "key") {
			hasAuth = true
			break
		}
	}

	// For certain categories, authentication should be required
	requiresAuth := []SemanticCategory{
		CategoryDataStorage,
		CategoryAuthentication,
		CategoryCommunication,
	}

	for _, category := range requiresAuth {
		if analysis.SemanticCategory == category && !hasAuth {
			result.Issues = append(result.Issues, ValidationIssue{
				IssueID:    fmt.Sprintf("missing_auth_%d", time.Now().UnixNano()),
				Severity:   ValidationSeverityError,
				Message:    "Missing authentication for sensitive capability",
				Field:      "authentication",
				Expected:   "authentication parameter",
				Actual:     "none",
				Suggestion: "Add authentication parameters to secure the capability",
			})
			result.Status = StatusFailed
			result.Score *= 0.5
			break
		}
	}
}

// validateCompliance validates compliance requirements
func (ve *ValidationEngine) validateCompliance(
	rule *ValidationRule,
	analysis *CapabilityAnalysis,
	result *ValidationResult,
) {
	// Check documentation completeness
	if analysis.Quality.DocumentationScore < 0.8 {
		result.Issues = append(result.Issues, ValidationIssue{
			IssueID:    fmt.Sprintf("compliance_docs_%d", time.Now().UnixNano()),
			Severity:   ValidationSeverityWarning,
			Message:    "Insufficient documentation for compliance",
			Field:      "documentation_score",
			Expected:   0.8,
			Actual:     analysis.Quality.DocumentationScore,
			Suggestion: "Improve documentation to meet compliance requirements",
		})
		result.Status = StatusWarning
		result.Score *= 0.9
	}

	// Check parameter coverage
	if analysis.Quality.ParameterCoverage < 1.0 {
		result.Issues = append(result.Issues, ValidationIssue{
			IssueID:    fmt.Sprintf("compliance_params_%d", time.Now().UnixNano()),
			Severity:   ValidationSeverityWarning,
			Message:    "Incomplete parameter documentation",
			Field:      "parameter_coverage",
			Expected:   1.0,
			Actual:     analysis.Quality.ParameterCoverage,
			Suggestion: "Document all parameters for compliance",
		})
		result.Status = StatusWarning
		result.Score *= 0.95
	}
}

// validateIntegration validates integration compatibility
func (ve *ValidationEngine) validateIntegration(
	rule *ValidationRule,
	analysis *CapabilityAnalysis,
	result *ValidationResult,
) {
	// Check for conflicting dependencies
	conflicts := 0
	for _, dep := range analysis.Dependencies {
		for _, conflict := range analysis.Conflicts {
			if dep == conflict {
				conflicts++
			}
		}
	}

	if conflicts > 0 {
		result.Issues = append(result.Issues, ValidationIssue{
			IssueID:    fmt.Sprintf("integration_conflicts_%d", time.Now().UnixNano()),
			Severity:   ValidationSeverityError,
			Message:    "Conflicting dependencies detected",
			Field:      "dependencies",
			Expected:   "no conflicts",
			Actual:     fmt.Sprintf("%d conflicts", conflicts),
			Suggestion: "Resolve dependency conflicts for proper integration",
		})
		result.Status = StatusFailed
		result.Score *= 0.7
	}
}

// Helper methods

func (ve *ValidationEngine) checkCondition(
	condition ValidationCondition,
	analysis *CapabilityAnalysis,
	context string,
) *ValidationIssue {
	value := ve.getFieldValue(condition.Field, analysis)
	if value == nil && condition.Optional {
		return nil
	}

	if !ve.evaluateCondition(condition.Operator, value, condition.Value) {
		return &ValidationIssue{
			IssueID:    fmt.Sprintf("%s_condition_%d", context, time.Now().UnixNano()),
			Severity:   ValidationSeverityWarning,
			Message:    condition.Description,
			Field:      condition.Field,
			Expected:   condition.Value,
			Actual:     value,
			Suggestion: fmt.Sprintf("Ensure %s meets the required condition", condition.Field),
		}
	}

	return nil
}

func (ve *ValidationEngine) getFieldValue(field string, analysis *CapabilityAnalysis) interface{} {
	// Use reflection to get field value from analysis
	v := reflect.ValueOf(analysis).Elem()
	
	// Handle nested field access (e.g., "Quality.OverallScore")
	fieldParts := strings.Split(field, ".")
	for _, part := range fieldParts {
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		v = v.FieldByName(part)
		if !v.IsValid() {
			return nil
		}
	}

	if v.IsValid() && v.CanInterface() {
		return v.Interface()
	}

	return nil
}

func (ve *ValidationEngine) evaluateCondition(
	operator ConditionOperator,
	actual, expected interface{},
) bool {
	switch operator {
	case OperatorEquals:
		return reflect.DeepEqual(actual, expected)
	case OperatorNotEquals:
		return !reflect.DeepEqual(actual, expected)
	case OperatorGreaterThan:
		return ve.compareNumbers(actual, expected) > 0
	case OperatorLessThan:
		return ve.compareNumbers(actual, expected) < 0
	case OperatorGreaterThanEqual:
		return ve.compareNumbers(actual, expected) >= 0
	case OperatorLessThanEqual:
		return ve.compareNumbers(actual, expected) <= 0
	case OperatorExists:
		return actual != nil
	case OperatorNotExists:
		return actual == nil
	default:
		return false
	}
}

func (ve *ValidationEngine) compareNumbers(a, b interface{}) int {
	aVal := ve.toFloat64(a)
	bVal := ve.toFloat64(b)
	
	if aVal > bVal {
		return 1
	} else if aVal < bVal {
		return -1
	}
	return 0
}

func (ve *ValidationEngine) toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case int32:
		return float64(val)
	default:
		return 0
	}
}

func (ve *ValidationEngine) getWorstStatus(current, new ValidationStatus) ValidationStatus {
	statusOrder := map[ValidationStatus]int{
		StatusPassed:  0,
		StatusSkipped: 1,
		StatusWarning: 2,
		StatusError:   3,
		StatusFailed:  4,
	}

	if statusOrder[new] > statusOrder[current] {
		return new
	}
	return current
}

func (ve *ValidationEngine) getApplicableRules(category SemanticCategory) []*ValidationRule {
	var rules []*ValidationRule
	
	ve.mutex.RLock()
	for _, rule := range ve.validationRules {
		if rule.Category == category || rule.Category == "" {
			rules = append(rules, rule)
		}
	}
	ve.mutex.RUnlock()
	
	return rules
}

func (ve *ValidationEngine) generateValidationRecommendations(issues []ValidationIssue) []string {
	var recommendations []string
	
	for _, issue := range issues {
		if issue.Suggestion != "" {
			recommendations = append(recommendations, issue.Suggestion)
		}
	}
	
	return recommendations
}

// Public methods for managing validation rules and results

func (ve *ValidationEngine) AddValidationRule(rule *ValidationRule) {
	ve.mutex.Lock()
	defer ve.mutex.Unlock()
	
	rule.CreatedAt = time.Now()
	rule.LastModified = time.Now()
	ve.validationRules[rule.RuleID] = rule
}

func (ve *ValidationEngine) GetValidationHistory(pluginName, capabilityName string) []*ValidationResult {
	ve.mutex.RLock()
	defer ve.mutex.RUnlock()
	
	key := fmt.Sprintf("%s:%s", pluginName, capabilityName)
	return ve.validationHistory[key]
}

func (ve *ValidationEngine) GetValidationRule(ruleID string) (*ValidationRule, bool) {
	ve.mutex.RLock()
	defer ve.mutex.RUnlock()
	
	rule, exists := ve.validationRules[ruleID]
	return rule, exists
}