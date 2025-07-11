# MCP Endpoint Landscape

## MCP Protocol Overview

MCPEG implements the full Model Context Protocol (MCP) specification, acting as an MCP server that exposes backend services through standardized MCP methods.

## Core MCP Concepts We Expose

### 1. **Resources** 
Static or dynamic data that can be read by LLMs
```
Example: file://database/schema, vault://secrets/api-keys
```

### 2. **Tools**
Functions that LLMs can call to perform actions
```
Example: query_database, get_weather, restart_service
```

### 3. **Prompts**
Reusable prompt templates with parameters
```
Example: sql_analyzer, incident_report, code_review
```

## MCP Protocol Methods (JSON-RPC 2.0)

### Server Lifecycle
```json
POST /mcp
{
  "jsonrpc": "2.0",
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-03-26",
    "capabilities": {
      "resources": {},
      "tools": {},
      "prompts": {},
      "logging": {}
    },
    "clientInfo": {
      "name": "claude-desktop",
      "version": "1.0.0"
    }
  },
  "id": 1
}
```

### Resource Methods
```json
// List available resources
{
  "method": "resources/list",
  "params": {}
}

// Read a specific resource
{
  "method": "resources/read", 
  "params": {
    "uri": "mysql://localhost:3306/schema/users"
  }
}
```

### Tool Methods
```json
// List available tools
{
  "method": "tools/list",
  "params": {}
}

// Call a tool
{
  "method": "tools/call",
  "params": {
    "name": "query_database",
    "arguments": {
      "query": "SELECT * FROM users WHERE active = true",
      "database": "prod"
    }
  }
}
```

### Prompt Methods
```json
// List available prompts
{
  "method": "prompts/list",
  "params": {}
}

// Get a prompt
{
  "method": "prompts/get",
  "params": {
    "name": "incident_analysis",
    "arguments": {
      "severity": "high",
      "service": "api"
    }
  }
}
```

## MCPEG Landscape Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        MCP Clients                             │
│  Claude Desktop │ VSCode Plugin │ Custom Apps │ API Clients    │
└─────────────────┬───────────────────────────────────────────────┘
                  │ JSON-RPC 2.0 over stdio/HTTP
                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                      MCPEG Gateway                             │
│                                                                 │
│  ┌─────────────────┐    ┌──────────────────────────────────┐   │
│  │   MCP Server    │    │      Configuration Manager       │   │
│  │                 │    │                                  │   │
│  │ • Protocol      │◄───┤ • Service Registration          │   │
│  │   Handler       │    │ • Dynamic Loading               │   │
│  │ • Method Router │    │ • Health Monitoring             │   │
│  │ • Transport     │    └──────────────────────────────────┘   │
│  │   Layer         │                                           │
│  └─────────┬───────┘                                           │
│            │                                                   │
│  ┌─────────▼──────────────────────────────────────────────┐   │
│  │                Service Router                          │   │
│  │  Routes MCP calls to appropriate service adapters     │   │
│  └─────────┬──────────────────────────────────────────────┘   │
│            │                                                   │
│  ┌─────────▼──────────────────────────────────────────────┐   │
│  │               Service Adapters                         │   │
│  │                                                        │   │
│  │ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │   │
│  │ │  MySQL   │ │  Vault   │ │ Weather  │ │  Script  │  │   │
│  │ │ Adapter  │ │ Adapter  │ │ Adapter  │ │ Adapter  │  │   │
│  │ └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘  │   │
│  └──────┼────────────┼────────────┼────────────┼────────┘   │
└─────────┼────────────┼────────────┼────────────┼────────────┘
          │            │            │            │
          ▼            ▼            ▼            ▼
    ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐
    │  MySQL   │ │HashiCorp │ │Weather   │ │  Local   │
    │Database  │ │  Vault   │ │   API    │ │ Scripts  │
    └──────────┘ └──────────┘ └──────────┘ └──────────┘
```

## Service-Specific MCP Exposures

### Database Services (MySQL, PostgreSQL)

**Resources:**
- `mysql://server/schema/{table}` - Table schema information
- `mysql://server/data/{table}?limit=100` - Sample data
- `mysql://server/indexes/{table}` - Index information

**Tools:**
- `query_database` - Execute SQL queries
- `explain_query` - Analyze query performance
- `get_table_stats` - Get table statistics

**Prompts:**
- `sql_optimization` - SQL query optimization guidance
- `schema_analysis` - Database schema review

### Secrets Management (Vault)

**Resources:**
- `vault://secrets/{path}` - Secret metadata (not values)
- `vault://policies/{name}` - Access policies

**Tools:**
- `get_secret` - Retrieve secret values
- `list_secrets` - List available secrets
- `rotate_secret` - Trigger secret rotation

**Prompts:**
- `security_audit` - Security configuration review
- `access_review` - Access policy analysis

### Weather Services

**Resources:**
- `weather://current/{location}` - Current conditions
- `weather://forecast/{location}` - Weather forecast

**Tools:**
- `get_weather` - Get current weather
- `get_forecast` - Get weather forecast
- `weather_alerts` - Get weather alerts

**Prompts:**
- `weather_analysis` - Weather pattern analysis
- `travel_advisory` - Travel weather recommendations

### Script Execution

**Resources:**
- `script://available` - List available scripts
- `script://logs/{script}` - Script execution logs

**Tools:**
- `run_script` - Execute a script
- `get_script_status` - Check script status
- `kill_script` - Stop running script

**Prompts:**
- `automation_help` - Script automation guidance
- `troubleshooting` - System troubleshooting steps

## Transport Endpoints

### 1. **stdio Transport** (Primary)
```bash
mcpeg serve --transport stdio --config config.yaml
```
- Used by Claude Desktop and similar MCP clients
- JSON-RPC over stdin/stdout
- Process-to-process communication

### 2. **HTTP Transport** (Secondary)
```bash
mcpeg serve --transport http --port 8080 --config config.yaml
```
- RESTful MCP over HTTP
- WebSocket upgrade for streaming
- Suitable for web applications

**HTTP Endpoints:**
```
POST /mcp/v1/initialize
POST /mcp/v1/resources/list
POST /mcp/v1/resources/read
POST /mcp/v1/tools/list
POST /mcp/v1/tools/call
POST /mcp/v1/prompts/list
POST /mcp/v1/prompts/get

# Management endpoints
GET  /health
GET  /metrics
GET  /services
POST /services/{name}/enable
POST /services/{name}/disable
```

## Capability Advertisement

When clients connect, MCPEG advertises its capabilities:

```json
{
  "capabilities": {
    "resources": {
      "subscribe": true,
      "listChanged": true
    },
    "tools": {
      "listChanged": true
    },
    "prompts": {
      "listChanged": true
    },
    "logging": {
      "level": "info"
    }
  },
  "serverInfo": {
    "name": "mcpeg",
    "version": "0.1.0"
  }
}
```

## Dynamic Service Discovery

Services can be enabled/disabled at runtime:

```yaml
# Configuration hot-reload
services:
  mysql:
    enabled: true
  vault:
    enabled: false  # Disabled, won't appear in MCP responses
  weather:
    enabled: true
```

## Error Handling

All MCP errors follow JSON-RPC 2.0 error format:

```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32603,
    "message": "Internal error",
    "data": {
      "type": "service_unavailable",
      "service": "mysql",
      "details": "Connection timeout after 30s",
      "suggested_actions": [
        "Check database connectivity",
        "Verify credentials",
        "Increase timeout setting"
      ]
    }
  },
  "id": 123
}
```

## Monitoring and Observability

**Metrics Exposed:**
- Total MCP requests per method
- Request latency percentiles
- Service adapter health status
- Active connection count
- Memory usage per service

**Logging:**
- Every MCP request/response logged
- Service adapter state changes
- Configuration updates
- Error conditions with context

This landscape provides a complete MCP gateway that can integrate any backend service while maintaining the standard MCP protocol interface for clients.