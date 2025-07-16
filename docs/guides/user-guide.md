# User Guide

Complete guide to using MCpeg (Model Context Protocol Enablement Gateway) for integrating external services with AI assistants.

## Overview

MCpeg is a lightweight service that provides a Model Context Protocol (MCP) API, enabling AI assistants to interact with external services through standardized tools, resources, and prompts.

## Core Concepts

### MCP Protocol
The Model Context Protocol is a JSON-RPC 2.0 based protocol that allows AI assistants to:
- **Call Tools:** Execute actions on external services
- **Read Resources:** Access data from various sources
- **Use Prompts:** Leverage predefined prompt templates

### Gateway Architecture
MCpeg acts as a gateway between AI assistants and external services:
```
AI Assistant ↔ MCpeg Gateway ↔ External Services
```

### Plugin System
MCpeg extends functionality through plugins:
- **Built-in Plugins:** Memory, Git, Editor
- **Custom Plugins:** Develop your own integrations
- **Hot Reloading:** Update plugins without restart

## Getting Started

### Basic Usage

1. **Start the Gateway:**
   ```bash
   mcpeg gateway --dev
   ```

2. **Test Connection:**
   ```bash
   curl http://localhost:8080/health
   ```

3. **Initialize MCP Session:**
   ```bash
   curl -X POST http://localhost:8080/mcp \
     -H "Content-Type: application/json" \
     -d '{
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
     }'
   ```

### Configuration

MCpeg uses YAML configuration files:

```yaml
# config/development.yaml
server:
  address: "0.0.0.0:8080"
  tls:
    enabled: false
  
logging:
  level: "debug"
  format: "json"

plugins:
  memory:
    enabled: true
    data_file: "data/memory_storage.json"
  git:
    enabled: true
    working_dir: "."
  editor:
    enabled: true
    max_file_size: 10485760
```

## Built-in Plugins

### Memory Plugin

Provides persistent key-value storage across sessions.

**Tools:**
- `memory_store` - Store key-value pairs
- `memory_retrieve` - Retrieve stored values
- `memory_delete` - Delete stored keys
- `memory_list` - List all stored keys

**Example Usage:**
```bash
# Store value
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "memory_store",
      "arguments": {
        "key": "project_status",
        "value": "In development"
      }
    }
  }'

# Retrieve value
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "memory_retrieve",
      "arguments": {
        "key": "project_status"
      }
    }
  }'
```

### Git Plugin

Provides version control operations.

**Tools:**
- `git_status` - Show repository status
- `git_commit` - Create commits
- `git_diff` - Show changes
- `git_log` - Show commit history
- `git_branch` - List/create branches

**Example Usage:**
```bash
# Check status
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "git_status",
      "arguments": {}
    }
  }'

# Create commit
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
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
  }'
```

### Editor Plugin

Provides file system operations.

**Tools:**
- `file_read` - Read file contents
- `file_write` - Write file contents
- `file_list` - List directory contents
- `file_delete` - Delete files
- `file_create` - Create new files

**Example Usage:**
```bash
# Read file
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "file_read",
      "arguments": {
        "path": "README.md"
      }
    }
  }'

# Write file
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "file_write",
      "arguments": {
        "path": "example.txt",
        "content": "Hello, World!"
      }
    }
  }'
```

## Resources

MCpeg provides access to various resources through the MCP protocol.

### List Resources
```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "resources/list",
    "params": {}
  }'
```

### Read Resource
```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "resources/read",
    "params": {
      "uri": "file:///path/to/file.txt"
    }
  }'
```

## Prompts

MCpeg provides predefined prompt templates for common tasks.

### List Prompts
```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "prompts/list",
    "params": {}
  }'
```

### Get Prompt
```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
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
  }'
```

## Claude Desktop Integration

### Configuration Setup

1. **Copy Configuration:**
   ```bash
   cp config/claude-desktop/claude_desktop_native_http.json ~/.config/claude-desktop/claude_desktop_config.json
   ```

2. **Start MCpeg:**
   ```bash
   mcpeg gateway --dev
   ```

3. **Restart Claude Desktop** to load the configuration

### Available Configurations

- **Native HTTP:** Direct HTTP connection (localhost)
- **SSH Tunnel:** Secure connection to remote server
- **HTTP Remote:** Direct HTTP to remote server

See `config/claude-desktop/README_CLAUDE_DESKTOP.md` for complete setup instructions.

## Advanced Usage

### Custom Configuration

Create custom configuration files:

```yaml
# config/custom.yaml
server:
  address: "0.0.0.0:9000"
  tls:
    enabled: true
    cert_path: "/path/to/cert.pem"
    key_path: "/path/to/key.pem"

auth:
  jwt:
    enabled: true
    secret_key: "your-secret-key"
    issuer: "mcpeg"

plugins:
  memory:
    enabled: true
    data_file: "/var/lib/mcpeg/memory.json"
  
  custom_plugin:
    enabled: true
    config_path: "/etc/mcpeg/custom-plugin.yaml"
```

### Environment Variables

Override configuration with environment variables:

```bash
export MCPEG_SERVER_ADDRESS="0.0.0.0:9000"
export MCPEG_LOG_LEVEL="info"
export MCPEG_PLUGINS_MEMORY_ENABLED="true"
export MCPEG_PLUGINS_MEMORY_DATA_FILE="/custom/path/memory.json"
```

### Plugin Management

#### List Available Plugins
```bash
mcpeg gateway --list-plugins
```

#### Enable/Disable Plugins
```yaml
plugins:
  memory:
    enabled: true
  git:
    enabled: false
  editor:
    enabled: true
```

#### Plugin Hot Reloading
```bash
# Reload specific plugin
curl -X POST http://localhost:8080/admin/plugins/reload \
  -H "Content-Type: application/json" \
  -d '{"plugin_name": "memory"}'

# Reload all plugins
curl -X POST http://localhost:8080/admin/plugins/reload-all
```

## Monitoring and Debugging

### Health Checks
```bash
curl http://localhost:8080/health
```

### Metrics
```bash
curl http://localhost:8080/metrics
```

### Logging

MCpeg provides structured logging optimized for LLM debugging:

```json
{
  "timestamp": "2024-01-01T12:00:00Z",
  "level": "info",
  "message": "mcp_request_processed",
  "method": "tools/call",
  "tool_name": "memory_store",
  "duration_ms": 15,
  "success": true
}
```

### Debug Mode
```bash
mcpeg gateway --dev --log-level debug
```

## Security

### Authentication

Enable JWT authentication:

```yaml
auth:
  jwt:
    enabled: true
    secret_key: "your-secure-secret-key"
    issuer: "mcpeg"
    expiration: "24h"
```

### TLS Configuration

Enable HTTPS:

```yaml
server:
  tls:
    enabled: true
    cert_path: "/path/to/cert.pem"
    key_path: "/path/to/key.pem"
```

### Access Control

Configure plugin access control:

```yaml
rbac:
  enabled: true
  policies:
    - name: "readonly"
      permissions:
        - "tools:read"
        - "resources:read"
    - name: "admin"
      permissions:
        - "tools:*"
        - "resources:*"
        - "admin:*"
```

## Performance Tuning

### Resource Limits
```yaml
server:
  max_concurrent_requests: 100
  request_timeout: "30s"
  read_timeout: "10s"
  write_timeout: "10s"
```

### Plugin Configuration
```yaml
plugins:
  memory:
    cache_size: 1000
    persist_interval: "5m"
  editor:
    max_file_size: 10485760
    allowed_extensions: [".go", ".js", ".py", ".md"]
```

## Troubleshooting

### Common Issues

1. **Port Already in Use:**
   ```bash
   lsof -i :8080
   # Change port in configuration
   ```

2. **Plugin Not Loading:**
   ```bash
   mcpeg gateway --dev --log-level debug
   # Check plugin logs
   ```

3. **Memory Issues:**
   ```bash
   # Check memory usage
   curl http://localhost:8080/health
   ```

### Debug Commands

```bash
# Validate configuration
mcpeg validate --config config/development.yaml

# Test plugin
mcpeg gateway --test-plugin memory

# Check connectivity
curl -v http://localhost:8080/health
```

## Best Practices

### Configuration Management
- Use environment-specific config files
- Override with environment variables
- Validate configurations before deployment

### Plugin Development
- Follow the plugin API specification
- Implement proper error handling
- Use structured logging

### Security
- Enable TLS in production
- Use strong JWT secrets
- Implement proper access controls

### Monitoring
- Set up health checks
- Monitor plugin performance
- Use structured logging for debugging

## Next Steps

- **Advanced Configuration:** [Configuration Guide](configuration.md)
- **API Integration:** [API Reference](../reference/api-reference.md)
- **Plugin Development:** [Plugin Development Guide](plugin-development.md)
- **Troubleshooting:** [Troubleshooting Guide](troubleshooting.md)
- **Contributing:** [Contributing Guide](../processes/contributing.md)

## Support

- **Documentation:** Complete reference documentation
- **Issues:** [GitHub Issues](https://github.com/osakka/mcpeg/issues)
- **Community:** [GitHub Discussions](https://github.com/osakka/mcpeg/discussions)
- **Examples:** [Example Configurations](../examples/)