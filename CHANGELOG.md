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
- Professional CLI interface following modern best practices
- Cross-platform build support for Linux, macOS, and Windows
- Comprehensive health checking and metrics collection
- LLM-optimized structured logging framework

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

### Security
- Circuit breaker pattern implementation for service protection
- Request validation framework with comprehensive error handling
- Secure default configurations for TLS and CORS