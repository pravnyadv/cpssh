#!/bin/sh
set -e

REPO="pravnyadv/cpssh"
BIN="cpssh"

# Pick a user-writable bin dir — no sudo needed.
# /opt/homebrew/bin  → Apple Silicon Macs (Homebrew default)
# /usr/local/bin     → Intel Macs with Homebrew (Homebrew makes it user-writable)
# ~/.local/bin       → fallback (Linux, or macOS without Homebrew)
if [ -d "/opt/homebrew/bin" ]; then
  INSTALL_DIR="/opt/homebrew/bin"
elif [ -w "/usr/local/bin" ]; then
  INSTALL_DIR="/usr/local/bin"
else
  INSTALL_DIR="$HOME/.local/bin"
  mkdir -p "$INSTALL_DIR"
fi

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)        ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# macOS: ensure pngpaste is available (used for clipboard image reading)
if [ "$OS" = "darwin" ] && ! command -v pngpaste >/dev/null 2>&1; then
  if ! command -v brew >/dev/null 2>&1; then
    echo "Homebrew is required to install pngpaste. Install it from https://brew.sh" >&2
    exit 1
  fi
  echo "Installing pngpaste..."
  brew install pngpaste
fi

# Fetch latest release version from GitHub
VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
  | grep '"tag_name"' | sed -E 's/.*"v?([^"]+)".*/\1/')

if [ -z "$VERSION" ]; then
  echo "Could not determine latest version. Check https://github.com/$REPO/releases" >&2
  exit 1
fi

URL="https://github.com/$REPO/releases/download/v${VERSION}/${BIN}_${OS}_${ARCH}.tar.gz"

echo "Installing $BIN v$VERSION ($OS/$ARCH) to $INSTALL_DIR..."

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

curl -fsSL "$URL" | tar -xz -C "$TMP"

if [ ! -f "$TMP/$BIN" ]; then
  echo "Binary not found in archive" >&2
  exit 1
fi

install -m 755 "$TMP/$BIN" "$INSTALL_DIR/$BIN"

echo ""
echo "$BIN installed to $INSTALL_DIR/$BIN"

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *) echo "Note: $INSTALL_DIR is not in your PATH. Add this to your shell config:" \
     && echo "  export PATH=\"$INSTALL_DIR:\$PATH\"" ;;
esac

echo "Run: $BIN setup"
