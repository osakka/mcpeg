# CLAUDE.md - AI Assistant Context

This document provides context and instructions for AI assistants working on the MCPEG project.

## Project Overview

MCPEG (Model Context Protocol Enablement Gateway) is a lightweight service that:
- Implements the full MCP protocol specification on one side
- Integrates with external services via REST APIs or binary calls on the other side
- Uses YAML configuration for service wiring
- Follows API-first development methodology

## Key Principles

1. **No Divergence from MCP Spec**: API schemas must be generated directly from official MCP protocol specifications
2. **Single Source of Truth**: Avoid redundancy; each piece of information exists in exactly one place
3. **API-First**: Always define APIs before implementation
4. **Code Generation**: Use code generators to ensure consistency between specs and implementation
5. **Documentation Currency**: Keep all documentation 100% up-to-date with automated processes

## Development Guidelines

### When making changes:
1. Check if it affects the MCP protocol compliance
2. Update relevant ADRs if architectural decisions are made
3. Ensure no redundant information is created
4. Verify changes align with API-first methodology
5. Run code generators after API schema changes

### Testing Commands:
[To be added based on chosen technology stack]

### Linting Commands:
[To be added based on chosen technology stack]

## Project Structure

- `/src/api/` - MCP API schema files (multiple files for single API)
- `/docs/adrs/` - Architecture Decision Records with timeline
- `/docs/guidelines/` - Development and contribution guidelines
- Configuration: YAML-based service wiring

## Technology Decisions

- **Language**: Go (ADR-005)
- **Configuration**: YAML (ADR-004)
- **Protocol**: MCP with JSON-RPC 2.0 (ADR-002)
- **First Adapter**: REST API (ADR-006)
- **Testing**: Built-in validation framework (ADR-007)

## Go-Specific Commands

### Building:
```bash
go build -o build/mcpeg cmd/mcpeg/main.go
```

### Testing:
```bash
go test ./...
go test -race ./...
go test -cover ./...
```

### Linting:
```bash
golangci-lint run
go vet ./...
```

### Code Generation:
```bash
go generate ./...
```

## Important Patterns

1. **Adapter Interface**: All adapters must implement the common Adapter interface
2. **Context Usage**: Always pass context.Context for cancellation and timeouts
3. **Error Wrapping**: Use `fmt.Errorf` with `%w` for error wrapping
4. **Structured Logging**: Use structured logging (when logger is added)
5. **Configuration Validation**: Validate all configuration at startup