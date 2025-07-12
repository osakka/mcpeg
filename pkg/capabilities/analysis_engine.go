package capabilities

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/osakka/mcpeg/internal/registry"
	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
)

// NewAnalysisEngine creates a new capability analysis engine
func NewAnalysisEngine(logger logging.Logger, metrics metrics.Metrics, config AnalysisConfig) *AnalysisEngine {
	return &AnalysisEngine{
		logger:     logger.WithComponent("capability-analysis"),
		metrics:    metrics,
		analyses:   make(map[string]*CapabilityAnalysis),
		relations:  make(map[string][]CapabilityRelation),
		categories: make(map[SemanticCategory][]*CapabilityAnalysis),
		config:     config,
	}
}

// AnalyzeCapability performs comprehensive analysis of a single capability
func (ae *AnalysisEngine) AnalyzeCapability(ctx context.Context, pluginName string, capability interface{}) (*CapabilityAnalysis, error) {
	startTime := time.Now()
	defer func() {
		ae.metrics.Observe("capability_analysis_duration_ms", float64(time.Since(startTime).Nanoseconds()/1e6))
	}()

	analysis := &CapabilityAnalysis{
		PluginName:    pluginName,
		AnalyzedAt:    time.Now(),
		Metadata:      make(map[string]interface{}),
		Relationships: []CapabilityRelation{},
	}

	// Determine capability type and extract details
	switch cap := capability.(type) {
	case registry.ToolDefinition:
		analysis.CapabilityName = cap.Name
		analysis.CapabilityType = CapabilityTool
		analysis.Description = cap.Description
		analysis.Parameters = ae.analyzeToolParameters(cap)
		
	case registry.ResourceDefinition:
		analysis.CapabilityName = cap.Name
		analysis.CapabilityType = CapabilityResource
		analysis.Description = cap.Description
		analysis.Parameters = ae.analyzeResourceParameters(cap)
		
	case registry.PromptDefinition:
		analysis.CapabilityName = cap.Name
		analysis.CapabilityType = CapabilityPrompt
		analysis.Description = cap.Description
		analysis.Parameters = ae.analyzePromptParameters(cap)
		
	default:
		return nil, fmt.Errorf("unsupported capability type: %T", capability)
	}

	// Perform semantic categorization
	analysis.SemanticCategory = ae.categorizeCapability(analysis)

	// Analyze quality metrics
	if ae.config.EnableQualityMetrics {
		analysis.Quality = ae.analyzeQuality(analysis)
	}

	// Initialize usage metrics
	if ae.config.EnableUsageTracking {
		analysis.Usage = UsageMetrics{
			CallCount:       0,
			SuccessRate:     1.0,
			AverageLatency:  0,
			ErrorRate:       0.0,
			LastUsed:        time.Time{},
			PopularityScore: 0.0,
			EfficiencyScore: 1.0,
		}
	}

	// Perform semantic analysis to identify dependencies and conflicts
	if ae.config.EnableSemanticAnalysis {
		ae.analyzeSemanticProperties(analysis)
	}

	// Store analysis
	key := fmt.Sprintf("%s:%s", pluginName, analysis.CapabilityName)
	ae.analyses[key] = analysis

	// Update category mapping
	ae.updateCategoryMapping(analysis)

	ae.logger.WithContext(ctx).Info("capability_analyzed",
		"plugin", pluginName,
		"capability", analysis.CapabilityName,
		"type", analysis.CapabilityType,
		"category", analysis.SemanticCategory,
		"quality_score", analysis.Quality.OverallScore,
	)

	ae.metrics.Inc("capabilities_analyzed_total")

	return analysis, nil
}

// analyzeToolParameters extracts and analyzes tool parameters
func (ae *AnalysisEngine) analyzeToolParameters(tool registry.ToolDefinition) []ParameterAnalysis {
	var params []ParameterAnalysis

	// Parse input schema if available
	if tool.InputSchema != nil {
		if properties, ok := tool.InputSchema["properties"].(map[string]interface{}); ok {
			required := []string{}
			if req, ok := tool.InputSchema["required"].([]interface{}); ok {
				for _, r := range req {
					if reqStr, ok := r.(string); ok {
						required = append(required, reqStr)
					}
				}
			}

			for name, prop := range properties {
				if propMap, ok := prop.(map[string]interface{}); ok {
					param := ParameterAnalysis{
						Name:         name,
						Required:     contains(required, name),
						Constraints:  make(map[string]interface{}),
						Examples:     []interface{}{},
					}

					if typeVal, ok := propMap["type"].(string); ok {
						param.Type = typeVal
					}

					if desc, ok := propMap["description"].(string); ok {
						param.Description = desc
					}

					// Analyze semantic role based on name and description
					param.SemanticRole = ae.analyzeParameterRole(name, param.Description)

					// Extract constraints
					for key, value := range propMap {
						if key != "type" && key != "description" {
							param.Constraints[key] = value
						}
					}

					params = append(params, param)
				}
			}
		}
	}

	return params
}

// analyzeResourceParameters extracts and analyzes resource parameters
func (ae *AnalysisEngine) analyzeResourceParameters(resource registry.ResourceDefinition) []ParameterAnalysis {
	// Resources typically don't have complex parameter schemas
	// But we can analyze the URI template for parameters
	return ae.extractURIParameters(resource.URI)
}

// analyzePromptParameters extracts and analyzes prompt parameters  
func (ae *AnalysisEngine) analyzePromptParameters(prompt registry.PromptDefinition) []ParameterAnalysis {
	var params []ParameterAnalysis

	// Analyze prompt arguments
	for _, arg := range prompt.Arguments {
		param := ParameterAnalysis{
			Name:         arg.Name,
			Required:     arg.Required,
			Description:  arg.Description,
			SemanticRole: "input",
			Constraints:  make(map[string]interface{}),
			Examples:     []interface{}{},
		}

		// Infer type from description or name
		param.Type = ae.inferParameterType(arg.Name, arg.Description)

		params = append(params, param)
	}

	return params
}

// categorizeCapability assigns semantic categories to capabilities
func (ae *AnalysisEngine) categorizeCapability(analysis *CapabilityAnalysis) SemanticCategory {
	name := strings.ToLower(analysis.CapabilityName)
	desc := strings.ToLower(analysis.Description)
	pluginName := strings.ToLower(analysis.PluginName)

	// Pattern-based categorization
	patterns := map[SemanticCategory][]string{
		CategoryDataStorage: {
			"store", "save", "persist", "cache", "database", "storage", "memory",
			"get", "retrieve", "fetch", "load", "read", "query",
		},
		CategoryFileSystem: {
			"file", "directory", "folder", "path", "edit", "write", "create",
			"delete", "move", "copy", "rename", "list", "browse",
		},
		CategoryVersionControl: {
			"git", "commit", "branch", "merge", "push", "pull", "clone",
			"version", "diff", "log", "status", "checkout",
		},
		CategoryCommunication: {
			"send", "receive", "message", "email", "chat", "notify",
			"webhook", "api", "request", "response", "http", "rest",
		},
		CategoryAnalysis: {
			"analyze", "parse", "validate", "check", "inspect", "examine",
			"test", "verify", "scan", "detect", "measure", "calculate",
		},
		CategoryTransformation: {
			"convert", "transform", "format", "encode", "decode", "compress",
			"extract", "process", "filter", "sort", "group", "aggregate",
		},
		CategoryMonitoring: {
			"monitor", "watch", "track", "log", "metric", "health",
			"status", "performance", "alert", "observe", "measure",
		},
		CategoryAuthentication: {
			"auth", "login", "token", "key", "credential", "permission",
			"access", "security", "verify", "validate", "authorize",
		},
	}

	// Score each category
	scores := make(map[SemanticCategory]int)
	
	for category, keywords := range patterns {
		for _, keyword := range keywords {
			if strings.Contains(name, keyword) {
				scores[category] += 3
			}
			if strings.Contains(desc, keyword) {
				scores[category] += 2
			}
			if strings.Contains(pluginName, keyword) {
				scores[category] += 1
			}
		}
	}

	// Find highest scoring category
	var bestCategory SemanticCategory
	maxScore := 0
	
	for category, score := range scores {
		if score > maxScore {
			maxScore = score
			bestCategory = category
		}
	}

	// Default to analysis if no clear category
	if bestCategory == "" {
		bestCategory = CategoryAnalysis
	}

	return bestCategory
}

// analyzeQuality assesses the quality of a capability
func (ae *AnalysisEngine) analyzeQuality(analysis *CapabilityAnalysis) QualityMetrics {
	quality := QualityMetrics{}

	// Documentation score based on description completeness
	if len(analysis.Description) > 0 {
		quality.DocumentationScore = min(1.0, float64(len(analysis.Description))/200.0)
	}

	// Parameter coverage score
	if len(analysis.Parameters) > 0 {
		documented := 0
		for _, param := range analysis.Parameters {
			if param.Description != "" {
				documented++
			}
		}
		quality.ParameterCoverage = float64(documented) / float64(len(analysis.Parameters))
	} else {
		quality.ParameterCoverage = 1.0
	}

	// Default scores for now - would be enhanced with actual analysis
	quality.ErrorHandling = 0.8
	quality.TestCoverage = 0.7
	quality.PerformanceScore = 0.8
	quality.ReliabilityScore = 0.8

	// Calculate overall score as weighted average
	quality.OverallScore = (
		quality.DocumentationScore*0.2 +
		quality.ParameterCoverage*0.2 +
		quality.ErrorHandling*0.15 +
		quality.TestCoverage*0.15 +
		quality.PerformanceScore*0.15 +
		quality.ReliabilityScore*0.15) / 1.0

	return quality
}

// analyzeSemanticProperties identifies dependencies and conflicts
func (ae *AnalysisEngine) analyzeSemanticProperties(analysis *CapabilityAnalysis) {
	// Analyze parameter types and names for dependencies
	for _, param := range analysis.Parameters {
		// File path parameters might depend on file system capabilities
		if strings.Contains(strings.ToLower(param.Name), "path") ||
		   strings.Contains(strings.ToLower(param.Name), "file") {
			analysis.Dependencies = append(analysis.Dependencies, "file_system")
		}

		// Authentication parameters
		if strings.Contains(strings.ToLower(param.Name), "token") ||
		   strings.Contains(strings.ToLower(param.Name), "auth") ||
		   strings.Contains(strings.ToLower(param.Name), "key") {
			analysis.Dependencies = append(analysis.Dependencies, "authentication")
		}
	}

	// Analyze what this capability provides
	switch analysis.SemanticCategory {
	case CategoryDataStorage:
		analysis.Provides = append(analysis.Provides, "data_persistence", "retrieval")
	case CategoryFileSystem:
		analysis.Provides = append(analysis.Provides, "file_operations", "path_resolution")
	case CategoryVersionControl:
		analysis.Provides = append(analysis.Provides, "version_tracking", "change_management")
	case CategoryAuthentication:
		analysis.Provides = append(analysis.Provides, "access_control", "identity_verification")
	}
}

// Helper functions

func (ae *AnalysisEngine) analyzeParameterRole(name, description string) string {
	name = strings.ToLower(name)
	desc := strings.ToLower(description)

	if strings.Contains(name, "output") || strings.Contains(desc, "result") {
		return "output"
	}
	if strings.Contains(name, "config") || strings.Contains(desc, "configuration") {
		return "configuration"
	}
	if strings.Contains(name, "context") || strings.Contains(desc, "context") {
		return "context"
	}
	return "input"
}

func (ae *AnalysisEngine) extractURIParameters(uri string) []ParameterAnalysis {
	// Extract parameters from URI template like /resource/{id}
	re := regexp.MustCompile(`\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(uri, -1)
	
	var params []ParameterAnalysis
	for _, match := range matches {
		if len(match) > 1 {
			params = append(params, ParameterAnalysis{
				Name:         match[1],
				Type:         "string",
				Required:     true,
				Description:  fmt.Sprintf("URI parameter: %s", match[1]),
				SemanticRole: "input",
				Constraints:  make(map[string]interface{}),
				Examples:     []interface{}{},
			})
		}
	}
	
	return params
}

func (ae *AnalysisEngine) inferParameterType(name, description string) string {
	name = strings.ToLower(name)
	desc := strings.ToLower(description)

	if strings.Contains(name, "count") || strings.Contains(name, "number") ||
	   strings.Contains(name, "size") || strings.Contains(name, "limit") {
		return "integer"
	}
	if strings.Contains(name, "enabled") || strings.Contains(name, "flag") ||
	   strings.Contains(desc, "true") || strings.Contains(desc, "false") {
		return "boolean"
	}
	if strings.Contains(name, "list") || strings.Contains(name, "array") {
		return "array"
	}
	return "string"
}

func (ae *AnalysisEngine) updateCategoryMapping(analysis *CapabilityAnalysis) {
	ae.categories[analysis.SemanticCategory] = append(ae.categories[analysis.SemanticCategory], analysis)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// GetAnalysis retrieves analysis for a specific capability
func (ae *AnalysisEngine) GetAnalysis(pluginName, capabilityName string) (*CapabilityAnalysis, bool) {
	key := fmt.Sprintf("%s:%s", pluginName, capabilityName)
	analysis, exists := ae.analyses[key]
	return analysis, exists
}

// GetAnalysesByCategory returns all analyses for a semantic category
func (ae *AnalysisEngine) GetAnalysesByCategory(category SemanticCategory) []*CapabilityAnalysis {
	return ae.categories[category]
}

// GetAllAnalyses returns all capability analyses
func (ae *AnalysisEngine) GetAllAnalyses() map[string]*CapabilityAnalysis {
	return ae.analyses
}