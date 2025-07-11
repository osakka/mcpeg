# ADR-007: Built-in Validation and Testing Framework

## Status

Accepted

## Context

Traditional approaches separate testing from the service implementation, requiring external CI/CD pipelines. For MCPEG, we want to:
- Enable self-validation of configurations
- Allow testing of adapter mappings without external dependencies
- Provide health checks and diagnostics
- Support dry-run modes for configuration changes
- Validate MCP protocol compliance

## Decision

We will build validation and testing capabilities directly into MCPEG as first-class features, including:
1. Configuration validation endpoints
2. Adapter testing mode with mocked backends
3. MCP protocol compliance checker
4. Health check and diagnostic endpoints
5. Request/response recording and replay

## Consequences

### Positive

- Zero additional infrastructure for testing
- Instant feedback on configuration changes
- Can validate in production environment
- Self-documenting test cases
- Easier debugging with built-in tools
- Can run validation before applying config changes

### Negative

- Increases binary size
- Additional attack surface in production
- More complex codebase
- Need to ensure test code doesn't affect production

### Neutral

- Blurs line between service and tooling
- Requires careful separation of concerns

## Implementation Design

### 1. Validation Endpoints
```
GET /v1/validate/config
POST /v1/validate/adapter
GET /v1/validate/mcp-compliance
```

### 2. Test Mode
```yaml
# In config
test_mode:
  enabled: true
  mock_responses:
    - pattern: "GET /api/*"
      response: 
        status: 200
        body: {"test": "data"}
```

### 3. Diagnostic Tools
```
GET /v1/diagnostics/health
GET /v1/diagnostics/config
GET /v1/diagnostics/requests?last=100
POST /v1/diagnostics/dry-run
```

### 4. MCP Compliance Testing
- Validate against official MCP test suite
- Check message format compliance
- Verify required method implementations
- Test error handling

## Security Considerations

- Test endpoints require authentication
- Separate port for diagnostic endpoints
- Rate limiting on validation endpoints
- No sensitive data in test responses
- Audit logging for all test operations

## Alternatives Considered

1. **External Test Suite**: Rejected to maintain single-binary simplicity
2. **Separate Test Binary**: Rejected to avoid deployment complexity
3. **No Built-in Testing**: Rejected as it would require external infrastructure

## References

- [Testing in Production](https://medium.com/@copyconstruct/testing-in-production-the-safe-way-18ca102d0ef1)
- [Self-Testing Code](https://martinfowler.com/bliki/SelfTestingCode.html)