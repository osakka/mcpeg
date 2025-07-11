#!/bin/bash
# MCpeg Gateway Service Installation Script

set -e

# Configuration
MCPEG_HOME="${MCPEG_HOME:-/opt/mcpeg}"
MCPEG_USER="${MCPEG_USER:-mcpeg}"
MCPEG_GROUP="${MCPEG_GROUP:-mcpeg}"
SERVICE_NAME="${SERVICE_NAME:-mcpeg}"
SERVICE_TYPE="${SERVICE_TYPE:-production}"  # production or development

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING:${NC} $1"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR:${NC} $1"
}

info() {
    echo -e "${BLUE}$1${NC}"
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        error "This script must be run as root"
        exit 1
    fi
}

# Create mcpeg user and group
create_user() {
    log "Creating mcpeg user and group..."
    
    # Create group if it doesn't exist
    if ! getent group "$MCPEG_GROUP" > /dev/null 2>&1; then
        groupadd --system "$MCPEG_GROUP"
        log "Created group: $MCPEG_GROUP"
    else
        log "Group already exists: $MCPEG_GROUP"
    fi
    
    # Create user if it doesn't exist
    if ! id "$MCPEG_USER" > /dev/null 2>&1; then
        useradd --system --gid "$MCPEG_GROUP" --home-dir "$MCPEG_HOME" \
                --shell /bin/false --comment "MCpeg Gateway" "$MCPEG_USER"
        log "Created user: $MCPEG_USER"
    else
        log "User already exists: $MCPEG_USER"
    fi
}

# Create necessary directories
create_directories() {
    log "Creating directories..."
    
    # Configuration directory
    mkdir -p /etc/mcpeg
    chown root:root /etc/mcpeg
    chmod 755 /etc/mcpeg
    
    # Log directory
    mkdir -p /var/log/mcpeg
    chown "$MCPEG_USER:$MCPEG_GROUP" /var/log/mcpeg
    chmod 755 /var/log/mcpeg
    
    # Run directory
    mkdir -p /var/run/mcpeg
    chown "$MCPEG_USER:$MCPEG_GROUP" /var/run/mcpeg
    chmod 755 /var/run/mcpeg
    
    # Data directory
    mkdir -p /opt/mcpeg/data
    chown "$MCPEG_USER:$MCPEG_GROUP" /opt/mcpeg/data
    chmod 755 /opt/mcpeg/data
    
    log "Directories created successfully"
}

# Install systemd service
install_service() {
    log "Installing systemd service..."
    
    local service_file
    if [[ "$SERVICE_TYPE" == "development" ]]; then
        service_file="$MCPEG_HOME/scripts/mcpeg-dev.service"
        SERVICE_NAME="mcpeg-dev"
    else
        service_file="$MCPEG_HOME/scripts/mcpeg.service"
        SERVICE_NAME="mcpeg"
    fi
    
    if [[ ! -f "$service_file" ]]; then
        error "Service file not found: $service_file"
        exit 1
    fi
    
    # Copy service file
    cp "$service_file" "/etc/systemd/system/$SERVICE_NAME.service"
    chown root:root "/etc/systemd/system/$SERVICE_NAME.service"
    chmod 644 "/etc/systemd/system/$SERVICE_NAME.service"
    
    log "Service file installed: /etc/systemd/system/$SERVICE_NAME.service"
    
    # Reload systemd
    systemctl daemon-reload
    log "Systemd daemon reloaded"
    
    # Enable service
    systemctl enable "$SERVICE_NAME"
    log "Service enabled: $SERVICE_NAME"
}

# Create default configuration
create_default_config() {
    log "Creating default configuration..."
    
    local config_file="/etc/mcpeg/gateway.yaml"
    if [[ "$SERVICE_TYPE" == "development" ]]; then
        config_file="/etc/mcpeg/gateway-dev.yaml"
    fi
    
    if [[ ! -f "$config_file" ]]; then
        cat > "$config_file" <<EOF
# MCpeg Gateway Configuration
# This is a default configuration - modify as needed

server:
  address: "0.0.0.0"
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 60s
  shutdown_timeout: 30s
  
  tls:
    enabled: false
    
  cors:
    enabled: true
    allow_origins: ["*"]
    allow_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    allow_headers: ["Content-Type", "Authorization", "X-Client-ID", "X-Session-ID"]
    
  middleware:
    compression:
      enabled: true
      level: 6
    rate_limit:
      enabled: false
      rps: 1000
      burst: 2000
    request_logging:
      enabled: true
      include_body: false
      exclude_paths: ["/health", "/metrics"]
      
  health_check:
    enabled: true
    endpoint: "/health"
    detailed: false

logging:
  level: "info"
  format: "json"
  output:
    console:
      enabled: true
      colorized: false
    file:
      enabled: true
      path: "/var/log/mcpeg/mcpeg.log"
      max_size: 100
      max_backups: 5
      max_age: 7
      compress: true

metrics:
  enabled: true
  address: "0.0.0.0"
  port: 9090
  prometheus:
    enabled: true
    endpoint: "/metrics"
    namespace: "mcpeg"
    subsystem: "gateway"

registry:
  discovery:
    static:
      enabled: true
      services: []
  load_balancer:
    strategy: "round_robin"
    health_aware: true
    circuit_breaker:
      enabled: true
      failure_threshold: 5
      recovery_timeout: 30s
  health_checks:
    enabled: true
    interval: 30s
    timeout: 5s

security:
  api_key:
    enabled: false
  jwt:
    enabled: false
  validation:
    enabled: true
    strict_mode: false
    validate_body: true

development:
  enabled: false
  admin_endpoints:
    enabled: false
    prefix: "/admin"
EOF
        
        # Set development-specific settings
        if [[ "$SERVICE_TYPE" == "development" ]]; then
            cat >> "$config_file" <<EOF

# Development-specific overrides
development:
  enabled: true
  debug_mode: true
  admin_endpoints:
    enabled: true
    prefix: "/admin"
    config_reload: true
    service_discovery: true
    health_checks: true

logging:
  level: "debug"
  output:
    console:
      colorized: true
EOF
        fi
        
        chown root:root "$config_file"
        chmod 644 "$config_file"
        log "Default configuration created: $config_file"
    else
        log "Configuration file already exists: $config_file"
    fi
}

# Set up logrotate
setup_logrotate() {
    log "Setting up logrotate..."
    
    cat > /etc/logrotate.d/mcpeg <<EOF
/var/log/mcpeg/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    sharedscripts
    postrotate
        systemctl reload mcpeg 2>/dev/null || true
    endscript
}
EOF
    
    chmod 644 /etc/logrotate.d/mcpeg
    log "Logrotate configuration created"
}

# Create symbolic links for management scripts
create_symlinks() {
    log "Creating symbolic links..."
    
    # Create links in /usr/local/bin
    ln -sf "$MCPEG_HOME/scripts/mcpeg-start.sh" /usr/local/bin/mcpeg-start
    ln -sf "$MCPEG_HOME/scripts/mcpeg-stop.sh" /usr/local/bin/mcpeg-stop
    ln -sf "$MCPEG_HOME/scripts/mcpeg-restart.sh" /usr/local/bin/mcpeg-restart
    ln -sf "$MCPEG_HOME/scripts/mcpeg-status.sh" /usr/local/bin/mcpeg-status
    
    log "Symbolic links created in /usr/local/bin"
}

# Show post-installation information
show_post_install_info() {
    echo ""
    info "=== MCpeg Gateway Installation Complete ==="
    echo ""
    echo "Service: $SERVICE_NAME"
    echo "User: $MCPEG_USER"
    echo "Group: $MCPEG_GROUP"
    echo "Home: $MCPEG_HOME"
    echo "Config: /etc/mcpeg/"
    echo "Logs: /var/log/mcpeg/"
    echo "PID: /var/run/mcpeg/"
    echo ""
    info "=== Available Commands ==="
    echo "systemctl start $SERVICE_NAME      # Start the service"
    echo "systemctl stop $SERVICE_NAME       # Stop the service"
    echo "systemctl restart $SERVICE_NAME    # Restart the service"
    echo "systemctl status $SERVICE_NAME     # Check service status"
    echo "systemctl enable $SERVICE_NAME     # Enable auto-start"
    echo "systemctl disable $SERVICE_NAME    # Disable auto-start"
    echo ""
    echo "mcpeg-start                        # Start using script"
    echo "mcpeg-stop                         # Stop using script"
    echo "mcpeg-restart                      # Restart using script"
    echo "mcpeg-status                       # Show detailed status"
    echo ""
    info "=== Next Steps ==="
    echo "1. Edit configuration: /etc/mcpeg/gateway.yaml"
    echo "2. Start the service: systemctl start $SERVICE_NAME"
    echo "3. Check status: systemctl status $SERVICE_NAME"
    echo "4. View logs: journalctl -u $SERVICE_NAME -f"
    echo ""
}

# Main execution
main() {
    log "MCpeg Gateway Service Installation"
    
    # Check root privileges
    check_root
    
    # Create user and group
    create_user
    
    # Create directories
    create_directories
    
    # Install systemd service
    install_service
    
    # Create default configuration
    create_default_config
    
    # Set up logrotate
    setup_logrotate
    
    # Create symbolic links
    create_symlinks
    
    # Show post-installation info
    show_post_install_info
}

# Show usage
usage() {
    echo "Usage: $0 [options]"
    echo "Options:"
    echo "  -t, --type TYPE      Service type: production or development (default: production)"
    echo "  -u, --user USER      User to run as (default: mcpeg)"
    echo "  -g, --group GROUP    Group to run as (default: mcpeg)"
    echo "  -h, --help           Show this help"
    echo ""
    echo "Environment variables:"
    echo "  MCPEG_HOME          MCpeg installation directory"
    echo "  MCPEG_USER          User to run as"
    echo "  MCPEG_GROUP         Group to run as"
    echo "  SERVICE_NAME        Service name"
    echo "  SERVICE_TYPE        Service type (production or development)"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -t|--type)
            SERVICE_TYPE="$2"
            shift 2
            ;;
        -u|--user)
            MCPEG_USER="$2"
            shift 2
            ;;
        -g|--group)
            MCPEG_GROUP="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Validate service type
if [[ "$SERVICE_TYPE" != "production" && "$SERVICE_TYPE" != "development" ]]; then
    error "Invalid service type: $SERVICE_TYPE (must be 'production' or 'development')"
    exit 1
fi

# Run main function
main "$@"