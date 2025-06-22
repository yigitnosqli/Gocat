#!/bin/bash

# GoCat RPM Package Builder
# Creates .rpm packages for RHEL/CentOS/Fedora systems

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
if [ $# -ne 2 ]; then
    echo "Usage: $0 <version> <architecture>"
    echo "Example: $0 1.0.0 x86_64"
    exit 1
fi

VERSION="$1"
ARCH="$2"
PACKAGE_NAME="gocat"
BUILD_ROOT="/tmp/gocat-rpm-build"
SPEC_FILE="${PACKAGE_NAME}.spec"

# Create build directories
create_build_structure() {
    log_info "Creating RPM build structure for $PACKAGE_NAME v$VERSION ($ARCH)..."
    
    rm -rf "$BUILD_ROOT"
    mkdir -p "$BUILD_ROOT"/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
    mkdir -p "$BUILD_ROOT/BUILDROOT/${PACKAGE_NAME}-${VERSION}-1.${ARCH}/usr/local/bin"
    mkdir -p "$BUILD_ROOT/BUILDROOT/${PACKAGE_NAME}-${VERSION}-1.${ARCH}/usr/share/doc/${PACKAGE_NAME}"
    mkdir -p "$BUILD_ROOT/BUILDROOT/${PACKAGE_NAME}-${VERSION}-1.${ARCH}/usr/share/man/man1"
    
    log_success "Build structure created"
}

# Create RPM spec file
create_spec_file() {
    log_info "Creating RPM spec file..."
    
    cat > "$BUILD_ROOT/SPECS/$SPEC_FILE" << EOF
Name:           $PACKAGE_NAME
Version:        $VERSION
Release:        1%{?dist}
Summary:        Modern netcat alternative written in Go

License:        MIT
URL:            https://github.com/ibrahmsql/gocat
Source0:        %{name}-%{version}.tar.gz

BuildArch:      noarch
Requires:       glibc

%description
GoCat is a modern, feature-rich alternative to netcat written in Go.
It provides enhanced functionality for network communication, debugging,
and penetration testing with improved IPv6 support, interactive modes,
and cross-platform compatibility.

%prep
# No prep needed for binary package

%build
# No build needed for binary package

%install
rm -rf %{buildroot}
mkdir -p %{buildroot}/usr/local/bin
mkdir -p %{buildroot}/usr/share/doc/%{name}
mkdir -p %{buildroot}/usr/share/man/man1

# Copy binary
cp %{_sourcedir}/gocat %{buildroot}/usr/local/bin/
chmod 755 %{buildroot}/usr/local/bin/gocat

# Create documentation
cat > %{buildroot}/usr/share/doc/%{name}/README << 'DOCEOF'
GoCat for RPM-based Systems
===========================

GoCat has been installed to /usr/local/bin/gocat

Usage:
  gocat --help
  gocat listen 8080
  gocat connect example.com 80

For more information, visit:
https://github.com/ibrahmsql/gocat
DOCEOF

# Create man page
cat > %{buildroot}/usr/share/man/man1/gocat.1 << 'MANEOF'
.TH GOCAT 1 "$(date +'%B %Y')" "GoCat $VERSION" "User Commands"
.SH NAME
gocat \- modern netcat alternative written in Go
.SH SYNOPSIS
.B gocat
[\fIOPTIONS\fR] \fICOMMAND\fR [\fIARGS\fR]
.SH DESCRIPTION
GoCat is a modern, feature-rich alternative to netcat written in Go. It provides enhanced functionality for network communication, debugging, and penetration testing.
.SH OPTIONS
.TP
\fB\-h, \-\-help\fR
Show help message
.TP
\fB\-v, \-\-version\fR
Show version information
.SH COMMANDS
.TP
\fBlisten\fR \fIPORT\fR
Listen on specified port
.TP
\fBconnect\fR \fIHOST\fR \fIPORT\fR
Connect to specified host and port
.SH EXAMPLES
.TP
Listen on port 8080:
.B gocat listen 8080
.TP
Connect to example.com on port 80:
.B gocat connect example.com 80
.SH AUTHOR
Written by Ibrahim SQL.
.SH REPORTING BUGS
Report bugs to: https://github.com/ibrahmsql/gocat/issues
.SH COPYRIGHT
Copyright © 2024 Ibrahim SQL. License MIT.
MANEOF

gzip -9 %{buildroot}/usr/share/man/man1/gocat.1

%files
/usr/local/bin/gocat
/usr/share/doc/%{name}/README
/usr/share/man/man1/gocat.1.gz

%post
# Create symlink for compatibility
if [ ! -e /usr/local/bin/nc ]; then
    ln -s /usr/local/bin/gocat /usr/local/bin/nc
fi

# Update man database
if command -v mandb >/dev/null 2>&1; then
    mandb -q 2>/dev/null || true
fi

echo "GoCat installed successfully!"
echo "Run 'gocat --help' to get started."

%preun
# Remove symlink
if [ -L /usr/local/bin/nc ] && [ "$(readlink /usr/local/bin/nc)" = "/usr/local/bin/gocat" ]; then
    rm -f /usr/local/bin/nc
fi

%changelog
* $(date +'%a %b %d %Y') Ibrahim SQL <ibrahim@example.com> - $VERSION-1
- Major version 2.0.0 release
- Enhanced performance and stability improvements
- Improved IPv6 support and connection handling
- Advanced SSL/TLS encryption capabilities
- Extended proxy support with authentication
- Enhanced command execution with security features
- Improved interactive mode with better UX
- Advanced port scanning with stealth options
- Better error handling and logging
- Cross-platform compatibility enhancements
- Updated documentation and man pages
- Bug fixes and security improvements

* Wed Oct 15 2024 Ibrahim SQL <ibrahim@example.com> - 1.0.0-1
- Initial RPM package release
EOF
    
    log_success "Spec file created"
}

# Copy binary and build package
build_package() {
    log_info "Building RPM package..."
    
    # Copy binary to sources
    if [ ! -f "./gocat" ]; then
        log_error "Binary not found. Please run 'make build' first."
    fi
    
    cp "./gocat" "$BUILD_ROOT/SOURCES/"
    
    # Build the package
    rpmbuild --define "_topdir $BUILD_ROOT" \
             --define "_sourcedir $BUILD_ROOT/SOURCES" \
             --buildroot "$BUILD_ROOT/BUILDROOT/${PACKAGE_NAME}-${VERSION}-1.${ARCH}" \
             -bb "$BUILD_ROOT/SPECS/$SPEC_FILE"
    
    # Move package to current directory
    RPM_FILE="$BUILD_ROOT/RPMS/$ARCH/${PACKAGE_NAME}-${VERSION}-1.${ARCH}.rpm"
    if [ -f "$RPM_FILE" ]; then
        mv "$RPM_FILE" "./"
        log_success "Package built successfully: ${PACKAGE_NAME}-${VERSION}-1.${ARCH}.rpm"
        
        # Show package info
        echo -e "\n${CYAN}Package Information:${NC}"
        rpm -qip "${PACKAGE_NAME}-${VERSION}-1.${ARCH}.rpm"
        
        # Show package contents
        echo -e "\n${CYAN}Package Contents:${NC}"
        rpm -qlp "${PACKAGE_NAME}-${VERSION}-1.${ARCH}.rpm"
        
        # Cleanup
        rm -rf "$BUILD_ROOT"
        
        echo -e "\n${GREEN}✅ RPM package ready: ${PACKAGE_NAME}-${VERSION}-1.${ARCH}.rpm${NC}"
        echo -e "${YELLOW}Install with: sudo rpm -i ${PACKAGE_NAME}-${VERSION}-1.${ARCH}.rpm${NC}"
    else
        log_error "Package build failed"
    fi
}

# Main execution
echo -e "${CYAN}GoCat RPM Package Builder${NC}\n"

# Check if rpmbuild is available
if ! command -v rpmbuild >/dev/null 2>&1; then
    log_error "rpmbuild not found. Please install rpm-build package."
fi

create_build_structure
create_spec_file
build_package

log_success "RPM package build completed!"