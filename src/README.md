# Source Code Directory

This directory contains all source code for the MCPEG project.

## Go Project Structure

- `/api/` - MCP API schema definitions and generated code
  - OpenAPI/JSON Schema specifications
  - Generated Go types and validators
  - No manual modifications to generated code
  
- `/cmd/mcpeg/` - Main application entry point
  - Command-line interface
  - Server initialization
  
- `/internal/` - Private application code
  - `/adapter/` - Service adapter interfaces and implementations
    - `/rest/` - REST API adapter
    - `/binary/` - Binary execution adapter (future)
  - `/config/` - Configuration loading and validation
  - `/mcp/` - MCP protocol implementation
    - `/server/` - MCP server implementation
    - `/handlers/` - Protocol method handlers
  - `/validation/` - Built-in validation framework
    - `/compliance/` - MCP compliance testing
    - `/config/` - Configuration validation
    - `/diagnostics/` - Diagnostic endpoints
    
- `/pkg/` - Public libraries (can be imported by external projects)
  - `/templates/` - Template engine for value substitution
  - `/transform/` - Response transformation utilities

## Development Guidelines

1. **Go Standards**: Follow standard Go project layout and idioms
2. **Generated Code**: Files in `/api/` are generated. Do not edit manually
3. **Internal vs Pkg**: Use `/internal/` for app-specific code, `/pkg/` for reusable libraries
4. **Testing**: Each package should have `*_test.go` files
5. **Interfaces**: Define interfaces in the package that uses them

## Build Commands

```bash
# Build the application
go build -o ../build/mcpeg cmd/mcpeg/main.go

# Run tests
go test ./...

# Run with race detector
go test -race ./...

# Generate code from schemas
go generate ./...

# Run linting
golangci-lint run

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o ../build/mcpeg-linux-amd64 cmd/mcpeg/main.go
GOOS=darwin GOARCH=amd64 go build -o ../build/mcpeg-darwin-amd64 cmd/mcpeg/main.go
GOOS=windows GOARCH=amd64 go build -o ../build/mcpeg-windows-amd64.exe cmd/mcpeg/main.go
```