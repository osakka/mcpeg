# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial project structure
- Research and analysis of MCP protocol specifications
- Documentation framework with ADRs and guidelines
- API-first development approach
- Comprehensive service registry with load balancing and circuit breaker patterns
- MCP JSON-RPC router with method-specific endpoint handling
- Gateway server with HTTP middleware stack and graceful shutdown
- OpenAPI-based code generation for Go types and handlers
- Unified `mcpeg` binary with subcommands (gateway, codegen, validate, version, help)
- Single source of truth build system with centralized `scripts/build.sh`
- **Plugin System Architecture**: Complete plugin framework with Memory, Git, and Editor services
- **Daemon Process Management**: Full daemon support with PID files, signal handling, and process control
- **Production Logging**: File logging with rotation, compression, and structured output
- **System Integration**: Systemd service files, management scripts, and installation automation
- **Process Control**: Built-in daemon commands (--daemon, --stop, --restart, --status, --log-rotate)
- Professional CLI interface following modern best practices
- Cross-platform build support for Linux, macOS, and Windows
- Comprehensive health checking and metrics collection
- LLM-optimized structured logging framework

#### Major Feature Implementation (Production-Ready)
- **YAML Configuration System**: Advanced loader with environment variable overrides and validation interfaces
- **MCP Response Validation**: Complete type-specific validation for all MCP 2025-03-26 specification response types
- **Real HTTP Health Checks**: Actual HTTP client implementation with authentication and circuit breaker integration
- **Production Prometheus Metrics**: Comprehensive metrics endpoint with HTTP, service, MCP, health, system, and business metrics
- **HTTP Middleware Stack**: 
  - Gzip compression with intelligent content-type detection and performance metrics
  - Rate limiting with sliding window algorithm and per-client tracking
  - CORS, recovery, and comprehensive request/response logging
- **Advanced Load Balancer**: 
  - Multiple strategies (round-robin, least-connections, weighted, hash-based, random)
  - Circuit breaker pattern with automatic failure detection
  - Health-aware routing with success rate monitoring
  - Real-time request tracking and latency measurement
- **Complete Service Discovery**:
  - DNS discovery with SRV record lookups and multi-domain support
  - Consul integration with full API integration and health filtering
  - Kubernetes integration with native API and RBAC authentication
  - Static configuration with endpoint parsing and metadata support
  - Automatic service registration with capability probing
- **Comprehensive Admin API**: 22 RESTful endpoints for complete gateway management
  - Service management (list, register, health, capabilities, statistics)
  - Discovery control (manual triggers, status, discovered services)
  - Load balancer management (statistics, circuit breaker control, strategies)
  - Configuration management (view, update, reload with validation)
  - System monitoring (runtime info, memory profiling, goroutine debugging)
  - Self-documenting API with built-in documentation endpoint

#### Branding and Identity
- **Official Logo**: Clean SVG logo representing the "peg" concept with connection visualization
- **Brand Consistency**: Clarified pronunciation as "MC peg" with "MCpeg" spelling throughout documentation
- **Visual Identity**: Logo incorporates hexagonal peg shape with connection lines showing gateway functionality

### Changed
- Migrated from separate `gateway` and `codegen` binaries to unified `mcpeg` binary
- Updated build system to use single source of truth pattern
- Improved CLI user experience with consistent subcommand interface
- Module path changed from `github.com/yourusername/mcpeg` to `github.com/osakka/mcpeg`

### Deprecated
- N/A

### Removed
- Separate binary builds for gateway and codegen (consolidated into unified binary)

### Fixed
- Import path corrections throughout codebase
- Interface implementation mismatches in metrics and logging
- Build script shell expansion issues with LDFLAGS
- Missing version parameter in HealthManager constructor
- Replaced all placeholder implementations with production-ready code
- JSON response handling with proper error management and content types
- Thread-safe operations across all components with proper mutex usage
- Memory management and resource cleanup in all background processes
- **Plugin Registration System**: Extended URL validation to accept `plugin://` scheme for internal plugin endpoints
- **Service Registry Integration**: Added plugin-aware health check bypass to prevent HTTP health checks on plugin URLs
- **Plugin Service Discovery**: All three built-in plugins (Memory, Git, Editor) now register successfully with service registry

### Security
- Circuit breaker pattern implementation for service protection
- Request validation framework with comprehensive error handling
- Secure default configurations for TLS and CORS
- **Admin API Authentication**: API key-based authentication for admin endpoints with configurable headers
- **TLS Configuration Management**: Fixed development mode TLS configuration loading and flag parsing
- **Comprehensive Plugin Testing**: Complete test coverage for plugin system security and functionality
- **Path Standardization**: Centralized path management eliminating hardcoded paths throughout codebase
- **Flag Normalization**: Standardized command-line flag processing with single source of truth architecture
- **MCP Plugin Integration Phase 1**: Complete RBAC-enabled plugin integration with MCP protocol
  - JWT authentication with RSA signature validation
  - Plugin-level access control with granular permissions
  - Direct plugin method invocation bypassing HTTP overhead
  - Unified MCP JSON-RPC gateway with plugin-aware routing
  - Automatic capability aggregation from accessible plugins
  - Comprehensive error handling and retry logic