#!/usr/bin/env bash
# Install the plugin's own skills globally for agents on this host
# (symlink so a git pull updates them everywhere).
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
dest="${1:-$HOME/.agents/skills}"
mkdir -p "$dest"
for name in sop reporting; do
  d="$here/$name"
  [ -d "$d" ] || continue
  ln -sfn "$d" "$dest/$name"
  echo "linked $dest/$name"
done
echo "remember: agents need WECOMBOT_API_URL and WECOMBOT_API_TOKEN"
