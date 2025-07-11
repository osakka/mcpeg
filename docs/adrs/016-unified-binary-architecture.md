# ADR-016: Unified Binary Architecture

## Status

Accepted

## Context

Previously, MCPEG was designed to build two separate binaries:
- `gateway` - The main MCP gateway server
- `codegen` - OpenAPI code generation tool

This design created several issues:
1. **Deployment Complexity**: Users needed to manage two separate binaries
2. **Distribution Overhead**: Two binaries to package, version, and distribute
3. **User Experience**: Inconsistent CLI interfaces and separate help systems
4. **Build Complexity**: Multiple build targets and release configurations
5. **Violates Single Source of Truth**: Two entry points for related functionality

The user identified this as a violation of the "single source of truth" principle, questioning why there should be separate binaries when the functionality could be unified under subcommands.

## Decision

Implement a unified `mcpeg` binary with subcommands for different functionality:

```bash
mcpeg gateway [options]     # Start MCP gateway server
mcpeg codegen [options]     # Generate Go code from OpenAPI specs  
mcpeg validate [options]    # Validate OpenAPI specifications
mcpeg version              # Show version information
mcpeg help                 # Show help information
```

### Implementation Details

1. **Unified Entry Point**: Create `cmd/mcpeg/main.go` as the single binary entry point
2. **Subcommand Architecture**: Use flag parsing to route to appropriate functionality
3. **Shared Components**: Reuse logging, metrics, and validation across subcommands
4. **Professional CLI**: Implement comprehensive help, version, and usage information
5. **Build System Update**: Modify build scripts to produce single `mcpeg` binary

### API Design

The unified binary maintains all existing functionality while providing a cleaner interface:
- Subcommands use their own flag sets for isolation
- Comprehensive help available at both global and subcommand levels
- Consistent error handling and logging across all subcommands
- Version information embedded at build time

## Consequences

### Positive

- **Single Source of Truth**: One binary to manage, deploy, and distribute
- **Better User Experience**: Professional CLI with logical subcommands
- **Simplified Build Process**: Single build target instead of multiple binaries
- **Easier Deployment**: One binary for all MCPEG functionality
- **Consistent Interface**: Unified help, version, and error handling
- **Modern CLI Best Practices**: Follows standard patterns like `docker`, `kubectl`, `git`
- **Reduced Maintenance**: Single codebase entry point to maintain

### Negative

- **Slightly Larger Binary**: Single binary contains all functionality (minimal impact)
- **Tight Coupling**: All functionality linked into one binary (acceptable trade-off)

### Neutral

- **Code Organization**: Requires careful organization of subcommand logic
- **Flag Namespace**: Need to manage flag conflicts between subcommands (mitigated by separate flag sets)

## Alternatives Considered

### Keep Separate Binaries
- **Rejected**: Violates single source of truth principle
- **Issues**: Deployment complexity, user confusion, maintenance overhead

### Plugin Architecture
- **Rejected**: Over-engineering for current scope
- **Issues**: Added complexity without clear benefits

### Shell Scripts Wrapper
- **Rejected**: Introduces platform dependencies and complexity
- **Issues**: Windows compatibility, error handling, version management

## References

- User feedback: "should those not be produced in a single binary with --gateway and --codegen flags"
- [XVC Methodology](https://github.com/osakka/xvc) - Single Source of Truth principle
- Modern CLI examples: Docker, Kubernetes kubectl, Git
- [ADR-003: API-First Development](003-api-first-development.md) - Related architectural principle
- [Build System Implementation](../../scripts/build.sh) - Updated build configuration