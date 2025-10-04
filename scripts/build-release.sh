#!/bin/bash
# GoCat Release Build Script
# Builds binaries for all platforms and creates release packages

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
VERSION=${1:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
BUILT_BY=$(whoami)

PROJECT_NAME="gocat"
DIST_DIR="dist"
RELEASE_DIR="release"

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║${NC}          ${GREEN}GoCat Release Builder${NC}                          ${BLUE}║${NC}"
echo -e "${BLUE}╠════════════════════════════════════════════════════════════╣${NC}"
echo -e "${BLUE}║${NC}  Version:    ${YELLOW}${VERSION}${NC}"
echo -e "${BLUE}║${NC}  Commit:     ${YELLOW}${GIT_COMMIT}${NC}"
echo -e "${BLUE}║${NC}  Branch:     ${YELLOW}${GIT_BRANCH}${NC}"
echo -e "${BLUE}║${NC}  Build Time: ${YELLOW}${BUILD_TIME}${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"

# Clean previous builds
echo -e "\n${YELLOW}→${NC} Cleaning previous builds..."
rm -rf ${DIST_DIR} ${RELEASE_DIR}
mkdir -p ${DIST_DIR} ${RELEASE_DIR}

# Build flags
LDFLAGS="-s -w \
    -X main.version=${VERSION} \
    -X main.buildTime=${BUILD_TIME} \
    -X main.gitCommit=${GIT_COMMIT} \
    -X main.gitBranch=${GIT_BRANCH} \
    -X main.builtBy=${BUILT_BY}"

# Platforms to build
declare -A PLATFORMS=(
    ["linux-amd64"]="linux amd64"
    ["linux-arm64"]="linux arm64"
    ["linux-arm"]="linux arm"
    ["darwin-amd64"]="darwin amd64"
    ["darwin-arm64"]="darwin arm64"
    ["windows-amd64"]="windows amd64"
    ["windows-arm64"]="windows arm64"
    ["freebsd-amd64"]="freebsd amd64"
)

# Build for each platform
echo -e "\n${YELLOW}→${NC} Building binaries..."
for platform in "${!PLATFORMS[@]}"; do
    IFS=' ' read -r GOOS GOARCH <<< "${PLATFORMS[$platform]}"
    
    output_name="${PROJECT_NAME}-${platform}"
    if [ "$GOOS" = "windows" ]; then
        output_name="${output_name}.exe"
    fi
    
    echo -e "  ${BLUE}▸${NC} Building ${GREEN}${platform}${NC}..."
    
    CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags="${LDFLAGS}" \
        -trimpath \
        -o "${DIST_DIR}/${output_name}" \
        . || {
            echo -e "  ${RED}✗${NC} Failed to build ${platform}"
            continue
        }
    
    echo -e "  ${GREEN}✓${NC} Built ${output_name}"
done

# Create checksums
echo -e "\n${YELLOW}→${NC} Generating checksums..."
cd ${DIST_DIR}
sha256sum * > SHA256SUMS
md5sum * > MD5SUMS
cd ..
echo -e "  ${GREEN}✓${NC} Checksums generated"

# Create release archives
echo -e "\n${YELLOW}→${NC} Creating release archives..."

for platform in "${!PLATFORMS[@]}"; do
    IFS=' ' read -r GOOS GOARCH <<< "${PLATFORMS[$platform]}"
    
    binary_name="${PROJECT_NAME}-${platform}"
    if [ "$GOOS" = "windows" ]; then
        binary_name="${binary_name}.exe"
    fi
    
    if [ ! -f "${DIST_DIR}/${binary_name}" ]; then
        continue
    fi
    
    archive_name="${PROJECT_NAME}-${VERSION}-${platform}"
    
    # Create temporary directory for archive
    temp_dir=$(mktemp -d)
    mkdir -p "${temp_dir}/${PROJECT_NAME}"
    
    # Copy files
    cp "${DIST_DIR}/${binary_name}" "${temp_dir}/${PROJECT_NAME}/${PROJECT_NAME}$([ "$GOOS" = "windows" ] && echo ".exe" || echo "")"
    cp README.md "${temp_dir}/${PROJECT_NAME}/" 2>/dev/null || true
    cp LICENSE "${temp_dir}/${PROJECT_NAME}/" 2>/dev/null || true
    
    # Create archive
    cd "${temp_dir}"
    if [ "$GOOS" = "windows" ]; then
        zip -r "${archive_name}.zip" ${PROJECT_NAME} > /dev/null
        mv "${archive_name}.zip" "${OLDPWD}/${RELEASE_DIR}/"
        echo -e "  ${GREEN}✓${NC} Created ${archive_name}.zip"
    else
        tar czf "${archive_name}.tar.gz" ${PROJECT_NAME}
        mv "${archive_name}.tar.gz" "${OLDPWD}/${RELEASE_DIR}/"
        echo -e "  ${GREEN}✓${NC} Created ${archive_name}.tar.gz"
    fi
    cd - > /dev/null
    
    rm -rf "${temp_dir}"
done

# Create Debian package
echo -e "\n${YELLOW}→${NC} Creating Debian package..."
if command -v dpkg-deb &> /dev/null; then
    DEB_DIR="${RELEASE_DIR}/deb"
    mkdir -p "${DEB_DIR}/DEBIAN"
    mkdir -p "${DEB_DIR}/usr/local/bin"
    mkdir -p "${DEB_DIR}/usr/share/doc/${PROJECT_NAME}"
    
    # Copy binary
    cp "${DIST_DIR}/${PROJECT_NAME}-linux-amd64" "${DEB_DIR}/usr/local/bin/${PROJECT_NAME}"
    chmod +x "${DEB_DIR}/usr/local/bin/${PROJECT_NAME}"
    
    # Create control file
    cat > "${DEB_DIR}/DEBIAN/control" << EOF
Package: ${PROJECT_NAME}
Version: ${VERSION#v}
Section: net
Priority: optional
Architecture: amd64
Maintainer: GoCat Team <ibrahimsql@proton.me>
Description: Modern netcat alternative written in Go
 GoCat is a modern, feature-rich netcat alternative with support for
 HTTP proxy, SSH tunneling, DNS tunneling, protocol conversion, and more.
Homepage: https://github.com/ibrahmsql/gocat
EOF
    
    # Copy docs
    cp README.md "${DEB_DIR}/usr/share/doc/${PROJECT_NAME}/" 2>/dev/null || true
    cp LICENSE "${DEB_DIR}/usr/share/doc/${PROJECT_NAME}/" 2>/dev/null || true
    
    # Build package
    dpkg-deb --build "${DEB_DIR}" "${RELEASE_DIR}/${PROJECT_NAME}_${VERSION#v}_amd64.deb" > /dev/null
    echo -e "  ${GREEN}✓${NC} Created ${PROJECT_NAME}_${VERSION#v}_amd64.deb"
    
    rm -rf "${DEB_DIR}"
else
    echo -e "  ${YELLOW}⚠${NC}  dpkg-deb not found, skipping Debian package"
fi

# Create RPM package (if rpmbuild available)
echo -e "\n${YELLOW}→${NC} Creating RPM package..."
if command -v rpmbuild &> /dev/null; then
    RPM_DIR="${RELEASE_DIR}/rpm"
    mkdir -p "${RPM_DIR}"/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
    
    # Create spec file
    cat > "${RPM_DIR}/SPECS/${PROJECT_NAME}.spec" << EOF
Name:           ${PROJECT_NAME}
Version:        ${VERSION#v}
Release:        1%{?dist}
Summary:        Modern netcat alternative written in Go

License:        MIT
URL:            https://github.com/ibrahmsql/gocat
Source0:        %{name}-%{version}.tar.gz

%description
GoCat is a modern, feature-rich netcat alternative with support for
HTTP proxy, SSH tunneling, DNS tunneling, protocol conversion, and more.

%prep

%build

%install
mkdir -p %{buildroot}/usr/local/bin
install -m 0755 ${PROJECT_NAME} %{buildroot}/usr/local/bin/${PROJECT_NAME}

%files
/usr/local/bin/${PROJECT_NAME}

%changelog
* $(date '+%a %b %d %Y') GoCat Team <ibrahimsql@proton.me> - ${VERSION#v}-1
- Release ${VERSION}
EOF
    
    # Copy binary to BUILD
    cp "${DIST_DIR}/${PROJECT_NAME}-linux-amd64" "${RPM_DIR}/BUILD/${PROJECT_NAME}"
    
    # Build RPM
    rpmbuild --define "_topdir ${PWD}/${RPM_DIR}" \
             --define "_rpmdir ${PWD}/${RELEASE_DIR}" \
             -bb "${RPM_DIR}/SPECS/${PROJECT_NAME}.spec" > /dev/null 2>&1 || true
    
    if ls ${RELEASE_DIR}/*/*.rpm 1> /dev/null 2>&1; then
        mv ${RELEASE_DIR}/*/*.rpm ${RELEASE_DIR}/
        echo -e "  ${GREEN}✓${NC} Created RPM package"
    else
        echo -e "  ${YELLOW}⚠${NC}  Failed to create RPM package"
    fi
    
    rm -rf "${RPM_DIR}"
else
    echo -e "  ${YELLOW}⚠${NC}  rpmbuild not found, skipping RPM package"
fi

# Generate release notes
echo -e "\n${YELLOW}→${NC} Generating release notes..."
cat > "${RELEASE_DIR}/RELEASE_NOTES.md" << EOF
# GoCat ${VERSION} Release Notes

## 🎉 What's New

### New Features
- ✅ HTTP Reverse Proxy with load balancing
- ✅ Multi-Port Listener
- ✅ Protocol Converter (TCP↔UDP, HTTP↔WebSocket)
- ✅ SSH Tunneling (local, remote, dynamic SOCKS)
- ✅ DNS Tunneling for covert channels

### Bug Fixes
- ✅ Fixed worker pool deadlock
- ✅ Fixed IPv6 address formatting
- ✅ Improved TLS security (minimum TLS 1.2)
- ✅ Fixed resource leaks
- ✅ Better error handling

### Improvements
- ✅ Enhanced statistics display
- ✅ Better logging with themes
- ✅ Improved documentation
- ✅ Code cleanup and optimization

## 📦 Installation

### Linux (Debian/Ubuntu)
\`\`\`bash
wget https://github.com/ibrahmsql/gocat/releases/download/${VERSION}/${PROJECT_NAME}_${VERSION#v}_amd64.deb
sudo dpkg -i ${PROJECT_NAME}_${VERSION#v}_amd64.deb
\`\`\`

### Linux (RPM)
\`\`\`bash
wget https://github.com/ibrahmsql/gocat/releases/download/${VERSION}/${PROJECT_NAME}-${VERSION#v}-1.x86_64.rpm
sudo rpm -i ${PROJECT_NAME}-${VERSION#v}-1.x86_64.rpm
\`\`\`

### macOS
\`\`\`bash
wget https://github.com/ibrahmsql/gocat/releases/download/${VERSION}/${PROJECT_NAME}-${VERSION}-darwin-amd64.tar.gz
tar xzf ${PROJECT_NAME}-${VERSION}-darwin-amd64.tar.gz
sudo mv ${PROJECT_NAME}/${PROJECT_NAME} /usr/local/bin/
\`\`\`

### Windows
Download \`${PROJECT_NAME}-${VERSION}-windows-amd64.zip\` and extract to your PATH.

## 📊 Checksums

See \`SHA256SUMS\` and \`MD5SUMS\` files for verification.

## 🔗 Links

- [Documentation](https://github.com/ibrahmsql/gocat)
- [Report Issues](https://github.com/ibrahmsql/gocat/issues)
- [Contribute](https://github.com/ibrahmsql/gocat/blob/main/CONTRIBUTING.md)

---

**Full Changelog**: https://github.com/ibrahmsql/gocat/compare/...${VERSION}
EOF

echo -e "  ${GREEN}✓${NC} Release notes generated"

# Summary
echo -e "\n${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║${NC}                  ${GREEN}Build Summary${NC}                           ${BLUE}║${NC}"
echo -e "${BLUE}╠════════════════════════════════════════════════════════════╣${NC}"
echo -e "${BLUE}║${NC}  Binaries:  ${GREEN}$(ls -1 ${DIST_DIR}/${PROJECT_NAME}-* 2>/dev/null | wc -l)${NC}"
echo -e "${BLUE}║${NC}  Archives:  ${GREEN}$(ls -1 ${RELEASE_DIR}/*.{tar.gz,zip} 2>/dev/null | wc -l)${NC}"
echo -e "${BLUE}║${NC}  Packages:  ${GREEN}$(ls -1 ${RELEASE_DIR}/*.{deb,rpm} 2>/dev/null | wc -l)${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"

echo -e "\n${GREEN}✓${NC} Release build complete!"
echo -e "${YELLOW}→${NC} Output directory: ${GREEN}${RELEASE_DIR}/${NC}"
echo -e "${YELLOW}→${NC} Binaries directory: ${GREEN}${DIST_DIR}/${NC}\n"
