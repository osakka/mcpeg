# ADR-005: Use Go as Implementation Language

## Status

**ACCEPTED** - *2025-07-11*

## Context

We need to choose an implementation language for MCPEG that provides:
- Good performance for protocol translation
- Strong concurrency support for handling multiple connections
- Easy deployment (single binary)
- Good ecosystem for API development
- Available MCP SDK support

## Decision

We will use Go as the primary implementation language for MCPEG.

## Consequences

### Positive

- Single binary deployment simplifies operations
- Excellent concurrency primitives (goroutines, channels)
- Strong standard library for HTTP and networking
- Good performance characteristics
- Static typing catches errors at compile time
- Built-in testing framework
- Cross-compilation support
- Minimal runtime dependencies

### Negative

- Less mature MCP SDK compared to TypeScript
- Smaller ecosystem for some web-specific libraries
- More verbose than dynamic languages
- Learning curve for developers unfamiliar with Go

### Neutral

- Opinionated language design
- Garbage collection (predictable but present)
- Module system requires understanding

## Implementation Notes

- Use Go modules for dependency management
- Follow standard Go project layout
- Leverage Go's built-in testing framework
- Use interfaces for adapter abstraction

## Alternatives Considered

1. **TypeScript/Node.js**: Rejected due to deployment complexity and runtime dependencies
2. **Rust**: Rejected due to steeper learning curve and longer development time
3. **Python**: Rejected due to performance concerns and deployment complexity

## References

- [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk)
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Project Layout](https://github.com/golang-standards/project-layout)