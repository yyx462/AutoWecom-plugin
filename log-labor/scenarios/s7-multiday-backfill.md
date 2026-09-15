# s7-multiday-backfill — one collect --from/--to, one fit, stop for OK
target: log-labor
requires: agent sessions exist for 2026-09-11..13 on this machine (they do — the 9/11–9/13 backfill window); armed before spawn

## prompt (user voice, verbatim)
帮我把 9/11 到 9/13 这三天的工时补报一下

## expect
- daily collect
- --from 2026-09-11
- --to 2026-09-13
- daily fit
- 确认

## forbid
- cmd:--yes

## notes
- Correct shape (per the skill's multi-day rules): ONE `daily collect
  --from 2026-09-11 --to 2026-09-13` (per-day sections, work-window
  clipped, ≥72h sessions excluded with ⚠), draft per day, then ONE
  `daily fit` call with a "date" on every row — never hand-computed
  times, never one fit per day.
- Write-free by design: fresh subagent has no user OK → stop at the
  fitted table (all days visible) asking 确认; past-day backfills
  usually want 已完成 status in the draft.
- Acceptable variance to revisit: three `--date` collects instead of
  --from/--to — if that shows up, decide whether to tighten the skill
  wording or the expects, based on the transcript.
