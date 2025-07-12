# MCpeg Project Structure

## Current Project Layout

```
mcpeg/
├── api/                     # API specifications
│   └── openapi/            # OpenAPI specification files
├── assets/                 # Project assets (logos, branding)
├── build/                  # Build artifacts and runtime data
│   └── runtime/           # Runtime files (PID, logs)
├── cmd/                    # Main applications
│   └── mcpeg/             # Unified mcpeg binary
│       └── main.go        # Entry point with subcommands
├── config/                 # Configuration files
│   ├── secrets/           # Secret management configs
│   ├── security/          # Security configurations (RBAC, TLS)
│   ├── services/          # Service definition files
│   ├── development.yaml   # Development configuration
│   ├── production.yaml    # Production configuration
│   └── mcpeg.yaml        # Default configuration
├── data/                   # Application data
│   ├── cache/             # Application cache
│   ├── logs/              # Application logs
│   └── runtime/           # Runtime data files
├── docs/                   # Documentation
│   ├── adrs/              # Architecture Decision Records
│   ├── architecture/      # System design documents
│   ├── guidelines/        # Development guidelines
│   └── testing/           # Testing documentation
├── internal/               # Private application code
│   ├── adapter/           # Service adapters
│   ├── config/            # Configuration management
│   ├── mcp/               # MCP protocol implementation
│   │   └── types/         # MCP type definitions
│   ├── plugins/           # Plugin integration system
│   │   └── data/          # Plugin data storage
│   ├── registry/          # Service registry implementation
│   ├── router/            # MCP routing implementation
│   └── server/            # Gateway server implementation
├── pkg/                    # Public packages
│   ├── auth/              # Authentication and authorization
│   ├── capabilities/      # Phase 2 intelligence system
│   ├── codegen/           # Code generation utilities
│   ├── concurrency/       # Concurrency management
│   ├── config/            # Configuration utilities
│   ├── context/           # Context propagation
│   ├── errors/            # Error handling framework
│   ├── health/            # Health checking system
│   ├── logging/           # LLM-optimized logging
│   ├── mcp/               # MCP protocol utilities
│   ├── metrics/           # Metrics collection
│   ├── paths/             # Path management utilities
│   ├── plugins/           # Plugin framework
│   │   ├── build/         # Plugin build artifacts
│   │   └── data/          # Plugin data
│   ├── process/           # Process management (daemon, PID)
│   ├── rbac/              # Role-based access control
│   ├── templates/         # Template processing
│   ├── transform/         # Data transformation
│   └── validation/        # Validation framework
├── scripts/                # Build and utility scripts
│   ├── build.sh           # Main build script (single source of truth)
│   ├── install-service.sh # System service installation
│   └── mcpeg-*.sh         # Management scripts
├── test/                   # Testing infrastructure
│   └── integration/       # Integration tests
│       ├── test_mcp_client.js        # Automated MCP test client
│       └── mcp-inspector-config.json # MCP Inspector configuration
├── trash/                  # Obsolete files (cleanup staging)
│   └── adr/               # Outdated ADR files
├── go.mod                  # Go module definition
├── go.sum                  # Go dependencies lock
├── Makefile               # Build automation (delegates to scripts/build.sh)
├── CHANGELOG.md           # Project changelog
├── CLAUDE.md              # AI assistant context and instructions
└── README.md              # Project overview
```

## Key Architecture Principles

### 1. Single Source of Truth
- **Build System**: All build logic centralized in `scripts/build.sh`
- **Configuration**: YAML-based with environment overrides
- **Documentation**: Each piece of information exists in exactly one place

### 2. Unified Binary Architecture
- **Single Executable**: `mcpeg` binary with subcommands (`gateway`, `codegen`, `validate`)
- **No Separate Binaries**: Consolidated from previous separate gateway/codegen binaries
- **Consistent Interface**: `mcpeg <subcommand>` pattern throughout

### 3. Module Organization
```go
// Correct import patterns
import (
    "github.com/osakka/mcpeg/pkg/logging"
    "github.com/osakka/mcpeg/internal/server"
    "github.com/osakka/mcpeg/pkg/capabilities"
)
```

### 4. Package Categories

#### `/cmd/` - Application Entry Points
- Contains main packages only
- Unified `mcpeg` binary with subcommand routing

#### `/internal/` - Private Application Code
- Cannot be imported by external projects
- Core gateway implementation
- MCP protocol handling
- Service registry and routing

#### `/pkg/` - Public Packages
- Can be imported by external projects
- Reusable components and utilities
- Framework-level functionality

#### `/config/` - Configuration Management
- Environment-specific configurations
- Security and RBAC policies
- Service definitions

#### `/test/` - Testing Infrastructure
- Integration tests
- MCP test clients
- Testing configurations

## Recent Structural Changes

### Phase 2 Intelligence System
Added comprehensive plugin discovery and intelligence:
- `pkg/capabilities/analysis_engine.go` - Thread-safe capability analysis
- `pkg/capabilities/discovery_engine.go` - Dynamic plugin discovery
- `pkg/capabilities/aggregation_engine.go` - Cross-plugin aggregation
- `pkg/capabilities/validation_engine.go` - Runtime validation

### Testing Infrastructure
Established production-ready testing:
- `test/integration/test_mcp_client.js` - Automated MCP test client
- `test/integration/mcp-inspector-config.json` - Inspector configuration
- `docs/testing/mcp-testing-methodology.md` - Testing documentation

### Documentation Consolidation
- Eliminated duplicate `docs/adr/` directory
- Maintained single source of truth in `docs/adrs/`
- Added comprehensive testing documentation

## Build and Development

### Standard Commands
```bash
# Build (single source of truth)
./scripts/build.sh build

# Development
make dev                    # Start development server
make test                   # Run tests
make validate              # Validate OpenAPI specs

# Production
make build-prod            # Cross-platform build
make release               # Create release artifacts
```

### File Organization Rules
1. **No duplication**: Each file has exactly one authoritative location
2. **Logical grouping**: Related functionality grouped by package
3. **Clear boundaries**: Public vs private interfaces well-defined
4. **Version control**: All source files under git, build artifacts ignored

## Integration Points

### Plugin System
- Built-in plugins: Memory, Git, Editor
- Plugin registration in service registry
- MCP endpoint exposure through unified API

### MCP Protocol
- JSON-RPC 2.0 compliance
- Method-specific routing
- Comprehensive validation

### Phase 2 Intelligence
- Thread-safe concurrent analysis
- Intelligent capability discovery
- Automated conflict resolution

This structure supports the XVC (Extreme Vibe Coding) methodology with single source of truth, no redundancy, and surgical precision in all changes.