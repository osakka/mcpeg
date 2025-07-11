# MCP Protocol Implementation

This package implements the Model Context Protocol (MCP) server functionality.

## Package Structure

```
mcp/
├── server/          # Core MCP server implementation
├── handlers/        # Protocol method handlers  
├── transport/       # Transport layer (stdio, HTTP)
├── types/          # MCP protocol types and schemas
└── router/         # Request routing to service adapters
```

## Core Components

### MCP Server
The main server that orchestrates all MCP protocol handling:

```go
type Server struct {
    config     *Config
    handlers   map[string]Handler
    router     *Router
    transports []Transport
    logger     logging.Logger
}
```

### Protocol Handlers
Each MCP method has a dedicated handler:

- `InitializeHandler` - Server initialization
- `ResourcesListHandler` - List available resources
- `ResourcesReadHandler` - Read specific resource
- `ToolsListHandler` - List available tools
- `ToolsCallHandler` - Execute tool
- `PromptsListHandler` - List available prompts
- `PromptsGetHandler` - Get prompt template

### Transport Layer
Supports multiple transport mechanisms:

- **stdio**: JSON-RPC over stdin/stdout (primary)
- **HTTP**: RESTful JSON-RPC over HTTP (secondary)

### Request Router
Routes MCP requests to appropriate service adapters based on:
- Resource URI schemes
- Tool names
- Service availability

## MCP Protocol Flow

```
Client Request → Transport → Server → Handler → Router → Service Adapter
                                                              ↓
Client Response ← Transport ← Server ← Handler ← Router ← Service Adapter
```

## Implementation Principles

1. **Protocol Compliance**: Strict adherence to MCP specification
2. **Service Isolation**: Each adapter handles its own MCP methods
3. **Error Propagation**: Detailed error context for troubleshooting
4. **Performance**: Async handling with controlled concurrency
5. **Observability**: Complete request/response logging

## Usage Example

```go
// Create MCP server
server := mcp.NewServer(config, logger)

// Register service adapters
server.RegisterAdapter("mysql", mysqlAdapter)
server.RegisterAdapter("vault", vaultAdapter)

// Start transports
server.StartStdio()
server.StartHTTP(":8080")

// Handle graceful shutdown
server.Shutdown(ctx)
```