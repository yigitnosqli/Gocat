#!/bin/bash

# GoCat - Modern Netcat Alternative Installation Script
# Supports multiple platforms and installation methods

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# ASCII Art Banner
print_banner() {
    echo -e "${CYAN}"
    cat << "EOF"
   ____       ____      _   
  / ___| ___ / ___|__ _| |_ 
 | |  _ / _ \ |   / _` | __|
 | |_| | (_) | |_| (_| | |_ 
  \____|\___/ \___\__,_|\__|
                            
Modern Netcat Alternative in Go
EOF
    echo -e "${NC}"
}

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect OS and Architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case $ARCH in
        x86_64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        armv7l) ARCH="arm" ;;
        *) log_error "Unsupported architecture: $ARCH"; exit 1 ;;
    esac
    
    case $OS in
        linux|darwin) ;;
        *) log_error "Unsupported OS: $OS"; exit 1 ;;
    esac
    
    log_info "Detected platform: $OS-$ARCH"
}

# Check dependencies
check_dependencies() {
    log_info "Checking dependencies..."
    
    if ! command -v curl >/dev/null 2>&1; then
        log_warning "curl not found, attempting to install..."
        case $OS in
            linux)
                if command -v apt >/dev/null 2>&1; then
                    sudo apt update && sudo apt install -y curl
                elif command -v yum >/dev/null 2>&1; then
                    sudo yum install -y curl
                elif command -v pacman >/dev/null 2>&1; then
                    sudo pacman -S --noconfirm curl
                else
                    log_error "Package manager not found. Please install curl manually."
                    exit 1
                fi
                ;;
            darwin)
                if command -v brew >/dev/null 2>&1; then
                    brew install curl
                else
                    log_error "Homebrew not found. Please install curl manually."
                    exit 1
                fi
                ;;
        esac
    fi
    
    log_success "Dependencies check completed"
}

# Get latest release version
get_latest_version() {
    log_info "Fetching latest release information..."
    
    REPO_URL="https://api.github.com/repos/ibrahmsql/gocat/releases/latest"
    VERSION=$(curl -s "$REPO_URL" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' | sed 's/^v//')
    
    if [ -z "$VERSION" ]; then
        log_error "Failed to fetch latest version"
        exit 1
    fi
    
    log_success "Latest version: v$VERSION"
}

# Download and install binary
install_binary() {
    log_info "Downloading gocat v$VERSION for $OS-$ARCH..."
    
    BINARY_NAME="gocat-v${VERSION}-${OS}-${ARCH}"
    if [ "$OS" = "windows" ]; then
        BINARY_NAME="${BINARY_NAME}.exe"
    fi
    
    DOWNLOAD_URL="https://github.com/ibrahmsql/gocat/releases/download/v${VERSION}/${BINARY_NAME}"
    TEMP_DIR=$(mktemp -d)
    TEMP_FILE="$TEMP_DIR/gocat"
    
    if curl -L "$DOWNLOAD_URL" -o "$TEMP_FILE"; then
        chmod +x "$TEMP_FILE"
        
        # Install to system path
        INSTALL_DIR="/usr/local/bin"
        if [ ! -w "$INSTALL_DIR" ]; then
            log_info "Installing to $INSTALL_DIR (requires sudo)..."
            sudo mv "$TEMP_FILE" "$INSTALL_DIR/gocat"
        else
            mv "$TEMP_FILE" "$INSTALL_DIR/gocat"
        fi
        
        log_success "gocat installed successfully to $INSTALL_DIR/gocat"
    else
        log_error "Failed to download gocat"
        exit 1
    fi
    
    # Cleanup
    rm -rf "$TEMP_DIR"
}

# Verify installation
verify_installation() {
    log_info "Verifying installation..."
    
    if command -v gocat >/dev/null 2>&1; then
        VERSION_OUTPUT=$(gocat --version 2>/dev/null || gocat -v 2>/dev/null || echo "unknown")
        log_success "Installation verified: $VERSION_OUTPUT"
        echo -e "\n${GREEN}🎉 gocat is ready to use!${NC}"
        echo -e "${CYAN}Try: gocat --help${NC}"
    else
        log_error "Installation verification failed"
        exit 1
    fi
}

# Main installation function
main() {
    print_banner
    
    log_info "Starting gocat installation..."
    
    detect_platform
    check_dependencies
    get_latest_version
    install_binary
    verify_installation
    
    echo -e "\n${GREEN}Installation completed successfully!${NC}"
    echo -e "${PURPLE}Thank you for using gocat!${NC}"
}

# Run main function
main "$@"