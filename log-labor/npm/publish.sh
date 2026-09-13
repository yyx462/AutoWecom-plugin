#!/usr/bin/env bash
# npm/publish.sh — local publish helper for the six log-labor packages.
# Rides your `npm login`; npm enforces 2FA per package, so expect one
# browser approval per package (up to 6 passkey taps). CI-side,
# release.yml prefers trusted publishing (OIDC, zero taps, zero tokens)
# once the workflow is registered on npmjs.com — this script is the
# bootstrap and the fallback. Platform packages publish first, wrapper
# LAST (avoids a window where the wrapper is live but its optional
# deps aren't).
#
#   Usage: npm/publish.sh [version]   # default: latest log-labor-v* tag
set -euo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"

command -v npm >/dev/null || { echo "npm/publish: FATAL — npm not found" >&2; exit 1; }
npm whoami >/dev/null 2>&1 || { echo "npm/publish: FATAL — not logged in; run: npm login" >&2; exit 1; }

VERSION="${1:-}"
if [ -z "$VERSION" ]; then
  VERSION="$(git -C "$HERE/../.." describe --tags --match 'log-labor-v*' --abbrev=0 2>/dev/null | sed 's/^log-labor-//')" || true
fi
[ -n "$VERSION" ] || { echo "npm/publish: FATAL — no version arg and no log-labor-v* tag reachable" >&2; exit 1; }

"$HERE/build.sh" "$VERSION"
cd "$HERE/../dist/npm/platforms"
for pkg in *; do
  echo "npm/publish: @log-labor/$pkg"
  ( cd "$pkg" && npm publish --access public )
done
echo "npm/publish: log-labor (wrapper — last by design)"
( cd ../wrapper && npm publish --access public )
echo "npm/publish: done — https://www.npmjs.com/package/log-labor"
