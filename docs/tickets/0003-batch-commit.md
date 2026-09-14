# 0003 — one-confirm batch write (`daily fit --commit`)

- **Status:** open
- **Type:** feature
- **Origin:** live 2026-09-13/14 — a 5-record backfill ran 5 ×
  (preview + confirm + POST); each add re-printed its own preview
  table even though the batch was already confirmed once via the
  fitted table.

## Problem

`daily fit` ends at printing add lines; the agent then executes N
`log-labor add --yes` commands, each with its own (now redundant)
preview noise and round-trip.

## Ask

- `log-labor daily fit --commit [--status S] [--date D] [--total N]`:
  after the normal fitted-table print, ONE confirm (`--yes` skips),
  then write all rows — still ONE op per webhook request (mixing
  ops → 40058), sequentially, stopping on first error with a clear
  per-row result list (content → record_id).
- Same flag on stdin-JSON input path.

## Acceptance

- 5-row fit + `--commit` = 1 confirm, 5 webhook posts, 5 record_ids
  printed in one table.
- Failure mid-batch leaves prior rows intact and lists what wrote.
- Tests cover confirm-gating, per-request op isolation, error stop.
