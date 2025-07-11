# Source Code Directory (Legacy)

This directory is maintained for API schema generation only.

## Contents

- `/api/` - MCP API schema definitions and generated code
  - OpenAPI/JSON Schema specifications
  - Generated Go types and validators
  - No manual modifications to generated code

## Note on Go Project Structure

The actual Go source code follows standard Go project layout at the repository root:

- `/cmd/mcpeg/` - Main application entry point
- `/internal/` - Private application code
- `/pkg/` - Public libraries
- `/go.mod` - Go module definition (at root)

This separation allows us to keep generated API schemas isolated while following Go conventions.

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