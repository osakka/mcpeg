package paths

import (
	"os"
	"path/filepath"
)

// PathConfig centralizes all file system path configuration
type PathConfig struct {
	// Base directories
	BuildDir   string `yaml:"build_dir"`
	DataDir    string `yaml:"data_dir"`
	LogsDir    string `yaml:"logs_dir"`
	RuntimeDir string `yaml:"runtime_dir"`
	CacheDir   string `yaml:"cache_dir"`
	ConfigDir  string `yaml:"config_dir"`

	// Specific files
	PIDFile string `yaml:"pid_file"`
	LogFile string `yaml:"log_file"`

	// Service-specific paths
	MemoryDataFile string `yaml:"memory_data_file"`
}

// DefaultPaths returns the default path configuration with build/ as base
func DefaultPaths() *PathConfig {
	buildDir := "build"

	return &PathConfig{
		BuildDir:   buildDir,
		DataDir:    filepath.Join(buildDir, "data"),
		LogsDir:    filepath.Join(buildDir, "logs"),
		RuntimeDir: filepath.Join(buildDir, "runtime"),
		CacheDir:   filepath.Join(buildDir, "cache"),
		ConfigDir:  "config",

		PIDFile: filepath.Join(buildDir, "runtime", "mcpeg.pid"),
		LogFile: filepath.Join(buildDir, "logs", "mcpeg.log"),

		MemoryDataFile: filepath.Join(buildDir, "data", "memory_storage.json"),
	}
}

// SystemPaths returns system-wide paths for production deployment
func SystemPaths() *PathConfig {
	return &PathConfig{
		BuildDir:   "/opt/mcpeg/build",
		DataDir:    "/var/lib/mcpeg",
		LogsDir:    "/var/log/mcpeg",
		RuntimeDir: "/var/run/mcpeg",
		CacheDir:   "/var/cache/mcpeg",
		ConfigDir:  "/etc/mcpeg",

		PIDFile: "/var/run/mcpeg/mcpeg.pid",
		LogFile: "/var/log/mcpeg/mcpeg.log",

		MemoryDataFile: "/var/lib/mcpeg/memory_storage.json",
	}
}

// EnsureDirectories creates all required directories
func (p *PathConfig) EnsureDirectories() error {
	dirs := []string{
		p.BuildDir,
		p.DataDir,
		p.LogsDir,
		p.RuntimeDir,
		p.CacheDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

// GetPIDFile returns the PID file path with fallback logic
func (p *PathConfig) GetPIDFile() string {
	if p.PIDFile != "" {
		return p.PIDFile
	}

	// Try system location first, then build directory
	systemPID := "/var/run/mcpeg/mcpeg.pid"
	if isWritable(filepath.Dir(systemPID)) {
		return systemPID
	}

	return filepath.Join(p.BuildDir, "runtime", "mcpeg.pid")
}

// GetLogFile returns the log file path with fallback logic
func (p *PathConfig) GetLogFile() string {
	if p.LogFile != "" {
		return p.LogFile
	}

	// Try system location first, then build directory
	systemLog := "/var/log/mcpeg/mcpeg.log"
	if isWritable(filepath.Dir(systemLog)) {
		return systemLog
	}

	return filepath.Join(p.BuildDir, "logs", "mcpeg.log")
}

// GetDataDir returns the data directory path
func (p *PathConfig) GetDataDir() string {
	if p.DataDir != "" {
		return p.DataDir
	}
	return filepath.Join(p.BuildDir, "data")
}

// GetCacheDir returns the cache directory path
func (p *PathConfig) GetCacheDir() string {
	if p.CacheDir != "" {
		return p.CacheDir
	}
	return filepath.Join(p.BuildDir, "cache")
}

// isWritable checks if a directory is writable
func isWritable(dir string) bool {
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return false
	}

	testFile := filepath.Join(dir, ".writetest")
	file, err := os.Create(testFile)
	if err != nil {
		return false
	}
	file.Close()
	os.Remove(testFile)
	return true
}

// GetDefaultConfigPath returns the default configuration file path
func GetDefaultConfigPath() string {
	// Check for environment variable first
	if configPath := os.Getenv("MCPEG_CONFIG"); configPath != "" {
		return configPath
	}

	// Try system config first
	systemConfig := "/etc/mcpeg/production.yaml"
	if _, err := os.Stat(systemConfig); err == nil {
		return systemConfig
	}

	// Fall back to local config
	return "config/production.yaml"
}

// GetDefaultPIDFile returns the default PID file path
func GetDefaultPIDFile() string {
	return DefaultPaths().GetPIDFile()
}

// GetDefaultLogFile returns the default log file path
func GetDefaultLogFile() string {
	return DefaultPaths().GetLogFile()
}
