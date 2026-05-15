#!/bin/bash
set -e

REPO="pravnyadv/cpssh"
INSTALL_DIR="/usr/local/bin"
VERSION="${1:-latest}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case $ARCH in
  x86_64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

if [ "$VERSION" = "latest" ]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
fi

if [ -z "$VERSION" ]; then
  echo "Could not determine latest version. Check your internet connection."
  exit 1
fi

FILENAME="cpssh_${OS}_${ARCH}"
URL="https://github.com/$REPO/releases/download/$VERSION/$FILENAME"

echo "Downloading cpssh $VERSION ($OS/$ARCH)..."
curl -fsSL "$URL" -o /tmp/cpssh
chmod +x /tmp/cpssh

if [ -w "$INSTALL_DIR" ]; then
  mv /tmp/cpssh "$INSTALL_DIR/cpssh"
else
  sudo mv /tmp/cpssh "$INSTALL_DIR/cpssh"
fi

# Remove Gatekeeper quarantine on macOS
if [ "$OS" = "darwin" ]; then
  xattr -dr com.apple.quarantine "$INSTALL_DIR/cpssh" 2>/dev/null || true
fi

echo ""
echo "Installed cpssh to $INSTALL_DIR/cpssh"
echo ""
echo "Run: cpssh setup"
