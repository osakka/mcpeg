# ADR-006: Prioritize REST API Adapters

## Status

**ACCEPTED** - *2025-07-11*

## Context

MCPEG needs to support multiple types of backend integrations:
- REST APIs
- Binary executables
- gRPC services
- GraphQL endpoints
- Message queues

We need to decide which adapter type to implement first to provide maximum value and establish patterns for future adapters.

## Decision

We will prioritize REST API adapters as the first implementation, focusing on:
- Generic REST client capabilities
- Flexible request/response mapping
- Common authentication methods (Bearer, Basic, API Key)
- JSON transformation utilities

## Consequences

### Positive

- REST APIs are the most common integration pattern
- Well-understood technology with mature tooling
- Covers majority of modern service integrations
- Simpler to implement than binary execution sandboxing
- Good test case for adapter pattern
- Easy to mock for testing

### Negative

- Doesn't immediately support legacy systems using binaries
- May need to revisit design when adding binary adapters
- REST-specific assumptions might leak into core

### Neutral

- Sets precedent for adapter design
- Influences configuration schema design

## Design Principles

1. **Generic Mapping**: Configuration-driven request/response mapping
2. **Template Support**: Use Go templates for dynamic values
3. **Transform Pipeline**: Support response transformation
4. **Error Mapping**: Map HTTP errors to MCP errors
5. **Retry Logic**: Built-in retry with exponential backoff

## Example Configuration

```yaml
services:
  - id: "api-service"
    type: "rest"
    base_url: "https://api.example.com"
    auth:
      type: "bearer"
      token: "${API_TOKEN}"
    
    tools:
      - name: "search"
        maps_to:
          method: "GET"
          path: "/search"
          query_params:
            q: "{{.input.query}}"
            limit: "{{.input.limit | default 10}}"
          transform:
            response: |
              {
                "results": .items,
                "total": .total_count
              }
```

## Alternatives Considered

1. **Binary Adapters First**: Rejected due to complexity of sandboxing and process management
2. **GraphQL First**: Rejected as REST is more universal
3. **All Adapters Simultaneously**: Rejected to maintain focus and quality

## References

- [REST API Design Best Practices](https://restfulapi.net/)
- [Go HTTP Client Best Practices](https://golang.org/pkg/net/http/)