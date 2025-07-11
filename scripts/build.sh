#!/bin/bash

# MCPEG Build Script - Single Source of Truth
# This script centralizes all build logic and configuration

set -euo pipefail

# Build configuration - SINGLE SOURCE OF TRUTH
readonly PROJECT_NAME="mcpeg"
readonly BINARY_DIR="build"
readonly VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')}"
readonly COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')}"
readonly BUILD_TIME="${BUILD_TIME:-$(date -u '+%Y-%m-%d_%H:%M:%S')}"

# Go build configuration
readonly LDFLAGS="-X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}"
readonly BUILD_FLAGS="-trimpath -ldflags \"${LDFLAGS}\""

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

# Ensure build directory exists
ensure_build_dir() {
    if [[ ! -d "${BINARY_DIR}" ]]; then
        log_info "Creating build directory: ${BINARY_DIR}"
        mkdir -p "${BINARY_DIR}"
    fi
}

# Build a single binary
build_binary() {
    local cmd_path="$1"
    local output_path="$2"
    local binary_name
    binary_name=$(basename "${cmd_path}")
    
    log_info "Building ${binary_name}..."
    
    if go build -trimpath -ldflags "${LDFLAGS}" -o "${output_path}" "./cmd/${cmd_path}"; then
        log_success "Built ${binary_name} -> ${output_path}"
    else
        log_error "Failed to build ${binary_name}"
        return 1
    fi
}

# Build development version (faster, no optimizations)
build_dev_binary() {
    local cmd_path="$1"
    local output_path="$2"
    local binary_name
    binary_name=$(basename "${cmd_path}")
    
    log_info "Building ${binary_name} (development)..."
    
    if go build -o "${output_path}" "./cmd/${cmd_path}"; then
        log_success "Built ${binary_name} (dev) -> ${output_path}"
    else
        log_error "Failed to build ${binary_name}"
        return 1
    fi
}

# Cross-compile for multiple platforms
build_cross_platform() {
    local cmd_path="$1"
    local base_name="$2"
    
    local platforms=(
        "linux/amd64"
        "linux/arm64"
        "darwin/amd64"
        "darwin/arm64"
        "windows/amd64"
    )
    
    for platform in "${platforms[@]}"; do
        local goos="${platform%/*}"
        local goarch="${platform#*/}"
        local ext=""
        
        if [[ "${goos}" == "windows" ]]; then
            ext=".exe"
        fi
        
        local output="${BINARY_DIR}/${base_name}-${goos}-${goarch}${ext}"
        
        log_info "Building ${base_name} for ${goos}/${goarch}..."
        
        if CGO_ENABLED=0 GOOS="${goos}" GOARCH="${goarch}" go build -trimpath -ldflags "${LDFLAGS}" -o "${output}" "./cmd/${cmd_path}"; then
            log_success "Built ${base_name} for ${goos}/${goarch} -> ${output}"
        else
            log_error "Failed to build ${base_name} for ${goos}/${goarch}"
            return 1
        fi
    done
}

# Clean build artifacts
clean() {
    log_info "Cleaning build artifacts..."
    rm -rf "${BINARY_DIR}"
    rm -f coverage.out coverage.html
    go clean -cache
    log_success "Cleaned build artifacts"
}

# Run tests
test() {
    log_info "Running tests..."
    if go test -v -race -coverprofile=coverage.out ./...; then
        log_success "Tests passed"
    else
        log_error "Tests failed"
        return 1
    fi
}

# Generate coverage report
coverage() {
    if [[ ! -f coverage.out ]]; then
        log_info "No coverage data found, running tests first..."
        test
    fi
    
    log_info "Generating coverage report..."
    go tool cover -html=coverage.out -o coverage.html
    log_success "Coverage report generated: coverage.html"
}

# Format code
fmt() {
    log_info "Formatting code..."
    go fmt ./...
    if command -v goimports >/dev/null 2>&1; then
        goimports -w .
    else
        log_warn "goimports not found, skipping import formatting"
    fi
    log_success "Code formatted"
}

# Tidy dependencies
tidy() {
    log_info "Tidying dependencies..."
    go mod tidy
    log_success "Dependencies tidied"
}

# Validate OpenAPI specifications
validate() {
    local mcpeg_binary="${BINARY_DIR}/mcpeg"
    
    if [[ ! -f "${mcpeg_binary}" ]]; then
        log_info "MCPEG binary not found, building first..."
        ensure_build_dir
        build_binary "mcpeg" "${mcpeg_binary}"
    fi
    
    log_info "Validating OpenAPI specifications..."
    if "${mcpeg_binary}" validate -spec-file api/openapi/mcp-gateway.yaml; then
        log_success "OpenAPI specification is valid"
    else
        log_error "OpenAPI specification validation failed"
        return 1
    fi
}

# Generate code from OpenAPI specs
generate() {
    local mcpeg_binary="${BINARY_DIR}/mcpeg"
    
    if [[ ! -f "${mcpeg_binary}" ]]; then
        log_info "MCPEG binary not found, building first..."
        ensure_build_dir
        build_binary "mcpeg" "${mcpeg_binary}"
    fi
    
    log_info "Generating code from OpenAPI specifications..."
    if "${mcpeg_binary}" codegen -spec-file api/openapi/mcp-gateway.yaml -output internal/generated -package generated; then
        log_success "Code generation completed"
    else
        log_error "Code generation failed"
        return 1
    fi
}

# Start development server
dev() {
    local mcpeg_binary="${BINARY_DIR}/mcpeg"
    
    if [[ ! -f "${mcpeg_binary}" ]]; then
        log_info "MCPEG binary not found, building first..."
        ensure_build_dir
        build_binary "mcpeg" "${mcpeg_binary}"
    fi
    
    log_info "Starting development server..."
    "${mcpeg_binary}" gateway -dev -log-level debug
}

# Create release archives
release() {
    log_info "Creating release archives..."
    
    local release_dir="${BINARY_DIR}/release"
    mkdir -p "${release_dir}"
    
    # Create archives for each platform
    local platforms=(
        "linux-amd64"
        "linux-arm64"
        "darwin-amd64"
        "darwin-arm64"
        "windows-amd64"
    )
    
    for platform in "${platforms[@]}"; do
        local archive_name="${PROJECT_NAME}-${VERSION}-${platform}"
        
        if [[ "${platform}" == *"windows"* ]]; then
            # Windows ZIP archive
            log_info "Creating Windows archive: ${archive_name}.zip"
            (cd "${BINARY_DIR}" && zip -q "${release_dir}/${archive_name}.zip" "mcpeg-${platform}.exe")
        else
            # Unix tar.gz archive
            log_info "Creating Unix archive: ${archive_name}.tar.gz"
            (cd "${BINARY_DIR}" && tar -czf "${release_dir}/${archive_name}.tar.gz" "mcpeg-${platform}")
        fi
    done
    
    log_success "Release archives created in ${release_dir}/"
}

# Show build information
info() {
    echo "MCPEG Build Information"
    echo "======================="
    echo "Project:    ${PROJECT_NAME}"
    echo "Version:    ${VERSION}"
    echo "Commit:     ${COMMIT}"
    echo "Build Time: ${BUILD_TIME}"
    echo "Build Dir:  ${BINARY_DIR}"
    echo ""
    echo "Go Environment:"
    go version
    echo "GOOS:       $(go env GOOS)"
    echo "GOARCH:     $(go env GOARCH)"
    echo "CGO:        $(go env CGO_ENABLED)"
}

# Show help
help() {
    cat << EOF
MCPEG Build Script - Single Source of Truth

Usage: $0 <command> [options]

Commands:
  build           Build all binaries for current platform
  build-dev       Build development binaries (faster, no optimizations)
  build-prod      Build production binaries for all platforms
  clean           Clean build artifacts
  test            Run tests
  coverage        Generate test coverage report
  fmt             Format code
  tidy            Tidy dependencies
  validate        Validate OpenAPI specifications
  generate        Generate code from OpenAPI specs
  dev             Start development server
  release         Create release archives
  info            Show build information
  help            Show this help

Examples:
  $0 build                    # Build for current platform
  $0 build-prod              # Cross-compile for all platforms
  $0 test                    # Run tests
  $0 dev                     # Start development server
  $0 clean && $0 build       # Clean rebuild

Environment Variables:
  VERSION         Override version (default: git describe)
  COMMIT          Override commit hash (default: git rev-parse)
  BUILD_TIME      Override build time (default: current time)

Build artifacts are placed in: ${BINARY_DIR}/
EOF
}

# Main command dispatcher
main() {
    local command="${1:-help}"
    
    case "${command}" in
        build)
            ensure_build_dir
            build_binary "mcpeg" "${BINARY_DIR}/mcpeg"
            ;;
        build-dev)
            ensure_build_dir
            build_dev_binary "mcpeg" "${BINARY_DIR}/mcpeg"
            ;;
        build-prod)
            ensure_build_dir
            build_cross_platform "mcpeg" "mcpeg"
            ;;
        clean)
            clean
            ;;
        test)
            test
            ;;
        coverage)
            coverage
            ;;
        fmt)
            fmt
            ;;
        tidy)
            tidy
            ;;
        validate)
            validate
            ;;
        generate)
            generate
            ;;
        dev)
            dev
            ;;
        release)
            # First build production binaries
            ensure_build_dir
            build_cross_platform "mcpeg" "mcpeg"
            # Then create archives
            release
            ;;
        info)
            info
            ;;
        help|--help|-h)
            help
            ;;
        *)
            log_error "Unknown command: ${command}"
            echo ""
            help
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"