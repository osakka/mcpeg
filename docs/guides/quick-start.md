# Quick Start Guide

Get MCpeg running in 5 minutes with this streamlined guide.

## Prerequisites

- Go 1.21+ installed
- Git installed
- Port 8080 available

## 1. Install MCpeg

### Option A: Build from Source (Recommended)

```bash
git clone https://github.com/osakka/mcpeg.git
cd mcpeg
./scripts/build.sh build
```

### Option B: Binary Release

```bash
curl -L https://github.com/osakka/mcpeg/releases/latest/download/mcpeg-linux-amd64.tar.gz -o mcpeg.tar.gz
tar -xzf mcpeg.tar.gz
```

## 2. Start MCpeg Gateway

```bash
# From source build
./build/mcpeg gateway --dev

# From binary
./mcpeg gateway --dev
```

You should see:
```
 __  __  ____  ____   ______ _____ 
|  \/  |/ ___||  _ \ |  ____/ ____|
| |\/| | |    | |_) || |__ | |  __ 
| |  | | |    |  __/ |  __|| | |_ |
| |  | | |____| |    | |___| |__| |
|_|  |_|\_____|_|    |______\_____|

Model Context Protocol Enablement Gateway
Starting on 0.0.0.0:8080 (TLS: false, Dev: true)
```

## 3. Test Installation

### Health Check
```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "healthy",
  "version": "v1.0.0",
  "uptime": "30s"
}
```

### MCP Protocol Test
```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{"tools":{}}}}'
```

## 4. Configure Claude Desktop

### Copy Configuration
```bash
cp config/claude-desktop/claude_desktop_native_http.json ~/.config/claude-desktop/claude_desktop_config.json
```

### Configuration Content
```json
{
  "mcpServers": {
    "mcpeg-gateway": {
      "transport": {
        "type": "http",
        "url": "http://localhost:8080/mcp",
        "headers": {
          "Content-Type": "application/json",
          "Accept": "application/json"
        }
      }
    }
  }
}
```

## 5. Test MCP Tools

### List Available Tools
```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

### Test Memory Plugin
```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"memory_store","arguments":{"key":"test","value":"Hello World"}}}'
```

## 6. Explore Built-in Plugins

MCpeg includes three built-in plugins:

### Memory Plugin
- **Purpose:** Persistent key-value storage
- **Tools:** `memory_store`, `memory_retrieve`, `memory_delete`
- **Usage:** Store context across sessions

### Git Plugin  
- **Purpose:** Version control operations
- **Tools:** `git_status`, `git_commit`, `git_diff`
- **Usage:** Manage code repositories

### Editor Plugin
- **Purpose:** File system operations
- **Tools:** `file_read`, `file_write`, `file_list`
- **Usage:** Edit and manage files

## Common Commands

### Development Mode
```bash
mcpeg gateway --dev --config config/development.yaml
```

### Production Mode
```bash
mcpeg gateway --config config/production.yaml
```

### List Plugins
```bash
mcpeg gateway --list-plugins
```

### Validate Configuration
```bash
mcpeg validate --config config/development.yaml
```

### Generate Code
```bash
mcpeg codegen --spec api/openapi/mcp-gateway.yaml
```

## Next Steps

- **Complete Setup:** Read the [Installation Guide](installation.md)
- **Configuration:** Check the [Configuration Guide](configuration.md)
- **API Usage:** Explore the [API Reference](../reference/api-reference.md)
- **Integration:** Review [User Guide](user-guide.md) for detailed usage
- **Development:** See [Plugin Development Guide](plugin-development.md)

## Need Help?

- **Issues:** Common problems in [Troubleshooting Guide](troubleshooting.md)
- **Documentation:** Full reference in [User Guide](user-guide.md)
- **Community:** [GitHub Discussions](https://github.com/osakka/mcpeg/discussions)

## Configuration Files

All Claude Desktop configurations are available in `config/claude-desktop/`:
- `claude_desktop_native_http.json` - Native HTTP (localhost)
- `claude_desktop_tunnel_config.json` - SSH tunnel (remote)
- `claude_desktop_http_config.json` - Direct HTTP (remote)
- `README_CLAUDE_DESKTOP.md` - Complete setup guide

You're now ready to use MCpeg with Claude Desktop! 🚀