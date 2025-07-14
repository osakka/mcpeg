# Claude Desktop Integration with MCpeg

This document explains how to configure Claude Desktop to connect to MCpeg using native HTTP transport.

## Quick Setup

1. **Start MCpeg Gateway:**
   ```bash
   cd /opt/mcpeg
   ./build/mcpeg gateway --dev --config config/development.yaml
   ```

2. **Configure Claude Desktop:**
   Copy the appropriate configuration to your Claude Desktop config directory:

   **For same machine (localhost):**
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

   **For remote machine:**
   ```json
   {
     "mcpServers": {
       "mcpeg-gateway": {
         "transport": {
           "type": "http",
           "url": "http://YOUR_SERVER_IP:8080/mcp",
           "headers": {
             "Content-Type": "application/json",
             "Accept": "application/json"
           }
         }
       }
     }
   }
   ```

## Configuration Files Provided

All configuration files are located in `config/claude-desktop/`:

- `claude_desktop_native_http.json` - Basic native HTTP config for localhost
- `claude_desktop_native_http_complete.json` - Complete config with environment variables
- `claude_desktop_http_config.json` - Alternative HTTP config
- `claude_desktop_tunnel_config.json` - SSH tunnel config for remote access
- `claude_desktop_sse_config.json` - Server-Sent Events config (if supported)
- `claude_desktop_config.json` - Generic configuration template

## MCpeg HTTP Endpoints

MCpeg exposes the MCP protocol over HTTP at these endpoints:

- **Primary MCP endpoint:** `POST /mcp` (JSON-RPC 2.0)
- **Method-specific endpoints:** (when enabled)
  - `POST /mcp/tools/list`
  - `POST /mcp/tools/call`
  - `POST /mcp/resources/list`
  - `POST /mcp/resources/read`
  - `POST /mcp/prompts/list`

## Built-in Plugins

MCpeg includes these built-in plugins:

- **Memory Plugin:** Persistent key-value storage across sessions
- **Git Plugin:** Git version control operations
- **Editor Plugin:** File system operations and editing capabilities

## Troubleshooting

1. **"no services available" error:**
   - Ensure MCpeg gateway is running with proper configuration
   - Check that built-in plugins are loaded successfully
   - Verify the correct endpoint URL (`/mcp` not `/api/v1`)

2. **Connection refused:**
   - Verify MCpeg is running on port 8080
   - Check firewall settings for remote connections
   - Ensure the correct IP address is used

3. **Cross-machine connectivity:**
   - Use the SSH tunnel config for secure remote access
   - Or configure MCpeg for external access and use HTTP config

## Server Configuration

Default MCpeg configuration (development.yaml):
- **Address:** `0.0.0.0:8080`
- **TLS:** Disabled (development mode)
- **Plugins:** Memory, Git, Editor (built-in)