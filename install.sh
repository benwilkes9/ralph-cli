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

# Get latest release tag — prefer jq for safe JSON parsing, fall back to sed
RELEASE_JSON="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest")"
if command -v jq >/dev/null 2>&1; then
  TAG="$(printf '%s' "$RELEASE_JSON" | jq -r '.tag_name')"
else
  TAG="$(printf '%s' "$RELEASE_JSON" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1)"
fi

# Validate tag format
if [ -z "$TAG" ] || [ "$TAG" = "null" ]; then
  echo "Failed to fetch latest release tag" >&2
  exit 1
fi
case "$TAG" in
  v[0-9]*) ;; # valid
  *)
    echo "Unexpected release tag format: $TAG" >&2
    exit 1
    ;;
esac

VERSION="${TAG#v}"
ARCHIVE="ralph-cli_${VERSION}_${OS}_${ARCH}.${EXT}"
CHECKSUMS="ralph-cli_${VERSION}_checksums.txt"
BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading ralph ${TAG} for ${OS}/${ARCH}..."
curl -fsSL -o "${TMPDIR}/${ARCHIVE}" "${BASE_URL}/${ARCHIVE}"

# Verify checksum if checksums file is published
if curl -fsSL -o "${TMPDIR}/${CHECKSUMS}" "${BASE_URL}/${CHECKSUMS}" 2>/dev/null; then
  EXPECTED="$(grep "${ARCHIVE}" "${TMPDIR}/${CHECKSUMS}" | awk '{print $1}')"
  if [ -n "$EXPECTED" ]; then
    if command -v sha256sum >/dev/null 2>&1; then
      ACTUAL="$(sha256sum "${TMPDIR}/${ARCHIVE}" | awk '{print $1}')"
    elif command -v shasum >/dev/null 2>&1; then
      ACTUAL="$(shasum -a 256 "${TMPDIR}/${ARCHIVE}" | awk '{print $1}')"
    else
      echo "Warning: no sha256sum or shasum found, skipping checksum verification" >&2
      ACTUAL=""
    fi
    if [ -n "$ACTUAL" ] && [ "$ACTUAL" != "$EXPECTED" ]; then
      echo "Checksum mismatch!" >&2
      echo "  expected: $EXPECTED" >&2
      echo "  actual:   $ACTUAL" >&2
      exit 1
    fi
    [ -n "$ACTUAL" ] && echo "Checksum verified."
  fi
fi

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
