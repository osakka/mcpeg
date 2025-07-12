package capabilities

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
)

// AggregationEngine handles capability aggregation and conflict resolution
type AggregationEngine struct {
	logger            logging.Logger
	metrics           metrics.Metrics
	discoveryEngine   *DiscoveryEngine
	analysisEngine    *AnalysisEngine

	// Aggregation state
	aggregatedCapabilities map[SemanticCategory]*CategoryAggregation
	globalCapabilityMap    map[string]*AggregatedCapability
	conflictResolutions    map[string]*ConflictResolution
	
	// Synchronization
	mutex sync.RWMutex

	// Configuration
	config AggregationConfig
}

// AggregationConfig configures the aggregation engine
type AggregationConfig struct {
	EnableAggregation      bool          `yaml:"enable_aggregation"`
	ConflictResolution     bool          `yaml:"conflict_resolution"`
	AutoConflictResolution bool          `yaml:"auto_conflict_resolution"`
	AggregationInterval    time.Duration `yaml:"aggregation_interval"`
	ConflictThreshold      float64       `yaml:"conflict_threshold"`
	SimilarityThreshold    float64       `yaml:"similarity_threshold"`
}

// CategoryAggregation represents aggregated capabilities for a semantic category
type CategoryAggregation struct {
	Category      SemanticCategory        `json:"category"`
	Capabilities  []*AggregatedCapability `json:"capabilities"`
	TotalCount    int                     `json:"total_count"`
	QualityScore  float64                 `json:"quality_score"`
	Coverage      float64                 `json:"coverage"`
	Conflicts     []CapabilityConflict    `json:"conflicts"`
	Recommendations []CategoryRecommendation `json:"recommendations"`
	LastUpdated   time.Time              `json:"last_updated"`
}

// AggregatedCapability represents a capability that may be provided by multiple plugins
type AggregatedCapability struct {
	Name           string                    `json:"name"`
	Category       SemanticCategory          `json:"category"`
	Description    string                    `json:"description"`
	Providers      []*CapabilityProvider     `json:"providers"`
	BestProvider   *CapabilityProvider       `json:"best_provider"`
	Alternatives   []*CapabilityProvider     `json:"alternatives"`
	QualityScore   float64                   `json:"quality_score"`
	Conflicts      []CapabilityConflict      `json:"conflicts"`
	Relationships  []AggregatedRelation      `json:"relationships"`
	Usage          AggregatedUsageMetrics    `json:"usage"`
	LastEvaluated  time.Time                 `json:"last_evaluated"`
}

// CapabilityProvider represents a plugin that provides a specific capability
type CapabilityProvider struct {
	PluginName      string                 `json:"plugin_name"`
	CapabilityName  string                 `json:"capability_name"`
	QualityScore    float64                `json:"quality_score"`
	UsageMetrics    UsageMetrics           `json:"usage_metrics"`
	ProviderRank    int                    `json:"provider_rank"`
	Advantages      []string               `json:"advantages"`
	Disadvantages   []string               `json:"disadvantages"`
	Compatibility   map[string]float64     `json:"compatibility"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// CapabilityConflict represents a conflict between capabilities
type CapabilityConflict struct {
	ConflictID      string                    `json:"conflict_id"`
	Type            ConflictType              `json:"type"`
	Severity        ConflictSeverity          `json:"severity"`
	Description     string                    `json:"description"`
	Participants    []*CapabilityProvider     `json:"participants"`
	Resolution      *ConflictResolution       `json:"resolution,omitempty"`
	Impact          ConflictImpact            `json:"impact"`
	DetectedAt      time.Time                 `json:"detected_at"`
	AutoResolvable  bool                      `json:"auto_resolvable"`
}

// ConflictType defines types of capability conflicts
type ConflictType string

const (
	ConflictFunctional   ConflictType = "functional"   // Same functionality, different implementations
	ConflictResource     ConflictType = "resource"     // Competing for same resources
	ConflictSemantic     ConflictType = "semantic"     // Different meanings for same operation
	ConflictTechnical    ConflictType = "technical"    // Technical incompatibilities
	ConflictSecurity     ConflictType = "security"     // Security policy conflicts
	ConflictPerformance  ConflictType = "performance"  // Performance trade-offs
)

// ConflictSeverity defines conflict severity levels
type ConflictSeverity string

const (
	SeverityLow      ConflictSeverity = "low"
	SeverityMedium   ConflictSeverity = "medium"
	SeverityHigh     ConflictSeverity = "high"
	SeverityCritical ConflictSeverity = "critical"
)

// ConflictImpact describes the impact of a conflict
type ConflictImpact struct {
	PerformanceImpact float64 `json:"performance_impact"`
	ReliabilityImpact float64 `json:"reliability_impact"`
	SecurityImpact    float64 `json:"security_impact"`
	UsabilityImpact   float64 `json:"usability_impact"`
	OverallImpact     float64 `json:"overall_impact"`
}

// ConflictResolution defines how a conflict should be resolved
type ConflictResolution struct {
	ResolutionID     string                    `json:"resolution_id"`
	Strategy         ResolutionStrategy        `json:"strategy"`
	Description      string                    `json:"description"`
	RecommendedProvider *CapabilityProvider    `json:"recommended_provider,omitempty"`
	Configuration    map[string]interface{}    `json:"configuration"`
	ExpectedOutcome  string                    `json:"expected_outcome"`
	Confidence       float64                   `json:"confidence"`
	CreatedAt        time.Time                 `json:"created_at"`
	Applied          bool                      `json:"applied"`
	AppliedAt        *time.Time                `json:"applied_at,omitempty"`
}

// ResolutionStrategy defines strategies for conflict resolution
type ResolutionStrategy string

const (
	StrategyPrefer       ResolutionStrategy = "prefer"       // Prefer one provider over others
	StrategyRoundRobin   ResolutionStrategy = "round_robin"  // Rotate between providers
	StrategyLoadBalance  ResolutionStrategy = "load_balance" // Balance based on load
	StrategyQuality      ResolutionStrategy = "quality"      // Use highest quality provider
	StrategyDisable      ResolutionStrategy = "disable"      // Disable conflicting providers
	StrategyIsolate      ResolutionStrategy = "isolate"      // Use different contexts
	StrategyMerge        ResolutionStrategy = "merge"        // Merge capabilities
)

// AggregatedRelation represents relationships in aggregated view
type AggregatedRelation struct {
	Type          RelationType `json:"type"`
	Target        string       `json:"target"`
	Strength      float64      `json:"strength"`
	Bidirectional bool         `json:"bidirectional"`
	Context       string       `json:"context"`
}

// AggregatedUsageMetrics combines usage metrics across providers
type AggregatedUsageMetrics struct {
	TotalCalls       int64         `json:"total_calls"`
	AverageSuccessRate float64     `json:"average_success_rate"`
	AverageLatency   time.Duration `json:"average_latency"`
	AverageErrorRate float64       `json:"average_error_rate"`
	PopularityScore  float64       `json:"popularity_score"`
	TrendDirection   string        `json:"trend_direction"`
}

// CategoryRecommendation provides recommendations for capability categories
type CategoryRecommendation struct {
	Type         RecommendationType `json:"type"`
	Priority     Priority           `json:"priority"`
	Category     SemanticCategory   `json:"category"`
	Description  string             `json:"description"`
	Impact       string             `json:"impact"`
	Effort       string             `json:"effort"`
	Actions      []RecommendedAction `json:"actions"`
}

// NewAggregationEngine creates a new capability aggregation engine
func NewAggregationEngine(
	logger logging.Logger,
	metrics metrics.Metrics,
	discoveryEngine *DiscoveryEngine,
	analysisEngine *AnalysisEngine,
	config AggregationConfig,
) *AggregationEngine {
	return &AggregationEngine{
		logger:                 logger.WithComponent("capability-aggregation"),
		metrics:                metrics,
		discoveryEngine:        discoveryEngine,
		analysisEngine:         analysisEngine,
		aggregatedCapabilities: make(map[SemanticCategory]*CategoryAggregation),
		globalCapabilityMap:    make(map[string]*AggregatedCapability),
		conflictResolutions:    make(map[string]*ConflictResolution),
		config:                 config,
	}
}

// AggregateCapabilities performs comprehensive capability aggregation
func (ae *AggregationEngine) AggregateCapabilities(ctx context.Context) error {
	startTime := time.Now()
	defer func() {
		ae.metrics.Observe("capability_aggregation_duration_ms", float64(time.Since(startTime).Nanoseconds()/1e6))
	}()

	ae.logger.WithContext(ctx).Info("capability_aggregation_started")

	// Get all discoveries
	discoveries := ae.discoveryEngine.GetAllDiscoveries()
	
	// Clear existing aggregations
	ae.mutex.Lock()
	ae.aggregatedCapabilities = make(map[SemanticCategory]*CategoryAggregation)
	ae.globalCapabilityMap = make(map[string]*AggregatedCapability)
	ae.mutex.Unlock()

	// Group capabilities by category and name
	categoryGroups := make(map[SemanticCategory]map[string][]*CapabilityAnalysis)
	
	for _, discovery := range discoveries {
		for _, capability := range discovery.Capabilities {
			category := capability.SemanticCategory
			name := capability.CapabilityName

			if categoryGroups[category] == nil {
				categoryGroups[category] = make(map[string][]*CapabilityAnalysis)
			}
			
			categoryGroups[category][name] = append(categoryGroups[category][name], capability)
		}
	}

	// Process each category
	for category, capabilities := range categoryGroups {
		aggregation := ae.aggregateCategory(ctx, category, capabilities)
		
		ae.mutex.Lock()
		ae.aggregatedCapabilities[category] = aggregation
		ae.mutex.Unlock()
	}

	// Detect and resolve conflicts
	if ae.config.ConflictResolution {
		conflicts := ae.detectConflicts(ctx)
		if ae.config.AutoConflictResolution {
			ae.resolveConflicts(ctx, conflicts)
		}
	}

	ae.logger.WithContext(ctx).Info("capability_aggregation_completed",
		"categories", len(ae.aggregatedCapabilities),
		"total_capabilities", len(ae.globalCapabilityMap),
	)

	ae.metrics.Inc("capability_aggregations_total")
	ae.metrics.Set("aggregated_categories_count", float64(len(ae.aggregatedCapabilities)))

	return nil
}

// aggregateCategory aggregates capabilities within a semantic category
func (ae *AggregationEngine) aggregateCategory(
	ctx context.Context,
	category SemanticCategory,
	capabilityGroups map[string][]*CapabilityAnalysis,
) *CategoryAggregation {
	
	aggregation := &CategoryAggregation{
		Category:        category,
		Capabilities:    []*AggregatedCapability{},
		TotalCount:      0,
		QualityScore:    0.0,
		Coverage:        0.0,
		Conflicts:       []CapabilityConflict{},
		Recommendations: []CategoryRecommendation{},
		LastUpdated:     time.Now(),
	}

	qualitySum := 0.0
	
	// Process each capability group (same name, different providers)
	for capabilityName, capabilities := range capabilityGroups {
		aggregated := ae.aggregateCapability(ctx, capabilityName, category, capabilities)
		aggregation.Capabilities = append(aggregation.Capabilities, aggregated)
		aggregation.TotalCount += len(capabilities)
		qualitySum += aggregated.QualityScore

		// Add to global map
		ae.globalCapabilityMap[capabilityName] = aggregated
	}

	// Calculate overall metrics
	if len(aggregation.Capabilities) > 0 {
		aggregation.QualityScore = qualitySum / float64(len(aggregation.Capabilities))
	}

	// Calculate coverage based on expected capabilities for this category
	expectedCapabilities := ae.getExpectedCapabilities(category)
	if len(expectedCapabilities) > 0 {
		providedCount := 0
		for _, expected := range expectedCapabilities {
			if _, exists := ae.globalCapabilityMap[expected]; exists {
				providedCount++
			}
		}
		aggregation.Coverage = float64(providedCount) / float64(len(expectedCapabilities))
	}

	// Generate category-specific recommendations
	aggregation.Recommendations = ae.generateCategoryRecommendations(ctx, aggregation)

	return aggregation
}

// aggregateCapability aggregates multiple providers of the same capability
func (ae *AggregationEngine) aggregateCapability(
	ctx context.Context,
	capabilityName string,
	category SemanticCategory,
	capabilities []*CapabilityAnalysis,
) *AggregatedCapability {
	
	aggregated := &AggregatedCapability{
		Name:          capabilityName,
		Category:      category,
		Providers:     []*CapabilityProvider{},
		Alternatives:  []*CapabilityProvider{},
		Conflicts:     []CapabilityConflict{},
		Relationships: []AggregatedRelation{},
		LastEvaluated: time.Now(),
	}

	qualitySum := 0.0
	totalCalls := int64(0)
	successRateSum := 0.0
	latencySum := time.Duration(0)
	errorRateSum := 0.0

	// Process each provider
	for i, capability := range capabilities {
		provider := &CapabilityProvider{
			PluginName:     capability.PluginName,
			CapabilityName: capability.CapabilityName,
			QualityScore:   capability.Quality.OverallScore,
			UsageMetrics:   capability.Usage,
			ProviderRank:   i + 1,
			Advantages:     []string{},
			Disadvantages:  []string{},
			Compatibility:  make(map[string]float64),
			Metadata:       capability.Metadata,
		}

		// Analyze provider advantages/disadvantages
		ae.analyzeProviderCharacteristics(provider, capability)

		aggregated.Providers = append(aggregated.Providers, provider)
		
		// Accumulate metrics
		qualitySum += capability.Quality.OverallScore
		totalCalls += capability.Usage.CallCount
		successRateSum += capability.Usage.SuccessRate
		latencySum += capability.Usage.AverageLatency
		errorRateSum += capability.Usage.ErrorRate

		// Use first capability's description as base
		if aggregated.Description == "" {
			aggregated.Description = capability.Description
		}
	}

	// Calculate aggregated metrics
	providerCount := len(capabilities)
	if providerCount > 0 {
		aggregated.QualityScore = qualitySum / float64(providerCount)
		aggregated.Usage = AggregatedUsageMetrics{
			TotalCalls:         totalCalls,
			AverageSuccessRate: successRateSum / float64(providerCount),
			AverageLatency:     latencySum / time.Duration(providerCount),
			AverageErrorRate:   errorRateSum / float64(providerCount),
			PopularityScore:    float64(totalCalls) / 1000.0, // Normalize
			TrendDirection:     "stable",
		}
	}

	// Rank providers by quality and performance
	ae.rankProviders(aggregated.Providers)
	
	// Set best provider and alternatives
	if len(aggregated.Providers) > 0 {
		aggregated.BestProvider = aggregated.Providers[0]
		if len(aggregated.Providers) > 1 {
			aggregated.Alternatives = aggregated.Providers[1:]
		}
	}

	// Detect conflicts between providers
	aggregated.Conflicts = ae.detectProviderConflicts(ctx, aggregated.Providers)

	return aggregated
}

// analyzeProviderCharacteristics identifies advantages and disadvantages of providers
func (ae *AggregationEngine) analyzeProviderCharacteristics(
	provider *CapabilityProvider,
	capability *CapabilityAnalysis,
) {
	// Analyze based on quality metrics
	if capability.Quality.DocumentationScore > 0.8 {
		provider.Advantages = append(provider.Advantages, "Well documented")
	} else if capability.Quality.DocumentationScore < 0.5 {
		provider.Disadvantages = append(provider.Disadvantages, "Poor documentation")
	}

	if capability.Quality.PerformanceScore > 0.8 {
		provider.Advantages = append(provider.Advantages, "High performance")
	} else if capability.Quality.PerformanceScore < 0.5 {
		provider.Disadvantages = append(provider.Disadvantages, "Performance concerns")
	}

	if capability.Quality.ReliabilityScore > 0.9 {
		provider.Advantages = append(provider.Advantages, "High reliability")
	} else if capability.Quality.ReliabilityScore < 0.6 {
		provider.Disadvantages = append(provider.Disadvantages, "Reliability issues")
	}

	// Analyze based on usage metrics
	if capability.Usage.SuccessRate > 0.95 {
		provider.Advantages = append(provider.Advantages, "High success rate")
	} else if capability.Usage.SuccessRate < 0.8 {
		provider.Disadvantages = append(provider.Disadvantages, "Low success rate")
	}

	if capability.Usage.AverageLatency < 100*time.Millisecond {
		provider.Advantages = append(provider.Advantages, "Low latency")
	} else if capability.Usage.AverageLatency > 1*time.Second {
		provider.Disadvantages = append(provider.Disadvantages, "High latency")
	}
}

// rankProviders sorts providers by their overall suitability score
func (ae *AggregationEngine) rankProviders(providers []*CapabilityProvider) {
	sort.Slice(providers, func(i, j int) bool {
		scoreI := ae.calculateProviderScore(providers[i])
		scoreJ := ae.calculateProviderScore(providers[j])
		return scoreI > scoreJ
	})

	// Update ranks
	for i, provider := range providers {
		provider.ProviderRank = i + 1
	}
}

// calculateProviderScore computes an overall score for a provider
func (ae *AggregationEngine) calculateProviderScore(provider *CapabilityProvider) float64 {
	// Weighted combination of quality and usage metrics
	qualityWeight := 0.4
	successRateWeight := 0.3
	latencyWeight := 0.2
	popularityWeight := 0.1

	// Normalize latency (lower is better)
	latencyScore := 1.0
	if provider.UsageMetrics.AverageLatency > 0 {
		latencyScore = 1.0 / (1.0 + float64(provider.UsageMetrics.AverageLatency.Milliseconds())/1000.0)
	}

	return provider.QualityScore*qualityWeight +
		provider.UsageMetrics.SuccessRate*successRateWeight +
		latencyScore*latencyWeight +
		provider.UsageMetrics.PopularityScore*popularityWeight
}

// detectConflicts identifies conflicts across all aggregated capabilities
func (ae *AggregationEngine) detectConflicts(ctx context.Context) []CapabilityConflict {
	var conflicts []CapabilityConflict

	ae.mutex.RLock()
	for _, aggregated := range ae.globalCapabilityMap {
		conflicts = append(conflicts, aggregated.Conflicts...)
	}
	ae.mutex.RUnlock()

	return conflicts
}

// detectProviderConflicts identifies conflicts between capability providers
func (ae *AggregationEngine) detectProviderConflicts(
	ctx context.Context,
	providers []*CapabilityProvider,
) []CapabilityConflict {
	var conflicts []CapabilityConflict

	// Check for functional conflicts (multiple providers of same capability)
	if len(providers) > 1 {
		conflictID := fmt.Sprintf("functional_conflict_%d", time.Now().Unix())
		
		conflict := CapabilityConflict{
			ConflictID:   conflictID,
			Type:         ConflictFunctional,
			Severity:     ae.assessConflictSeverity(providers),
			Description:  fmt.Sprintf("Multiple providers available for capability"),
			Participants: providers,
			Impact:       ae.calculateConflictImpact(providers),
			DetectedAt:   time.Now(),
			AutoResolvable: true,
		}

		conflicts = append(conflicts, conflict)
	}

	return conflicts
}

// assessConflictSeverity determines the severity of a conflict
func (ae *AggregationEngine) assessConflictSeverity(providers []*CapabilityProvider) ConflictSeverity {
	if len(providers) <= 2 {
		return SeverityLow
	}
	if len(providers) <= 4 {
		return SeverityMedium
	}
	return SeverityHigh
}

// calculateConflictImpact estimates the impact of a conflict
func (ae *AggregationEngine) calculateConflictImpact(providers []*CapabilityProvider) ConflictImpact {
	// Calculate variance in quality scores
	qualityVariance := ae.calculateQualityVariance(providers)
	
	return ConflictImpact{
		PerformanceImpact: qualityVariance * 0.3,
		ReliabilityImpact: qualityVariance * 0.4,
		SecurityImpact:    qualityVariance * 0.2,
		UsabilityImpact:   qualityVariance * 0.5,
		OverallImpact:     qualityVariance * 0.35,
	}
}

// calculateQualityVariance computes variance in provider quality scores
func (ae *AggregationEngine) calculateQualityVariance(providers []*CapabilityProvider) float64 {
	if len(providers) <= 1 {
		return 0.0
	}

	sum := 0.0
	for _, provider := range providers {
		sum += provider.QualityScore
	}
	mean := sum / float64(len(providers))

	varianceSum := 0.0
	for _, provider := range providers {
		diff := provider.QualityScore - mean
		varianceSum += diff * diff
	}

	return varianceSum / float64(len(providers))
}

// resolveConflicts applies automatic conflict resolution strategies
func (ae *AggregationEngine) resolveConflicts(ctx context.Context, conflicts []CapabilityConflict) {
	for _, conflict := range conflicts {
		resolution := ae.createResolution(ctx, conflict)
		if resolution != nil {
			ae.mutex.Lock()
			ae.conflictResolutions[conflict.ConflictID] = resolution
			ae.mutex.Unlock()

			ae.logger.WithContext(ctx).Info("conflict_resolved",
				"conflict_id", conflict.ConflictID,
				"strategy", resolution.Strategy,
				"confidence", resolution.Confidence,
			)

			ae.metrics.Inc("conflicts_resolved_total")
		}
	}
}

// createResolution creates a resolution strategy for a conflict
func (ae *AggregationEngine) createResolution(ctx context.Context, conflict CapabilityConflict) *ConflictResolution {
	switch conflict.Type {
	case ConflictFunctional:
		// For functional conflicts, prefer the highest quality provider
		return ae.createQualityBasedResolution(conflict)
		
	case ConflictResource:
		// For resource conflicts, use load balancing
		return ae.createLoadBalanceResolution(conflict)
		
	default:
		return nil
	}
}

// createQualityBasedResolution creates a resolution that prefers the highest quality provider
func (ae *AggregationEngine) createQualityBasedResolution(conflict CapabilityConflict) *ConflictResolution {
	if len(conflict.Participants) == 0 {
		return nil
	}

	// Find the highest quality provider
	var bestProvider *CapabilityProvider
	bestScore := -1.0

	for _, provider := range conflict.Participants {
		score := ae.calculateProviderScore(provider)
		if score > bestScore {
			bestScore = score
			bestProvider = provider
		}
	}

	return &ConflictResolution{
		ResolutionID:        fmt.Sprintf("quality_resolution_%d", time.Now().Unix()),
		Strategy:            StrategyQuality,
		Description:         fmt.Sprintf("Prefer highest quality provider: %s", bestProvider.PluginName),
		RecommendedProvider: bestProvider,
		Configuration:       map[string]interface{}{"preferred_provider": bestProvider.PluginName},
		ExpectedOutcome:     "Consistent high-quality capability provision",
		Confidence:          0.8,
		CreatedAt:           time.Now(),
		Applied:             false,
	}
}

// createLoadBalanceResolution creates a load balancing resolution
func (ae *AggregationEngine) createLoadBalanceResolution(conflict CapabilityConflict) *ConflictResolution {
	return &ConflictResolution{
		ResolutionID:    fmt.Sprintf("loadbalance_resolution_%d", time.Now().Unix()),
		Strategy:        StrategyLoadBalance,
		Description:     "Distribute load across all capable providers",
		Configuration:   map[string]interface{}{"strategy": "round_robin"},
		ExpectedOutcome: "Optimized resource utilization",
		Confidence:      0.7,
		CreatedAt:       time.Now(),
		Applied:         false,
	}
}

// Helper methods

func (ae *AggregationEngine) getExpectedCapabilities(category SemanticCategory) []string {
	// Define expected capabilities for each category
	expectedCapabilities := map[SemanticCategory][]string{
		CategoryDataStorage:   {"store", "retrieve", "delete", "list", "update"},
		CategoryFileSystem:    {"read", "write", "create", "delete", "list", "move", "copy"},
		CategoryVersionControl: {"commit", "push", "pull", "branch", "merge", "diff", "log"},
		CategoryCommunication: {"send", "receive", "notify", "subscribe"},
		CategoryAnalysis:      {"validate", "analyze", "parse", "check"},
		CategoryTransformation: {"convert", "transform", "encode", "decode"},
		CategoryMonitoring:    {"monitor", "alert", "measure", "track"},
		CategoryAuthentication: {"authenticate", "authorize", "validate"},
	}

	return expectedCapabilities[category]
}

func (ae *AggregationEngine) generateCategoryRecommendations(
	ctx context.Context,
	aggregation *CategoryAggregation,
) []CategoryRecommendation {
	var recommendations []CategoryRecommendation

	// Check coverage
	if aggregation.Coverage < 0.8 {
		recommendations = append(recommendations, CategoryRecommendation{
			Type:        RecommendationIntegration,
			Priority:    PriorityMedium,
			Category:    aggregation.Category,
			Description: fmt.Sprintf("Low capability coverage (%.1f%%) in %s category", aggregation.Coverage*100, aggregation.Category),
			Impact:      "Limited functionality",
			Effort:      "Medium",
			Actions: []RecommendedAction{
				{
					Type:        "add_capabilities",
					Description: "Add missing capabilities to improve coverage",
					Expected:    "Comprehensive category support",
				},
			},
		})
	}

	// Check quality
	if aggregation.QualityScore < 0.7 {
		recommendations = append(recommendations, CategoryRecommendation{
			Type:        RecommendationOptimization,
			Priority:    PriorityHigh,
			Category:    aggregation.Category,
			Description: fmt.Sprintf("Low average quality score (%.2f) in %s category", aggregation.QualityScore, aggregation.Category),
			Impact:      "Poor user experience",
			Effort:      "High",
			Actions: []RecommendedAction{
				{
					Type:        "improve_quality",
					Description: "Improve documentation, testing, and error handling",
					Expected:    "Higher quality capabilities",
				},
			},
		})
	}

	return recommendations
}

// Public methods for accessing aggregated data

func (ae *AggregationEngine) GetCategoryAggregation(category SemanticCategory) (*CategoryAggregation, bool) {
	ae.mutex.RLock()
	defer ae.mutex.RUnlock()
	
	aggregation, exists := ae.aggregatedCapabilities[category]
	return aggregation, exists
}

func (ae *AggregationEngine) GetAggregatedCapability(name string) (*AggregatedCapability, bool) {
	ae.mutex.RLock()
	defer ae.mutex.RUnlock()
	
	capability, exists := ae.globalCapabilityMap[name]
	return capability, exists
}

func (ae *AggregationEngine) GetConflictResolution(conflictID string) (*ConflictResolution, bool) {
	ae.mutex.RLock()
	defer ae.mutex.RUnlock()
	
	resolution, exists := ae.conflictResolutions[conflictID]
	return resolution, exists
}

func (ae *AggregationEngine) GetAllAggregations() map[SemanticCategory]*CategoryAggregation {
	ae.mutex.RLock()
	defer ae.mutex.RUnlock()
	
	// Return copy to prevent external modification
	aggregations := make(map[SemanticCategory]*CategoryAggregation)
	for k, v := range ae.aggregatedCapabilities {
		aggregations[k] = v
	}
	
	return aggregations
}