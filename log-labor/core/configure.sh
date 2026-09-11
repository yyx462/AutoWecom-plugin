#!/usr/bin/env bash
# core/configure.sh — the plugin.json `entry` for AutoWecom-core's deploy.
# Verb protocol (api 1): env | plan | install | health — all idempotent,
# all loud. Core's deploy.sh walks PLUGIN_DIR/*/ and runs these verbs;
# nothing here may hardcode host paths or print secrets.
#
# Environment (from core's deploy env):
#   LABOR_WEBHOOK_KEY    the sheet webhook key — REQUIRED for install
#   LABOR_PERSON_DEFAULT optional default corp userid (only informational
#                        here; the key itself lives in the broker env)
#   PLUGIN_REF           pinned ref of this repo (default master)
#   PLUGIN_STAMP_DIR     where stamps live (core passes state/plugins)
set -euo pipefail
here="$(cd "$(dirname "$0")/.." && pwd)"   # log-labor/
verb="${1:-plan}"
REF="${PLUGIN_REF:-master}"
STAMP_DIR="${PLUGIN_STAMP_DIR:-$here/dist}"
STAMP="$STAMP_DIR/log-labor.stamp"

artifact_hash() {
  ( cd "$here" && cat plugin.json core/configure.sh internal/skill/SKILL.md.tmpl skill/SKILL.md 2>/dev/null | shasum -a 256 | cut -d' ' -f1 )
}

case "$verb" in
  env)
    echo "LABOR_WEBHOOK_KEY"
    ;;
  plan)
    if [ -s "$STAMP" ] && [ "$(cat "$STAMP")" = "${REF}:$(artifact_hash)" ]; then
      echo "log-labor: up to date ($REF)"
    else
      echo "log-labor: install/refresh needed → skill mount ok, broker handler expects LABOR_WEBHOOK_KEY"
    fi
    ;;
  install)
    [ -n "${LABOR_WEBHOOK_KEY:-}" ] || {
      echo "log-labor: FATAL — LABOR_WEBHOOK_KEY missing in deploy env (deps/env)" >&2; exit 1; }
    [ -s "$here/skill/SKILL.md" ] || {
      echo "log-labor: FATAL — rendered skill missing (run: go generate ./internal/skill/)" >&2; exit 1; }
    case "${LABOR_WEBHOOK_KEY}" in
      http*) echo "log-labor: FATAL — LABOR_WEBHOOK_KEY looks like a URL; put the bare key value" >&2; exit 1 ;;
    esac
    if [ -s "$STAMP" ] && [ "$(cat "$STAMP")" = "${REF}:$(artifact_hash)" ]; then
      echo "log-labor: up to date ($REF) — skipped"
      exit 0
    fi
    mkdir -p "$STAMP_DIR"
    printf '%s:%s\n' "$REF" "$(artifact_hash)" > "$STAMP"
    echo "log-labor: installed (stamp $STAMP) — key passed through to broker env, never written here"
    ;;
  health)
    [ -s "$STAMP" ] || { echo "log-labor: not installed (no stamp)" >&2; exit 1; }
    [ "$(cat "$STAMP")" = "${REF}:$(artifact_hash)" ] || {
      echo "log-labor: stamp drift — run install" >&2; exit 1; }
    if [ -n "${LABOR_BROKER_URL:-}" ]; then
      curl -fsS -m 5 "${LABOR_BROKER_URL%/}/healthz" >/dev/null 2>&1 \
        && echo "log-labor: healthy (broker reachable)" \
        || { echo "log-labor: WARN broker unreachable" >&2; exit 1; }
    else
      echo "log-labor: healthy (stamp ok; no LABOR_BROKER_URL to probe)"
    fi
    ;;
  *)
    echo "core/configure.sh: unknown verb '$verb' (env|plan|install|health)" >&2
    exit 2
    ;;
esac
