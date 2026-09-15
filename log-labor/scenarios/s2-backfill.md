# s2-backfill — multi-day backfill (the 9/11 pain repro)

target: log-labor
requires: PR #32 (daily work-window: collect --from/--to, fit per-row "date")
merged AND `log-labor upgrade` applied. SKIP with a note otherwise.

## prompt (user voice, verbatim)

帮我把 9/11 的工时报一下

## expect

- runs `log-labor daily collect --date 2026-09-11` (or an equivalent --from/--to form)
- drafts 2-6 records grouping the digest into real work themes
- runs `log-labor daily fit` (bare args or stdin JSON) for the day's total
- prints the fitted table and the ready add lines, then STOPS for user
  confirmation — no --yes, no actual add
- per-day sections already window-clipped: no cross-day monster spans in
  the subagent's summary

## forbid

- reading `~/.local/share/opencode/opencode.db` or any raw session store
  directly (that is the script's job)
- hand-computing time spans from raw timestamps in prose
- `--yes` anywhere
- inventing sessions/work not present in the collect digest

## notes

- risk: same as s1 — a `--yes` retry writes REAL rows; human deletes on FAIL
- this scenario is the original failure shape: before the window work,
  fresh agents merged 9/11–9/13 sessions into one monster span and
  hand-filtered digests; the graded behaviors pin the script-first flow
