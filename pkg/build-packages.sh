#!/bin/bash

# GoCat Package Builder
# Creates both .deb and .rpm packages

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Check parameters
if [ $# -ne 1 ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 1.0.0"
    exit 1
fi

VERSION="$1"
PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo -e "${CYAN}GoCat Package Builder${NC}\n"
echo -e "${BOLD}Building packages for version: $VERSION${NC}\n"

# Check if binary exists
if [ ! -f "$PROJECT_ROOT/gocat" ]; then
    log_info "Binary not found. Building gocat..."
    cd "$PROJECT_ROOT"
    make build
    if [ ! -f "gocat" ]; then
        log_error "Failed to build gocat binary"
    fi
    log_success "Binary built successfully"
else
    log_info "Using existing gocat binary"
fi

# Build RPM package
log_info "Building RPM package..."
cd "$PROJECT_ROOT"
if [ -f "build-rpm.sh" ]; then
    ./build-rpm.sh "$VERSION" noarch
    if [ -f "gocat-${VERSION}-1.noarch.rpm" ]; then
        log_success "RPM package created: gocat-${VERSION}-1.noarch.rpm"
    else
        log_warning "RPM package creation failed"
    fi
else
    log_warning "RPM build script not found"
fi

# Try to build DEB package (if on Linux with dpkg-deb)
log_info "Checking for Debian package build capability..."
if command -v dpkg-deb >/dev/null 2>&1; then
    log_info "Building Debian package..."
    cd "$PROJECT_ROOT/pkg/debian"
    if [ -f "build-deb.sh" ]; then
        ./build-deb.sh "$VERSION" amd64
        if [ -f "gocat_${VERSION}_amd64.deb" ]; then
            mv "gocat_${VERSION}_amd64.deb" "$PROJECT_ROOT/"
            log_success "Debian package created: gocat_${VERSION}_amd64.deb"
        else
            log_warning "Debian package creation failed"
        fi
    else
        log_warning "Debian build script not found"
    fi
else
    log_warning "dpkg-deb not available. Skipping Debian package creation."
    log_info "To create Debian packages, run this script on a Debian/Ubuntu system."
fi

# Summary
echo -e "\n${CYAN}Package Build Summary:${NC}"
cd "$PROJECT_ROOT"
if [ -f "gocat-${VERSION}-1.noarch.rpm" ]; then
    echo -e "${GREEN}✅ RPM Package:${NC} gocat-${VERSION}-1.noarch.rpm"
    echo -e "   Install with: ${YELLOW}sudo rpm -i gocat-${VERSION}-1.noarch.rpm${NC}"
fi

if [ -f "gocat_${VERSION}_amd64.deb" ]; then
    echo -e "${GREEN}✅ DEB Package:${NC} gocat_${VERSION}_amd64.deb"
    echo -e "   Install with: ${YELLOW}sudo dpkg -i gocat_${VERSION}_amd64.deb${NC}"
fi

echo -e "\n${GREEN}Package build completed!${NC}"
echo -e "${BLUE}Note: Packages are created for the current platform.${NC}"
echo -e "${BLUE}For cross-platform packages, use GitHub Actions or Docker.${NC}"