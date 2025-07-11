# ADR-021: Daemon Process Management Architecture

## Status
**ACCEPTED** - *2025-07-11*

## Context

MCpeg Gateway requires production-ready daemon capabilities for server deployments. The system needs to support background operation, process lifecycle management, logging, and system integration while maintaining the single binary approach.

## Decision

Implement comprehensive daemon process management with the following architecture:

### Core Components

1. **PID File Management** (`pkg/process/pid.go`)
   - Process detection and validation
   - Stale PID file cleanup
   - Signal-based process control
   - Graceful shutdown coordination

2. **Daemon Mode Support** (`pkg/process/daemon.go`)
   - Process forking for background operation
   - Terminal detachment
   - File descriptor redirection
   - Environment setup for daemon mode

3. **Production Logging** (`pkg/logging/file_logger.go`)
   - File-based logging with rotation
   - Compression and retention policies
   - Buffered writes for performance
   - Multi-writer support (console + file)

4. **System Integration**
   - Systemd service files with security hardening
   - Management scripts for common operations
   - Installation automation
   - Signal handling (SIGTERM, SIGINT, SIGHUP, SIGUSR1)

### Command Interface

All daemon functionality integrated into single `mcpeg` binary:

```bash
# Daemon mode
mcpeg --daemon

# Process control
mcpeg --stop
mcpeg --restart
mcpeg --status
mcpeg --log-rotate

# System service
systemctl start mcpeg
systemctl status mcpeg
```

### Design Principles

1. **Single Binary**: All functionality in one executable
2. **Unix Compliance**: Following daemon best practices
3. **Production Ready**: Proper signal handling, logging, monitoring
4. **Zero Dependencies**: Self-contained deployment
5. **Security First**: Process isolation and privilege separation

## Implementation Details

### PID Management
- Atomic PID file operations with race condition protection
- Process existence validation using signal(0)
- Automatic cleanup on normal and abnormal termination
- Support for custom PID file locations

### Daemon Process Flow
```
1. Parse --daemon flag
2. Fork child process
3. Parent exits (returns to shell)
4. Child detaches from terminal (setsid)
5. Redirect stdin/stdout/stderr
6. Write PID file
7. Setup signal handlers
8. Start main application
```

### Logging Architecture
- **Console Logging**: Development and foreground mode
- **File Logging**: Production daemon mode with rotation
- **Structured Output**: JSON format for log aggregation
- **Multi-Writer**: Simultaneous console and file output

### Signal Handling
- `SIGTERM/SIGINT`: Graceful shutdown
- `SIGHUP`: Configuration reload
- `SIGUSR1`: Log rotation
- `SIGKILL`: Emergency termination (external)

## Consequences

### Positive
- ✅ **Production Ready**: Full daemon lifecycle management
- ✅ **Standard Compliance**: Follows Unix daemon conventions
- ✅ **Easy Deployment**: Single binary, no complex setup
- ✅ **Operational Excellence**: Rich monitoring and control
- ✅ **System Integration**: Native systemd support

### Negative
- ⚠️ **Complexity**: Additional process management code
- ⚠️ **Platform Specific**: Some features require Unix-like systems
- ⚠️ **Testing**: Daemon behavior harder to test

### Mitigations
- Comprehensive unit and integration tests
- Fallback modes for non-Unix platforms
- Clear documentation and examples
- Management scripts for common operations

## Files Created/Modified

### New Files
```
pkg/process/pid.go              # PID file management
pkg/process/daemon.go           # Daemon mode implementation
pkg/logging/file_logger.go      # File logging with rotation
scripts/mcpeg.service           # Production systemd service
scripts/mcpeg-dev.service       # Development systemd service
scripts/mcpeg-start.sh          # Start management script
scripts/mcpeg-stop.sh           # Stop management script
scripts/mcpeg-restart.sh        # Restart management script
scripts/mcpeg-status.sh         # Status reporting script
scripts/install-service.sh      # System installation script
```

### Modified Files
```
cmd/gateway/main.go             # Daemon flag handling and integration
internal/server/gateway_server.go  # Server lifecycle integration
```

## Alternatives Considered

### 1. External Process Manager
Use systemd, supervisor, or similar external tools.
- **Rejected**: Adds deployment complexity and external dependencies

### 2. Separate Daemon Binary
Create distinct `mcpegd` daemon binary.
- **Rejected**: Violates single binary principle, increases maintenance

### 3. Library-Only Approach
Provide daemon functionality as library only.
- **Rejected**: Poor user experience, requires custom integration

## Implementation Status

- ✅ PID file management implemented
- ✅ Daemon mode with process forking
- ✅ File logging with rotation and compression
- ✅ Signal handling for all major signals
- ✅ Systemd service files with security hardening
- ✅ Management scripts with error handling
- ✅ Installation automation
- ✅ Integration with main binary

## References

- [Unix Daemon Programming](https://man7.org/linux/man-pages/man7/daemon.7.html)
- [Systemd Service Files](https://www.freedesktop.org/software/systemd/man/systemd.service.html)
- [Go Process Management](https://pkg.go.dev/os/exec)
- [Signal Handling Best Practices](https://man7.org/linux/man-pages/man7/signal.7.html)

## Revision History

| Version | Date       | Changes                 | Author |
|---------|------------|-------------------------|---------|
| 1.0     | 2025-07-11 | Initial version         | Claude  |