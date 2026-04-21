#!/bin/sh
# sunbeams installer
#
# Usage:
#   curl -sSL https://asdfgasfhsn.github.io/sunbeams/install.sh | sh
#
# Environment variables:
#   SUNBEAMS_VERSION   — release tag to install (default: latest)
#                        e.g. SUNBEAMS_VERSION=v0.1.0
#   SUNBEAMS_PREFIX    — installation prefix (default: /usr/local)
#                        binary goes to $SUNBEAMS_PREFIX/bin/sunbeams

set -eu

SUNBEAMS_VERSION="${SUNBEAMS_VERSION:-latest}"
SUNBEAMS_PREFIX="${SUNBEAMS_PREFIX:-/usr/local}"

REPO="asdfgasfhsn/sunbeams"
BINARY_NAME="sunbeams"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

say() { printf '%s\n' "$*"; }
err() { printf 'error: %s\n' "$*" >&2; exit 1; }

# ---------------------------------------------------------------------------
# OS check
# ---------------------------------------------------------------------------

OS="$(uname -s)"
if [ "$OS" != "Linux" ]; then
  err "sunbeams targets Linux/KDE (detected: $OS). Install is not supported on this platform."
fi

# ---------------------------------------------------------------------------
# Arch detection
# ---------------------------------------------------------------------------

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)  GOARCH="amd64" ;;
  aarch64) GOARCH="arm64" ;;
  *)       err "Unsupported architecture: $ARCH. sunbeams releases only provide linux/amd64 and linux/arm64." ;;
esac

# ---------------------------------------------------------------------------
# Version resolution
# ---------------------------------------------------------------------------

if [ "$SUNBEAMS_VERSION" = "latest" ]; then
  say "Resolving latest release..."
  RESOLVED_URL="$(curl -sLI -o /dev/null -w '%{url_effective}' \
    "https://github.com/${REPO}/releases/latest")"
  TAG="$(printf '%s' "$RESOLVED_URL" | sed 's|.*/tag/||')"
  if [ -z "$TAG" ]; then
    err "Could not determine latest release tag from: $RESOLVED_URL"
  fi
else
  TAG="$SUNBEAMS_VERSION"
fi

# Goreleaser archive filenames use the bare version without a leading 'v'
VERSION="$(printf '%s' "$TAG" | sed 's/^v//')"

say "Installing sunbeams $TAG (linux/$GOARCH)..."

# ---------------------------------------------------------------------------
# Download URLs
# ---------------------------------------------------------------------------

ARCHIVE="sunbeams_${VERSION}_linux_${GOARCH}.tar.gz"
BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"
ARCHIVE_URL="${BASE_URL}/${ARCHIVE}"
CHECKSUMS_URL="${BASE_URL}/checksums.txt"

# ---------------------------------------------------------------------------
# Temp dir with cleanup trap
# ---------------------------------------------------------------------------

TMPDIR="$(mktemp -d)"
# shellcheck disable=SC2064  # We want $TMPDIR to expand now, not at trap time
trap "rm -rf '$TMPDIR'" EXIT

# ---------------------------------------------------------------------------
# Download
# ---------------------------------------------------------------------------

say "Downloading $ARCHIVE..."
curl -sSL -o "${TMPDIR}/${ARCHIVE}" "$ARCHIVE_URL"

say "Downloading checksums..."
curl -sSL -o "${TMPDIR}/checksums.txt" "$CHECKSUMS_URL"

# ---------------------------------------------------------------------------
# Checksum verification
# ---------------------------------------------------------------------------

if command -v sha256sum >/dev/null 2>&1; then
  SHASUM_CMD="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
  SHASUM_CMD="shasum -a 256"
else
  err "Cannot verify download: install coreutils (sha256sum) or shasum to verify the download."
fi

say "Verifying checksum..."
EXPECTED="$(grep " ${ARCHIVE}$" "${TMPDIR}/checksums.txt" | awk '{print $1}')"
if [ -z "$EXPECTED" ]; then
  err "Checksum entry for '${ARCHIVE}' not found in checksums.txt"
fi

ACTUAL="$(${SHASUM_CMD} "${TMPDIR}/${ARCHIVE}" | awk '{print $1}')"
if [ "$ACTUAL" != "$EXPECTED" ]; then
  err "Checksum mismatch for ${ARCHIVE}
  expected: $EXPECTED
  actual:   $ACTUAL"
fi

say "Checksum OK."

# ---------------------------------------------------------------------------
# Extract
# ---------------------------------------------------------------------------

tar -C "$TMPDIR" -xzf "${TMPDIR}/${ARCHIVE}"

BINARY_PATH="${TMPDIR}/${BINARY_NAME}"
if [ ! -f "$BINARY_PATH" ]; then
  # Goreleaser may nest the binary inside a subdirectory
  BINARY_PATH="$(find "$TMPDIR" -maxdepth 2 -name "$BINARY_NAME" -type f | head -1)"
fi

if [ -z "$BINARY_PATH" ] || [ ! -f "$BINARY_PATH" ]; then
  err "Could not find '${BINARY_NAME}' binary after extracting ${ARCHIVE}"
fi

# ---------------------------------------------------------------------------
# Install binary
# ---------------------------------------------------------------------------

INSTALL_DIR="${SUNBEAMS_PREFIX}/bin"
INSTALL_PATH="${INSTALL_DIR}/${BINARY_NAME}"

do_install() {
  install -d "$INSTALL_DIR"
  install -m 755 "$BINARY_PATH" "$INSTALL_PATH"
}

if [ -w "$INSTALL_DIR" ] || { [ ! -e "$INSTALL_DIR" ] && [ -w "$(dirname "$INSTALL_DIR")" ]; }; then
  do_install
else
  if command -v sudo >/dev/null 2>&1; then
    say "Requiring sudo to write to $INSTALL_DIR..."
    sudo install -d "$INSTALL_DIR"
    sudo install -m 755 "$BINARY_PATH" "$INSTALL_PATH"
  else
    err "Cannot write to $INSTALL_DIR and sudo is not available.
Re-run with a writable prefix:
  SUNBEAMS_PREFIX=\$HOME/.local $0"
  fi
fi

# ---------------------------------------------------------------------------
# Done
# ---------------------------------------------------------------------------

say ""
say "sunbeams $TAG installed to $INSTALL_PATH"
say ""
say "Next step — run the guided Bazzite installer:"
say "  sudo sunbeams install"
