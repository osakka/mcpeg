package capabilities

import (
	"time"

	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
)

// CapabilityType represents different types of plugin capabilities
type CapabilityType string

const (
	CapabilityTool     CapabilityType = "tool"
	CapabilityResource CapabilityType = "resource"
	CapabilityPrompt   CapabilityType = "prompt"
	CapabilityService  CapabilityType = "service"
)

// SemanticCategory represents the semantic domain of a capability
type SemanticCategory string

const (
	CategoryDataStorage   SemanticCategory = "data_storage"
	CategoryFileSystem    SemanticCategory = "file_system"
	CategoryVersionControl SemanticCategory = "version_control"
	CategoryCommunication SemanticCategory = "communication"
	CategoryAnalysis      SemanticCategory = "analysis"
	CategoryTransformation SemanticCategory = "transformation"
	CategoryMonitoring    SemanticCategory = "monitoring"
	CategoryAuthentication SemanticCategory = "authentication"
)

// CapabilityAnalysis represents the intelligent analysis of a plugin capability
type CapabilityAnalysis struct {
	PluginName       string                 `json:"plugin_name"`
	CapabilityName   string                 `json:"capability_name"`
	CapabilityType   CapabilityType         `json:"capability_type"`
	SemanticCategory SemanticCategory       `json:"semantic_category"`
	Description      string                 `json:"description"`
	Parameters       []ParameterAnalysis    `json:"parameters"`
	Dependencies     []string               `json:"dependencies"`
	Provides         []string               `json:"provides"`
	Conflicts        []string               `json:"conflicts"`
	Quality          QualityMetrics         `json:"quality"`
	Usage            UsageMetrics           `json:"usage"`
	Relationships    []CapabilityRelation   `json:"relationships"`
	Metadata         map[string]interface{} `json:"metadata"`
	AnalyzedAt       time.Time              `json:"analyzed_at"`
}

// ParameterAnalysis provides semantic analysis of capability parameters
type ParameterAnalysis struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Required     bool                   `json:"required"`
	Description  string                 `json:"description"`
	SemanticRole string                 `json:"semantic_role"` // input, output, configuration, context
	Constraints  map[string]interface{} `json:"constraints"`
	Examples     []interface{}          `json:"examples"`
}

// QualityMetrics represents the quality assessment of a capability
type QualityMetrics struct {
	DocumentationScore float64 `json:"documentation_score"`
	ParameterCoverage  float64 `json:"parameter_coverage"`
	ErrorHandling      float64 `json:"error_handling"`
	TestCoverage       float64 `json:"test_coverage"`
	PerformanceScore   float64 `json:"performance_score"`
	ReliabilityScore   float64 `json:"reliability_score"`
	OverallScore       float64 `json:"overall_score"`
}

// UsageMetrics tracks how capabilities are being used
type UsageMetrics struct {
	CallCount        int64         `json:"call_count"`
	SuccessRate      float64       `json:"success_rate"`
	AverageLatency   time.Duration `json:"average_latency"`
	ErrorRate        float64       `json:"error_rate"`
	LastUsed         time.Time     `json:"last_used"`
	PopularityScore  float64       `json:"popularity_score"`
	EfficiencyScore  float64       `json:"efficiency_score"`
}

// CapabilityRelation represents relationships between capabilities
type CapabilityRelation struct {
	Type        RelationType `json:"type"`
	Target      string       `json:"target"`        // plugin:capability format
	Strength    float64      `json:"strength"`      // 0.0 to 1.0
	Description string       `json:"description"`
	Confidence  float64      `json:"confidence"`    // AI confidence in this relationship
}

// RelationType defines types of relationships between capabilities
type RelationType string

const (
	RelationComplementary RelationType = "complementary" // Work well together
	RelationAlternative   RelationType = "alternative"   // Can substitute for each other
	RelationDependency    RelationType = "dependency"    // One requires the other
	RelationConflicting   RelationType = "conflicting"   // Cannot be used together
	RelationPipeline      RelationType = "pipeline"      // Output of one feeds input of another
	RelationComposition   RelationType = "composition"   // Combined they create new functionality
)

// AnalysisEngine performs intelligent analysis of plugin capabilities
type AnalysisEngine struct {
	logger  logging.Logger
	metrics metrics.Metrics

	// Analysis state
	analyses map[string]*CapabilityAnalysis // key: plugin:capability
	relations map[string][]CapabilityRelation
	categories map[SemanticCategory][]*CapabilityAnalysis

	// Configuration
	config AnalysisConfig
}

// AnalysisConfig configures the capability analysis engine
type AnalysisConfig struct {
	EnableSemanticAnalysis bool          `yaml:"enable_semantic_analysis"`
	EnableUsageTracking    bool          `yaml:"enable_usage_tracking"`
	EnableQualityMetrics   bool          `yaml:"enable_quality_metrics"`
	AnalysisInterval       time.Duration `yaml:"analysis_interval"`
	RelationThreshold      float64       `yaml:"relation_threshold"`
	CacheTimeout           time.Duration `yaml:"cache_timeout"`
}

// DiscoveryResult represents the result of plugin discovery
type DiscoveryResult struct {
	PluginName    string                `json:"plugin_name"`
	Capabilities  []*CapabilityAnalysis `json:"capabilities"`
	Dependencies  []string              `json:"dependencies"`
	Conflicts     []string              `json:"conflicts"`
	Recommendations []Recommendation    `json:"recommendations"`
	DiscoveredAt  time.Time             `json:"discovered_at"`
}

// Recommendation suggests actions based on capability analysis
type Recommendation struct {
	Type        RecommendationType `json:"type"`
	Priority    Priority           `json:"priority"`
	Description string             `json:"description"`
	Actions     []RecommendedAction `json:"actions"`
	Confidence  float64            `json:"confidence"`
}

// RecommendationType defines types of recommendations
type RecommendationType string

const (
	RecommendationOptimization RecommendationType = "optimization"
	RecommendationIntegration  RecommendationType = "integration"
	RecommendationConflict     RecommendationType = "conflict"
	RecommendationDependency   RecommendationType = "dependency"
	RecommendationUpgrade      RecommendationType = "upgrade"
)

// Priority defines recommendation priorities
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// RecommendedAction defines specific actions to take
type RecommendedAction struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Expected    string                 `json:"expected_outcome"`
}