#!/usr/bin/env bash
#
# OpenAPI CLI Installation Script
# This script downloads and installs the latest version of the OpenAPI CLI
# for Linux and macOS systems.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/speakeasy-api/openapi/main/scripts/install.sh | bash
#   or
#   wget -qO- https://raw.githubusercontent.com/speakeasy-api/openapi/main/scripts/install.sh | bash
#
# Options:
#   OPENAPI_INSTALL_DIR - Installation directory (default: /usr/local/bin)
#   OPENAPI_VERSION - Specific version to install (default: latest)
#

set -e

# Configuration
REPO="speakeasy-api/openapi"
DEFAULT_INSTALL_DIR="/usr/local/bin"
USER_INSTALL_DIR="$HOME/.local/bin"
VERSION="${OPENAPI_VERSION:-latest}"
BINARY_NAME="openapi"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect operating system
detect_os() {
    local os
    local uname_output="$(uname -s)"
    case "$uname_output" in
        Linux*)     os="Linux" ;;
        Darwin*)    os="Darwin" ;;
        CYGWIN*|MINGW*|MSYS*)    os="Windows" ;;
        *)
            log_error "Unsupported operating system: $uname_output"
            exit 1
            ;;
    esac
    echo "$os"
}

# Detect architecture
detect_arch() {
    local arch
    case "$(uname -m)" in
        x86_64|amd64)   arch="x86_64" ;;
        aarch64|arm64)  arch="arm64" ;;
        *)
            log_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac
    echo "$arch"
}

# Get latest version from GitHub
get_latest_version() {
    local latest_url="https://api.github.com/repos/${REPO}/releases/latest"
    local version
    
    if command -v curl >/dev/null 2>&1; then
        version=$(curl -fsSL "$latest_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    elif command -v wget >/dev/null 2>&1; then
        version=$(wget -qO- "$latest_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    else
        log_error "curl or wget is required to download the CLI"
        exit 1
    fi
    
    echo "$version"
}

# Determine installation directory
get_install_dir() {
    # If user specified a directory, use it
    if [ -n "$OPENAPI_INSTALL_DIR" ]; then
        echo "$OPENAPI_INSTALL_DIR"
        return
    fi
    
    # Try to use /usr/local/bin if we have write access
    if [ -w "$DEFAULT_INSTALL_DIR" ] || [ -w "$(dirname "$DEFAULT_INSTALL_DIR")" ]; then
        echo "$DEFAULT_INSTALL_DIR"
        return
    fi
    
    # Fall back to user directory
    log_info "No write access to $DEFAULT_INSTALL_DIR, using $USER_INSTALL_DIR instead"
    echo "$USER_INSTALL_DIR"
}

# Download and install
install_cli() {
    local INSTALL_DIR=$(get_install_dir)
    local os=$(detect_os)
    local arch=$(detect_arch)
    
    log_info "Detected OS: $os"
    log_info "Detected Architecture: $arch"
    log_info "Installation directory: $INSTALL_DIR"
    
    # Get version
    if [ "$VERSION" = "latest" ]; then
        VERSION=$(get_latest_version)
        log_info "Latest version: $VERSION"
    fi
    
    # Construct download URL based on OS
    local archive_name
    local archive_format
    if [ "$os" = "Windows" ]; then
        archive_name="${BINARY_NAME}_${os}_${arch}.zip"
        archive_format="zip"
    else
        archive_name="${BINARY_NAME}_${os}_${arch}.tar.gz"
        archive_format="tar.gz"
    fi
    
    local download_url="https://github.com/${REPO}/releases/download/${VERSION}/${archive_name}"
    
    log_info "Downloading from: $download_url"
    
    # Create temporary directory
    local tmp_dir=$(mktemp -d)
    trap "rm -rf $tmp_dir" EXIT
    
    # Download archive
    if command -v curl >/dev/null 2>&1; then
        if ! curl -fsSL "$download_url" -o "$tmp_dir/$archive_name"; then
            log_error "Failed to download from $download_url"
            exit 1
        fi
    elif command -v wget >/dev/null 2>&1; then
        if ! wget -q "$download_url" -O "$tmp_dir/$archive_name"; then
            log_error "Failed to download from $download_url"
            exit 1
        fi
    fi
    
    log_info "Download complete"
    
    # Extract archive based on format
    log_info "Extracting archive..."
    if [ "$archive_format" = "zip" ]; then
        if command -v unzip >/dev/null 2>&1; then
            unzip -q "$tmp_dir/$archive_name" -d "$tmp_dir"
        else
            log_error "unzip is required to extract the archive. Please install unzip and try again."
            exit 1
        fi
    else
        tar -xzf "$tmp_dir/$archive_name" -C "$tmp_dir"
    fi
    
    # Create install directory if it doesn't exist
    if [ ! -d "$INSTALL_DIR" ]; then
        log_info "Creating installation directory: $INSTALL_DIR"
        mkdir -p "$INSTALL_DIR" || {
            log_error "Failed to create $INSTALL_DIR. Try running with sudo or set OPENAPI_INSTALL_DIR to a writable location."
            exit 1
        }
    fi
    
    # Install binary (Windows binaries have .exe extension)
    local source_binary="$tmp_dir/$BINARY_NAME"
    local target_binary="$INSTALL_DIR/$BINARY_NAME"
    
    if [ "$os" = "Windows" ]; then
        source_binary="$tmp_dir/${BINARY_NAME}.exe"
        target_binary="$INSTALL_DIR/${BINARY_NAME}.exe"
    fi
    
    log_info "Installing to $target_binary..."
    if ! mv "$source_binary" "$target_binary"; then
        log_error "Failed to install to $INSTALL_DIR. Try running with sudo or set OPENAPI_INSTALL_DIR to a writable location."
        exit 1
    fi
    
    # Make executable (not needed on Windows, but doesn't hurt)
    chmod +x "$target_binary" 2>/dev/null || true
    
    log_info "OpenAPI CLI ${VERSION} has been installed to $target_binary"
    
    # Verify installation
    local cmd_to_check="$BINARY_NAME"
    if [ "$os" = "Windows" ]; then
        cmd_to_check="${BINARY_NAME}.exe"
    fi
    
    if command -v "$cmd_to_check" >/dev/null 2>&1; then
        log_info "Installation successful! Run '$BINARY_NAME --help' to get started."
    else
        log_warn "Installation complete, but $BINARY_NAME is not in your PATH."
        if [ "$os" = "Windows" ]; then
            log_warn "Add $INSTALL_DIR to your PATH environment variable."
        else
            log_warn "Add $INSTALL_DIR to your PATH by adding this to your ~/.bashrc or ~/.zshrc:"
            log_warn "  export PATH=\"\$PATH:$INSTALL_DIR\""
            log_warn ""
            log_warn "Then run: source ~/.bashrc  # or source ~/.zshrc"
        fi
    fi
}

# Main execution
main() {
    log_info "Installing OpenAPI CLI..."
    install_cli
}

main