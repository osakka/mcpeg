# MCPEG Project Structure

## Standard Go Layout

```
mcpeg/
├── cmd/                    # Main applications
│   └── mcpeg/             # The mcpeg CLI/server
│       └── main.go
├── internal/              # Private application code
│   ├── adapter/          # Service adapters
│   │   ├── rest/        # REST API adapter
│   │   └── binary/      # Binary execution adapter
│   ├── config/          # Configuration management
│   ├── mcp/            # MCP protocol implementation
│   │   ├── server/     # MCP server
│   │   └── handlers/   # Protocol handlers
│   └── validation/      # Validation framework
├── pkg/                   # Public libraries
│   ├── logging/         # LLM-optimized logger
│   ├── templates/       # Template engine
│   └── transform/       # Data transformation
├── src/                   # Schema generation only
│   └── api/            # Generated API schemas
├── docs/                  # Documentation
│   ├── adrs/           # Architecture decisions
│   ├── architecture/   # System design
│   └── guidelines/     # Development guides
├── build/                # Build artifacts
├── go.mod               # Go module definition
├── go.sum               # Go dependencies lock
└── README.md            # Project overview
```

## Key Points

1. **`go.mod` at root**: This is Go standard - never move it to `src/`
2. **`cmd/` for executables**: Each subdirectory is a main package
3. **`internal/` for private code**: Can't be imported by external projects
4. **`pkg/` for public libraries**: Can be imported by others
5. **`src/api/` for generated schemas**: Kept separate from Go code

## Import Examples

With this structure, imports look like:
```go
import (
    "github.com/yourusername/mcpeg/pkg/logging"
    "github.com/yourusername/mcpeg/internal/adapter"
)
```

Not:
```go
// Wrong - if go.mod was in src/
import "github.com/yourusername/mcpeg/src/pkg/logging"
```