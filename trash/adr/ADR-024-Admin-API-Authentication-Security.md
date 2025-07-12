# ADR-024: Admin API Authentication and Security

## Status
**ACCEPTED** - *2025-07-12*

## Context
The MCpeg gateway includes powerful admin endpoints for service management, configuration control, and system monitoring. These endpoints provide access to sensitive operations including service registration, configuration reloading, and system metrics. Without proper authentication, these endpoints pose a significant security risk in production environments.

During the security sweep, we identified that admin endpoints were accessible without any authentication mechanism, allowing unauthorized access to critical gateway management functions. This represented a major security gap that needed immediate remediation.

## Decision
We implemented comprehensive API key-based authentication for all admin endpoints:

1. **API Key Authentication**: Configurable API key validation with custom header support
2. **Conditional Authentication**: Authentication disabled when no API key is configured (development mode)
3. **Comprehensive Logging**: Detailed authentication attempt logging for security monitoring
4. **Metrics Integration**: Authentication success/failure metrics for monitoring

## Implementation Details

### Authentication Configuration
```yaml
# Production configuration
server:
  admin_api_key: "${MCPEG_ADMIN_API_KEY}"
  admin_api_header: "X-Admin-API-Key"

# Development configuration  
server:
  admin_api_key: ""  # Empty = authentication disabled
  admin_api_header: "X-Admin-API-Key"
```

### Middleware Implementation
```go
// internal/server/gateway_server.go
func (gs *GatewayServer) adminAuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Skip auth if no API key configured (development mode)
        if gs.config.AdminAPIKey == "" {
            next.ServeHTTP(w, r)
            return
        }

        headerName := gs.config.AdminAPIHeader
        if headerName == "" {
            headerName = "X-Admin-API-Key"
        }

        providedKey := r.Header.Get(headerName)
        if providedKey == "" {
            gs.metrics.Inc("admin_auth_missing_key")
            gs.logger.Warn("Admin API access denied: missing API key",
                "remote_addr", r.RemoteAddr,
                "user_agent", r.UserAgent(),
                "path", r.URL.Path)
            
            http.Error(w, "Unauthorized: API key required", http.StatusUnauthorized)
            return
        }

        if providedKey != gs.config.AdminAPIKey {
            gs.metrics.Inc("admin_auth_invalid_key")
            gs.logger.Warn("Admin API access denied: invalid API key",
                "remote_addr", r.RemoteAddr,
                "user_agent", r.UserAgent(),
                "path", r.URL.Path)
            
            http.Error(w, "Unauthorized: invalid API key", http.StatusUnauthorized)
            return
        }

        gs.metrics.Inc("admin_auth_success")
        gs.logger.Debug("Admin API access granted",
            "remote_addr", r.RemoteAddr,
            "path", r.URL.Path)

        next.ServeHTTP(w, r)
    })
}
```

### Router Integration
```go
// Apply authentication middleware to admin routes
if gs.config.EnableAdminEndpoints {
    adminRouter := router.PathPrefix("/admin").Subrouter()
    
    // Apply auth middleware if API key is configured
    if gs.config.AdminAPIKey != "" {
        adminRouter.Use(gs.adminAuthMiddleware)
    }
    
    // Register admin endpoints...
}
```

## Consequences

### Positive
- **Security Enhancement**: Admin endpoints protected from unauthorized access
- **Flexible Configuration**: Authentication can be disabled for development environments
- **Comprehensive Monitoring**: Authentication attempts logged and metrified for security analysis
- **Production Ready**: Environment variable-based API key configuration for secure deployment
- **Custom Headers**: Configurable authentication header names for integration flexibility

### Negative
- **Additional Configuration**: Requires API key management in production environments
- **Breaking Change**: Existing admin API clients need to be updated with authentication headers
- **Complexity**: Additional middleware logic and configuration validation required

## Files Modified
- `internal/server/gateway_server.go`: Added `AdminAPIKey` and `AdminAPIHeader` configuration fields
- `internal/server/gateway_server.go`: Implemented `adminAuthMiddleware` function
- `internal/server/admin_auth_test.go`: Created comprehensive test suite for authentication
- `config/production.yaml`: Added admin API authentication configuration
- `dev-config.yaml`: Added development-friendly authentication configuration

## Testing
The implementation includes comprehensive test coverage:

- **Auth Disabled**: Verifies access when no API key is configured
- **Auth Required**: Validates authentication requirement when API key is set
- **Correct Key**: Tests successful authentication with valid API key
- **Incorrect Key**: Verifies rejection of invalid API keys
- **Custom Headers**: Tests configurable authentication header names

```go
func TestAdminAuthMiddleware(t *testing.T) {
    // Test cases covering all authentication scenarios
    // with proper mock setup and validation
}
```

## References
- [Admin API Documentation](../admin-api.md)
- [Security Configuration Guide](../security.md)
- [ADR-016: Admin API Design](ADR-016-Admin-API-Design.md)
- [ADR-021: Production Security](ADR-021-Production-Security.md)