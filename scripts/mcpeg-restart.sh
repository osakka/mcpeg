#!/bin/bash
# MCpeg Gateway Restart Script

set -e

# Configuration
MCPEG_HOME="${MCPEG_HOME:-/opt/mcpeg}"
MCPEG_CONFIG="${MCPEG_CONFIG:-/etc/mcpeg/gateway.yaml}"
MCPEG_PID_FILE="${MCPEG_PID_FILE:-/var/run/mcpeg/mcpeg.pid}"
MCPEG_LOG_FILE="${MCPEG_LOG_FILE:-/var/log/mcpeg/mcpeg.log}"
MCPEG_USER="${MCPEG_USER:-mcpeg}"

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

# Restart using built-in restart command
restart_builtin() {
    log "Restarting MCpeg Gateway using built-in restart command..."
    
    if [[ "$EUID" -eq 0 ]]; then
        # Running as root, switch to mcpeg user
        sudo -u "$MCPEG_USER" "$MCPEG_HOME/build/mcpeg" \
            --restart \
            --config "$MCPEG_CONFIG" \
            --pid-file "$MCPEG_PID_FILE" \
            --log-file "$MCPEG_LOG_FILE"
    else
        # Already running as non-root user
        "$MCPEG_HOME/build/mcpeg" \
            --restart \
            --config "$MCPEG_CONFIG" \
            --pid-file "$MCPEG_PID_FILE" \
            --log-file "$MCPEG_LOG_FILE"
    fi
}

# Restart using separate stop and start scripts
restart_scripts() {
    log "Restarting MCpeg Gateway using separate stop and start..."
    
    # Stop first
    if [[ -x "$MCPEG_HOME/scripts/mcpeg-stop.sh" ]]; then
        "$MCPEG_HOME/scripts/mcpeg-stop.sh" --pid-file "$MCPEG_PID_FILE" --user "$MCPEG_USER"
    else
        error "Stop script not found: $MCPEG_HOME/scripts/mcpeg-stop.sh"
        return 1
    fi
    
    # Wait a moment
    sleep 2
    
    # Start again
    if [[ -x "$MCPEG_HOME/scripts/mcpeg-start.sh" ]]; then
        "$MCPEG_HOME/scripts/mcpeg-start.sh" \
            --config "$MCPEG_CONFIG" \
            --pid-file "$MCPEG_PID_FILE" \
            --log-file "$MCPEG_LOG_FILE" \
            --user "$MCPEG_USER"
    else
        error "Start script not found: $MCPEG_HOME/scripts/mcpeg-start.sh"
        return 1
    fi
}

# Wait for service to be ready
wait_for_service() {
    local max_attempts=30
    local attempt=0
    
    log "Waiting for MCpeg Gateway to be ready..."
    
    while [[ $attempt -lt $max_attempts ]]; do
        if [[ -f "$MCPEG_PID_FILE" ]]; then
            local pid=$(cat "$MCPEG_PID_FILE")
            if kill -0 "$pid" 2>/dev/null; then
                # Try to check if service is responding
                # This is a simplified check - in a real implementation,
                # you might want to check the health endpoint
                log "MCpeg Gateway is ready (PID: $pid)"
                return 0
            fi
        fi
        
        sleep 1
        ((attempt++))
    done
    
    error "MCpeg Gateway did not become ready within ${max_attempts} seconds"
    return 1
}

# Main execution
main() {
    local use_builtin="${1:-true}"
    
    log "MCpeg Gateway Restart Script"
    
    # Check if gateway binary exists
    if [[ ! -x "$MCPEG_HOME/build/mcpeg" ]]; then
        error "MCpeg gateway binary not found or not executable: $MCPEG_HOME/build/mcpeg"
        exit 1
    fi
    
    # Try built-in restart first
    if [[ "$use_builtin" == "true" ]]; then
        if restart_builtin; then
            log "MCpeg Gateway restarted successfully using built-in command"
            exit 0
        else
            warn "Built-in restart command failed, falling back to script method"
        fi
    fi
    
    # Fall back to script method
    if restart_scripts; then
        if wait_for_service; then
            log "MCpeg Gateway restarted successfully"
        else
            error "MCpeg Gateway restart completed but service is not ready"
            exit 1
        fi
    else
        error "Failed to restart MCpeg Gateway"
        exit 1
    fi
}

# Show usage
usage() {
    echo "Usage: $0 [options]"
    echo "Options:"
    echo "  -s, --scripts        Use separate stop/start scripts instead of built-in restart"
    echo "  -c, --config FILE    Configuration file (default: /etc/mcpeg/gateway.yaml)"
    echo "  -p, --pid-file FILE  PID file path (default: /var/run/mcpeg/mcpeg.pid)"
    echo "  -l, --log-file FILE  Log file path (default: /var/log/mcpeg/mcpeg.log)"
    echo "  -u, --user USER      User to run as (default: mcpeg)"
    echo "  -h, --help           Show this help"
    echo ""
    echo "Environment variables:"
    echo "  MCPEG_HOME          MCpeg installation directory"
    echo "  MCPEG_CONFIG        Configuration file path"
    echo "  MCPEG_PID_FILE      PID file path"
    echo "  MCPEG_LOG_FILE      Log file path"
    echo "  MCPEG_USER          User to run as"
}

# Parse command line arguments
USE_BUILTIN=true

while [[ $# -gt 0 ]]; do
    case $1 in
        -s|--scripts)
            USE_BUILTIN=false
            shift
            ;;
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
main "$USE_BUILTIN"