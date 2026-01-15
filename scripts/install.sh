#!/bin/bash
#
# zap installer script for macOS and Linux
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/itda-work/zap/main/scripts/install.sh | bash
#
# Options:
#   ZAP_VERSION=v0.3.0  Install specific version
#   ZAP_INSTALL_DIR=~/bin  Install to custom directory
#

set -euo pipefail

REPO="itda-work/zap"
BINARY_NAME="zap"
DEFAULT_INSTALL_DIR="/usr/local/bin"
TMP_FILE=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info() {
    echo -e "${BLUE}==>${NC} $1"
}

success() {
    echo -e "${GREEN}==>${NC} $1"
}

warn() {
    echo -e "${YELLOW}==>${NC} $1"
}

error() {
    echo -e "${RED}==>${NC} $1" >&2
    exit 1
}

# Detect OS
detect_os() {
    local os
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    case "$os" in
        darwin) echo "macos" ;;
        linux) echo "linux" ;;
        *) error "Unsupported OS: $os" ;;
    esac
}

# Detect architecture
detect_arch() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        x86_64|amd64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) error "Unsupported architecture: $arch" ;;
    esac
}

# Get latest version from GitHub API
get_latest_version() {
    local url="https://api.github.com/repos/${REPO}/releases/latest"
    local version

    if command -v curl &> /dev/null; then
        version=$(curl -fsSL "$url" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    elif command -v wget &> /dev/null; then
        version=$(wget -qO- "$url" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    else
        error "curl or wget is required"
    fi

    if [[ -z "$version" ]]; then
        error "Failed to get latest version"
    fi

    echo "$version"
}

# Download binary
download_binary() {
    local version="$1"
    local os="$2"
    local arch="$3"
    local dest="$4"

    local filename="${BINARY_NAME}-${os}-${arch}"
    local url="https://github.com/${REPO}/releases/download/${version}/${filename}"

    info "Downloading ${filename}..."

    if command -v curl &> /dev/null; then
        curl -fsSL -o "$dest" "$url"
    elif command -v wget &> /dev/null; then
        wget -q -O "$dest" "$url"
    fi

    if [[ ! -f "$dest" ]]; then
        error "Download failed"
    fi
}

main() {
    info "Installing zap..."

    # Detect platform
    local os arch
    os=$(detect_os)
    arch=$(detect_arch)
    info "Detected platform: ${os}/${arch}"

    # Get version
    local version="${ZAP_VERSION:-}"
    if [[ -z "$version" ]]; then
        info "Fetching latest version..."
        version=$(get_latest_version)
    fi
    info "Version: ${version}"

    # Determine install directory
    local install_dir="${ZAP_INSTALL_DIR:-$DEFAULT_INSTALL_DIR}"
    local install_path="${install_dir}/${BINARY_NAME}"

    # Create temp file
    TMP_FILE=$(mktemp)
    trap 'rm -f "$TMP_FILE" 2>/dev/null || true' EXIT

    # Download
    download_binary "$version" "$os" "$arch" "$TMP_FILE"

    # Make executable
    chmod +x "$TMP_FILE"

    # Install
    info "Installing to ${install_path}..."

    # Create directory if it doesn't exist
    if [[ ! -d "$install_dir" ]]; then
        if mkdir -p "$install_dir" 2>/dev/null; then
            :
        else
            warn "Requesting sudo permission to create ${install_dir}"
            sudo mkdir -p "$install_dir"
        fi
    fi

    if [[ -w "$install_dir" ]]; then
        mv "$TMP_FILE" "$install_path"
    else
        warn "Requesting sudo permission to install to ${install_dir}"
        sudo mv "$TMP_FILE" "$install_path"
    fi

    # Verify
    if [[ -x "$install_path" ]]; then
        success "Successfully installed zap ${version}"
        echo ""
        "$install_path" version
        echo ""

        # Check if install_dir is in PATH
        if [[ ":$PATH:" != *":$install_dir:"* ]]; then
            warn "Note: ${install_dir} is not in your PATH"
            echo "  Add to your shell profile:"
            echo "    export PATH=\"\$PATH:${install_dir}\""
        fi
    else
        error "Installation failed"
    fi
}

main "$@"
