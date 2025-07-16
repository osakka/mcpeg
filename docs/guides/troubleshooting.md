# Troubleshooting Guide

Common issues and solutions for MCpeg deployment and usage.

## Installation Issues

### Go Version Compatibility

**Problem:** Build fails with Go version error
```
go: mcpeg requires Go 1.21 or later
```

**Solution:**
```bash
# Check Go version
go version

# Install Go 1.21+
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

### Build Script Permissions

**Problem:** Build script not executable
```
./scripts/build.sh: Permission denied
```

**Solution:**
```bash
chmod +x scripts/build.sh
./scripts/build.sh build
```

### Missing Dependencies

**Problem:** Build fails with missing dependencies
```
package github.com/some/package: cannot find package
```

**Solution:**
```bash
go mod tidy
go mod download
./scripts/build.sh build
```

## Server Startup Issues

### Port Already in Use

**Problem:** Server fails to start
```
[ERROR] failed to start server: listen tcp :8080: bind: address already in use
```

**Solution:**
```bash
# Check what's using port 8080
lsof -i :8080
sudo netstat -tlnp | grep :8080

# Kill process or change port
sudo kill -9 <PID>
# OR
mcpeg gateway --address 0.0.0.0:8081
```

### Configuration File Not Found

**Problem:** Configuration file error
```
[ERROR] config file not found: config/production.yaml
```

**Solution:**
```bash
# Check file exists
ls -la config/production.yaml

# Use absolute path
mcpeg gateway --config $(pwd)/config/production.yaml

# Use default development config
mcpeg gateway --dev
```

### Permission Denied Errors

**Problem:** Cannot write to data directory
```
[ERROR] failed to create data file: permission denied
```

**Solution:**
```bash
# Create data directory with proper permissions
sudo mkdir -p /var/lib/mcpeg
sudo chown mcpeg:mcpeg /var/lib/mcpeg
sudo chmod 755 /var/lib/mcpeg

# Or use local directory
mkdir -p data
mcpeg gateway --dev
```

## TLS/SSL Issues

### Certificate Errors

**Problem:** TLS certificate validation fails
```
[ERROR] tls: failed to load certificate: no such file or directory
```

**Solution:**
```bash
# Generate self-signed certificate for testing
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes

# Or disable TLS for development
mcpeg gateway --dev --tls=false
```

### Certificate Path Issues

**Problem:** Certificate files not accessible
```
[ERROR] tls: failed to load certificate: permission denied
```

**Solution:**
```bash
# Check certificate permissions
ls -la /etc/ssl/certs/mcpeg.pem
ls -la /etc/ssl/private/mcpeg.key

# Fix permissions
sudo chmod 644 /etc/ssl/certs/mcpeg.pem
sudo chmod 600 /etc/ssl/private/mcpeg.key
sudo chown mcpeg:mcpeg /etc/ssl/private/mcpeg.key
```

## Plugin Issues

### Plugin Not Loading

**Problem:** Plugin fails to load
```
[ERROR] failed to load plugin: plugin not found
```

**Solution:**
```bash
# Check plugin configuration
mcpeg gateway --list-plugins

# Enable plugin in config
vim config/development.yaml
# Set enabled: true for the plugin

# Check plugin dependencies
mcpeg gateway --test-plugin memory
```

### Memory Plugin Issues

**Problem:** Memory plugin data file corruption
```
[ERROR] memory plugin: failed to load data file: invalid JSON
```

**Solution:**
```bash
# Backup corrupted file
cp data/memory_storage.json data/memory_storage.json.bak

# Reset memory plugin data
echo '{}' > data/memory_storage.json

# Or delete and restart
rm data/memory_storage.json
mcpeg gateway --dev
```

### Git Plugin Issues

**Problem:** Git plugin fails in non-git directory
```
[ERROR] git plugin: not a git repository
```

**Solution:**
```bash
# Initialize git repository
git init

# Or set different working directory
export MCPEG_PLUGINS_GIT_WORKING_DIR="/path/to/git/repo"

# Or disable git plugin
mcpeg gateway --disable-plugin git
```

### Editor Plugin Issues

**Problem:** File size limit exceeded
```
[ERROR] editor plugin: file size exceeds maximum limit
```

**Solution:**
```bash
# Increase file size limit in config
vim config/development.yaml
# Set max_file_size: 52428800  # 50MB

# Or use environment variable
export MCPEG_PLUGINS_EDITOR_MAX_FILE_SIZE=52428800
```

## MCP Protocol Issues

### JSON-RPC Errors

**Problem:** Invalid JSON-RPC request
```
{"jsonrpc":"2.0","error":{"code":-32700,"message":"Parse error"},"id":null}
```

**Solution:**
```bash
# Validate JSON format
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{"tools":{}}}}'

# Check content-type header
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

### Method Not Found

**Problem:** MCP method not available
```
{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"},"id":1}
```

**Solution:**
```bash
# List available methods
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{"tools":{}}}}'

# Check plugin is enabled
mcpeg gateway --list-plugins
```

### No Services Available

**Problem:** No services available for method
```
{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal error","data":"no services available for method: initialize"},"id":null}
```

**Solution:**
```bash
# Ensure plugins are loaded
mcpeg gateway --dev --log-level debug

# Check service registry
curl http://localhost:8080/debug/services

# Restart with fresh configuration
mcpeg gateway --dev --config config/development.yaml
```

## Claude Desktop Integration Issues

### Connection Refused

**Problem:** Claude Desktop cannot connect to MCpeg
```
Connection refused to http://localhost:8080/mcp
```

**Solution:**
```bash
# Check MCpeg is running
curl http://localhost:8080/health

# Check firewall
sudo ufw status
sudo ufw allow 8080

# Check Claude Desktop config
cat ~/.config/claude-desktop/claude_desktop_config.json
```

### Authentication Errors

**Problem:** JWT authentication fails
```
{"error":"invalid token"}
```

**Solution:**
```bash
# Disable JWT for testing
export MCPEG_AUTH_JWT_ENABLED=false

# Or configure JWT properly
export MCPEG_AUTH_JWT_SECRET_KEY="your-secret-key"
export MCPEG_AUTH_JWT_ISSUER="mcpeg"
```

### Configuration Not Loading

**Problem:** Claude Desktop doesn't load MCpeg server
```
MCpeg server not available in Claude Desktop
```

**Solution:**
```bash
# Check config file location
ls -la ~/.config/claude-desktop/claude_desktop_config.json

# Validate JSON syntax
python -m json.tool ~/.config/claude-desktop/claude_desktop_config.json

# Restart Claude Desktop after config changes
```

## Performance Issues

### High Memory Usage

**Problem:** MCpeg consuming excessive memory
```
[WARN] memory usage: 1.2GB (threshold: 1GB)
```

**Solution:**
```bash
# Check memory usage
curl http://localhost:8080/metrics | grep memory

# Reduce plugin cache sizes
vim config/production.yaml
# Set cache_size: 100 for plugins

# Enable garbage collection tuning
export GOGC=50
```

### Slow Response Times

**Problem:** API responses are slow
```
[WARN] request duration: 5.2s (threshold: 1s)
```

**Solution:**
```bash
# Check system load
top
htop

# Increase worker pool size
vim config/production.yaml
# Set worker_pool_size: 20

# Enable request timeout
export MCPEG_SERVER_REQUEST_TIMEOUT=30s
```

### Connection Timeouts

**Problem:** Requests timing out
```
[ERROR] request timeout: context deadline exceeded
```

**Solution:**
```bash
# Increase timeouts in config
vim config/production.yaml
# Set read_timeout: 30s
# Set write_timeout: 30s

# Or use environment variables
export MCPEG_SERVER_READ_TIMEOUT=30s
export MCPEG_SERVER_WRITE_TIMEOUT=30s
```

## Logging and Debugging

### Enable Debug Logging

```bash
# Start with debug logging
mcpeg gateway --dev --log-level debug

# Or set environment variable
export MCPEG_LOGGING_LEVEL=debug
mcpeg gateway
```

### Structured Logging

```bash
# View logs in JSON format
mcpeg gateway --dev | jq '.'

# Filter specific log levels
mcpeg gateway --dev | jq 'select(.level == "ERROR")'

# Filter by component
mcpeg gateway --dev | jq 'select(.component == "plugin")'
```

### Health Check Debugging

```bash
# Check overall health
curl http://localhost:8080/health

# Check specific components
curl http://localhost:8080/health | jq '.components'

# Check plugin health
curl http://localhost:8080/health | jq '.plugins'
```

## Network Issues

### Firewall Configuration

**Problem:** External connections blocked
```
Connection refused from remote host
```

**Solution:**
```bash
# Allow port 8080
sudo ufw allow 8080
sudo iptables -A INPUT -p tcp --dport 8080 -j ACCEPT

# Or bind to specific interface
mcpeg gateway --address 192.168.1.100:8080
```

### DNS Resolution Issues

**Problem:** Cannot resolve external services
```
[ERROR] failed to resolve hostname: no such host
```

**Solution:**
```bash
# Check DNS configuration
cat /etc/resolv.conf
nslookup google.com

# Use IP addresses instead of hostnames
# Or configure custom DNS
export MCPEG_DNS_SERVERS="8.8.8.8,8.8.4.4"
```

## Database Issues

### Memory Plugin Data Corruption

**Problem:** Memory plugin data file corrupted
```
[ERROR] memory plugin: failed to parse data file
```

**Solution:**
```bash
# Backup and reset
cp data/memory_storage.json data/memory_storage.json.backup
echo '{}' > data/memory_storage.json

# Or use different data file
export MCPEG_PLUGINS_MEMORY_DATA_FILE="/tmp/memory_clean.json"
```

## System Resource Issues

### Disk Space

**Problem:** Insufficient disk space
```
[ERROR] failed to write log file: no space left on device
```

**Solution:**
```bash
# Check disk usage
df -h

# Clean up log files
sudo logrotate -f /etc/logrotate.d/mcpeg

# Or use different log location
export MCPEG_LOGGING_OUTPUT="/tmp/mcpeg.log"
```

### File Descriptor Limits

**Problem:** Too many open files
```
[ERROR] failed to accept connection: too many open files
```

**Solution:**
```bash
# Check current limits
ulimit -n

# Increase limit
ulimit -n 4096

# Or set in systemd service
echo "LimitNOFILE=4096" >> /etc/systemd/system/mcpeg.service
```

## Diagnostic Commands

### System Information

```bash
# Check system resources
mcpeg gateway --system-info

# Check configuration
mcpeg validate --config config/production.yaml --verbose

# Check plugin status
mcpeg gateway --list-plugins --verbose
```

### Network Diagnostics

```bash
# Test connectivity
curl -v http://localhost:8080/health

# Check listening ports
netstat -tlnp | grep mcpeg
ss -tlnp | grep mcpeg

# Test MCP protocol
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"ping","params":{}}'
```

### Performance Diagnostics

```bash
# Check metrics
curl http://localhost:8080/metrics

# Monitor in real-time
watch -n 1 'curl -s http://localhost:8080/health | jq'

# Profile performance
go tool pprof http://localhost:8080/debug/pprof/profile
```

## Common Error Codes

### HTTP Error Codes

- **400 Bad Request** - Invalid JSON or malformed request
- **401 Unauthorized** - Authentication required or invalid token
- **403 Forbidden** - Access denied by RBAC policy
- **404 Not Found** - Endpoint or resource not found
- **429 Too Many Requests** - Rate limiting active
- **500 Internal Server Error** - Server error, check logs
- **503 Service Unavailable** - Server overloaded or maintenance

### JSON-RPC Error Codes

- **-32700** - Parse error (invalid JSON)
- **-32600** - Invalid request (not JSON-RPC 2.0)
- **-32601** - Method not found
- **-32602** - Invalid params
- **-32603** - Internal error
- **-32000 to -32099** - Custom application errors

## Getting Help

### Log Analysis

```bash
# View recent errors
journalctl -u mcpeg --since "1 hour ago" --grep ERROR

# Follow logs in real-time
journalctl -u mcpeg -f

# Search for specific errors
grep -i "failed to" /var/log/mcpeg/gateway.log
```

### Community Support

- **GitHub Issues:** [Report bugs and feature requests](https://github.com/osakka/mcpeg/issues)
- **Discussions:** [Community support](https://github.com/osakka/mcpeg/discussions)
- **Documentation:** [Complete guides](../README.md)

### Debug Information to Include

When reporting issues, include:

1. **Version information:**
   ```bash
   mcpeg --version
   ```

2. **Configuration (sanitized):**
   ```bash
   mcpeg validate --config config/production.yaml
   ```

3. **System information:**
   ```bash
   uname -a
   go version
   ```

4. **Error logs:**
   ```bash
   journalctl -u mcpeg --since "1 hour ago"
   ```

5. **Network information:**
   ```bash
   netstat -tlnp | grep mcpeg
   ```

## Prevention

### Regular Maintenance

```bash
# Update dependencies
go mod tidy
go get -u all

# Clean build artifacts
make clean

# Rotate logs
sudo logrotate -f /etc/logrotate.d/mcpeg

# Check disk space
df -h

# Monitor memory usage
free -h
```

### Health Monitoring

```bash
# Set up monitoring alerts
curl http://localhost:8080/health | jq '.status == "healthy"'

# Monitor plugin health
curl http://localhost:8080/health | jq '.plugins | to_entries[] | select(.value != "healthy")'

# Check memory usage
curl http://localhost:8080/metrics | grep memory_usage
```

### Backup Strategies

```bash
# Backup configuration
cp -r config/ backup/config-$(date +%Y%m%d)/

# Backup plugin data
cp -r data/ backup/data-$(date +%Y%m%d)/

# Backup logs
cp /var/log/mcpeg/gateway.log backup/logs-$(date +%Y%m%d).log
```