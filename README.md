# MCpeg
## Model Context Protocol Enablement Gateway

<div align="center">
  <img src="assets/logo.svg" alt="MCpeg Logo" width="200"/>
  
  **Pronounced "MC peg" • The Peg That Connects Model Contexts**
</div>

> ⚠️ **EXPERIMENTAL**: MCpeg is experimental software under heavy development. Built using the [XVC (Extreme Vibe Coding)](https://github.com/osakka/xvc) framework for rapid human-LLM collaboration. APIs and functionality may change significantly. Not recommended for production use.

**MCpeg** is a gateway service that provides a Model Context Protocol (MCP) API on one side and integrates with external services via API calls or binary invocations on the other side. Like a peg that connects different pieces, MCpeg bridges the gap between MCP-compliant clients and diverse backend services.

## Overview

**MCpeg** acts as a bridge between MCP-compliant clients and various backend services, providing:
- **MCP Protocol Support** - Model Context Protocol implementation
- **Gateway Architecture** - Request routing with load balancing capabilities
- **Service Discovery** - DNS, Consul, Kubernetes, and static configuration support
- **Core Features** - Circuit breaker patterns, rate limiting, compression, metrics
- **Admin API** - RESTful endpoints for gateway management and monitoring
- **Service Integration** - REST APIs, binary calls, and capability detection
- **YAML Configuration** - Configuration system with environment variable overrides
- **API-First Development** - Generated code from MCP specifications
- **Observability** - Prometheus metrics, structured logging, health checks
- **Plugin System** - MCP Plugin Integration with specialized endpoints
  - **Plugin Discovery** - Plugin discovery with capability analysis and dependency resolution
  - **Inter-Plugin Communication** - Message passing and event bus between plugins
  - **Hot Reloading** - Plugin updates with operation tracking and rollback
  - **Built-in Services** - Memory, Git, and Editor services with extensible architecture
- **Daemon Process Management** - Daemon support with PID files, signal handling, and process control
- **System Integration** - Systemd service files, management scripts, and installation

## Project Structure

See [Project Structure Guide](docs/architecture/project-structure.md) for detailed layout.

- `/cmd` - Application entry points
- `/internal` - Private application code  
- `/pkg` - Public Go packages
- `/api` - OpenAPI specifications and generated schemas
- `/build` - Build artifacts (runtime data, binaries)
- `/assets` - Logo and branding assets
- `/docs` - All documentation
  - `/adrs` - Architecture Decision Records
  - `/architecture` - System design documents
  - `/development` - Development guides (XVC, structure, etc.)
  - `/guidelines` - Coding and process guidelines

## Development Methodology: XVC Framework

This project follows the [XVC (Extreme Vibe Coding)](https://github.com/osakka/xvc) principles for human-LLM collaboration. See our [XVC Methodology Guide](docs/guidelines/xvc-methodology.md) for details.

### Core XVC Principles Applied:

1. **Single Source of Truth**: All API definitions derive from official MCP specifications
2. **No Redundancy**: Each piece of information exists in exactly one place  
3. **Surgical Precision**: Every change is intentional and well-documented
4. **Bar-Raising Solutions**: Only implement patterns that improve the overall system
5. **Forward Progress Only**: No regression, always building on solid foundations
6. **Always Solve Never Mask**: Address root causes, not symptoms

### Additional Development Principles:

- **API-First**: Define APIs before implementation
- **Code Generation**: Generate code from schemas to ensure consistency
- **LLM-Optimized Logging**: Every log entry contains complete context for troubleshooting
- **100% Observability**: An LLM can understand system state from logs alone

## Project Status

**Current Phase**: Active Development

Major features implemented:
- ✅ **Core Implementation** - Basic gateway functionality with MCP protocol support
- ✅ **Gateway Features** - Load balancing, service discovery, circuit breakers, rate limiting
- ✅ **Admin API** - RESTful endpoints for gateway management
- ✅ **Observability** - Prometheus metrics, structured logging, health monitoring
- ✅ **MCP Support** - Model Context Protocol implementation
- ✅ **Plugin System** - Memory, Git, and Editor services with extensible architecture
- ✅ **MCP Plugin Integration** - RBAC-enabled plugin access through MCP JSON-RPC API
- ✅ **Daemon Process Management** - Daemon with PID files, signal handling, systemd integration
- ✅ **Quality Standards** - Thread-safe operations, error handling, resource cleanup
- ✅ **Testing Infrastructure** - MCP test client with validation coverage
- 📋 All decisions documented in ADRs following XVC methodology
- 🔍 LLM-debuggable through comprehensive logging

**MCpeg** provides a solid foundation for connecting model contexts with backend services.

## Key Features

### 🚀 **Gateway Architecture**
- **Request Routing** - Load balancing with multiple strategies (round-robin, least-connections, weighted, hash-based)
- **Circuit Breaker Protection** - Failure detection and isolation with configurable thresholds
- **Health-Aware Load Balancing** - Success rate monitoring and service filtering
- **HTTP Middleware Stack** - Gzip compression, rate limiting, CORS, request logging, and panic recovery

### 🔍 **Service Discovery**
- **DNS Discovery** - SRV record lookups with multi-domain support
- **Consul Integration** - API integration with health filtering and metadata extraction
- **Kubernetes Integration** - API with RBAC authentication and label selectors
- **Static Configuration** - File-based service definitions with capability detection
- **Auto-Registration** - Discovered services register with the gateway

### 📊 **Observability**
- **Prometheus Metrics** - Metrics for HTTP requests, services, load balancer, health, and system resources
- **Structured Logging** - LLM-optimized logs with context for troubleshooting
- **Health Endpoints** - Liveness, readiness, and health status checking
- **Admin API** - RESTful endpoints for monitoring, configuration, and management
- **Security Features** - API key authentication for admin endpoints with audit logging
- **Testing Coverage** - Test suite for plugin system, authentication, and service integration
- **Standardized Architecture** - Centralized path management and flag processing

### ⚙️ **Configuration**
- **YAML Configuration** - Configuration loading with environment variable overrides
- **Hot Configuration Updates** - Runtime configuration changes via Admin API
- **Security Controls** - Validation and sanitization for configuration updates
- **Environment-Specific Configs** - Development and production configuration profiles

## Getting Started

> **Note**: This software is under active development. APIs may evolve as features are refined and extended.

### Prerequisites

- Go 1.21 or later
- Docker (optional, for containerized deployment)

### Building

MCPEG uses a **single source of truth** build system. All build configuration is centralized in `scripts/build.sh`:

```bash
# Using Make (delegates to build script)
make build

# Or use the build script directly
./scripts/build.sh build
```

Available build commands:

```bash
# Core building
make build          # Build for current platform
make build-dev      # Development build (faster)
make build-prod     # Cross-compile for all platforms

# Development
make dev            # Start development server
make test           # Run tests
make validate       # Validate OpenAPI specs
make fmt            # Format code

# Release
make release        # Create release archives
make clean          # Clean build artifacts

# Get help
make help           # Show all available commands
```

### Running

Start the gateway:

```bash
# Development mode
make dev

# Or run the binary directly
./build/mcpeg gateway -dev
```

Generate code from OpenAPI specs:

```bash
make validate       # Validate OpenAPI specification
make generate       # Generate Go code from specs

# Or use the unified binary directly
./build/mcpeg codegen -spec-file api/openapi/mcp-gateway.yaml -output internal/generated
./build/mcpeg validate -spec-file api/openapi/mcp-gateway.yaml
```

### Build Artifacts

All build artifacts are placed in the `build/` directory:
- `build/mcpeg` - Unified **MCpeg** binary with gateway and codegen functionality
- `build/release/` - Release archives for distribution

The build system follows the **single source of truth** principle:
- All build configuration is in `scripts/build.sh`
- Makefile delegates to the build script
- No duplication of build logic

## Quick Start

### Development Mode
```bash
# Build the binary
make build

# Run in development mode
./build/mcpeg --dev

# Check status
./build/mcpeg --status
```

### Production Daemon Mode
```bash
# Start as daemon
./build/mcpeg --daemon

# Control daemon
./build/mcpeg --stop
./build/mcpeg --restart
./build/mcpeg --status --verbose

# Log rotation
./build/mcpeg --log-rotate
```

### System Service Installation
```bash
# Install as systemd service
sudo ./scripts/install-service.sh

# Control via systemd
sudo systemctl start mcpeg
sudo systemctl enable mcpeg
sudo systemctl status mcpeg

# View logs
journalctl -u mcpeg -f
```

### Management Scripts
```bash
# Using management scripts
./scripts/mcpeg-start.sh
./scripts/mcpeg-stop.sh
./scripts/mcpeg-restart.sh
./scripts/mcpeg-status.sh --verbose --logs
```

## Contributing

This project uses XVC methodology. When contributing:
1. Ensure changes align with XVC principles
2. Maintain single source of truth
3. Document decisions in ADRs
4. Write LLM-optimized logs
5. Never mask problems - solve root causes

## License

MIT License - see [LICENSE](LICENSE) file for details.