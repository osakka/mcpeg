package process

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/osakka/mcpeg/pkg/logging"
)

// DaemonConfig configures daemon mode
type DaemonConfig struct {
	PIDFile    string
	LogFile    string
	WorkingDir string
	User       string
	Group      string
	Umask      int
	Background bool
}

// DaemonManager handles process daemonization
type DaemonManager struct {
	config DaemonConfig
	logger logging.Logger
}

// NewDaemonManager creates a new daemon manager
func NewDaemonManager(config DaemonConfig, logger logging.Logger) *DaemonManager {
	return &DaemonManager{
		config: config,
		logger: logger.WithComponent("daemon_manager"),
	}
}

// Daemonize starts the process as a daemon
func (dm *DaemonManager) Daemonize() error {
	if !dm.config.Background {
		return nil // Not running in background mode
	}

	dm.logger.Info("starting_daemon_mode",
		"pid_file", dm.config.PIDFile,
		"log_file", dm.config.LogFile,
		"working_dir", dm.config.WorkingDir)

	// Fork the process
	return dm.forkProcess()
}

// forkProcess creates a daemon process using fork/exec
func (dm *DaemonManager) forkProcess() error {
	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Prepare command arguments (remove --daemon flag to avoid infinite loop)
	args := dm.filterDaemonArgs(os.Args[1:])

	// Create the command
	cmd := exec.Command(execPath, args...)

	// Set up daemon environment
	if err := dm.setupDaemonEnvironment(cmd); err != nil {
		return fmt.Errorf("failed to setup daemon environment: %w", err)
	}

	// Start the daemon process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon process: %w", err)
	}

	// Log successful daemon start
	dm.logger.Info("daemon_started",
		"daemon_pid", cmd.Process.Pid,
		"executable", execPath)

	// Exit parent process
	os.Exit(0)
	return nil // This line will never be reached
}

// setupDaemonEnvironment configures the daemon process environment
func (dm *DaemonManager) setupDaemonEnvironment(cmd *exec.Cmd) error {
	// Set working directory
	if dm.config.WorkingDir != "" {
		cmd.Dir = dm.config.WorkingDir
	}

	// Set up file descriptors for daemon
	if err := dm.setupDaemonFiles(cmd); err != nil {
		return err
	}

	// Set process attributes
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Create new session (detach from terminal)
	}

	// Set environment variables
	cmd.Env = append(os.Environ(), "MCPEG_DAEMON=1")

	return nil
}

// setupDaemonFiles configures stdin/stdout/stderr for daemon
func (dm *DaemonManager) setupDaemonFiles(cmd *exec.Cmd) error {
	// Redirect stdin to /dev/null
	devNull, err := os.Open("/dev/null")
	if err != nil {
		return fmt.Errorf("failed to open /dev/null: %w", err)
	}
	cmd.Stdin = devNull

	// Set up stdout and stderr
	if dm.config.LogFile != "" {
		// Redirect to log file
		logFile, err := dm.openLogFile()
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	} else {
		// Redirect to /dev/null
		cmd.Stdout = devNull
		cmd.Stderr = devNull
	}

	return nil
}

// openLogFile opens the log file for daemon output
func (dm *DaemonManager) openLogFile() (*os.File, error) {
	// Create log directory if it doesn't exist
	logDir := filepath.Dir(dm.config.LogFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory %s: %w", logDir, err)
	}

	// Open log file with append mode
	logFile, err := os.OpenFile(dm.config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", dm.config.LogFile, err)
	}

	return logFile, nil
}

// filterDaemonArgs removes daemon-specific flags from command arguments
func (dm *DaemonManager) filterDaemonArgs(args []string) []string {
	var filtered []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Skip daemon-related flags
		if arg == "--daemon" || arg == "-daemon" {
			continue
		}

		// Skip flag with value
		if arg == "--pid-file" || arg == "--log-file" {
			i++ // Skip next argument (the value)
			continue
		}

		// Skip combined flag=value format
		if len(arg) > 11 && arg[:11] == "--pid-file=" {
			continue
		}
		if len(arg) > 11 && arg[:11] == "--log-file=" {
			continue
		}

		filtered = append(filtered, arg)
	}

	// Add updated flags for daemon process
	if dm.config.PIDFile != "" {
		filtered = append(filtered, "--pid-file", dm.config.PIDFile)
	}
	if dm.config.LogFile != "" {
		filtered = append(filtered, "--log-file", dm.config.LogFile)
	}

	return filtered
}

// IsDaemon checks if the current process is running as a daemon
func IsDaemon() bool {
	return os.Getenv("MCPEG_DAEMON") == "1"
}

// DetachFromTerminal detaches the current process from the terminal
func (dm *DaemonManager) DetachFromTerminal() error {
	// Change to root directory to avoid holding any directory
	if dm.config.WorkingDir != "" {
		if err := os.Chdir(dm.config.WorkingDir); err != nil {
			return fmt.Errorf("failed to change working directory to %s: %w", dm.config.WorkingDir, err)
		}
	} else {
		if err := os.Chdir("/"); err != nil {
			return fmt.Errorf("failed to change to root directory: %w", err)
		}
	}

	// Set umask
	if dm.config.Umask > 0 {
		syscall.Umask(dm.config.Umask)
	} else {
		syscall.Umask(0) // Default: no restrictions
	}

	return nil
}

// ValidateDaemonConfig validates the daemon configuration
func ValidateDaemonConfig(config DaemonConfig) error {
	// Validate PID file
	if config.PIDFile != "" {
		if err := ValidatePIDFile(config.PIDFile); err != nil {
			return fmt.Errorf("invalid PID file: %w", err)
		}
	}

	// Validate log file
	if config.LogFile != "" {
		logDir := filepath.Dir(config.LogFile)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("cannot create log directory %s: %w", logDir, err)
		}

		// Check if we can write to log file
		testFile := config.LogFile + ".test"
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			return fmt.Errorf("cannot write to log file %s: %w", config.LogFile, err)
		}
		os.Remove(testFile) // Clean up
	}

	// Validate working directory
	if config.WorkingDir != "" {
		if stat, err := os.Stat(config.WorkingDir); err != nil {
			return fmt.Errorf("working directory %s does not exist: %w", config.WorkingDir, err)
		} else if !stat.IsDir() {
			return fmt.Errorf("working directory %s is not a directory", config.WorkingDir)
		}
	}

	return nil
}

// GetDefaultDaemonConfig returns default daemon configuration
// Note: PIDFile and LogFile should be set by caller using config.GetDefaultPIDFile() and config.GetDefaultLogFile()
func GetDefaultDaemonConfig() DaemonConfig {
	return DaemonConfig{
		PIDFile:    "build/runtime/mcpeg.pid",
		LogFile:    "build/logs/mcpeg.log",
		WorkingDir: "",
		User:       "",
		Group:      "",
		Umask:      0,
		Background: false,
	}
}

// Note: GetDefaultLogFile has been moved to pkg/config/paths.go to centralize path management

// SetupSignalHandlers sets up signal handlers for daemon mode
func (dm *DaemonManager) SetupSignalHandlers(pidManager *PIDManager) {
	// This would set up additional signal handlers specific to daemon mode
	// like SIGHUP for config reload, SIGUSR1 for log rotation, etc.

	dm.logger.Info("daemon_signal_handlers_setup")
}

// DaemonStatus represents the daemon status
type DaemonStatus struct {
	Mode       string `json:"mode"` // "foreground" or "daemon"
	PIDFile    string `json:"pid_file"`
	LogFile    string `json:"log_file"`
	WorkingDir string `json:"working_dir"`
}

// GetDaemonStatus returns the current daemon status
func (dm *DaemonManager) GetDaemonStatus() DaemonStatus {
	mode := "foreground"
	if IsDaemon() {
		mode = "daemon"
	}

	workingDir, _ := os.Getwd()

	return DaemonStatus{
		Mode:       mode,
		PIDFile:    dm.config.PIDFile,
		LogFile:    dm.config.LogFile,
		WorkingDir: workingDir,
	}
}

// DaemonInterface defines the daemon manager interface
type DaemonInterface interface {
	Daemonize() error
	DetachFromTerminal() error
	SetupSignalHandlers(pidManager *PIDManager)
	GetDaemonStatus() DaemonStatus
}

// Ensure DaemonManager implements DaemonInterface
var _ DaemonInterface = (*DaemonManager)(nil)
