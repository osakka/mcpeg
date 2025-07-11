# ADR-004: Use YAML for Service Configuration

## Status

Proposed

## Context

MCPEG needs a configuration format that:
- Is human-readable and editable
- Supports complex service wiring and mappings
- Has good tooling support
- Can express relationships between MCP endpoints and backend services
- Supports validation and schema definitions

## Decision

We will use YAML as the configuration format for MCPEG with:
- A defined YAML schema for validation
- Support for environment variable substitution
- Clear structure for mapping MCP resources/tools to backend services
- Include configuration for both API endpoints and binary executables

Example structure:
```yaml
version: "1.0"
services:
  - name: "filesystem"
    type: "binary"
    command: "curl"
    mcp_tools:
      - name: "read_file"
        maps_to: "GET /files/{path}"
  - name: "database"
    type: "api"
    base_url: "https://api.example.com"
    mcp_resources:
      - pattern: "db://*"
        maps_to: "/query"
```

## Consequences

### Positive

- Human-readable and easy to edit
- Widespread tooling support
- Good for configuration management systems
- Supports comments for documentation
- Can be validated against schemas
- Familiar to most developers

### Negative

- Whitespace sensitivity can cause errors
- Less performant than binary formats
- Potential for ambiguous parsing in edge cases

### Neutral

- Need to define comprehensive schema
- Requires YAML parser dependency

## Alternatives Considered

1. **JSON**: Rejected due to lack of comments and verbosity
2. **TOML**: Rejected due to less widespread adoption and tooling
3. **HCL (HashiCorp Configuration Language)**: Rejected as too specific to HashiCorp ecosystem
4. **XML**: Rejected due to verbosity and complexity
5. **Protocol Buffers**: Rejected as too complex for configuration

## References

- [YAML Specification](https://yaml.org/spec/)
- [YAML Schema Validation](https://json-schema.org/)
- [Configuration as Code](https://www.atlassian.com/continuous-delivery/principles/configuration-as-code)