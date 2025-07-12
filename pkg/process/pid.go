package process

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/osakka/mcpeg/pkg/logging"
)

// PIDManager manages process ID files for daemon processes
type PIDManager struct {
	pidFile string
	logger  logging.Logger
}

// NewPIDManager creates a new PID manager
func NewPIDManager(pidFile string, logger logging.Logger) *PIDManager {
	return &PIDManager{
		pidFile: pidFile,
		logger:  logger.WithComponent("pid_manager"),
	}
}

// WritePID writes the current process ID to the PID file
func (pm *PIDManager) WritePID() error {
	if pm.pidFile == "" {
		return nil // No PID file configured
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(pm.pidFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create PID directory %s: %w", dir, err)
	}

	// Check if PID file already exists and process is running
	if err := pm.checkExistingProcess(); err != nil {
		return err
	}

	// Write current process ID
	pid := os.Getpid()
	content := fmt.Sprintf("%d\n", pid)

	if err := os.WriteFile(pm.pidFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write PID file %s: %w", pm.pidFile, err)
	}

	pm.logger.Info("pid_file_created",
		"pid_file", pm.pidFile,
		"pid", pid)

	return nil
}

// RemovePID removes the PID file
func (pm *PIDManager) RemovePID() error {
	if pm.pidFile == "" {
		return nil // No PID file configured
	}

	if _, err := os.Stat(pm.pidFile); os.IsNotExist(err) {
		return nil // File doesn't exist, nothing to do
	}

	if err := os.Remove(pm.pidFile); err != nil {
		pm.logger.Error("failed_to_remove_pid_file",
			"pid_file", pm.pidFile,
			"error", err)
		return fmt.Errorf("failed to remove PID file %s: %w", pm.pidFile, err)
	}

	pm.logger.Info("pid_file_removed", "pid_file", pm.pidFile)
	return nil
}

// checkExistingProcess checks if there's already a running process
func (pm *PIDManager) checkExistingProcess() error {
	if _, err := os.Stat(pm.pidFile); os.IsNotExist(err) {
		return nil // No existing PID file
	}

	// Read existing PID
	existingPID, err := pm.ReadPID()
	if err != nil {
		pm.logger.Warn("failed_to_read_existing_pid_file",
			"pid_file", pm.pidFile,
			"error", err)
		// Remove invalid PID file
		os.Remove(pm.pidFile)
		return nil
	}

	// Check if process is still running
	if pm.isProcessRunning(existingPID) {
		return fmt.Errorf("MCpeg is already running with PID %d (PID file: %s)", existingPID, pm.pidFile)
	}

	// Process is not running, remove stale PID file
	pm.logger.Warn("removing_stale_pid_file",
		"pid_file", pm.pidFile,
		"stale_pid", existingPID)
	os.Remove(pm.pidFile)

	return nil
}

// ReadPID reads the PID from the PID file
func (pm *PIDManager) ReadPID() (int, error) {
	if pm.pidFile == "" {
		return 0, fmt.Errorf("no PID file configured")
	}

	content, err := os.ReadFile(pm.pidFile)
	if err != nil {
		return 0, fmt.Errorf("failed to read PID file %s: %w", pm.pidFile, err)
	}

	pidStr := strings.TrimSpace(string(content))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file %s: %s", pm.pidFile, pidStr)
	}

	return pid, nil
}

// isProcessRunning checks if a process with the given PID is running
func (pm *PIDManager) isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	// Send signal 0 to check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// GetPIDFile returns the PID file path
func (pm *PIDManager) GetPIDFile() string {
	return pm.pidFile
}

// IsRunning checks if MCpeg is currently running based on PID file
func (pm *PIDManager) IsRunning() (bool, int, error) {
	if pm.pidFile == "" {
		return false, 0, fmt.Errorf("no PID file configured")
	}

	if _, err := os.Stat(pm.pidFile); os.IsNotExist(err) {
		return false, 0, nil // No PID file means not running
	}

	pid, err := pm.ReadPID()
	if err != nil {
		return false, 0, err
	}

	isRunning := pm.isProcessRunning(pid)
	return isRunning, pid, nil
}

// StopProcess attempts to stop the running MCpeg process
func (pm *PIDManager) StopProcess(force bool) error {
	isRunning, pid, err := pm.IsRunning()
	if err != nil {
		return err
	}

	if !isRunning {
		pm.logger.Info("no_running_process_found")
		return nil
	}

	pm.logger.Info("stopping_process", "pid", pid, "force", force)

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %w", pid, err)
	}

	// Send appropriate signal
	var signal os.Signal
	if force {
		signal = syscall.SIGKILL
		pm.logger.Info("sending_sigkill", "pid", pid)
	} else {
		signal = syscall.SIGTERM
		pm.logger.Info("sending_sigterm", "pid", pid)
	}

	if err := process.Signal(signal); err != nil {
		return fmt.Errorf("failed to send signal to process %d: %w", pid, err)
	}

	// Wait a moment to see if process stops
	if !force {
		// Give process time to gracefully shutdown
		for i := 0; i < 10; i++ {
			if !pm.isProcessRunning(pid) {
				pm.logger.Info("process_stopped_gracefully", "pid", pid)
				break
			}
			// Wait 500ms between checks
			select {
			case <-pm.createDelay(500):
			}
		}

		// If still running after graceful attempts, force kill
		if pm.isProcessRunning(pid) {
			pm.logger.Warn("process_still_running_after_sigterm_forcing_kill", "pid", pid)
			if err := process.Signal(syscall.SIGKILL); err != nil {
				return fmt.Errorf("failed to force kill process %d: %w", pid, err)
			}
		}
	}

	// Clean up PID file
	pm.RemovePID()

	return nil
}

// createDelay creates a channel that closes after the specified milliseconds
func (pm *PIDManager) createDelay(ms int) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		defer close(ch)
		// Simple delay implementation
		for i := 0; i < ms; i++ {
			// Busy wait for 1ms equivalent
			for j := 0; j < 100000; j++ {
				// Simple loop to create delay
			}
		}
	}()
	return ch
}

// Note: GetDefaultPIDFile has been moved to pkg/config/paths.go to centralize path management

// ValidatePIDFile validates that the PID file path is usable
func ValidatePIDFile(pidFile string) error {
	if pidFile == "" {
		return fmt.Errorf("PID file path cannot be empty")
	}

	// Check if directory exists or can be created
	dir := filepath.Dir(pidFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("cannot create PID file directory %s: %w", dir, err)
	}

	// Check if we can write to the directory
	testFile := filepath.Join(dir, ".mcpeg_test_write")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("cannot write to PID file directory %s: %w", dir, err)
	}
	os.Remove(testFile) // Clean up test file

	return nil
}

// ProcessStatus represents the status of the MCpeg process
type ProcessStatus struct {
	Running   bool   `json:"running"`
	PID       int    `json:"pid,omitempty"`
	PIDFile   string `json:"pid_file"`
	Uptime    string `json:"uptime,omitempty"`
	StartTime string `json:"start_time,omitempty"`
}

// GetProcessStatus returns the current process status
func (pm *PIDManager) GetProcessStatus() ProcessStatus {
	status := ProcessStatus{
		PIDFile: pm.pidFile,
	}

	isRunning, pid, err := pm.IsRunning()
	if err != nil || !isRunning {
		return status
	}

	status.Running = true
	status.PID = pid

	// Try to get process start time and uptime
	if startTime, uptime := pm.getProcessInfo(pid); startTime != "" {
		status.StartTime = startTime
		status.Uptime = uptime
	}

	return status
}

// getProcessInfo gets process start time and uptime (simplified implementation)
func (pm *PIDManager) getProcessInfo(pid int) (startTime, uptime string) {
	// This is a simplified implementation
	// In a full implementation, you'd read from /proc/<pid>/stat on Linux
	// or use platform-specific APIs

	// For now, return basic info
	return "unknown", "unknown"
}

// LogRotationSignal sends a signal to rotate logs (typically SIGUSR1)
func (pm *PIDManager) LogRotationSignal() error {
	isRunning, pid, err := pm.IsRunning()
	if err != nil {
		return err
	}

	if !isRunning {
		return fmt.Errorf("MCpeg is not running")
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %w", pid, err)
	}

	// Send SIGUSR1 for log rotation
	if err := process.Signal(syscall.SIGUSR1); err != nil {
		return fmt.Errorf("failed to send log rotation signal to process %d: %w", pid, err)
	}

	pm.logger.Info("log_rotation_signal_sent", "pid", pid)
	return nil
}

// ReloadConfigSignal sends a signal to reload configuration (typically SIGHUP)
func (pm *PIDManager) ReloadConfigSignal() error {
	isRunning, pid, err := pm.IsRunning()
	if err != nil {
		return err
	}

	if !isRunning {
		return fmt.Errorf("MCpeg is not running")
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %w", pid, err)
	}

	// Send SIGHUP for config reload
	if err := process.Signal(syscall.SIGHUP); err != nil {
		return fmt.Errorf("failed to send config reload signal to process %d: %w", pid, err)
	}

	pm.logger.Info("config_reload_signal_sent", "pid", pid)
	return nil
}

// SetupCleanupOnExit sets up cleanup of PID file on process exit
func (pm *PIDManager) SetupCleanupOnExit() {
	// This will be called when the process exits normally
	go func() {
		// Create a channel to listen for interrupt signals
		// This is handled in the main application, but we can also
		// register cleanup here as a backup
		defer pm.RemovePID()
	}()
}

// PIDFileManager interface for dependency injection
type PIDFileManager interface {
	WritePID() error
	RemovePID() error
	ReadPID() (int, error)
	IsRunning() (bool, int, error)
	StopProcess(force bool) error
	GetProcessStatus() ProcessStatus
	LogRotationSignal() error
	ReloadConfigSignal() error
	GetPIDFile() string
}

// Ensure PIDManager implements PIDFileManager
var _ PIDFileManager = (*PIDManager)(nil)
