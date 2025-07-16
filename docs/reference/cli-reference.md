# CLI Reference

Complete reference for the MCpeg command-line interface.

## Overview

MCpeg provides a unified binary with multiple subcommands for different operations:

```bash
mcpeg [global-options] <command> [command-options] [arguments]
```

## Global Options

### Common Flags

```bash
--help, -h          Show help information
--version, -v       Show version information
--config PATH       Configuration file path (default: config/development.yaml)
--log-level LEVEL   Log level: debug, info, warn, error (default: info)
--quiet, -q         Suppress output except errors
--verbose           Enable verbose output
```

### Configuration Override

```bash
--config-override KEY=VALUE    Override configuration values
```

**Examples:**
```bash
mcpeg gateway --config-override server.address=0.0.0.0:9000
mcpeg gateway --config-override plugins.memory.enabled=false
```

## Commands

### Gateway Command

Start the MCpeg gateway server.

```bash
mcpeg gateway [options]
```

#### Options

```bash
--dev                    Enable development mode
--daemon                 Run as daemon process
--address HOST:PORT      Server address (default: 0.0.0.0:8080)
--tls                    Enable TLS
--cert-path PATH         TLS certificate path
--key-path PATH          TLS private key path
--pid-file PATH          PID file path
--log-file PATH          Log file path
--dry-run               Validate configuration without starting
--list-plugins          List available plugins and exit
--enable-plugin NAME     Enable specific plugin
--disable-plugin NAME    Disable specific plugin
--test-plugin NAME       Test specific plugin and exit
--metrics               Enable metrics endpoint
--health                Enable health check endpoint
--debug                 Enable debug endpoints
```

#### Examples

```bash
# Start in development mode
mcpeg gateway --dev

# Start with custom configuration
mcpeg gateway --config config/production.yaml

# Start with TLS enabled
mcpeg gateway --tls --cert-path cert.pem --key-path key.pem

# Start as daemon
mcpeg gateway --daemon --pid-file /var/run/mcpeg.pid

# Start with plugin management
mcpeg gateway --disable-plugin git --enable-plugin memory

# Test configuration without starting
mcpeg gateway --config config/production.yaml --dry-run

# List available plugins
mcpeg gateway --list-plugins
```

### Validate Command

Validate configuration files and settings.

```bash
mcpeg validate [options]
```

#### Options

```bash
--config PATH           Configuration file to validate
--schema PATH           JSON schema file for validation
--strict               Enable strict validation mode
--format FORMAT        Output format: text, json, yaml (default: text)
--check-plugins        Validate plugin configurations
--check-permissions    Check file permissions
--check-network        Check network connectivity
--check-dependencies   Check external dependencies
```

#### Examples

```bash
# Validate default configuration
mcpeg validate

# Validate specific configuration
mcpeg validate --config config/production.yaml

# Strict validation with plugin checks
mcpeg validate --strict --check-plugins

# Validate with JSON output
mcpeg validate --format json

# Comprehensive validation
mcpeg validate --check-plugins --check-permissions --check-network
```

### Codegen Command

Generate code from OpenAPI specifications.

```bash
mcpeg codegen [options]
```

#### Options

```bash
--spec PATH             OpenAPI specification file
--output DIR            Output directory (default: generated/)
--language LANG         Target language: go, python, javascript, typescript
--package NAME          Package name for generated code
--template DIR          Custom template directory
--client               Generate client code
--server               Generate server code
--models               Generate model code only
--overwrite            Overwrite existing files
--dry-run              Show what would be generated
```

#### Examples

```bash
# Generate Go client code
mcpeg codegen --spec api/openapi/mcp-gateway.yaml --language go --client

# Generate Python server code
mcpeg codegen --spec api/openapi/mcp-gateway.yaml --language python --server

# Generate models only
mcpeg codegen --spec api/openapi/mcp-gateway.yaml --models --language typescript

# Show what would be generated
mcpeg codegen --spec api/openapi/mcp-gateway.yaml --dry-run
```

### Plugin Command

Manage MCpeg plugins.

```bash
mcpeg plugin <subcommand> [options]
```

#### Subcommands

##### List Plugins
```bash
mcpeg plugin list [options]
```

**Options:**
```bash
--format FORMAT        Output format: text, json, yaml (default: text)
--enabled             Show only enabled plugins
--disabled            Show only disabled plugins
--detailed            Show detailed plugin information
```

##### Install Plugin
```bash
mcpeg plugin install <plugin-name> [options]
```

**Options:**
```bash
--source URL          Plugin source URL or path
--version VERSION     Specific version to install
--force              Force installation
--dry-run            Show what would be installed
```

##### Remove Plugin
```bash
mcpeg plugin remove <plugin-name> [options]
```

**Options:**
```bash
--force              Force removal
--keep-config        Keep plugin configuration
```

##### Enable Plugin
```bash
mcpeg plugin enable <plugin-name>
```

##### Disable Plugin
```bash
mcpeg plugin disable <plugin-name>
```

##### Plugin Info
```bash
mcpeg plugin info <plugin-name> [options]
```

**Options:**
```bash
--format FORMAT      Output format: text, json, yaml (default: text)
```

#### Examples

```bash
# List all plugins
mcpeg plugin list

# List enabled plugins in JSON format
mcpeg plugin list --enabled --format json

# Show detailed plugin information
mcpeg plugin info memory --format json

# Install plugin from URL
mcpeg plugin install custom-plugin --source https://example.com/plugin.tar.gz

# Enable/disable plugins
mcpeg plugin enable memory
mcpeg plugin disable git
```

### Config Command

Configuration management utilities.

```bash
mcpeg config <subcommand> [options]
```

#### Subcommands

##### Show Config
```bash
mcpeg config show [options]
```

**Options:**
```bash
--config PATH         Configuration file path
--format FORMAT       Output format: yaml, json, text (default: yaml)
--key KEY            Show specific configuration key
--effective          Show effective configuration (after overrides)
```

##### Set Config
```bash
mcpeg config set <key> <value> [options]
```

**Options:**
```bash
--config PATH         Configuration file path
--create             Create configuration file if not exists
```

##### Get Config
```bash
mcpeg config get <key> [options]
```

**Options:**
```bash
--config PATH         Configuration file path
--format FORMAT       Output format: text, json, yaml (default: text)
```

##### Merge Config
```bash
mcpeg config merge <source-config> [options]
```

**Options:**
```bash
--config PATH         Target configuration file
--output PATH         Output file (default: update in place)
--strategy STRATEGY   Merge strategy: merge, replace, append (default: merge)
```

#### Examples

```bash
# Show current configuration
mcpeg config show

# Show effective configuration
mcpeg config show --effective

# Show specific key
mcpeg config get server.address

# Set configuration value
mcpeg config set server.address 0.0.0.0:9000

# Merge configurations
mcpeg config merge config/custom.yaml --config config/production.yaml
```

### Health Command

Health check utilities.

```bash
mcpeg health [options]
```

#### Options

```bash
--endpoint URL          Health check endpoint (default: http://localhost:8080/health)
--timeout DURATION      Request timeout (default: 5s)
--format FORMAT         Output format: text, json, yaml (default: text)
--detailed             Show detailed health information
--watch               Watch health status continuously
--interval DURATION    Watch interval (default: 30s)
--quiet               Only show health status
```

#### Examples

```bash
# Check health status
mcpeg health

# Check health with detailed output
mcpeg health --detailed --format json

# Watch health status
mcpeg health --watch --interval 10s

# Check remote server health
mcpeg health --endpoint http://remote-server:8080/health
```

### Version Command

Show version information.

```bash
mcpeg version [options]
```

#### Options

```bash
--format FORMAT        Output format: text, json, yaml (default: text)
--short               Show short version only
--build-info          Show build information
--dependencies        Show dependency versions
```

#### Examples

```bash
# Show version
mcpeg version

# Show detailed version information
mcpeg version --build-info --dependencies

# Show version in JSON format
mcpeg version --format json
```

### Completion Command

Generate shell completion scripts.

```bash
mcpeg completion <shell>
```

#### Supported Shells

- `bash`
- `zsh`
- `fish`
- `powershell`

#### Examples

```bash
# Generate bash completion
mcpeg completion bash > /etc/bash_completion.d/mcpeg

# Generate zsh completion
mcpeg completion zsh > ~/.zsh/completions/_mcpeg

# Generate fish completion
mcpeg completion fish > ~/.config/fish/completions/mcpeg.fish
```

## Environment Variables

MCpeg respects these environment variables:

### Configuration

```bash
MCPEG_CONFIG_FILE          Configuration file path
MCPEG_LOG_LEVEL            Log level (debug, info, warn, error)
MCPEG_DEVELOPMENT_MODE     Enable development mode (true/false)
```

### Server Configuration

```bash
MCPEG_SERVER_ADDRESS       Server address (default: 0.0.0.0:8080)
MCPEG_SERVER_TLS_ENABLED   Enable TLS (true/false)
MCPEG_SERVER_TLS_CERT_PATH TLS certificate path
MCPEG_SERVER_TLS_KEY_PATH  TLS private key path
```

### Plugin Configuration

```bash
MCPEG_PLUGINS_MEMORY_ENABLED     Enable memory plugin (true/false)
MCPEG_PLUGINS_MEMORY_DATA_FILE   Memory plugin data file path
MCPEG_PLUGINS_GIT_ENABLED        Enable git plugin (true/false)
MCPEG_PLUGINS_GIT_WORKING_DIR    Git plugin working directory
MCPEG_PLUGINS_EDITOR_ENABLED     Enable editor plugin (true/false)
MCPEG_PLUGINS_EDITOR_MAX_FILE_SIZE Editor plugin max file size
```

### Authentication

```bash
MCPEG_AUTH_JWT_ENABLED     Enable JWT authentication (true/false)
MCPEG_AUTH_JWT_SECRET_KEY  JWT secret key
MCPEG_AUTH_JWT_ISSUER      JWT issuer
MCPEG_AUTH_JWT_EXPIRATION  JWT expiration duration
```

## Exit Codes

MCpeg uses standard exit codes:

- `0` - Success
- `1` - General error
- `2` - Configuration error
- `3` - Plugin error
- `4` - Network error
- `5` - Authentication error
- `6` - Permission error
- `7` - Validation error
- `8` - Resource error (disk space, memory, etc.)
- `9` - Service unavailable

## Configuration File Locations

MCpeg searches for configuration files in this order:

1. `--config` flag value
2. `MCPEG_CONFIG_FILE` environment variable
3. `./config/development.yaml` (current directory)
4. `~/.mcpeg/config.yaml` (user home directory)
5. `/etc/mcpeg/config.yaml` (system-wide)

## Logging

### Log Levels

- `debug` - Detailed debugging information
- `info` - General information messages
- `warn` - Warning messages
- `error` - Error messages only

### Log Formats

- `text` - Human-readable text format
- `json` - Structured JSON format (default)

### Log Outputs

- `stdout` - Standard output (default)
- `stderr` - Standard error
- `file` - Log file path

## Examples

### Development Workflow

```bash
# Start development server
mcpeg gateway --dev

# Validate configuration
mcpeg validate --config config/development.yaml

# Test plugin
mcpeg plugin info memory

# Check health
mcpeg health --detailed
```

### Production Deployment

```bash
# Validate production configuration
mcpeg validate --config config/production.yaml --strict

# Start production server
mcpeg gateway --config config/production.yaml --daemon

# Monitor health
mcpeg health --watch --interval 30s

# Check logs
tail -f /var/log/mcpeg/gateway.log
```

### Plugin Management

```bash
# List all plugins
mcpeg plugin list --detailed

# Enable specific plugins
mcpeg plugin enable memory
mcpeg plugin enable git

# Test plugin configuration
mcpeg plugin info editor --format json
```

### Configuration Management

```bash
# Show current configuration
mcpeg config show --effective

# Update configuration
mcpeg config set server.address 0.0.0.0:9000
mcpeg config set plugins.memory.enabled true

# Merge configurations
mcpeg config merge config/custom.yaml
```

## Troubleshooting

### Common Issues

1. **Command not found:**
   ```bash
   export PATH=$PATH:/usr/local/bin
   which mcpeg
   ```

2. **Configuration errors:**
   ```bash
   mcpeg validate --config config/production.yaml
   ```

3. **Permission errors:**
   ```bash
   sudo chown -R mcpeg:mcpeg /var/lib/mcpeg
   chmod 755 /var/lib/mcpeg
   ```

4. **Plugin issues:**
   ```bash
   mcpeg plugin list --detailed
   mcpeg plugin info problematic-plugin
   ```

### Debug Mode

Enable debug mode for detailed output:

```bash
mcpeg gateway --dev --log-level debug
```

### Dry Run

Test configuration without starting:

```bash
mcpeg gateway --config config/production.yaml --dry-run
```

## Shell Integration

### Bash Completion

```bash
# Install completion
mcpeg completion bash > /etc/bash_completion.d/mcpeg
source /etc/bash_completion.d/mcpeg

# Or for current session
eval "$(mcpeg completion bash)"
```

### Zsh Completion

```bash
# Install completion
mcpeg completion zsh > ~/.zsh/completions/_mcpeg
# Add to ~/.zshrc
fpath=(~/.zsh/completions $fpath)
autoload -U compinit && compinit
```

### Fish Completion

```bash
# Install completion
mcpeg completion fish > ~/.config/fish/completions/mcpeg.fish
```

## Advanced Usage

### Scripting

```bash
#!/bin/bash
set -e

# Start MCpeg with error handling
if ! mcpeg gateway --config config/production.yaml --daemon; then
    echo "Failed to start MCpeg gateway"
    exit 1
fi

# Wait for health check
while ! mcpeg health --quiet; do
    echo "Waiting for MCpeg to be healthy..."
    sleep 5
done

echo "MCpeg is now running and healthy"
```

### Monitoring

```bash
#!/bin/bash
# Monitor MCpeg health
while true; do
    if ! mcpeg health --quiet; then
        echo "MCpeg health check failed"
        # Restart or alert
    fi
    sleep 30
done
```

### Configuration Validation

```bash
#!/bin/bash
# Validate all configuration files
for config in config/*.yaml; do
    if mcpeg validate --config "$config"; then
        echo "✓ $config is valid"
    else
        echo "✗ $config is invalid"
        exit 1
    fi
done
```

## Best Practices

1. **Use configuration files** instead of command-line flags for production
2. **Validate configurations** before deployment
3. **Use environment variables** for sensitive information
4. **Enable logging** with appropriate levels
5. **Monitor health** continuously in production
6. **Use shell completion** for improved CLI experience
7. **Test plugins** before enabling in production
8. **Use daemon mode** for production deployments

## Next Steps

- **Installation:** [Installation Guide](../guides/installation.md)
- **Configuration:** [Configuration Guide](../guides/configuration.md)
- **API Usage:** [API Reference](api-reference.md)
- **Plugin Development:** [Plugin Development Guide](../guides/plugin-development.md)
- **Troubleshooting:** [Troubleshooting Guide](../guides/troubleshooting.md)