package capabilities

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/osakka/mcpeg/internal/registry"
	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
	"github.com/osakka/mcpeg/pkg/plugins"
)

// DiscoveryEngine performs intelligent plugin discovery with dependency resolution
type DiscoveryEngine struct {
	logger          logging.Logger
	metrics         metrics.Metrics
	analysisEngine  *AnalysisEngine
	pluginManager   *plugins.PluginManager
	serviceRegistry *registry.ServiceRegistry

	// Discovery state
	discoveries map[string]*DiscoveryResult
	dependencyGraph map[string][]string
	conflictMatrix  map[string][]string
	
	// Synchronization
	mutex sync.RWMutex

	// Configuration
	config DiscoveryConfig
}

// DiscoveryConfig configures the discovery engine
type DiscoveryConfig struct {
	AutoDiscovery          bool          `yaml:"auto_discovery"`
	DiscoveryInterval      time.Duration `yaml:"discovery_interval"`
	DependencyResolution   bool          `yaml:"dependency_resolution"`
	ConflictDetection      bool          `yaml:"conflict_detection"`
	RecommendationEngine   bool          `yaml:"recommendation_engine"`
	MaxDiscoveryDepth      int           `yaml:"max_discovery_depth"`
	ConcurrentAnalysis     int           `yaml:"concurrent_analysis"`
	ReanalysisThreshold    time.Duration `yaml:"reanalysis_threshold"`
}

// NewDiscoveryEngine creates a new plugin discovery engine
func NewDiscoveryEngine(
	logger logging.Logger,
	metrics metrics.Metrics,
	analysisEngine *AnalysisEngine,
	pluginManager *plugins.PluginManager,
	serviceRegistry *registry.ServiceRegistry,
	config DiscoveryConfig,
) *DiscoveryEngine {
	return &DiscoveryEngine{
		logger:          logger.WithComponent("plugin-discovery"),
		metrics:         metrics,
		analysisEngine:  analysisEngine,
		pluginManager:   pluginManager,
		serviceRegistry: serviceRegistry,
		discoveries:     make(map[string]*DiscoveryResult),
		dependencyGraph: make(map[string][]string),
		conflictMatrix:  make(map[string][]string),
		config:          config,
	}
}

// DiscoverPlugin performs comprehensive discovery and analysis of a plugin
func (de *DiscoveryEngine) DiscoverPlugin(ctx context.Context, pluginName string) (*DiscoveryResult, error) {
	startTime := time.Now()
	defer func() {
		de.metrics.Observe("plugin_discovery_duration_ms", float64(time.Since(startTime).Nanoseconds()/1e6))
	}()

	de.logger.WithContext(ctx).Info("plugin_discovery_started", "plugin", pluginName)

	// Get plugin from plugin manager
	plugin, exists := de.pluginManager.GetPlugin(pluginName)
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", pluginName)
	}

	result := &DiscoveryResult{
		PluginName:      pluginName,
		Capabilities:    []*CapabilityAnalysis{},
		Dependencies:    []string{},
		Conflicts:       []string{},
		Recommendations: []Recommendation{},
		DiscoveredAt:    time.Now(),
	}

	// Analyze all capabilities concurrently
	capabilityChan := make(chan *CapabilityAnalysis, 100)
	errorChan := make(chan error, 100)
	
	var wg sync.WaitGroup
	
	// Analyze tools
	tools := plugin.GetTools()
	for _, tool := range tools {
		wg.Add(1)
		go func(tool registry.ToolDefinition) {
			defer wg.Done()
			analysis, err := de.analysisEngine.AnalyzeCapability(ctx, pluginName, tool)
			if err != nil {
				errorChan <- fmt.Errorf("tool analysis failed for %s: %w", tool.Name, err)
				return
			}
			capabilityChan <- analysis
		}(tool)
	}

	// Analyze resources
	resources := plugin.GetResources()
	for _, resource := range resources {
		wg.Add(1)
		go func(resource registry.ResourceDefinition) {
			defer wg.Done()
			analysis, err := de.analysisEngine.AnalyzeCapability(ctx, pluginName, resource)
			if err != nil {
				errorChan <- fmt.Errorf("resource analysis failed for %s: %w", resource.Name, err)
				return
			}
			capabilityChan <- analysis
		}(resource)
	}

	// Analyze prompts
	prompts := plugin.GetPrompts()
	for _, prompt := range prompts {
		wg.Add(1)
		go func(prompt registry.PromptDefinition) {
			defer wg.Done()
			analysis, err := de.analysisEngine.AnalyzeCapability(ctx, pluginName, prompt)
			if err != nil {
				errorChan <- fmt.Errorf("prompt analysis failed for %s: %w", prompt.Name, err)
				return
			}
			capabilityChan <- analysis
		}(prompt)
	}

	// Wait for all analyses to complete
	go func() {
		wg.Wait()
		close(capabilityChan)
		close(errorChan)
	}()

	// Collect results
	var analysisErrors []error
	for {
		select {
		case analysis, ok := <-capabilityChan:
			if !ok {
				capabilityChan = nil
				break
			}
			result.Capabilities = append(result.Capabilities, analysis)

		case err, ok := <-errorChan:
			if !ok {
				errorChan = nil
				break
			}
			analysisErrors = append(analysisErrors, err)
		}
		
		if capabilityChan == nil && errorChan == nil {
			break
		}
	}

	// Log any analysis errors but continue
	for _, err := range analysisErrors {
		de.logger.WithContext(ctx).Warn("capability_analysis_error", "error", err.Error())
	}

	// Perform dependency resolution
	if de.config.DependencyResolution {
		result.Dependencies = de.resolveDependencies(ctx, result.Capabilities)
	}

	// Perform conflict detection
	if de.config.ConflictDetection {
		result.Conflicts = de.detectConflicts(ctx, result.Capabilities)
	}

	// Generate recommendations
	if de.config.RecommendationEngine {
		result.Recommendations = de.generateRecommendations(ctx, result)
	}

	// Update internal state
	de.mutex.Lock()
	de.discoveries[pluginName] = result
	de.updateDependencyGraph(pluginName, result.Dependencies)
	de.updateConflictMatrix(pluginName, result.Conflicts)
	de.mutex.Unlock()

	de.logger.WithContext(ctx).Info("plugin_discovery_completed",
		"plugin", pluginName,
		"capabilities", len(result.Capabilities),
		"dependencies", len(result.Dependencies),
		"conflicts", len(result.Conflicts),
		"recommendations", len(result.Recommendations),
	)

	de.metrics.Inc("plugins_discovered_total")
	de.metrics.Set("plugin_capabilities_count", float64(len(result.Capabilities)))

	return result, nil
}

// resolveDependencies identifies dependencies between plugin capabilities
func (de *DiscoveryEngine) resolveDependencies(ctx context.Context, capabilities []*CapabilityAnalysis) []string {
	dependencies := make(map[string]bool)

	for _, cap := range capabilities {
		// Analyze explicit dependencies
		for _, dep := range cap.Dependencies {
			dependencies[dep] = true
		}

		// Analyze semantic dependencies
		switch cap.SemanticCategory {
		case CategoryFileSystem:
			// File operations might depend on path validation
			dependencies["path_validation"] = true
			
		case CategoryVersionControl:
			// Git operations typically depend on file system
			dependencies["file_system"] = true
			
		case CategoryDataStorage:
			// Storage might depend on serialization
			dependencies["serialization"] = true
		}

		// Analyze parameter-based dependencies
		for _, param := range cap.Parameters {
			if param.Type == "object" {
				dependencies["object_validation"] = true
			}
			if param.SemanticRole == "output" {
				dependencies["result_formatting"] = true
			}
		}
	}

	// Convert to sorted slice
	var deps []string
	for dep := range dependencies {
		deps = append(deps, dep)
	}
	sort.Strings(deps)

	return deps
}

// detectConflicts identifies potential conflicts between capabilities
func (de *DiscoveryEngine) detectConflicts(ctx context.Context, capabilities []*CapabilityAnalysis) []string {
	conflicts := make(map[string]bool)

	// Check for explicit conflicts
	for _, cap := range capabilities {
		for _, conflict := range cap.Conflicts {
			conflicts[conflict] = true
		}
	}

	// Detect implicit conflicts
	categoryCount := make(map[SemanticCategory]int)
	for _, cap := range capabilities {
		categoryCount[cap.SemanticCategory]++
	}

	// Multiple storage mechanisms might conflict
	if categoryCount[CategoryDataStorage] > 1 {
		conflicts["multiple_storage_backends"] = true
	}

	// Multiple authentication mechanisms might conflict
	if categoryCount[CategoryAuthentication] > 1 {
		conflicts["multiple_auth_providers"] = true
	}

	// Convert to sorted slice
	var conflictList []string
	for conflict := range conflicts {
		conflictList = append(conflictList, conflict)
	}
	sort.Strings(conflictList)

	return conflictList
}

// generateRecommendations creates intelligent recommendations based on discovery
func (de *DiscoveryEngine) generateRecommendations(ctx context.Context, result *DiscoveryResult) []Recommendation {
	var recommendations []Recommendation

	// Analyze capability coverage
	categoryCount := make(map[SemanticCategory]int)
	qualitySum := make(map[SemanticCategory]float64)
	
	for _, cap := range result.Capabilities {
		categoryCount[cap.SemanticCategory]++
		qualitySum[cap.SemanticCategory] += cap.Quality.OverallScore
	}

	// Recommend improvements for low-quality capabilities
	for category, count := range categoryCount {
		if count > 0 {
			avgQuality := qualitySum[category] / float64(count)
			if avgQuality < 0.7 {
				recommendations = append(recommendations, Recommendation{
					Type:        RecommendationOptimization,
					Priority:    PriorityMedium,
					Description: fmt.Sprintf("Consider improving %s capabilities quality (current: %.2f)", category, avgQuality),
					Actions: []RecommendedAction{
						{
							Type:        "improve_documentation",
							Description: "Add comprehensive documentation and examples",
							Parameters:  map[string]interface{}{"category": category},
							Expected:    "Improved user experience and capability adoption",
						},
					},
					Confidence: 0.8,
				})
			}
		}
	}

	// Recommend dependency resolution
	if len(result.Dependencies) > 0 {
		missingDeps := de.findMissingDependencies(result.Dependencies)
		if len(missingDeps) > 0 {
			recommendations = append(recommendations, Recommendation{
				Type:        RecommendationDependency,
				Priority:    PriorityHigh,
				Description: fmt.Sprintf("Missing dependencies detected: %v", missingDeps),
				Actions: []RecommendedAction{
					{
						Type:        "install_dependencies",
						Description: "Install or enable required dependencies",
						Parameters:  map[string]interface{}{"dependencies": missingDeps},
						Expected:    "Full plugin functionality available",
					},
				},
				Confidence: 0.9,
			})
		}
	}

	// Recommend conflict resolution
	if len(result.Conflicts) > 0 {
		recommendations = append(recommendations, Recommendation{
			Type:        RecommendationConflict,
			Priority:    PriorityCritical,
			Description: fmt.Sprintf("Conflicts detected: %v", result.Conflicts),
			Actions: []RecommendedAction{
				{
					Type:        "resolve_conflicts",
					Description: "Review and resolve capability conflicts",
					Parameters:  map[string]interface{}{"conflicts": result.Conflicts},
					Expected:    "Stable plugin operation without conflicts",
				},
			},
			Confidence: 0.95,
		})
	}

	// Recommend integration opportunities
	integrationOpportunities := de.findIntegrationOpportunities(result.Capabilities)
	for _, opportunity := range integrationOpportunities {
		recommendations = append(recommendations, opportunity)
	}

	return recommendations
}

// findMissingDependencies identifies dependencies that are not available
func (de *DiscoveryEngine) findMissingDependencies(dependencies []string) []string {
	var missing []string
	
	// Check against available capabilities across all plugins
	de.mutex.RLock()
	availableCapabilities := make(map[string]bool)
	for _, discovery := range de.discoveries {
		for _, cap := range discovery.Capabilities {
			for _, provided := range cap.Provides {
				availableCapabilities[provided] = true
			}
		}
	}
	de.mutex.RUnlock()

	for _, dep := range dependencies {
		if !availableCapabilities[dep] {
			missing = append(missing, dep)
		}
	}

	return missing
}

// findIntegrationOpportunities identifies opportunities for capability integration
func (de *DiscoveryEngine) findIntegrationOpportunities(capabilities []*CapabilityAnalysis) []Recommendation {
	var recommendations []Recommendation

	// Look for complementary capabilities
	for i, cap1 := range capabilities {
		for j, cap2 := range capabilities {
			if i >= j {
				continue
			}

			// Check if capabilities can form pipelines
			if de.canFormPipeline(cap1, cap2) {
				recommendations = append(recommendations, Recommendation{
					Type:        RecommendationIntegration,
					Priority:    PriorityMedium,
					Description: fmt.Sprintf("Pipeline opportunity: %s → %s", cap1.CapabilityName, cap2.CapabilityName),
					Actions: []RecommendedAction{
						{
							Type:        "create_pipeline",
							Description: "Create automated pipeline between capabilities",
							Parameters: map[string]interface{}{
								"source": cap1.CapabilityName,
								"target": cap2.CapabilityName,
							},
							Expected: "Streamlined workflow automation",
						},
					},
					Confidence: 0.7,
				})
			}
		}
	}

	return recommendations
}

// canFormPipeline determines if two capabilities can form a data pipeline
func (de *DiscoveryEngine) canFormPipeline(cap1, cap2 *CapabilityAnalysis) bool {
	// Check if first capability has output parameters
	hasOutput := false
	for _, param := range cap1.Parameters {
		if param.SemanticRole == "output" {
			hasOutput = true
			break
		}
	}

	// Check if second capability has input parameters
	hasInput := false
	for _, param := range cap2.Parameters {
		if param.SemanticRole == "input" {
			hasInput = true
			break
		}
	}

	// Check semantic compatibility
	semanticMatch := false
	switch {
	case cap1.SemanticCategory == CategoryDataStorage && cap2.SemanticCategory == CategoryAnalysis:
		semanticMatch = true
	case cap1.SemanticCategory == CategoryFileSystem && cap2.SemanticCategory == CategoryVersionControl:
		semanticMatch = true
	case cap1.SemanticCategory == CategoryTransformation && cap2.SemanticCategory == CategoryCommunication:
		semanticMatch = true
	}

	return hasOutput && hasInput && semanticMatch
}

// Helper methods for updating internal state

func (de *DiscoveryEngine) updateDependencyGraph(pluginName string, dependencies []string) {
	de.dependencyGraph[pluginName] = dependencies
}

func (de *DiscoveryEngine) updateConflictMatrix(pluginName string, conflicts []string) {
	de.conflictMatrix[pluginName] = conflicts
}

// GetDiscoveryResult retrieves the discovery result for a plugin
func (de *DiscoveryEngine) GetDiscoveryResult(pluginName string) (*DiscoveryResult, bool) {
	de.mutex.RLock()
	defer de.mutex.RUnlock()
	
	result, exists := de.discoveries[pluginName]
	return result, exists
}

// GetAllDiscoveries returns all plugin discoveries
func (de *DiscoveryEngine) GetAllDiscoveries() map[string]*DiscoveryResult {
	de.mutex.RLock()
	defer de.mutex.RUnlock()
	
	// Return a copy to prevent external modification
	discoveries := make(map[string]*DiscoveryResult)
	for k, v := range de.discoveries {
		discoveries[k] = v
	}
	
	return discoveries
}

// GetDependencyGraph returns the current dependency graph
func (de *DiscoveryEngine) GetDependencyGraph() map[string][]string {
	de.mutex.RLock()
	defer de.mutex.RUnlock()
	
	// Return a copy
	graph := make(map[string][]string)
	for k, v := range de.dependencyGraph {
		deps := make([]string, len(v))
		copy(deps, v)
		graph[k] = deps
	}
	
	return graph
}