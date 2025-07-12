package mcp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
	"github.com/osakka/mcpeg/pkg/plugins"
)

// PluginHotReload manages hot reloading and updates of plugins
type PluginHotReload struct {
	pluginManager *plugins.PluginManager
	logger        logging.Logger
	metrics       metrics.Metrics
	config        PluginHotReloadConfig

	// Hot reload state
	reloadInProgress map[string]*ReloadOperation
	pluginVersions   map[string]string
	reloadHistory    []ReloadHistoryEntry
	mutex            sync.RWMutex

	// Dependency tracking
	dependencyGraph map[string][]string
	reverseDeps     map[string][]string
}

// PluginHotReloadConfig configures hot reloading behavior
type PluginHotReloadConfig struct {
	// Reload settings
	EnableHotReload      bool          `yaml:"enable_hot_reload"`
	ReloadTimeout        time.Duration `yaml:"reload_timeout"`
	MaxConcurrentReloads int           `yaml:"max_concurrent_reloads"`
	SafeModeEnabled      bool          `yaml:"safe_mode_enabled"`

	// Backup and rollback
	EnableBackup          bool          `yaml:"enable_backup"`
	BackupRetentionPeriod time.Duration `yaml:"backup_retention_period"`
	AutoRollbackOnFailure bool          `yaml:"auto_rollback_on_failure"`

	// Health checking
	HealthCheckAfterReload bool          `yaml:"health_check_after_reload"`
	HealthCheckTimeout     time.Duration `yaml:"health_check_timeout"`

	// Versioning
	EnableVersioning      bool `yaml:"enable_versioning"`
	RequireVersionUpgrade bool `yaml:"require_version_upgrade"`

	// Dependency management
	ResolveDependencies   bool `yaml:"resolve_dependencies"`
	FailOnDependencyError bool `yaml:"fail_on_dependency_error"`
}

// ReloadOperation tracks an ongoing plugin reload operation
type ReloadOperation struct {
	ID              string                 `json:"id"`
	PluginName      string                 `json:"plugin_name"`
	OldVersion      string                 `json:"old_version"`
	NewVersion      string                 `json:"new_version"`
	Status          ReloadStatus           `json:"status"`
	StartTime       time.Time              `json:"start_time"`
	EndTime         *time.Time             `json:"end_time,omitempty"`
	Error           string                 `json:"error,omitempty"`
	Steps           []ReloadStep           `json:"steps"`
	AffectedPlugins []string               `json:"affected_plugins"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// ReloadStep represents a step in the reload process
type ReloadStep struct {
	Name        string        `json:"name"`
	Status      StepStatus    `json:"status"`
	StartTime   time.Time     `json:"start_time"`
	EndTime     *time.Time    `json:"end_time,omitempty"`
	Duration    time.Duration `json:"duration"`
	Error       string        `json:"error,omitempty"`
	Description string        `json:"description"`
}

// ReloadHistoryEntry tracks historical reload operations
type ReloadHistoryEntry struct {
	Operation *ReloadOperation `json:"operation"`
	Timestamp time.Time        `json:"timestamp"`
	Success   bool             `json:"success"`
	Duration  time.Duration    `json:"duration"`
}

// Plugin backup information
type PluginBackup struct {
	PluginName string                 `json:"plugin_name"`
	Version    string                 `json:"version"`
	BackupTime time.Time              `json:"backup_time"`
	BackupData map[string]interface{} `json:"backup_data"`
	Config     plugins.PluginConfig   `json:"config"`
}

// Enums
type ReloadStatus int

const (
	ReloadStatusPending ReloadStatus = iota
	ReloadStatusInProgress
	ReloadStatusCompleted
	ReloadStatusFailed
	ReloadStatusRolledBack
)

type StepStatus int

const (
	StepStatusPending StepStatus = iota
	StepStatusInProgress
	StepStatusCompleted
	StepStatusFailed
	StepStatusSkipped
)

// NewPluginHotReload creates a new plugin hot reload manager
func NewPluginHotReload(
	pluginManager *plugins.PluginManager,
	logger logging.Logger,
	metrics metrics.Metrics,
) *PluginHotReload {
	config := defaultPluginHotReloadConfig()

	phr := &PluginHotReload{
		pluginManager:    pluginManager,
		logger:           logger.WithComponent("plugin_hotreload"),
		metrics:          metrics,
		config:           config,
		reloadInProgress: make(map[string]*ReloadOperation),
		pluginVersions:   make(map[string]string),
		reloadHistory:    make([]ReloadHistoryEntry, 0),
		dependencyGraph:  make(map[string][]string),
		reverseDeps:      make(map[string][]string),
	}

	// Initialize current plugin versions
	phr.initializePluginVersions()

	return phr
}

// ReloadPlugin performs a hot reload of a specific plugin
func (phr *PluginHotReload) ReloadPlugin(ctx context.Context, pluginName string, newPlugin plugins.Plugin) (*ReloadOperation, error) {
	if !phr.config.EnableHotReload {
		return nil, fmt.Errorf("hot reload is not enabled")
	}

	phr.mutex.Lock()
	defer phr.mutex.Unlock()

	// Check if reload is already in progress
	if op, inProgress := phr.reloadInProgress[pluginName]; inProgress {
		return op, fmt.Errorf("reload already in progress for plugin %s", pluginName)
	}

	// Check concurrent reload limit
	if len(phr.reloadInProgress) >= phr.config.MaxConcurrentReloads {
		return nil, fmt.Errorf("maximum concurrent reloads reached (%d)", phr.config.MaxConcurrentReloads)
	}

	// Get current plugin
	currentPlugin, exists := phr.pluginManager.GetPlugin(pluginName)
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", pluginName)
	}

	// Create reload operation
	operation := &ReloadOperation{
		ID:         generateReloadID(),
		PluginName: pluginName,
		OldVersion: currentPlugin.Version(),
		NewVersion: newPlugin.Version(),
		Status:     ReloadStatusPending,
		StartTime:  time.Now(),
		Steps:      make([]ReloadStep, 0),
		Metadata:   make(map[string]interface{}),
	}

	// Validate version upgrade if required
	if phr.config.RequireVersionUpgrade {
		if !phr.isVersionUpgrade(currentPlugin.Version(), newPlugin.Version()) {
			return nil, fmt.Errorf("new version %s is not an upgrade from %s", newPlugin.Version(), currentPlugin.Version())
		}
	}

	// Calculate affected plugins (dependencies)
	if phr.config.ResolveDependencies {
		operation.AffectedPlugins = phr.calculateAffectedPlugins(pluginName)
	}

	phr.reloadInProgress[pluginName] = operation

	// Start reload process asynchronously
	go phr.executeReload(ctx, operation, currentPlugin, newPlugin)

	phr.logger.Info("plugin_reload_started",
		"plugin", pluginName,
		"old_version", operation.OldVersion,
		"new_version", operation.NewVersion,
		"operation_id", operation.ID)

	return operation, nil
}

// GetReloadStatus returns the status of a reload operation
func (phr *PluginHotReload) GetReloadStatus(operationID string) (*ReloadOperation, error) {
	phr.mutex.RLock()
	defer phr.mutex.RUnlock()

	// Check in progress operations
	for _, op := range phr.reloadInProgress {
		if op.ID == operationID {
			return op, nil
		}
	}

	// Check history
	for _, entry := range phr.reloadHistory {
		if entry.Operation.ID == operationID {
			return entry.Operation, nil
		}
	}

	return nil, fmt.Errorf("operation %s not found", operationID)
}

// GetActiveReloads returns all currently active reload operations
func (phr *PluginHotReload) GetActiveReloads() []*ReloadOperation {
	phr.mutex.RLock()
	defer phr.mutex.RUnlock()

	operations := make([]*ReloadOperation, 0, len(phr.reloadInProgress))
	for _, op := range phr.reloadInProgress {
		operations = append(operations, op)
	}
	return operations
}

// GetReloadHistory returns recent reload operations
func (phr *PluginHotReload) GetReloadHistory(limit int) []ReloadHistoryEntry {
	phr.mutex.RLock()
	defer phr.mutex.RUnlock()

	if limit <= 0 || limit > len(phr.reloadHistory) {
		limit = len(phr.reloadHistory)
	}

	// Return most recent entries
	start := len(phr.reloadHistory) - limit
	return phr.reloadHistory[start:]
}

// CancelReload attempts to cancel an ongoing reload operation
func (phr *PluginHotReload) CancelReload(operationID string) error {
	phr.mutex.Lock()
	defer phr.mutex.Unlock()

	for _, op := range phr.reloadInProgress {
		if op.ID == operationID {
			if op.Status == ReloadStatusInProgress {
				// Mark for cancellation - the actual cancellation is handled by the reload goroutine
				op.Metadata["cancel_requested"] = true
				phr.logger.Info("plugin_reload_cancel_requested",
					"operation_id", operationID,
					"plugin", op.PluginName)
				return nil
			}
			return fmt.Errorf("operation %s cannot be cancelled in status %d", operationID, op.Status)
		}
	}

	return fmt.Errorf("operation %s not found or not active", operationID)
}

// RollbackPlugin rolls back a plugin to its previous version
func (phr *PluginHotReload) RollbackPlugin(ctx context.Context, pluginName string) error {
	phr.mutex.Lock()
	defer phr.mutex.Unlock()

	// Find the most recent successful reload in history
	var lastSuccessfulReload *ReloadHistoryEntry
	for i := len(phr.reloadHistory) - 1; i >= 0; i-- {
		entry := &phr.reloadHistory[i]
		if entry.Operation.PluginName == pluginName && entry.Success {
			lastSuccessfulReload = entry
			break
		}
	}

	if lastSuccessfulReload == nil {
		return fmt.Errorf("no previous version found for plugin %s", pluginName)
	}

	phr.logger.Info("plugin_rollback_started",
		"plugin", pluginName,
		"target_version", lastSuccessfulReload.Operation.OldVersion)

	// Create rollback operation (this is a placeholder - in a real implementation,
	// we would need to store plugin artifacts and recreate them)
	operation := &ReloadOperation{
		ID:         generateReloadID(),
		PluginName: pluginName,
		OldVersion: phr.pluginVersions[pluginName],
		NewVersion: lastSuccessfulReload.Operation.OldVersion,
		Status:     ReloadStatusRolledBack,
		StartTime:  time.Now(),
		Steps:      make([]ReloadStep, 0),
		Metadata:   map[string]interface{}{"is_rollback": true},
	}

	// Add to history
	endTime := time.Now()
	operation.EndTime = &endTime
	phr.reloadHistory = append(phr.reloadHistory, ReloadHistoryEntry{
		Operation: operation,
		Timestamp: time.Now(),
		Success:   true,
		Duration:  time.Since(operation.StartTime),
	})

	phr.pluginVersions[pluginName] = operation.NewVersion

	phr.logger.Info("plugin_rollback_completed",
		"plugin", pluginName,
		"version", operation.NewVersion)

	return nil
}

// GetPluginVersions returns current versions of all plugins
func (phr *PluginHotReload) GetPluginVersions() map[string]string {
	phr.mutex.RLock()
	defer phr.mutex.RUnlock()

	versions := make(map[string]string)
	for name, version := range phr.pluginVersions {
		versions[name] = version
	}
	return versions
}

// Private methods

func (phr *PluginHotReload) executeReload(ctx context.Context, operation *ReloadOperation, oldPlugin, newPlugin plugins.Plugin) {
	defer func() {
		phr.mutex.Lock()
		delete(phr.reloadInProgress, operation.PluginName)
		phr.mutex.Unlock()
	}()

	operation.Status = ReloadStatusInProgress
	steps := []string{
		"validate_new_plugin",
		"backup_current_plugin",
		"shutdown_old_plugin",
		"register_new_plugin",
		"initialize_new_plugin",
		"health_check",
		"update_dependencies",
	}

	success := true
	for _, stepName := range steps {
		step := ReloadStep{
			Name:        stepName,
			Status:      StepStatusInProgress,
			StartTime:   time.Now(),
			Description: phr.getStepDescription(stepName),
		}

		// Check for cancellation
		if phr.isReloadCancelled(operation) {
			step.Status = StepStatusSkipped
			step.Error = "Operation cancelled"
			success = false
			break
		}

		// Execute step
		err := phr.executeReloadStep(ctx, stepName, operation, oldPlugin, newPlugin)
		endTime := time.Now()
		step.EndTime = &endTime
		step.Duration = endTime.Sub(step.StartTime)

		if err != nil {
			step.Status = StepStatusFailed
			step.Error = err.Error()
			success = false

			phr.logger.Error("plugin_reload_step_failed",
				"plugin", operation.PluginName,
				"step", stepName,
				"error", err)

			if phr.config.AutoRollbackOnFailure {
				phr.logger.Info("auto_rollback_triggered", "plugin", operation.PluginName)
				// Implement auto-rollback logic here
			}
			break
		} else {
			step.Status = StepStatusCompleted
		}

		operation.Steps = append(operation.Steps, step)
	}

	// Complete operation
	endTime := time.Now()
	operation.EndTime = &endTime
	if success {
		operation.Status = ReloadStatusCompleted
		phr.pluginVersions[operation.PluginName] = operation.NewVersion
	} else {
		operation.Status = ReloadStatusFailed
	}

	// Add to history
	phr.mutex.Lock()
	phr.reloadHistory = append(phr.reloadHistory, ReloadHistoryEntry{
		Operation: operation,
		Timestamp: time.Now(),
		Success:   success,
		Duration:  time.Since(operation.StartTime),
	})
	phr.mutex.Unlock()

	phr.metrics.Inc("plugin_reloads_total", "plugin", operation.PluginName, "success", fmt.Sprintf("%t", success))
	phr.metrics.Observe("plugin_reload_duration", time.Since(operation.StartTime).Seconds(), "plugin", operation.PluginName)

	phr.logger.Info("plugin_reload_completed",
		"plugin", operation.PluginName,
		"success", success,
		"duration", time.Since(operation.StartTime))
}

func (phr *PluginHotReload) executeReloadStep(ctx context.Context, stepName string, operation *ReloadOperation, oldPlugin, newPlugin plugins.Plugin) error {
	switch stepName {
	case "validate_new_plugin":
		return phr.validateNewPlugin(newPlugin)
	case "backup_current_plugin":
		return phr.backupCurrentPlugin(oldPlugin)
	case "shutdown_old_plugin":
		return oldPlugin.Shutdown(ctx)
	case "register_new_plugin":
		return phr.pluginManager.RegisterPlugin(newPlugin)
	case "initialize_new_plugin":
		config := plugins.PluginConfig{
			Name:   newPlugin.Name(),
			Config: make(map[string]interface{}),
		}
		return phr.pluginManager.InitializePlugin(ctx, newPlugin.Name(), config)
	case "health_check":
		if phr.config.HealthCheckAfterReload {
			return newPlugin.HealthCheck(ctx)
		}
		return nil
	case "update_dependencies":
		return phr.updateDependencies(operation.PluginName)
	default:
		return fmt.Errorf("unknown reload step: %s", stepName)
	}
}

func (phr *PluginHotReload) initializePluginVersions() {
	plugins := phr.pluginManager.ListPlugins()
	for name, plugin := range plugins {
		phr.pluginVersions[name] = plugin.Version()
	}
}

func (phr *PluginHotReload) isVersionUpgrade(oldVersion, newVersion string) bool {
	// Simple version comparison - in a real implementation,
	// you would use proper semantic versioning
	return newVersion > oldVersion
}

func (phr *PluginHotReload) calculateAffectedPlugins(pluginName string) []string {
	// Return plugins that depend on this plugin
	return phr.reverseDeps[pluginName]
}

func (phr *PluginHotReload) isReloadCancelled(operation *ReloadOperation) bool {
	if cancel, exists := operation.Metadata["cancel_requested"]; exists {
		return cancel.(bool)
	}
	return false
}

func (phr *PluginHotReload) getStepDescription(stepName string) string {
	descriptions := map[string]string{
		"validate_new_plugin":   "Validate the new plugin implementation",
		"backup_current_plugin": "Create backup of current plugin state",
		"shutdown_old_plugin":   "Gracefully shutdown the old plugin",
		"register_new_plugin":   "Register the new plugin with the manager",
		"initialize_new_plugin": "Initialize the new plugin",
		"health_check":          "Perform health check on the new plugin",
		"update_dependencies":   "Update dependent plugins",
	}
	return descriptions[stepName]
}

func (phr *PluginHotReload) validateNewPlugin(plugin plugins.Plugin) error {
	// Basic validation
	if plugin.Name() == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	if plugin.Version() == "" {
		return fmt.Errorf("plugin version cannot be empty")
	}
	return nil
}

func (phr *PluginHotReload) backupCurrentPlugin(plugin plugins.Plugin) error {
	if !phr.config.EnableBackup {
		return nil
	}

	// Create backup - this is a placeholder implementation
	backup := PluginBackup{
		PluginName: plugin.Name(),
		Version:    plugin.Version(),
		BackupTime: time.Now(),
		BackupData: map[string]interface{}{
			"tools":     plugin.GetTools(),
			"resources": plugin.GetResources(),
			"prompts":   plugin.GetPrompts(),
		},
	}

	phr.logger.Info("plugin_backed_up",
		"plugin", backup.PluginName,
		"version", backup.Version)

	return nil
}

func (phr *PluginHotReload) updateDependencies(pluginName string) error {
	affectedPlugins := phr.reverseDeps[pluginName]
	for _, depPlugin := range affectedPlugins {
		phr.logger.Debug("updating_dependent_plugin",
			"plugin", pluginName,
			"dependent", depPlugin)
		// Update dependency references
	}
	return nil
}

// ID generator
func generateReloadID() string {
	return fmt.Sprintf("reload_%d", time.Now().UnixNano())
}

func defaultPluginHotReloadConfig() PluginHotReloadConfig {
	return PluginHotReloadConfig{
		EnableHotReload:        true,
		ReloadTimeout:          60 * time.Second,
		MaxConcurrentReloads:   3,
		SafeModeEnabled:        true,
		EnableBackup:           true,
		BackupRetentionPeriod:  24 * time.Hour,
		AutoRollbackOnFailure:  true,
		HealthCheckAfterReload: true,
		HealthCheckTimeout:     10 * time.Second,
		EnableVersioning:       true,
		RequireVersionUpgrade:  false,
		ResolveDependencies:    true,
		FailOnDependencyError:  false,
	}
}
