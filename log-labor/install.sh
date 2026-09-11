#!/usr/bin/env bash
# install.sh — installer for the log-labor CLI.
#
#   git clone https://git.sh.nint.com/ying.yuxiang/AutoWecom-plugin
#   ./AutoWecom-plugin/log-labor/install.sh
#
# (The instance signs everyone in — REQUIRE_SIGNIN_VIEW — so anonymous
# `curl … | sh` against raw URLs gets a login page, not a script. If
# your Gitea ever allows anonymous raw, the piped one-liner works too.)
#
# Order: build from the checkout this script lives in (no network),
# else a Gitea release tarball (log-labor-vTAG), else a fresh shallow
# clone + source build. Never touches credentials — `log-labor init`
# does that.
set -euo pipefail

BASE_URL="${LOG_LABOR_BASE_URL:-https://git.sh.nint.com/ying.yuxiang/AutoWecom-plugin}"
REF="${LOG_LABOR_REF:-master}"
VERSION="${LOG_LABOR_VERSION:-v0.1.0}"
BIN_DIR="${LOG_LABOR_BIN_DIR:-$HOME/.local/bin}"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"          # darwin|linux
ARCH="$(uname -m)"; [ "$ARCH" = "arm64" ] || [ "$ARCH" = "aarch64" ] && ARCH=arm64 || ARCH=amd64
[ "$OS" = "darwin" ] || [ "$OS" = "linux" ] || { echo "install.sh: unsupported os $OS" >&2; exit 1; }

TAG="log-labor-${VERSION}"
TGZ="log-labor_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="$BASE_URL/releases/download/$TAG/$TGZ"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

echo "install.sh: looking for a source to build…"
SRC=""
HERE="$(cd "$(dirname "$0")" && pwd)"
if [ -f "$HERE/go.mod" ]; then
  echo "install.sh: building from the local checkout at $HERE"
  command -v go >/dev/null || { echo "install.sh: FATAL — go not found; install go or use a release tarball" >&2; exit 1; }
  ( cd "$HERE" && go build -o "$TMP/log-labor" ./cmd/log-labor )
  SRC="$TMP/log-labor"
fi
if [ -z "$SRC" ]; then
  echo "install.sh: no local checkout — trying release $TAG ($OS/$ARCH)…"
  if curl -fsSL -m 60 -o "$TMP/$TGZ" "$URL" && [ "$(head -c2 "$TMP/$TGZ" | xxd -p)" = "1f8b" ]; then
    tar -xzf "$TMP/$TGZ" -C "$TMP" && SRC="$TMP/log-labor"
  fi
fi
if [ -z "$SRC" ]; then
  echo "install.sh: no usable release tarball — trying a source build from ${REF}…"
  git clone --depth 1 --branch "$REF" "$BASE_URL.git" "$TMP/src" 2>/dev/null \
    || git clone --depth 1 --branch "$REF" "$BASE_URL" "$TMP/src" 2>/dev/null \
    || { echo "install.sh: FATAL — no release and no git access to $BASE_URL" >&2; exit 1; }
  command -v go >/dev/null || { echo "install.sh: FATAL — go not found; install go or ask for a release tarball" >&2; exit 1; }
  ( cd "$TMP/src/log-labor" && go build -o "$TMP/log-labor" ./cmd/log-labor )
  SRC="$TMP/log-labor"
fi

mkdir -p "$BIN_DIR"
if [ -w "$BIN_DIR" ]; then
  mv "$SRC" "$BIN_DIR/log-labor"
else
  echo "install.sh: $BIN_DIR not writable — using sudo"
  sudo mv "$SRC" "$BIN_DIR/log-labor"
fi
chmod +x "$BIN_DIR/log-labor"
echo "installed: $BIN_DIR/log-labor"

case ":$PATH:" in
  *":$BIN_DIR:"*) ;;
  *) echo "NOTE: $BIN_DIR is not on your PATH — add it (e.g. in ~/.zshrc: export PATH=\"$BIN_DIR:\$PATH\")" ;;
esac

cat <<'NEXT'

next steps:
  log-labor init            # webhook key + your corp id (zhang.san form)
  log-labor doctor          # verify
  log-labor skill install   # give your agents the skill
NEXT
