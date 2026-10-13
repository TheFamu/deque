#!/bin/bash

set -euo pipefail

sudo -v

# Get latest release number
VERSION=$(curl -s https://api.github.com/repos/TheFamu/deque/releases/latest | grep '"tag_name":' | sed -E 's/.*"tag_name": "([^"]+)".*/\1/')

case "$(uname -s)" in
    Linux)  OS=linux ;;
    Darwin) OS=darwin ;;
    *) echo "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
    x86_64|amd64)  ARCH=amd64 ;;
    arm64|aarch64) ARCH=arm64 ;;
    *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

# Download tarball
curl -sLO "https://github.com/TheFamu/deque/releases/download/${VERSION}/deque_${VERSION}_${OS}_${ARCH}.tar.gz"

# Download checksums
curl -sLO "https://github.com/TheFamu/deque/releases/download/${VERSION}/checksums.txt"

# Check checksums
case "$(uname -s)" in
    Linux)  if ! sha256sum -c checksums.txt --ignore-missing &>/dev/null; then echo "Failed checksums!" >&2; fi ;;
    Darwin) if ! shasum -a 256 -c checksums.txt --ignore-missing &>/dev/null; then echo "Failed checksums!" >&2; fi ;;
    *) echo "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac

# Extract and install
tar -xzf "deque_${VERSION}_${OS}_${ARCH}.tar.gz"
sudo install -m 755 "deque_${VERSION}_${OS}_${ARCH}/deque" /usr/local/bin/deque

# Test it works
VER=$(/usr/local/bin/deque -version | awk '{print $2}')

if [[ $VERSION == $VER ]]; then
    echo "installed successfully!"
    /usr/local/bin/deque -version
else
    echo "Problem installing latest version!" >&2
fi

# Cleanup
rm -rf "deque_${VERSION}_${OS}_${ARCH}.tar.gz" "deque_${VERSION}_${OS}_${ARCH}" "checksums.txt"

