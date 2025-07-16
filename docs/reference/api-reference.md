# API Reference

Complete reference for the MCpeg Model Context Protocol API.

## Overview

MCpeg implements the Model Context Protocol (MCP) specification version 2025-03-26 over JSON-RPC 2.0. All API calls use HTTP POST requests to the `/mcp` endpoint.

## Base URL

```
POST http://localhost:8080/mcp
```

## Authentication

### JWT Authentication (Optional)

When JWT authentication is enabled, include the authorization header:

```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'
```

### No Authentication (Development)

For development mode, no authentication is required:

```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'
```

## Core MCP Methods

### Initialize

Establishes a connection and negotiates capabilities.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-03-26",
    "capabilities": {
      "tools": {},
      "resources": {},
      "prompts": {}
    }
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2025-03-26",
    "capabilities": {
      "tools": {
        "listChanged": true
      },
      "resources": {
        "subscribe": true,
        "listChanged": true
      },
      "prompts": {
        "listChanged": true
      }
    },
    "serverInfo": {
      "name": "mcpeg",
      "version": "1.0.0"
    }
  }
}
```

### Ping

Health check method to verify server availability.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "ping",
  "params": {}
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "status": "healthy",
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

## Tools API

### List Tools

Get all available tools from enabled plugins.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/list",
  "params": {}
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "tools": [
      {
        "name": "memory_store",
        "description": "Store a key-value pair in persistent memory",
        "inputSchema": {
          "type": "object",
          "properties": {
            "key": {
              "type": "string",
              "description": "The key to store"
            },
            "value": {
              "type": "string",
              "description": "The value to store"
            }
          },
          "required": ["key", "value"]
        }
      },
      {
        "name": "git_status",
        "description": "Show the status of the git repository",
        "inputSchema": {
          "type": "object",
          "properties": {}
        }
      }
    ]
  }
}
```

### Call Tool

Execute a specific tool with provided arguments.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "memory_store",
    "arguments": {
      "key": "user_preference",
      "value": "dark_mode"
    }
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Successfully stored key 'user_preference' with value 'dark_mode'"
      }
    ]
  }
}
```

## Resources API

### List Resources

Get all available resources from enabled plugins.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "resources/list",
  "params": {}
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "resources": [
      {
        "uri": "file:///README.md",
        "name": "README.md",
        "description": "Project README file",
        "mimeType": "text/markdown"
      },
      {
        "uri": "memory://stored_keys",
        "name": "Memory Keys",
        "description": "List of all stored memory keys",
        "mimeType": "application/json"
      }
    ]
  }
}
```

### Read Resource

Read the contents of a specific resource.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "resources/read",
  "params": {
    "uri": "file:///README.md"
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "contents": [
      {
        "uri": "file:///README.md",
        "mimeType": "text/markdown",
        "text": "# MCpeg\n\nModel Context Protocol Enablement Gateway..."
      }
    ]
  }
}
```

### Subscribe to Resource

Subscribe to changes in a resource (if supported).

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "resources/subscribe",
  "params": {
    "uri": "memory://stored_keys"
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "subscribed": true
  }
}
```

## Prompts API

### List Prompts

Get all available prompt templates.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "prompts/list",
  "params": {}
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "prompts": [
      {
        "name": "code_review",
        "description": "Review code for quality and best practices",
        "arguments": [
          {
            "name": "language",
            "description": "Programming language",
            "required": true
          },
          {
            "name": "context",
            "description": "Code context or purpose",
            "required": false
          }
        ]
      }
    ]
  }
}
```

### Get Prompt

Get a specific prompt template with arguments.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "prompts/get",
  "params": {
    "name": "code_review",
    "arguments": {
      "language": "go",
      "context": "web service"
    }
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "description": "Review code for quality and best practices",
    "messages": [
      {
        "role": "user",
        "content": {
          "type": "text",
          "text": "Please review this Go code for a web service. Focus on code quality, best practices, performance, and security concerns."
        }
      }
    ]
  }
}
```

## Built-in Plugin APIs

### Memory Plugin

#### Store Value

**Tool:** `memory_store`

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "memory_store",
    "arguments": {
      "key": "project_config",
      "value": "{\"theme\":\"dark\",\"language\":\"en\"}"
    }
  }
}
```

#### Retrieve Value

**Tool:** `memory_retrieve`

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "memory_retrieve",
    "arguments": {
      "key": "project_config"
    }
  }
}
```

#### List All Keys

**Tool:** `memory_list`

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "memory_list",
    "arguments": {}
  }
}
```

#### Delete Key

**Tool:** `memory_delete`

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "memory_delete",
    "arguments": {
      "key": "project_config"
    }
  }
}
```

### Git Plugin

#### Git Status

**Tool:** `git_status`

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "git_status",
    "arguments": {}
  }
}
```

#### Git Commit

**Tool:** `git_commit`

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "git_commit",
    "arguments": {
      "message": "Add new feature",
      "add_all": true
    }
  }
}
```

#### Git Diff

**Tool:** `git_diff`

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "git_diff",
    "arguments": {
      "staged": false
    }
  }
}
```

#### Git Log

**Tool:** `git_log`

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "git_log",
    "arguments": {
      "limit": 10
    }
  }
}
```

### Editor Plugin

#### Read File

**Tool:** `file_read`

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "file_read",
    "arguments": {
      "path": "src/main.go"
    }
  }
}
```

#### Write File

**Tool:** `file_write`

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "file_write",
    "arguments": {
      "path": "src/example.go",
      "content": "package main\n\nfunc main() {\n    println(\"Hello, World!\")\n}"
    }
  }
}
```

#### List Directory

**Tool:** `file_list`

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "file_list",
    "arguments": {
      "path": "src"
    }
  }
}
```

#### Delete File

**Tool:** `file_delete`

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "file_delete",
    "arguments": {
      "path": "src/example.go"
    }
  }
}
```

## Administrative APIs

### Health Check

**Endpoint:** `GET /health`

**Response:**
```json
{
  "status": "healthy",
  "version": "v1.0.0",
  "uptime": "2h30m15s",
  "timestamp": "2024-01-01T12:00:00Z",
  "components": {
    "gateway": "healthy",
    "plugins": "healthy",
    "memory": "healthy",
    "git": "healthy",
    "editor": "healthy"
  }
}
```

### Metrics

**Endpoint:** `GET /metrics`

**Response:** (Prometheus format)
```
# HELP mcpeg_requests_total Total number of requests
# TYPE mcpeg_requests_total counter
mcpeg_requests_total{method="tools/call"} 150

# HELP mcpeg_request_duration_seconds Request duration in seconds
# TYPE mcpeg_request_duration_seconds histogram
mcpeg_request_duration_seconds_bucket{method="tools/call",le="0.1"} 120
mcpeg_request_duration_seconds_bucket{method="tools/call",le="0.5"} 145
mcpeg_request_duration_seconds_bucket{method="tools/call",le="1.0"} 150
```

### Plugin Management

#### List Plugins

**Endpoint:** `GET /admin/plugins`

**Response:**
```json
{
  "plugins": [
    {
      "name": "memory",
      "version": "1.0.0",
      "enabled": true,
      "status": "healthy",
      "description": "Persistent key-value storage service"
    },
    {
      "name": "git",
      "version": "1.0.0",
      "enabled": true,
      "status": "healthy",
      "description": "Git version control operations"
    }
  ]
}
```

#### Reload Plugin

**Endpoint:** `POST /admin/plugins/reload`

**Request:**
```json
{
  "plugin_name": "memory"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Plugin 'memory' reloaded successfully"
}
```

## Error Handling

### Standard JSON-RPC Errors

#### Parse Error (-32700)
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32700,
    "message": "Parse error"
  },
  "id": null
}
```

#### Invalid Request (-32600)
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32600,
    "message": "Invalid Request"
  },
  "id": null
}
```

#### Method Not Found (-32601)
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32601,
    "message": "Method not found"
  },
  "id": 1
}
```

#### Invalid Params (-32602)
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32602,
    "message": "Invalid params",
    "data": "Missing required parameter: key"
  },
  "id": 1
}
```

#### Internal Error (-32603)
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32603,
    "message": "Internal error",
    "data": "Plugin execution failed"
  },
  "id": 1
}
```

### Custom Application Errors

#### Plugin Not Found (-32000)
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32000,
    "message": "Plugin not found",
    "data": "Plugin 'custom' is not enabled or installed"
  },
  "id": 1
}
```

#### Authentication Required (-32001)
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32001,
    "message": "Authentication required",
    "data": "JWT token is missing or invalid"
  },
  "id": 1
}
```

#### Access Denied (-32002)
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32002,
    "message": "Access denied",
    "data": "Insufficient permissions for this operation"
  },
  "id": 1
}
```

## Rate Limiting

MCpeg supports rate limiting to prevent abuse:

**Rate Limit Headers:**
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1640995200
```

**Rate Limit Exceeded Response:**
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32003,
    "message": "Rate limit exceeded",
    "data": "Too many requests. Limit: 100 per minute"
  },
  "id": 1
}
```

## WebSocket Support

MCpeg supports WebSocket connections for real-time communication:

**Connection:** `ws://localhost:8080/ws`

**WebSocket Message Format:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "memory_store",
    "arguments": {
      "key": "test",
      "value": "websocket"
    }
  }
}
```

## SDK Examples

### JavaScript/Node.js

```javascript
const axios = require('axios');

class MCpegClient {
  constructor(baseUrl = 'http://localhost:8080') {
    this.baseUrl = baseUrl;
    this.requestId = 0;
  }

  async call(method, params = {}) {
    const response = await axios.post(`${this.baseUrl}/mcp`, {
      jsonrpc: '2.0',
      id: ++this.requestId,
      method,
      params
    });
    
    if (response.data.error) {
      throw new Error(response.data.error.message);
    }
    
    return response.data.result;
  }

  async initialize() {
    return this.call('initialize', {
      protocolVersion: '2025-03-26',
      capabilities: { tools: {}, resources: {}, prompts: {} }
    });
  }

  async listTools() {
    return this.call('tools/list');
  }

  async callTool(name, arguments) {
    return this.call('tools/call', { name, arguments });
  }
}

// Usage
const client = new MCpegClient();
await client.initialize();
const tools = await client.listTools();
const result = await client.callTool('memory_store', { key: 'test', value: 'hello' });
```

### Python

```python
import requests
import json

class MCpegClient:
    def __init__(self, base_url='http://localhost:8080'):
        self.base_url = base_url
        self.request_id = 0
    
    def call(self, method, params=None):
        self.request_id += 1
        payload = {
            'jsonrpc': '2.0',
            'id': self.request_id,
            'method': method,
            'params': params or {}
        }
        
        response = requests.post(f'{self.base_url}/mcp', json=payload)
        data = response.json()
        
        if 'error' in data:
            raise Exception(data['error']['message'])
        
        return data['result']
    
    def initialize(self):
        return self.call('initialize', {
            'protocolVersion': '2025-03-26',
            'capabilities': {'tools': {}, 'resources': {}, 'prompts': {}}
        })
    
    def list_tools(self):
        return self.call('tools/list')
    
    def call_tool(self, name, arguments):
        return self.call('tools/call', {'name': name, 'arguments': arguments})

# Usage
client = MCpegClient()
client.initialize()
tools = client.list_tools()
result = client.call_tool('memory_store', {'key': 'test', 'value': 'hello'})
```

### Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

type MCpegClient struct {
    BaseURL   string
    RequestID int
}

type JSONRPCRequest struct {
    JSONRPC string      `json:"jsonrpc"`
    ID      int         `json:"id"`
    Method  string      `json:"method"`
    Params  interface{} `json:"params"`
}

type JSONRPCResponse struct {
    JSONRPC string      `json:"jsonrpc"`
    ID      int         `json:"id"`
    Result  interface{} `json:"result"`
    Error   *JSONRPCError `json:"error"`
}

type JSONRPCError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    string `json:"data"`
}

func (c *MCpegClient) Call(method string, params interface{}) (interface{}, error) {
    c.RequestID++
    request := JSONRPCRequest{
        JSONRPC: "2.0",
        ID:      c.RequestID,
        Method:  method,
        Params:  params,
    }
    
    body, err := json.Marshal(request)
    if err != nil {
        return nil, err
    }
    
    resp, err := http.Post(c.BaseURL+"/mcp", "application/json", bytes.NewBuffer(body))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var response JSONRPCResponse
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return nil, err
    }
    
    if response.Error != nil {
        return nil, fmt.Errorf("JSON-RPC error: %s", response.Error.Message)
    }
    
    return response.Result, nil
}

// Usage
func main() {
    client := &MCpegClient{BaseURL: "http://localhost:8080"}
    
    // Initialize
    result, err := client.Call("initialize", map[string]interface{}{
        "protocolVersion": "2025-03-26",
        "capabilities": map[string]interface{}{
            "tools": map[string]interface{}{},
        },
    })
    if err != nil {
        panic(err)
    }
    
    // List tools
    tools, err := client.Call("tools/list", nil)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Tools: %v\n", tools)
}
```

## Best Practices

### Request Optimization

1. **Batch Requests:** Use batch JSON-RPC requests when possible
2. **Connection Reuse:** Keep HTTP connections alive
3. **Timeout Handling:** Set appropriate timeouts for long-running operations
4. **Error Handling:** Always check for JSON-RPC errors

### Security

1. **Authentication:** Use JWT tokens in production
2. **HTTPS:** Always use HTTPS in production
3. **Rate Limiting:** Implement client-side rate limiting
4. **Input Validation:** Validate all inputs before sending to MCpeg

### Performance

1. **Connection Pooling:** Use connection pooling for multiple requests
2. **Caching:** Cache frequently accessed data
3. **Compression:** Enable gzip compression for large payloads
4. **Monitoring:** Monitor request latency and error rates

## OpenAPI Specification

The complete OpenAPI specification is available at:
- **File:** `api/openapi/mcp-gateway.yaml`
- **Endpoint:** `GET /openapi.json`
- **Documentation:** `GET /docs` (if enabled)

## Next Steps

- **Installation:** [Installation Guide](../guides/installation.md)
- **Configuration:** [Configuration Guide](../guides/configuration.md)
- **Plugin Development:** [Plugin Development Guide](../guides/plugin-development.md)
- **Troubleshooting:** [Troubleshooting Guide](../guides/troubleshooting.md)