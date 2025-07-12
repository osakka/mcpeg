# ADR-023: Development Configuration and TLS Management

## Status
**ACCEPTED** - *2025-07-12*

## Context
During the comprehensive codebase sweep, we identified critical issues with TLS configuration management in development mode. The development mode flag (`--dev`) was not being properly parsed, preventing the TLS disable functionality from working correctly. This created confusion for developers who expected TLS to be automatically disabled in development mode but encountered connection errors when TLS certificates were not available.

The core problem was in the `runGateway()` function which didn't accept command-line arguments, causing flag parsing to fail silently and preventing development mode overrides from being applied.

## Decision
We implemented a comprehensive fix for development configuration and TLS management:

1. **Fixed Flag Parsing**: Modified `runGateway()` to accept arguments and use `flag.NewFlagSet` for proper flag parsing
2. **Enhanced Dev Mode Overrides**: Extended `applyDevModeOverrides()` to explicitly disable TLS in development mode
3. **Configuration Validation**: Ensured TLS settings are properly applied based on the parsed development flag

## Implementation Details

### Flag Parsing Fix
```go
// cmd/mcpeg/main.go
func runGateway(args []string) error {
    flagSet := flag.NewFlagSet("gateway", flag.ExitOnError)
    devMode := flagSet.Bool("dev", false, "Enable development mode")
    configFile := flagSet.String("config", "", "Configuration file path")
    
    if err := flagSet.Parse(args); err != nil {
        return fmt.Errorf("failed to parse flags: %w", err)
    }
    // ...
}
```

### TLS Disable in Development Mode
```go
// internal/config/loader.go
func applyDevModeOverrides(app *Config) {
    if app.Development.Enabled {
        // Disable TLS for development mode
        app.gatewayConfig.Server.TLS.Enabled = false
        // Other development overrides...
    }
}
```

### Configuration Structure
- **Production**: TLS enabled by default with certificate paths
- **Development**: TLS explicitly disabled with empty API keys for testing

## Consequences

### Positive
- Development mode now works correctly with `--dev` flag
- TLS is automatically disabled in development, preventing certificate errors
- Clear separation between development and production TLS configurations
- Improved developer experience with proper flag parsing
- Consistent configuration management across all deployment modes

### Negative
- Additional complexity in flag parsing logic
- Need to maintain separate TLS configurations for different environments
- Developers must be aware of TLS implications when switching between modes

## Files Modified
- `cmd/mcpeg/main.go`: Updated `runGateway()` function signature and flag parsing
- `internal/config/loader.go`: Enhanced `applyDevModeOverrides()` with TLS disable
- `dev-config.yaml`: Added TLS disabled configuration for development
- `config/production.yaml`: Maintained TLS enabled for production

## Testing
- Verified `--dev` flag properly disables TLS in development mode
- Tested flag parsing with various argument combinations
- Confirmed production mode maintains TLS enabled by default
- Validated configuration loading with environment variable overrides

## References
- [MCpeg Configuration Management](../configuration.md)
- [Development Setup Guide](../development.md)
- [ADR-015: Configuration Management](ADR-015-Configuration-Management.md)
- [ADR-020: Development Mode Enhancements](ADR-020-Development-Mode-Enhancements.md)