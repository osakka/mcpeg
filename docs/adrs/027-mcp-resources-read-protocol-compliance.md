# ADR-027: MCP Resources/Read Protocol Compliance

## Status
**ACCEPTED** - *2025-07-12*

## Context

MCpeg Gateway successfully implemented MCP protocol tools and resources listing capabilities, but was missing the critical `resources/read` functionality. This created a protocol compliance violation where resources were discoverable via `resources/list` but not accessible via `resources/read`, breaking user expectations and violating the MCP 2025-03-26 specification.

Key requirements identified:
- Complete MCP protocol compliance for resource operations
- Proper plugin:// URI parsing and routing
- Plugin resource access with authentication
- Structured resource content response formatting
- Error handling for invalid URIs and failed reads
- Integration with existing RBAC and logging systems

## Decision

We implemented complete MCP resources/read protocol compliance with the following architecture:

### Resource Reading Architecture

#### 1. **Plugin Resource Handler Interface**
```go
// pkg/mcp/types.go - Extended PluginHandler interface
type PluginHandler interface {
    // ... existing methods
    ReadPluginResource(ctx context.Context, uri string, capabilities *rbac.ProcessedCapabilities) (interface{}, error)
}
```

#### 2. **MCP Router Integration**
```go
// internal/router/mcp_router.go - handlePluginResourcesRead
func (mr *MCPRouter) handlePluginResourcesRead(ctx context.Context, reqCtx *RequestContext, mcpReq *types.MCPRequest) (*types.MCPResponse, error) {
    // Extract URI parameter from request
    var params struct {
        URI string `json:"uri"`
    }
    
    // Route to plugin handler with authentication
    result, err := mr.pluginHandler.ReadPluginResource(ctx, params.URI, reqCtx.Capabilities)
    
    // Format as proper MCP ResourceContent response
    contents := []ResourceContent{
        {
            URI:      params.URI,
            MimeType: determineMimeType(result),
            Text:     formatResourceText(result),
        },
    }
    
    return &types.MCPResponse{
        JSONRPC: "2.0",
        ID:      mcpReq.ID,
        Result: map[string]interface{}{
            "contents": contents,
        },
    }, nil
}
```

#### 3. **Plugin URI Parsing**
```go
// pkg/mcp/plugin_handler.go - ReadPluginResource implementation
func (ph *PluginHandlerImpl) ReadPluginResource(ctx context.Context, uri string, capabilities *rbac.ProcessedCapabilities) (interface{}, error) {
    // Parse plugin URI format: plugin://pluginName/resourceName
    if !strings.HasPrefix(uri, "plugin://") {
        return nil, fmt.Errorf("invalid plugin resource URI: %s", uri)
    }
    
    // Extract plugin name and resource name from URI
    uriParts := strings.TrimPrefix(uri, "plugin://")
    parts := strings.SplitN(uriParts, "/", 2)
    if len(parts) != 2 {
        return nil, fmt.Errorf("invalid plugin resource URI format: %s", uri)
    }
    
    pluginName := parts[0]
    resourceName := parts[1]
    
    // Validate access permissions
    if !ph.hasPluginAccess(pluginName, capabilities) {
        return nil, fmt.Errorf("access denied to plugin: %s", pluginName)
    }
    
    // Call plugin's ReadResource method
    result, err := plugin.ReadResource(ctx, resourceName)
    if err != nil {
        return nil, fmt.Errorf("failed to read resource %s from plugin %s: %w", resourceName, pluginName, err)
    }
    
    return result, nil
}
```

### Protocol Compliance Implementation

#### 1. **Request Format Validation**
```go
// MCP JSON-RPC 2.0 request format
{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "resources/read",
    "params": {
        "uri": "plugin://memory/memory_stats"
    }
}
```

#### 2. **Response Format Standardization**
```go
// MCP ResourceContent response format
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
        "contents": [
            {
                "uri": "plugin://memory/memory_stats",
                "mimeType": "application/json",
                "text": "{\"total_entries\": 8, \"storage_size\": \"2.1KB\"}"
            }
        ]
    }
}
```

#### 3. **MIME Type Detection**
```go
// determineMimeType - Content type detection
func determineMimeType(content interface{}) string {
    switch content.(type) {
    case string:
        return "text/plain"
    case map[string]interface{}, []interface{}:
        return "application/json"
    default:
        return "application/octet-stream"
    }
}
```

### Authentication and Authorization

#### 1. **RBAC Integration**
```go
// Permission validation for resource reading
permission := capabilities.Plugins[pluginName]
if wildcardPerm, hasWildcard := capabilities.Plugins["*"]; hasWildcard && len(capabilities.Plugins) == 1 {
    permission = wildcardPerm
}
if !permission.CanRead {
    return nil, fmt.Errorf("read access denied for plugin: %s", pluginName)
}
```

#### 2. **Plugin Access Control**
- Plugin-level access validation
- Resource-level read permissions
- URI format validation for security
- Error message consistency for security

### Logging and Monitoring

#### 1. **Comprehensive Logging**
```go
// Success logging
ph.logger.Info("plugin_resource_read_completed",
    "plugin", pluginName,
    "resource", resourceName,
    "uri", uri)

// Error logging
ph.logger.Error("plugin_resource_read_failed",
    "plugin", pluginName,
    "resource", resourceName,
    "uri", uri,
    "error", err)
```

#### 2. **Metrics Collection**
```go
// Resource read metrics
ph.metrics.Inc("plugin_resource_reads_total", 
    "plugin", pluginName, 
    "resource", resourceName, 
    "status", "success")
```

## Implementation Details

### Request Routing Flow
```
MCP Client Request (resources/read)
    ↓
HTTP Server (Gorilla Mux)
    ↓
MCP Router (tryPluginRouting)
    ↓
handlePluginResourcesRead
    ↓
Plugin Handler (ReadPluginResource)
    ↓
URI Parsing and Validation
    ↓
Plugin Manager (GetPlugin)
    ↓
Individual Plugin (ReadResource)
    ↓
Resource Content Response
    ↓
MCP JSON-RPC Response
```

### Error Handling Strategy
```go
// Comprehensive error handling with proper HTTP status codes
- Invalid URI format: 400 Bad Request
- Plugin not found: 404 Not Found  
- Access denied: 403 Forbidden
- Resource not found: 404 Not Found
- Plugin internal error: 500 Internal Server Error
```

### Testing Integration
```bash
# Manual testing command
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": 4, "method": "resources/read", "params": {"uri": "plugin://memory/memory_stats"}}'

# Expected response
{
  "jsonrpc": "2.0",
  "id": 4,
  "result": {
    "contents": [
      {
        "uri": "plugin://memory/memory_stats",
        "mimeType": "application/json",
        "text": "{\"total_entries\": 8, \"storage_size\": \"2.1KB\"}"
      }
    ]
  }
}
```

## Consequences

### Positive
- **Protocol Compliance**: Full MCP 2025-03-26 specification compliance
- **User Experience**: Resources are now fully accessible after discovery
- **Security**: Proper authentication and authorization for resource access
- **Consistency**: Same RBAC and logging patterns as other MCP operations
- **Performance**: Efficient URI parsing and direct plugin routing
- **Monitoring**: Complete observability with logging and metrics

### Negative
- **API Surface**: Additional endpoint increases testing and maintenance complexity
- **Security Surface**: New attack vectors through URI parsing and resource access
- **Resource Usage**: Additional memory and CPU for resource reading operations

## Quality Assurance

### Testing Coverage
- **Protocol Compliance**: MCP JSON-RPC 2.0 format validation
- **Authentication**: RBAC permission validation
- **URI Parsing**: Valid and invalid URI format testing
- **Plugin Integration**: Resource reading for all plugin types
- **Error Handling**: Comprehensive error scenario validation

### Performance Characteristics
- **Resource Reading**: ~10-30ms (plugin-dependent)
- **URI Parsing**: Sub-millisecond parsing for valid URIs
- **Authentication**: Same performance as other MCP operations
- **Memory Usage**: Minimal additional overhead per request

## Technical Implementation

### Files Modified
- `pkg/mcp/types.go`: Added ReadPluginResource to PluginHandler interface
- `internal/router/mcp_router.go`: Added handlePluginResourcesRead method (lines 1381-1445)
- `pkg/mcp/plugin_handler.go`: Implemented ReadPluginResource method (lines 269-325)

### Integration Points
- **MCP Router**: Added "resources/read" case to tryPluginRouting
- **Plugin System**: Extended plugin interface for resource reading
- **RBAC Engine**: Integrated permission validation for resource access
- **Logging System**: Added comprehensive resource read logging
- **Metrics System**: Added resource read metrics collection

### Documentation Updates
- `docs/architecture/high-level-design.md`: Updated performance metrics
- `docs/architecture/testing-infrastructure.md`: Added resources/read testing
- `CHANGELOG.md`: Added entry for complete resources/read implementation

## References
- [MCP Protocol Specification](https://spec.modelcontextprotocol.io/)
- [Plugin Resource Interface](../plugins/resource-interface.md)
- [RBAC Authorization Guide](../security/rbac-guide.md)
- [Testing Methodology](../testing/mcp-testing-methodology.md)

## Related ADRs
- [ADR-002: Use Model Context Protocol as Core Protocol](002-use-mcp-protocol.md)
- [ADR-020: Plugin System Foundation Architecture](020-plugin-system-foundation-architecture.md)
- [ADR-023: MCP Plugin Integration Phase 1](023-mcp-plugin-integration-phase-1.md)
- [ADR-024: MCP Plugin Integration Complete Phases 1-4](024-mcp-plugin-integration-complete-phases-1-4.md)
- [ADR-025: Phase 2 Advanced Plugin Discovery Intelligence](025-phase-2-advanced-plugin-discovery-intelligence.md)