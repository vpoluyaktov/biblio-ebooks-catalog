#!/bin/bash

# Build script for biblio-catalog
# Builds with ICU support for proper Unicode case conversion in SQLite

set -e

echo "🔨 Building biblio-catalog..."

# Get version information
VERSION=$(cat VERSION 2>/dev/null || echo "0.1.0")
BUILD_DATE=$(date -u '+%Y-%m-%d_%H:%M:%S_UTC')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
if [ "$ARCH" = "x86_64" ]; then
    GOARCH="amd64"
elif [ "$ARCH" = "arm64" ] || [ "$ARCH" = "aarch64" ]; then
    GOARCH="arm64"
else
    GOARCH="amd64"
fi

echo "  OS: $OS"
echo "  Architecture: $GOARCH"
echo "  Version: $VERSION"
echo "  Commit: $GIT_COMMIT"

# Build flags
LDFLAGS="-X main.Version=$VERSION -X main.BuildDate=$BUILD_DATE -X main.GitCommit=$GIT_COMMIT"

# Build with ICU support for Unicode case conversion
# Requires: libicu-dev (apt) or icu4c (brew)
echo "  Building with ICU support..."
CGO_ENABLED=1 go build -tags "icu" -ldflags "$LDFLAGS" -o biblio-catalog .

echo "✅ Build complete: biblio-catalog"
echo ""
echo "Note: ICU support requires libicu libraries at runtime."
echo "  - Linux: apt install libicu-dev"
echo "  - macOS: brew install icu4c"
