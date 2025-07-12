package rbac

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/osakka/mcpeg/pkg/auth"
	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
	"gopkg.in/yaml.v3"
)

// Engine handles RBAC policy evaluation and token processing
type Engine struct {
	jwtValidator  *auth.JWTValidator
	policies      map[string]*Policy
	defaultPolicy string
	logger        logging.Logger
	metrics       metrics.Metrics
	cacheTTL      time.Duration
	cache         map[string]*cacheEntry
}

type cacheEntry struct {
	capabilities *ProcessedCapabilities
	cachedAt     time.Time
}

// Config configures the RBAC engine
type Config struct {
	PolicyPath    string         `yaml:"policy_path"`
	DefaultPolicy string         `yaml:"default_policy"`
	CacheTTL      time.Duration  `yaml:"cache_ttl"`
	JWTConfig     auth.JWTConfig `yaml:"jwt"`
}

// NewEngine creates a new RBAC engine
func NewEngine(config Config, logger logging.Logger, metrics metrics.Metrics) (*Engine, error) {
	// Initialize JWT validator
	jwtValidator, err := auth.NewJWTValidator(config.JWTConfig, logger, metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT validator: %w", err)
	}

	engine := &Engine{
		jwtValidator:  jwtValidator,
		policies:      make(map[string]*Policy),
		defaultPolicy: config.DefaultPolicy,
		logger:        logger,
		metrics:       metrics,
		cacheTTL:      config.CacheTTL,
		cache:         make(map[string]*cacheEntry),
	}

	if engine.cacheTTL == 0 {
		engine.cacheTTL = 5 * time.Minute // Default 5 minute cache
	}

	// Load policies
	if err := engine.LoadPolicies(config.PolicyPath); err != nil {
		return nil, fmt.Errorf("failed to load policies: %w", err)
	}

	return engine, nil
}

// ProcessToken validates a JWT token and returns processed capabilities
func (e *Engine) ProcessToken(tokenString string) (*ProcessedCapabilities, error) {
	timer := e.metrics.Time("rbac_token_processing_duration")
	defer timer.Stop()

	// Check cache first
	if cached := e.getFromCache(tokenString); cached != nil {
		e.metrics.Inc("rbac_cache_hits")
		return cached, nil
	}

	e.metrics.Inc("rbac_cache_misses")

	// Validate JWT token
	claims, err := e.jwtValidator.ValidateToken(tokenString)
	if err != nil {
		e.metrics.Inc("rbac_token_validation_errors")
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	// Process capabilities
	capabilities, err := e.processCapabilities(claims)
	if err != nil {
		e.metrics.Inc("rbac_capability_processing_errors")
		return nil, fmt.Errorf("capability processing failed: %w", err)
	}

	// Cache the result
	e.addToCache(tokenString, capabilities)

	e.metrics.Inc("rbac_token_processing_success")
	e.logger.Debug("rbac_token_processed",
		"user_id", capabilities.UserID,
		"roles", capabilities.Roles,
		"plugin_count", len(capabilities.Plugins))

	return capabilities, nil
}

// processCapabilities converts JWT claims to processed capabilities
func (e *Engine) processCapabilities(claims *auth.JWTClaims) (*ProcessedCapabilities, error) {
	capabilities := &ProcessedCapabilities{
		UserID:    claims.Subject,
		Roles:     claims.Roles,
		Plugins:   make(map[string]PluginPermission),
		ExpiresAt: time.Unix(claims.ExpiresAt, 0),
		SessionID: claims.SessionID,
	}

	// Apply policies for each role
	for _, role := range claims.Roles {
		if err := e.applyRolePolicy(role, capabilities); err != nil {
			e.logger.Warn("rbac_role_policy_application_failed",
				"role", role,
				"user_id", claims.Subject,
				"error", err)
			continue
		}
	}

	// If no policies applied and we have a default, apply it
	if len(capabilities.Plugins) == 0 && e.defaultPolicy != "" {
		if err := e.applyRolePolicy(e.defaultPolicy, capabilities); err != nil {
			e.logger.Warn("rbac_default_policy_application_failed",
				"default_policy", e.defaultPolicy,
				"user_id", claims.Subject,
				"error", err)
		}
	}

	return capabilities, nil
}

// applyRolePolicy applies a role's policy to the capabilities
func (e *Engine) applyRolePolicy(role string, capabilities *ProcessedCapabilities) error {
	policy, exists := e.policies[role]
	if !exists {
		return fmt.Errorf("policy not found for role: %s", role)
	}

	e.logger.Debug("rbac_applying_policy", "role", role, "policy", policy.Name)

	for _, rule := range policy.Rules {
		permission := e.calculatePermissions(rule.Permissions)

		// Handle wildcard plugin
		if rule.Plugin == "*" {
			capabilities.Plugins["*"] = permission
		} else {
			// Merge with existing permissions (union)
			if existing, exists := capabilities.Plugins[rule.Plugin]; exists {
				capabilities.Plugins[rule.Plugin] = e.mergePermissions(existing, permission)
			} else {
				capabilities.Plugins[rule.Plugin] = permission
			}
		}

		e.metrics.Inc("rbac_rules_applied", "role", role, "plugin", rule.Plugin)
	}

	return nil
}

// calculatePermissions converts permission strings to PluginPermission
func (e *Engine) calculatePermissions(permissions []string) PluginPermission {
	perm := PluginPermission{}

	for _, p := range permissions {
		switch strings.ToLower(p) {
		case "read":
			perm.CanRead = true
		case "write":
			perm.CanWrite = true
		case "execute":
			perm.CanExecute = true
		case "admin":
			perm.CanAdmin = true
		}
	}

	return perm
}

// mergePermissions merges two permissions (union)
func (e *Engine) mergePermissions(existing, new PluginPermission) PluginPermission {
	return PluginPermission{
		CanRead:    existing.CanRead || new.CanRead,
		CanWrite:   existing.CanWrite || new.CanWrite,
		CanExecute: existing.CanExecute || new.CanExecute,
		CanAdmin:   existing.CanAdmin || new.CanAdmin,
	}
}

// LoadPolicies loads RBAC policies from a file or directory
func (e *Engine) LoadPolicies(policyPath string) error {
	if policyPath == "" {
		e.logger.Info("rbac_no_policy_path_configured_using_defaults")
		e.loadDefaultPolicies()
		return nil
	}

	info, err := os.Stat(policyPath)
	if err != nil {
		return fmt.Errorf("policy path does not exist: %w", err)
	}

	if info.IsDir() {
		return e.loadPoliciesFromDirectory(policyPath)
	} else {
		return e.loadPoliciesFromFile(policyPath)
	}
}

func (e *Engine) loadPoliciesFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read policy file: %w", err)
	}

	var config PolicyConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse policy file: %w", err)
	}

	for name, policy := range config.Policies {
		e.policies[name] = &policy
		e.logger.Info("rbac_policy_loaded", "name", name, "rules", len(policy.Rules))
	}

	if config.Default != "" {
		e.defaultPolicy = config.Default
	}

	e.logger.Info("rbac_policies_loaded", "count", len(e.policies), "file", filename)
	return nil
}

func (e *Engine) loadPoliciesFromDirectory(dirPath string) error {
	files, err := filepath.Glob(filepath.Join(dirPath, "*.yaml"))
	if err != nil {
		return fmt.Errorf("failed to find policy files: %w", err)
	}

	for _, file := range files {
		if err := e.loadPoliciesFromFile(file); err != nil {
			e.logger.Warn("rbac_policy_file_load_failed", "file", file, "error", err)
			continue
		}
	}

	return nil
}

func (e *Engine) loadDefaultPolicies() {
	// Load built-in default policies
	defaultPolicies := map[string]*Policy{
		"admin": {
			Name:        "Administrator",
			Description: "Full access to all plugins",
			Rules: []Rule{
				{
					Plugin:      "*",
					Permissions: []string{"read", "write", "execute", "admin"},
				},
			},
		},
		"readonly": {
			Name:        "Read Only",
			Description: "Read-only access to memory plugin",
			Rules: []Rule{
				{
					Plugin:      "memory",
					Permissions: []string{"read"},
				},
			},
		},
	}

	for name, policy := range defaultPolicies {
		e.policies[name] = policy
		e.logger.Info("rbac_default_policy_loaded", "name", name)
	}

	e.defaultPolicy = "readonly"
}

// Cache management
func (e *Engine) getFromCache(token string) *ProcessedCapabilities {
	if entry, exists := e.cache[token]; exists {
		if time.Since(entry.cachedAt) < e.cacheTTL && entry.capabilities.IsValid() {
			return entry.capabilities
		}
		// Remove expired entry
		delete(e.cache, token)
	}
	return nil
}

func (e *Engine) addToCache(token string, capabilities *ProcessedCapabilities) {
	e.cache[token] = &cacheEntry{
		capabilities: capabilities,
		cachedAt:     time.Now(),
	}

	// Simple cache cleanup - remove if cache gets too large
	if len(e.cache) > 1000 {
		e.cleanupCache()
	}
}

func (e *Engine) cleanupCache() {
	now := time.Now()
	for token, entry := range e.cache {
		if now.Sub(entry.cachedAt) > e.cacheTTL {
			delete(e.cache, token)
		}
	}
	e.logger.Debug("rbac_cache_cleaned", "remaining_entries", len(e.cache))
}

// GetPolicyNames returns the names of all loaded policies
func (e *Engine) GetPolicyNames() []string {
	names := make([]string, 0, len(e.policies))
	for name := range e.policies {
		names = append(names, name)
	}
	return names
}

// ValidatePolicy validates a policy configuration
func (e *Engine) ValidatePolicy(policy *Policy) error {
	if policy.Name == "" {
		return fmt.Errorf("policy name is required")
	}

	if len(policy.Rules) == 0 {
		return fmt.Errorf("policy must have at least one rule")
	}

	for i, rule := range policy.Rules {
		if rule.Plugin == "" {
			return fmt.Errorf("rule %d: plugin name is required", i)
		}

		if len(rule.Permissions) == 0 {
			return fmt.Errorf("rule %d: at least one permission is required", i)
		}

		// Validate permission values
		validPerms := map[string]bool{"read": true, "write": true, "execute": true, "admin": true}
		for _, perm := range rule.Permissions {
			if !validPerms[strings.ToLower(perm)] {
				return fmt.Errorf("rule %d: invalid permission '%s'", i, perm)
			}
		}
	}

	return nil
}
