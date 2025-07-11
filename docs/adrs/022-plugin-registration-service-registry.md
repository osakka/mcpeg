# ADR-022: Plugin Registration and Service Registry Integration

## Status
**ACCEPTED** - *2025-07-11*

## Context

MCpeg Gateway's plugin system encountered service registration validation failures when plugins attempted to register with the service registry. The core issues were:

1. **URL Validation Rejection**: Plugin endpoints using `plugin://internal` scheme were rejected by URL validation expecting HTTP/HTTPS
2. **Health Check Failures**: Service registry attempted HTTP health checks on non-HTTP plugin endpoints
3. **Service Unavailability**: Registration failures prevented plugin services from becoming available

## Decision

Implement specialized handling for plugin endpoints in the service registry to support internal plugin architecture while maintaining validation for external HTTP services.

### Core Solutions

#### 1. **Extended URL Validation for Plugin Schemes**

**File**: `/opt/mcpeg/pkg/validation/validator.go`

```go
func (v *Validator) validateURL(value interface{}) bool {
    str, ok := value.(string)
    if !ok {
        return false
    }
    
    // Accept HTTP/HTTPS URLs and internal plugin URLs
    httpRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
    pluginRegex := regexp.MustCompile(`^plugin://[^\s]*$`)
    
    return httpRegex.MatchString(str) || pluginRegex.MatchString(str)
}
```

**Rationale**: Plugins use internal URLs that don't conform to HTTP scheme requirements but need to pass validation for service registration.

#### 2. **Plugin-Aware Health Check Bypass**

**File**: `/opt/mcpeg/internal/registry/service_registry.go`

```go
func (sr *ServiceRegistry) performHealthCheck(ctx context.Context, service *RegisteredService) error {
    // Skip health checks for plugin endpoints - they are managed internally
    if strings.HasPrefix(service.Endpoint, "plugin://") {
        sr.logger.Debug("skipping_health_check_for_plugin",
            "service_id", service.ID,
            "service_name", service.Name,
            "endpoint", service.Endpoint)
        return sr.updateServiceHealth(service, HealthHealthy, nil, time.Since(startTime))
    }
    
    // Continue with HTTP health checks for external services...
}
```

**Rationale**: Plugin endpoints are internal components managed by the plugin system, not external HTTP services requiring health verification.

### Design Principles

1. **Separation of Concerns**: Plugin registration handled differently from external service registration
2. **Backward Compatibility**: External HTTP services continue to use full validation and health checks
3. **Internal Trust**: Plugin endpoints are trusted components managed by the gateway itself
4. **Validation Consistency**: URL validation rules extended rather than bypassed

## Implementation Details

### Plugin Registration Flow

```
1. Plugin System → Create ServiceRegistrationRequest with "plugin://internal"
2. Service Registry → Validate request (now accepts plugin:// URLs)
3. Service Registry → Skip HTTP health check for plugin endpoints
4. Service Registry → Mark plugin as healthy and active
5. Plugin Service → Available for MCP routing
```

### Affected Components

- **Validator**: Extended URL regex to accept `plugin://` scheme
- **Service Registry**: Added plugin endpoint detection and health check bypass
- **Plugin Integration**: Continues using `plugin://internal` endpoints
- **MCP Router**: Routes to plugin services normally after successful registration

### Error Resolution

**Before**: 
```
[ERROR] failed_to_register_plugin_service [error [service_registry:validation] 
register_service: Invalid registration request]
```

**After**:
```
[INFO] service_registration_completed [service_id mcp_plugin-memory-xxx 
name memory type mcp_plugin status active health healthy]
```

## Consequences

### Positive
- ✅ **Plugin System Operational**: All three built-in plugins (Memory, Git, Editor) register successfully
- ✅ **Service Discovery**: Plugins appear in service registry and are available for routing
- ✅ **Backward Compatibility**: External HTTP services unaffected by changes
- ✅ **Health Management**: Plugin health managed by plugin system rather than HTTP checks
- ✅ **Single Source of Truth**: Unified binary handles both plugins and external services

### Negative
- ⚠️ **Special Case Logic**: Service registry now has plugin-specific code paths
- ⚠️ **URL Scheme Proliferation**: Multiple URL schemes supported (http, https, plugin)
- ⚠️ **Health Check Gap**: Plugin health not verified via standard HTTP mechanisms

### Mitigations
- Plugin health managed by dedicated plugin system health checks
- Clear separation between plugin and external service handling
- Comprehensive logging for plugin registration debugging
- Documentation of plugin URL scheme requirements

## Files Modified

### New Functionality
```
pkg/validation/validator.go           # Extended URL validation for plugin://
internal/registry/service_registry.go # Plugin health check bypass
```

### Dependencies
- No new external dependencies
- Uses existing `strings.HasPrefix()` for plugin endpoint detection
- Leverages existing health status management

## Testing Results

### Plugin Registration Success
```bash
# All three plugins registered successfully:
[INFO] service_registration_completed [service_id mcp_plugin-memory-xxx 
name memory type mcp_plugin version 1.0.0 endpoint plugin://internal 
status active health healthy]

[INFO] service_registration_completed [service_id mcp_plugin-git-xxx 
name git type mcp_plugin version 1.0.0 endpoint plugin://internal 
status active health healthy]

[INFO] service_registration_completed [service_id mcp_plugin-editor-xxx 
name editor type mcp_plugin version 1.0.0 endpoint plugin://internal 
status active health healthy]
```

### Plugin Capabilities
- **Memory Plugin**: 5 tools, 2 resources, 2 prompts
- **Git Plugin**: 8 tools, 2 resources, 2 prompts  
- **Editor Plugin**: 7 tools, 2 resources, 2 prompts

## Alternatives Considered

### 1. Disable Health Checks Globally
**Rejected**: Would compromise external service health monitoring

### 2. Use HTTP URLs for Plugins
**Rejected**: Would require plugin HTTP endpoints, adding complexity

### 3. Separate Plugin Registry
**Rejected**: Would violate single service registry principle

### 4. Plugin-Specific Registration API
**Rejected**: Would duplicate registration logic and complexity

## Future Considerations

1. **Plugin Health Protocols**: Consider developing plugin-specific health check protocols
2. **URL Scheme Registry**: Formalize supported URL schemes for different service types
3. **Plugin Discovery**: Extend service discovery to handle plugin capabilities
4. **Health Check Plugins**: Consider plugins that can health-check other plugins

## References

- [ADR-020: Plugin System Architecture](020-plugin-system-architecture.md)
- [Service Registry Implementation](../../internal/registry/service_registry.go)
- [Plugin Integration](../../internal/plugins/integration.go)
- [URL Validation Logic](../../pkg/validation/validator.go)

## Revision History

| Version | Date       | Changes                 | Author |
|---------|------------|-------------------------|---------|
| 1.0     | 2025-07-11 | Initial implementation  | Claude  |