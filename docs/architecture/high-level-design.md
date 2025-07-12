# MCpeg High-Level Design

## System Architecture Overview

MCpeg is a production-ready Model Context Protocol (MCP) gateway that bridges MCP-compliant clients with diverse plugin-based services. The architecture follows XVC (Extreme Vibe Coding) principles with single source of truth, no redundancy, and surgical precision.

## Core Architecture Components

### 1. Unified Binary Architecture
```
┌─────────────────────────────────────┐
│           mcpeg Binary              │
├─────────────────────────────────────┤
│  gateway  │ codegen │ validate      │
│           │         │               │
│ Main      │ OpenAPI │ Spec          │
│ Server    │ Code    │ Validation    │
│           │ Gen     │               │
└─────────────────────────────────────┘
```

**Key Features**:
- Single executable with subcommand routing
- Consolidated from separate gateway/codegen binaries
- Consistent CLI interface: `mcpeg <subcommand>`

### 2. Gateway Server Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    MCpeg Gateway Server                     │
├─────────────────────────────────────────────────────────────┤
│  HTTP Server (Gorilla Mux)                                 │
│  ┌─────────────────┬─────────────────┬─────────────────┐   │
│  │   MCP Router    │  Admin API      │  Health/Metrics │   │
│  │                 │                 │                 │   │
│  │ JSON-RPC 2.0    │ 22 Endpoints    │ Prometheus      │   │
│  │ Method Routing  │ Service Mgmt    │ Health Checks   │   │
│  └─────────────────┴─────────────────┴─────────────────┘   │
├─────────────────────────────────────────────────────────────┤
│  Middleware Stack                                           │
│  ┌─────────┬─────────┬─────────┬─────────┬─────────────┐   │
│  │  CORS   │  Gzip   │  Rate   │ Logging │  Recovery   │   │
│  │         │ Comp.   │ Limit   │         │             │   │
│  └─────────┴─────────┴─────────┴─────────┴─────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### 3. Plugin System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Plugin Ecosystem                        │
├─────────────────────────────────────────────────────────────┤
│  Plugin Manager                                             │
│  ┌─────────────────┬─────────────────┬─────────────────┐   │
│  │  Memory Plugin  │   Git Plugin    │ Editor Plugin   │   │
│  │                 │                 │                 │   │
│  │ 5 tools         │ 8 tools         │ 7 tools         │   │
│  │ 2 resources     │ 2 resources     │ 2 resources     │   │
│  │ 2 prompts       │ 2 prompts       │ 2 prompts       │   │
│  └─────────────────┴─────────────────┴─────────────────┘   │
├─────────────────────────────────────────────────────────────┤
│  Service Registry Integration                               │
│  ┌─────────────────┬─────────────────┬─────────────────┐   │
│  │  Load Balancer  │ Health Checks   │ Circuit Breaker │   │
│  │                 │                 │                 │   │
│  │ Round Robin     │ Plugin-Aware    │ Failure         │   │
│  │ Least Conn      │ Bypass          │ Detection       │   │
│  │ Weighted        │                 │                 │   │
│  └─────────────────┴─────────────────┴─────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### 4. Phase 2 Intelligence System

```
┌─────────────────────────────────────────────────────────────┐
│            Phase 2 Advanced Plugin Intelligence            │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┬─────────────────┬─────────────────┐   │
│  │ Analysis Engine │Discovery Engine │Aggregation Eng  │   │
│  │                 │                 │                 │   │
│  │ Semantic Cat.   │ Dependency Res  │ Conflict Res    │   │
│  │ Quality Metrics │ Conflict Detect │ Provider Rank   │   │
│  │ Thread-Safe     │ Recommendations │ Cross-Plugin    │   │
│  │ RWMutex Sync    │                 │                 │   │
│  └─────────────────┴─────────────────┴─────────────────┘   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Validation Engine                      │   │
│  │                                                     │   │
│  │ Runtime Validation │ Policy Enforcement │ Monitoring│   │
│  │ 6 Rule Types       │ Auto Remediation   │ Trends    │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Data Flow Architecture

### 1. MCP Request Flow
```
Client Request (JSON-RPC 2.0)
    ↓
HTTP Server (Gorilla Mux)
    ↓
Middleware Stack (CORS, Gzip, Rate Limit, Logging)
    ↓
MCP Router (Method-specific routing)
    ↓
Plugin Handler (RBAC, JWT Auth)
    ↓
Plugin Manager (Direct method invocation)
    ↓
Individual Plugin (Memory/Git/Editor)
    ↓
Plugin Response (Structured)
    ↓
JSON-RPC Response
```

### 2. Phase 2 Intelligence Flow
```
Plugin Registration
    ↓
Analysis Engine (Semantic Categorization)
    ↓
Discovery Engine (Dependency Resolution)
    ↓
Aggregation Engine (Conflict Resolution)
    ↓
Validation Engine (Policy Enforcement)
    ↓
Service Registry (Health-aware routing)
```

## Key Architectural Patterns

### 1. Single Source of Truth
- **Build System**: Centralized in `scripts/build.sh`
- **Configuration**: YAML-based with environment overrides
- **Documentation**: Each piece of information exists once
- **ADRs**: Canonical decision records in `docs/adrs/`

### 2. Plugin-First Design
- **Built-in Plugins**: Memory, Git, Editor services
- **Unified Interface**: Consistent plugin contract
- **Service Registry**: Automatic plugin registration
- **MCP Exposure**: Direct plugin-to-MCP mapping

### 3. Thread-Safe Concurrency
- **RWMutex Synchronization**: All shared state protected
- **Concurrent Analysis**: Parallel capability processing
- **Race Condition Prevention**: Zero concurrent map writes
- **Defensive Programming**: Immutable return copies

### 4. Production-Ready Reliability
- **Circuit Breaker Pattern**: Automatic failure isolation
- **Health Monitoring**: Comprehensive health checking
- **Graceful Degradation**: Continue operation on partial failures
- **Comprehensive Logging**: LLM-optimized structured logs

## Component Integration Points

### 1. Plugin → Service Registry
```go
// Plugin registration with service registry
serviceRegistry.RegisterPlugin(pluginName, pluginInstance)

// Automatic capability exposure
tools := plugin.GetTools()
resources := plugin.GetResources()
prompts := plugin.GetPrompts()
```

### 2. MCP Router → Plugin Manager
```go
// Direct plugin method invocation
result, err := pluginManager.CallTool(ctx, toolName, args)

// Bypass HTTP overhead for internal plugins
response := plugin.ExecuteTool(toolName, parameters)
```

### 3. Phase 2 Intelligence → Plugin System
```go
// Capability analysis with thread safety
analysis := analysisEngine.AnalyzeCapability(ctx, pluginName, capability)

// Discovery with dependency resolution
result := discoveryEngine.DiscoverPlugin(ctx, pluginName)
```

## Configuration Architecture

### 1. Environment-Specific Configs
- `config/development.yaml` - Development settings
- `config/production.yaml` - Production settings
- `config/mcpeg.yaml` - Default configuration

### 2. Security Configuration
- `config/security/rbac_policies.yaml` - Access control
- `config/security/tls.yaml` - TLS settings
- `config/secrets/` - Secret management

## Testing Architecture

### 1. Automated Testing
- **Integration Tests**: `test/integration/test_mcp_client.js`
- **Protocol Compliance**: Full MCP JSON-RPC 2.0 validation
- **100% Success Rate**: All 32 capabilities verified

### 2. Interactive Testing
- **MCP Inspector**: Visual testing interface
- **Manual Validation**: cURL-based protocol testing
- **Real-world Testing**: Claude Desktop integration

## Performance Characteristics

### 1. Response Times (Local Testing)
- **Tool Discovery**: ~10-50ms
- **Tool Execution**: ~50-200ms (tool-dependent)
- **Resource Access**: ~5-20ms
- **Prompt Retrieval**: ~5-20ms

### 2. Concurrency
- **Thread-Safe Operations**: Zero race conditions
- **Concurrent Requests**: Unlimited concurrent handling
- **Plugin Isolation**: Per-request plugin execution
- **Resource Management**: Proper cleanup and memory management

## Security Architecture

### 1. Authentication & Authorization
- **JWT Token Validation**: RSA signature verification
- **RBAC Engine**: Role-based access control
- **Plugin-level Permissions**: Granular capability access
- **API Key Authentication**: Admin endpoint protection

### 2. Input Validation
- **JSON-RPC Validation**: Protocol compliance checking
- **Parameter Validation**: Type and constraint validation
- **Request Sanitization**: Input cleaning and validation
- **Response Validation**: Output format verification

## Deployment Architecture

### 1. Daemon Mode
- **Process Management**: PID files and signal handling
- **Systemd Integration**: Native service integration
- **Log Rotation**: Automatic log management
- **Graceful Shutdown**: Clean process termination

### 2. Monitoring & Observability
- **Prometheus Metrics**: Comprehensive metrics collection
- **Health Endpoints**: Liveness and readiness checks
- **Structured Logging**: LLM-optimized log format
- **Admin API**: 22 management endpoints

This architecture provides a robust, scalable, and maintainable foundation for enterprise MCP gateway deployments while maintaining the flexibility to adapt and extend as requirements evolve.