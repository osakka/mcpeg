#!/bin/bash
# MCpeg Gateway Start Script

set -e

# Configuration
MCPEG_HOME="${MCPEG_HOME:-/opt/mcpeg}"
MCPEG_CONFIG="${MCPEG_CONFIG:-/etc/mcpeg/gateway.yaml}"
MCPEG_PID_FILE="${MCPEG_PID_FILE:-$MCPEG_HOME/build/runtime/mcpeg.pid}"
MCPEG_LOG_FILE="${MCPEG_LOG_FILE:-$MCPEG_HOME/build/logs/mcpeg.log}"
MCPEG_USER="${MCPEG_USER:-mcpeg}"
MCPEG_GROUP="${MCPEG_GROUP:-mcpeg}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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

# Check if MCpeg is already running
check_running() {
    if [[ -f "$MCPEG_PID_FILE" ]]; then
        local pid=$(cat "$MCPEG_PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            return 0  # Running
        else
            warn "Stale PID file found, removing..."
            rm -f "$MCPEG_PID_FILE"
        fi
    fi
    return 1  # Not running
}

# Create necessary directories
setup_directories() {
    log "Setting up directories..."
    
    # Create run directory
    local run_dir=$(dirname "$MCPEG_PID_FILE")
    if [[ ! -d "$run_dir" ]]; then
        sudo mkdir -p "$run_dir"
        sudo chown "$MCPEG_USER:$MCPEG_GROUP" "$run_dir"
        sudo chmod 755 "$run_dir"
    fi
    
    # Create log directory
    local log_dir=$(dirname "$MCPEG_LOG_FILE")
    if [[ ! -d "$log_dir" ]]; then
        sudo mkdir -p "$log_dir"
        sudo chown "$MCPEG_USER:$MCPEG_GROUP" "$log_dir"
        sudo chmod 755 "$log_dir"
    fi
}

# Validate configuration
validate_config() {
    log "Validating configuration..."
    
    if [[ ! -f "$MCPEG_CONFIG" ]]; then
        error "Configuration file not found: $MCPEG_CONFIG"
        return 1
    fi
    
    if [[ ! -x "$MCPEG_HOME/build/mcpeg" ]]; then
        error "MCpeg gateway binary not found or not executable: $MCPEG_HOME/build/mcpeg"
        return 1
    fi
    
    log "Configuration validated successfully"
}

# Start MCpeg daemon
start_daemon() {
    log "Starting MCpeg Gateway..."
    
    # Start as daemon
    if [[ "$EUID" -eq 0 ]]; then
        # Running as root, switch to mcpeg user
        sudo -u "$MCPEG_USER" "$MCPEG_HOME/build/mcpeg" gateway \
            --daemon \
            --config "$MCPEG_CONFIG" \
            --pid-file "$MCPEG_PID_FILE" \
            --log-file "$MCPEG_LOG_FILE"
    else
        # Already running as non-root user
        "$MCPEG_HOME/build/mcpeg" gateway \
            --daemon \
            --config "$MCPEG_CONFIG" \
            --pid-file "$MCPEG_PID_FILE" \
            --log-file "$MCPEG_LOG_FILE"
    fi
    
    # Wait for PID file to be created
    local attempts=0
    while [[ $attempts -lt 30 ]]; do
        if [[ -f "$MCPEG_PID_FILE" ]]; then
            local pid=$(cat "$MCPEG_PID_FILE")
            if kill -0 "$pid" 2>/dev/null; then
                log "MCpeg Gateway started successfully (PID: $pid)"
                return 0
            fi
        fi
        sleep 1
        ((attempts++))
    done
    
    error "Failed to start MCpeg Gateway"
    return 1
}

# Main execution
main() {
    log "MCpeg Gateway Start Script"
    
    # Check if already running
    if check_running; then
        local pid=$(cat "$MCPEG_PID_FILE")
        warn "MCpeg Gateway is already running (PID: $pid)"
        exit 0
    fi
    
    # Setup directories
    setup_directories
    
    # Validate configuration
    if ! validate_config; then
        exit 1
    fi
    
    # Start daemon
    if ! start_daemon; then
        exit 1
    fi
    
    log "MCpeg Gateway startup complete"
}

# Show usage
usage() {
    echo "Usage: $0 [options]"
    echo "Options:"
    echo "  -c, --config FILE    Configuration file (default: /etc/mcpeg/gateway.yaml)"
    echo "  -p, --pid-file FILE  PID file path (default: /var/run/mcpeg/mcpeg.pid)"
    echo "  -l, --log-file FILE  Log file path (default: /var/log/mcpeg/mcpeg.log)"
    echo "  -u, --user USER      User to run as (default: mcpeg)"
    echo "  -g, --group GROUP    Group to run as (default: mcpeg)"
    echo "  -h, --help           Show this help"
    echo ""
    echo "Environment variables:"
    echo "  MCPEG_HOME          MCpeg installation directory"
    echo "  MCPEG_CONFIG        Configuration file path"
    echo "  MCPEG_PID_FILE      PID file path"
    echo "  MCPEG_LOG_FILE      Log file path"
    echo "  MCPEG_USER          User to run as"
    echo "  MCPEG_GROUP         Group to run as"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -c|--config)
            MCPEG_CONFIG="$2"
            shift 2
            ;;
        -p|--pid-file)
            MCPEG_PID_FILE="$2"
            shift 2
            ;;
        -l|--log-file)
            MCPEG_LOG_FILE="$2"
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

# Run main function
main "$@"