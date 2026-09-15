#!/usr/bin/env bash
# reset.sh — per-target state snapshot/restore for yyx-skill-test runs.
#
# Convention: lives at <target-skill-source>/skilltest/reset.sh; the target
# name is the skill dir name (override: YYX_SKILL_TEST_TARGET). State lives
# under ${YYX_SKILL_TEST_HOME:-~/.yyxSkill}/state/<target>/ — OUT of band
# from the live config paths so tested subagents never find the backups.
#
# Usage:
#   reset.sh snapshot   cp -Rp every SCOPE path into state/<target>/latest
#   reset.sh restore    wipe live SCOPE paths, copy back from latest, perms
#   reset.sh verify     masked diff live vs latest (never prints secrets)
#   reset.sh arm        write state/armed.json + snapshot (+ marker for guard)
#   reset.sh disarm     remove state/armed.json
#
# SCOPE: every path the target's scenarios may mutate. For log-labor:
# the whole ~/.config/log-labor dir (config.json incl. key/person/sheet,
# sources registry, lastcheck) PLUS the agent skill dirs that
# `log-labor skill install` can re-render (the 2026-09-14 s4 incident:
# a --help probe executed a real install and clobbered SKILL.md x4).
# Scenario forbids on `skill install` remain as belt-and-braces.
set -euo pipefail

SKILL_DIR="$(cd "$(dirname "$0")/.." && pwd)"
TARGET="${YYX_SKILL_TEST_TARGET:-$(basename "$SKILL_DIR")}"
STATE_ROOT="${YYX_SKILL_TEST_HOME:-$HOME/.yyxSkill}/state/$TARGET"
LATEST="$STATE_ROOT/latest"
ARMED="$STATE_ROOT/../armed.json"
MARKER="$STATE_ROOT/last-run.snapped"

# per-target scope: every mutable path a scenario may touch
SCOPE=("$HOME/.config/log-labor"
       "$HOME/.config/opencode/skills/log-labor"
       "$HOME/.claude/skills/log-labor"
       "$HOME/.agents/skills/log-labor"
       "$HOME/.codex/skills/log-labor")

mask_summary() { # masked view of a config.json — person/sheet only, never key
  local f="$1" label="$2"
  if [ -f "$f" ]; then
    python3 - "$f" "$label" <<'PY'
import json, sys
try:
    d = json.load(open(sys.argv[1]))
    print(f"{sys.argv[2]}: person={d.get('person','?')} sheet={d.get('sheet','?')} "
          f"keys={sorted(k for k in d if k != 'key')}")
except Exception as e:
    print(f"{sys.argv[2]}: <unreadable: {e}>")
PY
  else
    echo "$label: <absent>"
  fi
}

cmd="${1:-}"
case "$cmd" in
snapshot)
  rm -rf "$LATEST"
  for src in "${SCOPE[@]}"; do
    [ -e "$src" ] || { echo "snapshot: skip (absent) $src"; continue; }
    rel="${src#/}"
    mkdir -p "$LATEST/$(dirname "$rel")"
    cp -Rp "$src" "$LATEST/$rel"
    echo "snapshot: $src -> $LATEST/$rel"
  done
  python3 -c "import json,time;json.dump({'target':'$TARGET','ts':int(time.time()*1000)},open('$LATEST/.meta','w'))"
  chmod -R go-rwx "$LATEST"
  echo "snapshot: done ($TARGET)"
  ;;
restore)
  [ -d "$LATEST" ] || { echo "restore: no snapshot at $LATEST" >&2; exit 1; }
  for src in "${SCOPE[@]}"; do
    rel="${src#/}"
    if [ -e "$LATEST/$rel" ]; then
      rm -rf "$src"
      mkdir -p "$(dirname "$src")"
      cp -Rp "$LATEST/$rel" "$src"
      echo "restore: $src"
    else
      echo "restore: no snapshot for $src (left live state untouched)"
    fi
  done
  # per-target perm + health re-check
  [ -f "$HOME/.config/log-labor/config.json" ] && chmod 600 "$HOME/.config/log-labor/config.json"
  command -v log-labor >/dev/null && log-labor doctor >/dev/null 2>&1 \
    && echo "restore: log-labor doctor ok" || echo "restore: doctor skipped/failed"
  mask_summary "$HOME/.config/log-labor/config.json" "live"
  ;;
verify)
  echo "== masked state ($TARGET) =="
  for src in "${SCOPE[@]}"; do
    rel="${src#/}"
    mask_summary "$src/config.json" "live  "
    mask_summary "$LATEST/$rel/config.json" "latest"
    diff -rq "$src" "$LATEST/$rel" >/dev/null 2>&1 \
      && echo "match : $src" || echo "DIFFER: $src"
  done
  ;;
arm)
  mkdir -p "$STATE_ROOT"
  python3 -c "import json,time;json.dump({'target':'$TARGET','reset_script':'$0','run_id':'$(date +%Y%m%d-%H%M%S)-$$','ts':int(time.time()*1000)},open('$ARMED','w'))"
  chmod 600 "$ARMED"
  "$0" snapshot
  : > "$MARKER"
  echo "arm: $ARMED"
  ;;
disarm)
  rm -f "$ARMED" "$MARKER"
  echo "disarm: done"
  ;;
*)
  sed -n '2,20p' "$0" | sed 's/^# \{0,1\}//'
  exit 2
  ;;
esac
