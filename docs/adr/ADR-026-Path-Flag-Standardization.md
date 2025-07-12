# ADR-026: Path and Flag Standardization

## Status
**ACCEPTED** - *2025-07-12*

## Context
During comprehensive codebase analysis, we identified significant inconsistencies in path management and command-line flag usage across MCpeg. The system had hardcoded paths scattered throughout modules, inconsistent flag processing, and lacked a single source of truth for file system management. This created maintenance challenges and violated the principle of centralized configuration management.

Key issues included:
- Hardcoded paths in multiple modules (`./mcpeg.pid`, `./logs/`, `./data/`, `.cache/openapi`)
- Mixed flag usage patterns and inconsistent processing
- Backward compatibility wrappers creating multiple sources of truth
- Different modules using different default path conventions

## Decision
We implemented complete path and flag standardization with zero backward compatibility:

1. **Centralized Path Management**: Created `pkg/paths/paths.go` as single source of truth
2. **Build Directory Consolidation**: All runtime artifacts moved to `build/` hierarchy
3. **Flag Normalization**: Standardized all command-line flag processing
4. **Eliminated Backward Compatibility**: Removed all compatibility wrappers

## Implementation Details

### Centralized Path Architecture
```go
// pkg/paths/paths.go - Single source of truth
type PathConfig struct {
    BuildDir    string `yaml:"build_dir"`
    DataDir     string `yaml:"data_dir"`
    LogsDir     string `yaml:"logs_dir"`
    RuntimeDir  string `yaml:"runtime_dir"`
    CacheDir    string `yaml:"cache_dir"`
    PIDFile     string `yaml:"pid_file"`
    LogFile     string `yaml:"log_file"`
    MemoryDataFile string `yaml:"memory_data_file"`
}

func DefaultPaths() *PathConfig {
    buildDir := "build"
    return &PathConfig{
        BuildDir:    buildDir,
        DataDir:     filepath.Join(buildDir, "data"),
        LogsDir:     filepath.Join(buildDir, "logs"),
        RuntimeDir:  filepath.Join(buildDir, "runtime"),
        CacheDir:    filepath.Join(buildDir, "cache"),
        PIDFile:     filepath.Join(buildDir, "runtime", "mcpeg.pid"),
        LogFile:     filepath.Join(buildDir, "logs", "mcpeg.log"),
        MemoryDataFile: filepath.Join(buildDir, "data", "memory_storage.json"),
    }
}
```

### Build Directory Structure
```
build/
├── mcpeg                # Main binary
├── data/                # Persistent data (memory service)
├── logs/                # Log files
├── runtime/             # PID files, sockets, temp files
├── cache/               # Cache files (OpenAPI, etc.)
└── generated/           # Generated code output
```

### Flag Standardization
```go
// Proper Go flag syntax (no dashes in flag names)
flagSet.StringVar(&app.configFile, "config", paths.GetDefaultConfigPath(), "Path to configuration file")
flagSet.BoolVar(&app.daemon, "daemon", false, "Run in daemon mode")
flagSet.StringVar(&app.pidFile, "pid-file", paths.GetDefaultPIDFile(), "Path to PID file")

// Go's flag package automatically supports both -flag and --flag usage
```

### Backward Compatibility Removal
```go
// Before (pkg/config/paths.go):
func GetDefaultConfigPath() string {
    return paths.GetDefaultConfigPath() // Wrapper function
}

// After (pkg/config/paths.go):
// Note: Path functions have been moved to pkg/paths/paths.go
// No backward compatibility - use paths package directly
```

### Module Updates
- **Memory Service**: `pkg/plugins/memory_service.go` uses `paths.DefaultPaths().GetDataDir()`
- **OpenAPI Parser**: `pkg/codegen/openapi_parser.go` uses `"build/cache/openapi"`
- **Process Management**: `pkg/process/` modules reference centralized path functions
- **Main Binary**: `cmd/mcpeg/main.go` imports `pkg/paths` directly

## Consequences

### Positive
- **Single Source of Truth**: All path management centralized in one module
- **No Hardcoded Paths**: Eliminated all scattered path definitions
- **Consistent Flag Processing**: Standardized command-line interface
- **Maintainable Architecture**: Changes to paths require updates in one location only
- **Clean Build Structure**: All artifacts organized under `build/` directory
- **Zero Divergence**: No backward compatibility means no conflicting implementations

### Negative
- **Breaking Change**: Existing scripts/tools need to update to new paths
- **Migration Required**: Old installations need to move files to new structure
- **No Gradual Migration**: Complete cutover required due to no backward compatibility

## Files Modified
- `pkg/paths/paths.go`: New centralized path management module
- `pkg/config/paths.go`: Removed backward compatibility wrappers
- `cmd/mcpeg/main.go`: Updated to use centralized paths and proper flag syntax
- `pkg/plugins/memory_service.go`: Updated to use centralized data directory
- `pkg/codegen/openapi_parser.go`: Updated cache path to use build directory
- `pkg/process/daemon.go`: Removed hardcoded path functions
- `pkg/process/pid.go`: Removed hardcoded path functions
- `config/production.yaml`: Updated TLS certificate paths
- `scripts/mcpeg-start.sh`: Updated to use build directory structure

## Testing
- **Flag Processing**: Verified both `-flag` and `--flag` syntax work correctly
- **Path Resolution**: Confirmed all modules use centralized path configuration
- **Daemon Operation**: Tested daemon start/stop with new path structure
- **Build Process**: Verified all artifacts output to correct build directories
- **Health Checks**: Confirmed service responds correctly with new configuration

## References
- [Path Management Documentation](../paths.md)
- [Command Line Interface Guide](../cli.md)
- [Build System Documentation](../build.md)
- [Single Source of Truth Principles](../architecture/principles.md)