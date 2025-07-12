# ADR-002: Use Model Context Protocol as Core Protocol

## Status
**ACCEPTED** - *2025-07-11*

## Context

We need a standardized protocol for enabling Large Language Models (LLMs) to interact with external services and data sources. The Model Context Protocol (MCP) has emerged as an open standard that:
- Provides a well-defined specification for LLM-service communication
- Has official SDKs in multiple languages
- Is actively maintained by Anthropic and the community
- Offers a flexible architecture for various integration patterns

## Decision

We will use the Model Context Protocol (MCP) as the core protocol for MCPEG. This means:
- Implementing full MCP protocol compliance on the client-facing side
- Generating our API schemas directly from MCP specifications with zero divergence
- Using MCP's resource, tool, and prompt concepts as our primary abstractions
- Following MCP's JSON-RPC 2.0 message format

## Consequences

### Positive

- Immediate compatibility with MCP-compliant clients (Claude Desktop, etc.)
- Well-tested protocol with established patterns
- Active community and ongoing development
- Clear specification reduces ambiguity
- Multiple transport options (stdio, HTTP)

### Negative

- Locked into MCP's design decisions
- Must track and implement specification updates
- Limited flexibility in protocol extensions

### Neutral

- Must maintain strict compliance with specifications
- Need to set up automated schema generation from MCP specs

## Alternatives Considered

1. **Custom Protocol**: Rejected due to lack of ecosystem and increased development effort
2. **OpenAPI Only**: Rejected because it lacks LLM-specific concepts like prompts and sampling
3. **GraphQL**: Rejected due to complexity and lack of LLM-specific features
4. **gRPC**: Rejected because MCP's JSON-RPC approach is more accessible

## References

- [Model Context Protocol Website](https://modelcontextprotocol.io)
- [MCP Specification](https://github.com/modelcontextprotocol/specification)
- [MCP TypeScript SDK](https://github.com/modelcontextprotocol/typescript-sdk)