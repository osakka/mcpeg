# ADR-020: Plugin System Foundation Architecture

## Status
**ACCEPTED** - *2025-07-12*

## Context

MCpeg Gateway required a comprehensive plugin system to extend functionality beyond core MCP protocol handling. The system needed to support dynamic service registration, lifecycle management, and seamless integration with the existing gateway infrastructure while maintaining the single binary architecture.

Key requirements identified:
- Dynamic plugin loading and lifecycle management
- Service registry integration for automatic discovery
- Multiple plugin types (editor, git, memory services)
- Hot reloading capabilities for production flexibility
- Comprehensive error handling and isolation
- Performance optimization for high-frequency operations

## Decision

We implemented a foundational plugin system architecture with the following core components and design principles:

### Plugin Architecture Framework

#### 1. **Plugin Interface Standardization**
```go
// pkg/plugins/plugin.go - Core plugin interface
type Plugin interface {
    Initialize(ctx context.Context, config map[string]interface{}) error
    GetMetadata() PluginMetadata
    HandleRequest(ctx context.Context, request interface{}) (interface{}, error)
    Shutdown(ctx context.Context) error
    HealthCheck(ctx context.Context) error
}

type PluginMetadata struct {
    Name        string            `json:"name"`
    Version     string            `json:"version"`
    Type        PluginType        `json:"type"`
    Capabilities []string         `json:"capabilities"`
    Dependencies []string         `json:"dependencies"`
    Config      map[string]interface{} `json:"config"`
}
```

#### 2. **Plugin Type System**
```go
// Supported plugin types with specific capabilities
type PluginType string

const (
    PluginTypeEditor   PluginType = "editor"    // File editing operations
    PluginTypeGit      PluginType = "git"       // Git repository management
    PluginTypeMemory   PluginType = "memory"    // Persistent data storage
    PluginTypeGeneric  PluginType = "generic"   // Custom functionality
)
```

#### 3. **Plugin Loader Architecture**
```go
// pkg/plugins/loader.go - Dynamic loading system
type PluginLoader struct {
    plugins    map[string]Plugin
    registry   *registry.ServiceRegistry
    config     *config.PluginConfig
    logger     *logging.Logger
    healthMgr  *health.Manager
}

func (l *PluginLoader) LoadPlugin(pluginPath string) error {
    // 1. Validate plugin binary/module
    // 2. Initialize plugin with configuration
    // 3. Register with service registry
    // 4. Setup health monitoring
    // 5. Enable request routing
}
```

### Core Plugin Services

#### 1. **Editor Service Plugin** (`pkg/plugins/editor_service.go`)
```go
// Comprehensive file editing capabilities
type EditorService struct {
    config     *EditorConfig
    logger     *logging.Logger
    workspaceManager *WorkspaceManager
    syntaxEngine     *SyntaxEngine
}

// Key capabilities:
- File read/write operations with atomic guarantees
- Multi-format support (text, JSON, YAML, code files)
- Syntax validation and formatting
- Workspace management with isolation
- Version control integration hooks
```

#### 2. **Git Service Plugin** (`pkg/plugins/git_service.go`)
```go
// Advanced Git repository management
type GitService struct {
    config       *GitConfig
    logger       *logging.Logger
    repoManager  *RepositoryManager
    authProvider *AuthenticationProvider
}

// Key capabilities:
- Repository cloning, status, and operations
- Branch management and merging
- Authentication with multiple providers
- Commit history analysis
- Remote synchronization
```

#### 3. **Memory Service Plugin** (`pkg/plugins/memory_service.go`)
```go
// Persistent data storage with JSON backend
type MemoryService struct {
    config      *MemoryConfig
    logger      *logging.Logger
    storage     *PersistentStorage
    cacheLayer  *CacheManager
}

// Key capabilities:
- JSON-based persistent storage
- In-memory caching for performance
- Atomic read/write operations
- Data validation and schema enforcement
- Backup and recovery mechanisms
```

### Integration Architecture

#### 1. **Service Registry Integration**
```go
// internal/registry/service_registry.go - Plugin registration
func (r *ServiceRegistry) RegisterPlugin(plugin Plugin) error {
    metadata := plugin.GetMetadata()
    
    // Register plugin capabilities
    for _, capability := range metadata.Capabilities {
        r.capabilityMap[capability] = plugin
    }
    
    // Setup routing rules
    r.routingTable[metadata.Name] = plugin
    
    // Enable health monitoring
    r.healthManager.RegisterPlugin(plugin)
    
    return nil
}
```

#### 2. **Request Routing Integration**
```go
// internal/router/mcp_router.go - Plugin request routing
func (r *MCPRouter) RouteToPlugin(ctx context.Context, request MCPRequest) (MCPResponse, error) {
    // 1. Identify target plugin by capability
    plugin := r.registry.GetPluginByCapability(request.Method)
    
    // 2. Apply middleware (auth, rate limiting, logging)
    ctx = r.middleware.Apply(ctx, request)
    
    // 3. Route to plugin with error handling
    response, err := plugin.HandleRequest(ctx, request)
    
    // 4. Process response and return
    return r.processResponse(response, err)
}
```

#### 3. **Health Monitoring Integration**
```go
// pkg/health/checkers.go - Plugin health monitoring
func (h *HealthManager) MonitorPlugin(plugin Plugin) {
    go func() {
        ticker := time.NewTicker(30 * time.Second)
        for range ticker.C {
            if err := plugin.HealthCheck(context.Background()); err != nil {
                h.logger.Error("Plugin health check failed", 
                    "plugin", plugin.GetMetadata().Name,
                    "error", err)
                h.handlePluginFailure(plugin)
            }
        }
    }()
}
```

## Implementation Details

### Plugin Lifecycle Management
```go
// Complete plugin lifecycle with error handling
type PluginLifecycle struct {
    phases []LifecyclePhase
}

const (
    PhaseLoading      LifecyclePhase = "loading"
    PhaseInitializing LifecyclePhase = "initializing"
    PhaseRunning      LifecyclePhase = "running"
    PhaseStopping     LifecyclePhase = "stopping"
    PhaseStopped      LifecyclePhase = "stopped"
    PhaseFailed       LifecyclePhase = "failed"
)
```

### Configuration Management
```yaml
# config/mcpeg.yaml - Plugin configuration
plugins:
  editor:
    enabled: true
    config:
      workspace_dir: "./build/data/workspaces"
      max_file_size: "10MB"
      allowed_extensions: [".go", ".yaml", ".json", ".md"]
  
  git:
    enabled: true
    config:
      default_branch: "main"
      auth_methods: ["ssh", "https"]
      clone_depth: 50
  
  memory:
    enabled: true
    config:
      storage_path: "./build/data/memory_storage.json"
      cache_size: "100MB"
      backup_interval: "1h"
```

### Error Handling and Isolation
```go
// Plugin error isolation prevents system-wide failures
func (l *PluginLoader) HandlePluginError(plugin Plugin, err error) {
    metadata := plugin.GetMetadata()
    
    l.logger.Error("Plugin error occurred",
        "plugin", metadata.Name,
        "version", metadata.Version,
        "error", err,
        "action", "isolating_plugin")
    
    // Isolate failed plugin
    l.registry.DisablePlugin(metadata.Name)
    
    // Attempt recovery if configured
    if l.config.AutoRecover {
        go l.attemptPluginRecovery(plugin)
    }
}
```

## Consequences

### Positive
- **Extensible Architecture**: Easy addition of new functionality through plugins
- **Service Integration**: Seamless integration with existing service registry
- **Lifecycle Management**: Complete plugin lifecycle with health monitoring
- **Error Isolation**: Plugin failures don't affect core gateway functionality
- **Performance Optimized**: Efficient routing and caching mechanisms
- **Production Ready**: Comprehensive error handling and monitoring

### Negative
- **Complexity Increase**: Additional architectural complexity for plugin management
- **Resource Overhead**: Memory and CPU overhead for plugin isolation
- **Dependency Management**: Complex dependency resolution between plugins
- **Testing Complexity**: Increased testing surface area for plugin interactions

## Future Extensions Enabled

This foundation architecture enables:
1. **Hot Reloading**: Dynamic plugin updates without service restart
2. **Plugin Communication**: Inter-plugin messaging and coordination
3. **Advanced Discovery**: Intelligent capability analysis and optimization
4. **Security Policies**: RBAC and access control for plugin operations
5. **Marketplace Integration**: Plugin distribution and version management

## Technical Implementation

### Files Created
- `pkg/plugins/plugin.go`: Core plugin interface and types
- `pkg/plugins/loader.go`: Plugin loading and lifecycle management
- `pkg/plugins/editor_service.go`: File editing service implementation
- `pkg/plugins/git_service.go`: Git repository management service
- `pkg/plugins/memory_service.go`: Persistent data storage service
- `internal/plugins/integration.go`: Service registry integration

### Integration Points
- **Service Registry**: Plugin registration and capability mapping
- **MCP Router**: Request routing to appropriate plugins
- **Health Manager**: Plugin health monitoring and failure handling
- **Configuration System**: Plugin-specific configuration management

### Quality Assurance
- **Unit Testing**: Comprehensive tests for each plugin type
- **Integration Testing**: Plugin interaction and lifecycle testing
- **Performance Testing**: Load testing for plugin routing efficiency
- **Error Testing**: Failure scenarios and recovery mechanism validation

## References
- [Plugin Development Guide](../plugins/development-guide.md)
- [Plugin Configuration Reference](../plugins/configuration.md)
- [Service Registry Integration](../architecture/service-registry.md)
- [Health Monitoring Architecture](../architecture/health-monitoring.md)

## Related ADRs
- [ADR-010: Multi-Service Gateway](010-multi-service-gateway.md)
- [ADR-016: Unified Binary Architecture](016-unified-binary-architecture.md)
- [ADR-021: Daemon Process Management](021-daemon-process-management.md)
- [ADR-022: Plugin Registration Service Registry](022-plugin-registration-service-registry.md)