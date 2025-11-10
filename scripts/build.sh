#!/bin/bash
# Cross-platform build script for Scanner Service
# Builds binaries for Windows, Linux, and macOS

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Scanner Service - Cross-Platform Build${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# Configuration
APP_NAME="scanserver"
VERSION="1.0.0"
BUILD_DIR="build"
CMD_PATH="cmd/scanserver"

# Clean build directory
echo -e "${YELLOW}Cleaning build directory...${NC}"
rm -rf $BUILD_DIR
mkdir -p $BUILD_DIR

# Build info
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# LD FLAGS for version info
LDFLAGS="-s -w"
LDFLAGS="$LDFLAGS -X main.Version=$VERSION"
LDFLAGS="$LDFLAGS -X main.BuildTime=$BUILD_TIME"
LDFLAGS="$LDFLAGS -X main.GitCommit=$GIT_COMMIT"

# Build function
build() {
    local OS=$1
    local ARCH=$2
    local OUTPUT_NAME="${APP_NAME}"

    if [ "$OS" = "windows" ]; then
        OUTPUT_NAME="${APP_NAME}.exe"
    fi

    local OUTPUT_PATH="${BUILD_DIR}/${APP_NAME}-${OS}-${ARCH}/${OUTPUT_NAME}"

    echo -e "${YELLOW}Building for ${OS}/${ARCH}...${NC}"

    mkdir -p "$(dirname $OUTPUT_PATH)"

    CGO_ENABLED=0 GOOS=$OS GOARCH=$ARCH go build \
        -ldflags "$LDFLAGS" \
        -o "$OUTPUT_PATH" \
        "./${CMD_PATH}"

    if [ $? -eq 0 ]; then
        local SIZE=$(du -h "$OUTPUT_PATH" | cut -f1)
        echo -e "${GREEN}✓ Built ${OS}/${ARCH} successfully (${SIZE})${NC}"

        # Copy additional files
        cp -r web "${BUILD_DIR}/${APP_NAME}-${OS}-${ARCH}/"
        cp config.example.yaml "${BUILD_DIR}/${APP_NAME}-${OS}-${ARCH}/" 2>/dev/null || true
        cp README.md "${BUILD_DIR}/${APP_NAME}-${OS}-${ARCH}/" 2>/dev/null || true

        # Create archive
        cd $BUILD_DIR
        if [ "$OS" = "windows" ]; then
            zip -r "${APP_NAME}-${OS}-${ARCH}-v${VERSION}.zip" "${APP_NAME}-${OS}-${ARCH}" > /dev/null
            echo -e "${GREEN}✓ Created ${APP_NAME}-${OS}-${ARCH}-v${VERSION}.zip${NC}"
        else
            tar -czf "${APP_NAME}-${OS}-${ARCH}-v${VERSION}.tar.gz" "${APP_NAME}-${OS}-${ARCH}"
            echo -e "${GREEN}✓ Created ${APP_NAME}-${OS}-${ARCH}-v${VERSION}.tar.gz${NC}"
        fi
        cd ..
    else
        echo -e "${RED}✗ Failed to build ${OS}/${ARCH}${NC}"
        return 1
    fi
}

# Build for all platforms
echo ""
echo -e "${YELLOW}Starting builds...${NC}"
echo ""

# Windows
build windows amd64
build windows arm64

# Linux
build linux amd64
build linux arm64
build linux arm

# macOS
build darwin amd64
build darwin arm64

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Build completed successfully!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Binaries location: ${BUILD_DIR}/"
echo ""
ls -lh ${BUILD_DIR}/*.{zip,tar.gz} 2>/dev/null || true
