#!/bin/sh
set -e

REPO="benwilkes9/ralph-cli"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS
OS="$(uname -s)"
case "$OS" in
  Darwin) OS="darwin" ;;
  Linux)  OS="linux" ;;
  *)
    echo "Unsupported OS: $OS" >&2
    exit 1
    ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

# Determine archive format
EXT="tar.gz"
if [ "$OS" = "darwin" ]; then
  EXT="zip"
fi

# Get latest release tag
TAG="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"//;s/".*//')"
if [ -z "$TAG" ]; then
  echo "Failed to fetch latest release tag" >&2
  exit 1
fi

VERSION="${TAG#v}"
ARCHIVE="ralph-cli_${VERSION}_${OS}_${ARCH}.${EXT}"
URL="https://github.com/${REPO}/releases/download/${TAG}/${ARCHIVE}"

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading ralph ${TAG} for ${OS}/${ARCH}..."
curl -fsSL -o "${TMPDIR}/${ARCHIVE}" "$URL"

# Extract
if [ "$EXT" = "zip" ]; then
  unzip -q "${TMPDIR}/${ARCHIVE}" -d "$TMPDIR"
else
  tar -xzf "${TMPDIR}/${ARCHIVE}" -C "$TMPDIR"
fi

# Install
mkdir -p "$INSTALL_DIR"
cp "${TMPDIR}/ralph" "${INSTALL_DIR}/ralph"
chmod +x "${INSTALL_DIR}/ralph"

echo "Installed ralph ${TAG} to ${INSTALL_DIR}/ralph"
