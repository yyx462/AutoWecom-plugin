#!/usr/bin/env bash
# npm/build.sh — build all release artifacts and assemble the npm
# packages (esbuild/@openai/codex pattern: wrapper + one optional
# platform package per target).
#
#   Usage: npm/build.sh <version>     (v-prefix optional, e.g. v0.1.1)
#
# Outputs under dist/:
#   log-labor_<version>_<os>_<arch>.tar.gz   ×5 — GitHub Release assets
#   npm/wrapper/                   wrapper package (launcher + package.json)
#   npm/platforms/<npm-os>-<npm-cpu>/        @log-labor/* platform packages
#
# Name contract: the tarball names must byte-match what install.sh and
# install.ps1 compute; the npm platform packages live under the
# @log-labor org scope (@esbuild / @openai convention — unscoped
# hyphenated names trip npm's spam detection, see log-labor-win32-x64
# 2026-09-13). npm version = <version> minus the leading v.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
NPM_SRC="$ROOT/npm"
cd "$ROOT"

VERSION="${1:?usage: npm/build.sh <version> (e.g. v0.1.1)}"
case "$VERSION" in v*) ;; *) VERSION="v$VERSION" ;; esac
NPMV="${VERSION#v}"
SCOPE="@log-labor"
LDFLAGS="-X git.sh.nint.com/ying.yuxiang/AutoWecom-plugin/log-labor/internal/config.Version=${VERSION}"

rm -rf dist
mkdir -p dist/stage dist/npm/platforms

for target in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64; do
  os="${target%/*}"; arch="${target#*/}"
  exe="log-labor"; [ "$os" = "windows" ] && exe="log-labor.exe"
  echo "npm/build: ${os}/${arch}"
  GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 go build -trimpath -ldflags "$LDFLAGS" -o "dist/stage/$exe" ./cmd/log-labor
  tar -czf "dist/log-labor_${VERSION}_${os}_${arch}.tar.gz" -C dist/stage "$exe"

  # npm's spelling: GOOS windows→win32, GOARCH amd64→x64 (os/cpu fields
  # must match process.platform / process.arch exactly).
  nos="$os";  [ "$os"   = "windows" ] && nos="win32"
  narch="$arch"; [ "$arch" = "amd64" ] && narch="x64"
  pkg="dist/npm/platforms/${nos}-${narch}"
  mkdir -p "$pkg"
  cp "dist/stage/$exe" "$pkg/$exe"
  cat > "$pkg/package.json" <<EOF
{
  "name": "${SCOPE}/${nos}-${narch}",
  "version": "${NPMV}",
  "description": "log-labor binary for ${os}/${arch} — installed via the log-labor wrapper's optionalDependencies",
  "os": ["${nos}"],
  "cpu": ["${narch}"],
  "files": ["${exe}"],
  "publishConfig": { "access": "public" }
}
EOF
  rm "dist/stage/$exe"
done

pkg="dist/npm/wrapper"
mkdir -p "$pkg/bin"
cp "$NPM_SRC/wrapper/bin/log-labor.js" "$pkg/bin/log-labor.js"
cp "$NPM_SRC/wrapper/README.md" "$pkg/README.md"
sed -e "s/@VERSION@/${NPMV}/g" "$NPM_SRC/wrapper/package.json.tmpl" > "$pkg/package.json"

echo "npm/build: tarballs + packages ready:"
ls dist/*.tar.gz
ls dist/npm/platforms
ls dist/npm/wrapper
