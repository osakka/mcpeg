#!/bin/bash
# MCpeg Gateway Stop Script

set -e

# Configuration
MCPEG_HOME="${MCPEG_HOME:-/opt/mcpeg}"
MCPEG_PID_FILE="${MCPEG_PID_FILE:-/var/run/mcpeg/mcpeg.pid}"
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

# Check if MCpeg is running
check_running() {
    if [[ -f "$MCPEG_PID_FILE" ]]; then
        local pid=$(cat "$MCPEG_PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            echo "$pid"
            return 0  # Running
        else
            warn "Stale PID file found, removing..."
            rm -f "$MCPEG_PID_FILE"
        fi
    fi
    return 1  # Not running
}

# Stop MCpeg daemon using built-in stop command
stop_daemon_builtin() {
    log "Stopping MCpeg Gateway using built-in stop command..."
    
    if [[ "$EUID" -eq 0 ]]; then
        # Running as root, switch to mcpeg user
        sudo -u "$MCPEG_USER" "$MCPEG_HOME/build/mcpeg" \
            --stop \
            --pid-file "$MCPEG_PID_FILE"
    else
        # Already running as non-root user
        "$MCPEG_HOME/build/mcpeg" \
            --stop \
            --pid-file "$MCPEG_PID_FILE"
    fi
}

# Stop MCpeg daemon using signals
stop_daemon_signal() {
    local pid="$1"
    local force="${2:-false}"
    
    log "Stopping MCpeg Gateway (PID: $pid)..."
    
    if [[ "$force" == "true" ]]; then
        log "Sending SIGKILL to process $pid"
        kill -KILL "$pid" 2>/dev/null || true
    else
        log "Sending SIGTERM to process $pid"
        kill -TERM "$pid" 2>/dev/null || true
        
        # Wait for graceful shutdown
        local attempts=0
        while [[ $attempts -lt 30 ]]; do
            if ! kill -0 "$pid" 2>/dev/null; then
                log "Process $pid stopped gracefully"
                return 0
            fi
            sleep 1
            ((attempts++))
        done
        
        warn "Process $pid did not stop gracefully, forcing..."
        kill -KILL "$pid" 2>/dev/null || true
    fi
    
    # Wait for process to actually stop
    local attempts=0
    while [[ $attempts -lt 10 ]]; do
        if ! kill -0 "$pid" 2>/dev/null; then
            log "Process $pid stopped"
            return 0
        fi
        sleep 1
        ((attempts++))
    done
    
    error "Failed to stop process $pid"
    return 1
}

# Clean up PID file
cleanup_pidfile() {
    if [[ -f "$MCPEG_PID_FILE" ]]; then
        log "Removing PID file: $MCPEG_PID_FILE"
        rm -f "$MCPEG_PID_FILE"
    fi
}

# Main execution
main() {
    local force="${1:-false}"
    local use_builtin="${2:-true}"
    
    log "MCpeg Gateway Stop Script"
    
    # Check if running
    local pid
    if ! pid=$(check_running); then
        log "MCpeg Gateway is not running"
        exit 0
    fi
    
    # Try built-in stop command first
    if [[ "$use_builtin" == "true" ]] && [[ -x "$MCPEG_HOME/build/mcpeg" ]]; then
        if stop_daemon_builtin; then
            log "MCpeg Gateway stopped successfully"
            exit 0
        else
            warn "Built-in stop command failed, falling back to signal method"
        fi
    fi
    
    # Fall back to signal method
    if stop_daemon_signal "$pid" "$force"; then
        cleanup_pidfile
        log "MCpeg Gateway stopped successfully"
    else
        error "Failed to stop MCpeg Gateway"
        exit 1
    fi
}

# Show usage
usage() {
    echo "Usage: $0 [options]"
    echo "Options:"
    echo "  -f, --force          Force stop using SIGKILL"
    echo "  -s, --signal         Use signal method instead of built-in stop"
    echo "  -p, --pid-file FILE  PID file path (default: /var/run/mcpeg/mcpeg.pid)"
    echo "  -u, --user USER      User to run as (default: mcpeg)"
    echo "  -h, --help           Show this help"
    echo ""
    echo "Environment variables:"
    echo "  MCPEG_HOME          MCpeg installation directory"
    echo "  MCPEG_PID_FILE      PID file path"
    echo "  MCPEG_USER          User to run as"
}

# Parse command line arguments
FORCE=false
USE_BUILTIN=true

while [[ $# -gt 0 ]]; do
    case $1 in
        -f|--force)
            FORCE=true
            shift
            ;;
        -s|--signal)
            USE_BUILTIN=false
            shift
            ;;
        -p|--pid-file)
            MCPEG_PID_FILE="$2"
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
main "$FORCE" "$USE_BUILTIN"