package rbac

import (
	"time"
)

// ProcessedCapabilities represents the processed and filtered capabilities for a user
type ProcessedCapabilities struct {
	UserID    string                      `json:"user_id"`
	Roles     []string                    `json:"roles"`
	Plugins   map[string]PluginPermission `json:"plugins"`
	ExpiresAt time.Time                   `json:"expires_at"`
	SessionID string                      `json:"session_id,omitempty"`
}

// PluginPermission defines what actions a user can perform on a plugin
type PluginPermission struct {
	CanRead    bool `json:"can_read"`
	CanWrite   bool `json:"can_write"`
	CanExecute bool `json:"can_execute"`
	CanAdmin   bool `json:"can_admin"`
}

// Policy represents an RBAC policy configuration
type Policy struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Rules       []Rule `yaml:"rules"`
}

// Rule defines access rules for plugins
type Rule struct {
	Plugin      string   `yaml:"plugin"`
	Permissions []string `yaml:"permissions"` // read, write, execute, admin
	Conditions  []string `yaml:"conditions,omitempty"`
}

// PolicyConfig represents the complete RBAC configuration
type PolicyConfig struct {
	Policies map[string]Policy `yaml:"policies"`
	Default  string            `yaml:"default"` // Default policy for unknown roles
}

// HasPermission checks if capabilities allow a specific action on a plugin
func (pc *ProcessedCapabilities) HasPermission(pluginName string, permission string) bool {
	// Check direct plugin permission
	if pluginPerm, exists := pc.Plugins[pluginName]; exists {
		return pc.checkPermission(pluginPerm, permission)
	}

	// Check wildcard permission
	if wildcardPerm, exists := pc.Plugins["*"]; exists {
		return pc.checkPermission(wildcardPerm, permission)
	}

	return false
}

func (pc *ProcessedCapabilities) checkPermission(perm PluginPermission, permission string) bool {
	switch permission {
	case "read":
		return perm.CanRead
	case "write":
		return perm.CanWrite
	case "execute":
		return perm.CanExecute
	case "admin":
		return perm.CanAdmin
	default:
		return false
	}
}

// IsValid checks if the capabilities are still valid
func (pc *ProcessedCapabilities) IsValid() bool {
	return time.Now().Before(pc.ExpiresAt)
}

// GetAllowedPlugins returns a list of plugins the user has any access to
func (pc *ProcessedCapabilities) GetAllowedPlugins() []string {
	plugins := make([]string, 0, len(pc.Plugins))
	for plugin, perm := range pc.Plugins {
		if plugin != "*" && (perm.CanRead || perm.CanWrite || perm.CanExecute || perm.CanAdmin) {
			plugins = append(plugins, plugin)
		}
	}
	return plugins
}
