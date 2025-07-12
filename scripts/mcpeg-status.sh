#!/bin/bash
# MCpeg Gateway Status Script

set -e

# Configuration
MCPEG_HOME="${MCPEG_HOME:-/opt/mcpeg}"
MCPEG_PID_FILE="${MCPEG_PID_FILE:-/var/run/mcpeg/mcpeg.pid}"
MCPEG_LOG_FILE="${MCPEG_LOG_FILE:-/var/log/mcpeg/mcpeg.log}"
MCPEG_USER="${MCPEG_USER:-mcpeg}"

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

# Get process status using built-in status command
get_status_builtin() {
    if [[ -x "$MCPEG_HOME/build/mcpeg" ]]; then
        if [[ "$EUID" -eq 0 ]]; then
            # Running as root, switch to mcpeg user
            sudo -u "$MCPEG_USER" "$MCPEG_HOME/build/mcpeg" \
                --status \
                --pid-file "$MCPEG_PID_FILE" 2>/dev/null
        else
            # Already running as non-root user
            "$MCPEG_HOME/build/mcpeg" \
                --status \
                --pid-file "$MCPEG_PID_FILE" 2>/dev/null
        fi
        return $?
    fi
    return 1
}

# Get process status manually
get_status_manual() {
    local pid_file="$1"
    
    if [[ ! -f "$pid_file" ]]; then
        echo "Status: Stopped"
        echo "PID File: $pid_file (not found)"
        return 1
    fi
    
    local pid=$(cat "$pid_file")
    if [[ -z "$pid" ]] || ! [[ "$pid" =~ ^[0-9]+$ ]]; then
        echo "Status: Stopped"
        echo "PID File: $pid_file (invalid PID)"
        return 1
    fi
    
    if kill -0 "$pid" 2>/dev/null; then
        echo "Status: Running"
        echo "PID: $pid"
        echo "PID File: $pid_file"
        
        # Get process information
        if [[ -r "/proc/$pid/stat" ]]; then
            local stat=($(cat "/proc/$pid/stat"))
            local start_time="${stat[21]}"
            local state="${stat[2]}"
            
            echo "Process State: $state"
            
            # Calculate uptime (simplified)
            local uptime_seconds=$(awk '{print int($1)}' /proc/uptime)
            local boot_time=$(awk '/btime/ {print $2}' /proc/stat)
            local process_start=$((boot_time + start_time / 100))
            local current_time=$(date +%s)
            local process_uptime=$((current_time - process_start))
            
            if [[ $process_uptime -gt 0 ]]; then
                local days=$((process_uptime / 86400))
                local hours=$(((process_uptime % 86400) / 3600))
                local minutes=$(((process_uptime % 3600) / 60))
                local seconds=$((process_uptime % 60))
                
                echo "Uptime: ${days}d ${hours}h ${minutes}m ${seconds}s"
            fi
        fi
        
        return 0
    else
        echo "Status: Stopped"
        echo "PID File: $pid_file (stale PID: $pid)"
        return 1
    fi
}

# Show log file information
show_log_info() {
    local log_file="$1"
    
    if [[ -f "$log_file" ]]; then
        local size=$(stat -c%s "$log_file" 2>/dev/null || echo "0")
        local modified=$(stat -c%Y "$log_file" 2>/dev/null || echo "0")
        local readable_size=$(numfmt --to=iec-i --suffix=B "$size" 2>/dev/null || echo "${size}B")
        
        echo "Log File: $log_file"
        echo "Log Size: $readable_size"
        
        if [[ $modified -gt 0 ]]; then
            echo "Last Modified: $(date -d @$modified 2>/dev/null || echo 'unknown')"
        fi
    else
        echo "Log File: $log_file (not found)"
    fi
}

# Show system information
show_system_info() {
    echo ""
    info "=== System Information ==="
    echo "Hostname: $(hostname)"
    echo "OS: $(uname -s -r)"
    echo "Architecture: $(uname -m)"
    echo "Current User: $(whoami)"
    echo "Load Average: $(uptime | awk -F'load average:' '{print $2}' | xargs)"
    
    # Memory usage
    if command -v free >/dev/null 2>&1; then
        local mem_info=$(free -h | grep '^Mem:')
        echo "Memory: $mem_info"
    fi
    
    # Disk usage for log directory
    local log_dir=$(dirname "$MCPEG_LOG_FILE")
    if [[ -d "$log_dir" ]]; then
        local disk_usage=$(df -h "$log_dir" 2>/dev/null | tail -1)
        echo "Disk Usage (log): $disk_usage"
    fi
}

# Show recent log entries
show_recent_logs() {
    local log_file="$1"
    local lines="${2:-10}"
    
    if [[ -f "$log_file" ]] && [[ -r "$log_file" ]]; then
        echo ""
        info "=== Recent Log Entries (last $lines lines) ==="
        tail -n "$lines" "$log_file" 2>/dev/null || echo "Unable to read log file"
    fi
}

# Check service connectivity
check_connectivity() {
    local config_file="${1:-/etc/mcpeg/gateway.yaml}"
    
    # This is a simplified check - in a real implementation,
    # you would parse the config file to get the actual address and port
    local address="localhost"
    local port="8080"
    
    if command -v curl >/dev/null 2>&1; then
        if curl -s --connect-timeout 5 "http://$address:$port/health" >/dev/null 2>&1; then
            echo "Service Connectivity: OK (http://$address:$port/health)"
        else
            echo "Service Connectivity: FAILED (http://$address:$port/health)"
        fi
    elif command -v wget >/dev/null 2>&1; then
        if wget -q --timeout=5 -O /dev/null "http://$address:$port/health" 2>/dev/null; then
            echo "Service Connectivity: OK (http://$address:$port/health)"
        else
            echo "Service Connectivity: FAILED (http://$address:$port/health)"
        fi
    else
        echo "Service Connectivity: UNKNOWN (curl/wget not available)"
    fi
}

# Main execution
main() {
    local verbose="${1:-false}"
    local show_logs="${2:-false}"
    local log_lines="${3:-10}"
    
    echo "MCpeg Gateway Status"
    echo "===================="
    echo ""
    
    # Try built-in status first
    if get_status_builtin; then
        echo ""
        info "Status retrieved using built-in command"
    else
        # Fall back to manual status
        get_status_manual "$MCPEG_PID_FILE"
    fi
    
    echo ""
    
    # Show log information
    show_log_info "$MCPEG_LOG_FILE"
    
    # Check connectivity
    echo ""
    check_connectivity
    
    # Show system information if verbose
    if [[ "$verbose" == "true" ]]; then
        show_system_info
    fi
    
    # Show recent logs if requested
    if [[ "$show_logs" == "true" ]]; then
        show_recent_logs "$MCPEG_LOG_FILE" "$log_lines"
    fi
}

# Show usage
usage() {
    echo "Usage: $0 [options]"
    echo "Options:"
    echo "  -v, --verbose        Show detailed system information"
    echo "  -l, --logs           Show recent log entries"
    echo "  -n, --lines N        Number of log lines to show (default: 10)"
    echo "  -p, --pid-file FILE  PID file path (default: /var/run/mcpeg/mcpeg.pid)"
    echo "  -L, --log-file FILE  Log file path (default: /var/log/mcpeg/mcpeg.log)"
    echo "  -u, --user USER      User to run as (default: mcpeg)"
    echo "  -h, --help           Show this help"
    echo ""
    echo "Environment variables:"
    echo "  MCPEG_HOME          MCpeg installation directory"
    echo "  MCPEG_PID_FILE      PID file path"
    echo "  MCPEG_LOG_FILE      Log file path"
    echo "  MCPEG_USER          User to run as"
}

# Parse command line arguments
VERBOSE=false
SHOW_LOGS=false
LOG_LINES=10

while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -l|--logs)
            SHOW_LOGS=true
            shift
            ;;
        -n|--lines)
            LOG_LINES="$2"
            shift 2
            ;;
        -p|--pid-file)
            MCPEG_PID_FILE="$2"
            shift 2
            ;;
        -L|--log-file)
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
main "$VERBOSE" "$SHOW_LOGS" "$LOG_LINES"