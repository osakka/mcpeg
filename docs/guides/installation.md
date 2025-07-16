# Installation Guide

This guide provides comprehensive installation instructions for MCpeg (Model Context Protocol Enablement Gateway).

## Prerequisites

### System Requirements
- **Operating System:** Linux, macOS, or Windows (with WSL2)
- **Go Runtime:** Go 1.21 or later
- **Memory:** Minimum 512MB RAM, 1GB recommended
- **Disk Space:** 100MB for binary and dependencies
- **Network:** Port 8080 available (default, configurable)

### Development Dependencies
- **Git:** For version control and repository cloning
- **Make:** For build automation (optional, scripts work directly)
- **curl:** For API testing and health checks

## Installation Methods

### Method 1: Binary Release (Recommended)

1. **Download Latest Release:**
   ```bash
   curl -L https://github.com/osakka/mcpeg/releases/latest/download/mcpeg-linux-amd64.tar.gz -o mcpeg.tar.gz
   tar -xzf mcpeg.tar.gz
   sudo mv mcpeg /usr/local/bin/
   ```

2. **Verify Installation:**
   ```bash
   mcpeg --version
   ```

### Method 2: Build from Source

1. **Clone Repository:**
   ```bash
   git clone https://github.com/osakka/mcpeg.git
   cd mcpeg
   ```

2. **Build MCpeg:**
   ```bash
   ./scripts/build.sh build
   ```

3. **Install Binary:**
   ```bash
   sudo cp build/mcpeg /usr/local/bin/
   ```

### Method 3: Docker Container

1. **Run MCpeg Container:**
   ```bash
   docker run -p 8080:8080 mcpeg/mcpeg:latest gateway
   ```

2. **Persist Configuration:**
   ```bash
   docker run -p 8080:8080 -v $(pwd)/config:/app/config mcpeg/mcpeg:latest gateway --config /app/config/production.yaml
   ```

## Configuration

### Basic Configuration

1. **Create Configuration Directory:**
   ```bash
   mkdir -p ~/.mcpeg/config
   ```

2. **Copy Default Configuration:**
   ```bash
   cp config/development.yaml ~/.mcpeg/config/mcpeg.yaml
   ```

3. **Edit Configuration:**
   ```bash
   nano ~/.mcpeg/config/mcpeg.yaml
   ```

### Environment Variables

Set these environment variables for production:

```bash
export MCPEG_SERVER_ADDRESS="0.0.0.0:8080"
export MCPEG_LOG_LEVEL="info"
export MCPEG_DEVELOPMENT_MODE="false"
export MCPEG_TLS_ENABLED="true"
export MCPEG_TLS_CERT_PATH="/path/to/cert.pem"
export MCPEG_TLS_KEY_PATH="/path/to/key.pem"
```

## Service Setup

### Systemd Service (Linux)

1. **Create Service File:**
   ```bash
   sudo tee /etc/systemd/system/mcpeg.service << EOF
   [Unit]
   Description=MCpeg Gateway Service
   After=network.target
   
   [Service]
   Type=simple
   User=mcpeg
   Group=mcpeg
   ExecStart=/usr/local/bin/mcpeg gateway --config /etc/mcpeg/config.yaml
   Restart=always
   RestartSec=10
   
   [Install]
   WantedBy=multi-user.target
   EOF
   ```

2. **Enable and Start Service:**
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable mcpeg
   sudo systemctl start mcpeg
   ```

### macOS LaunchDaemon

1. **Create LaunchDaemon:**
   ```bash
   sudo tee /Library/LaunchDaemons/com.mcpeg.gateway.plist << EOF
   <?xml version="1.0" encoding="UTF-8"?>
   <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
   <plist version="1.0">
   <dict>
       <key>Label</key>
       <string>com.mcpeg.gateway</string>
       <key>ProgramArguments</key>
       <array>
           <string>/usr/local/bin/mcpeg</string>
           <string>gateway</string>
           <string>--config</string>
           <string>/etc/mcpeg/config.yaml</string>
       </array>
       <key>RunAtLoad</key>
       <true/>
   </dict>
   </plist>
   EOF
   ```

2. **Load Service:**
   ```bash
   sudo launchctl load /Library/LaunchDaemons/com.mcpeg.gateway.plist
   ```

## Verification

### Health Check

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "healthy",
  "version": "v1.0.0",
  "uptime": "5m30s",
  "components": {
    "gateway": "healthy",
    "plugins": "healthy"
  }
}
```

### MCP Protocol Test

```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{"tools":{}}}}'
```

## Post-Installation

### Configure Plugins

1. **List Available Plugins:**
   ```bash
   mcpeg gateway --list-plugins
   ```

2. **Configure Plugin Settings:**
   Edit the configuration file to enable/disable plugins:
   ```yaml
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

### Security Hardening

1. **Enable TLS:**
   ```yaml
   server:
     tls:
       enabled: true
       cert_path: "/path/to/cert.pem"
       key_path: "/path/to/key.pem"
   ```

2. **Configure Authentication:**
   ```yaml
   auth:
     jwt:
       enabled: true
       secret_key: "your-secret-key"
       issuer: "mcpeg"
   ```

## Troubleshooting

### Common Issues

1. **Port Already in Use:**
   ```bash
   # Check what's using port 8080
   lsof -i :8080
   # Change port in configuration
   ```

2. **Permission Denied:**
   ```bash
   # Ensure user has permissions
   sudo chown -R mcpeg:mcpeg /var/lib/mcpeg
   ```

3. **Configuration Not Found:**
   ```bash
   # Verify config file location
   mcpeg gateway --config /path/to/config.yaml --dry-run
   ```

### Log Files

- **Development:** Console output
- **Production:** `/var/log/mcpeg/gateway.log`
- **Docker:** `docker logs <container_id>`

## Next Steps

- Read the [Quick Start Guide](quick-start.md) for immediate usage
- Review [Configuration Guide](configuration.md) for advanced setup
- Explore [API Reference](../reference/api-reference.md) for integration
- Check [Troubleshooting Guide](troubleshooting.md) for common issues

## Support

- **Documentation:** [User Guide](user-guide.md)
- **Issues:** [GitHub Issues](https://github.com/osakka/mcpeg/issues)
- **Community:** [Discussions](https://github.com/osakka/mcpeg/discussions)