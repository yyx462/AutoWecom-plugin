# 0002 — local write journal of record_ids

- **Status:** open
- **Type:** feature
- **Origin:** live 2026-09-13/14 — correcting an older row means
  scrolling the agent transcript for its `record_id` (the webhook is
  write-only: no read), so anything not visible anymore is edited by
  hand in the sheet UI.

## Problem

`add`/`update` print `record_id` once; nothing durable keeps it. Hours
corrections (`update --record-id R -h <h>`) only work while R is still
in view.

## Ask

- Append-only journal at `~/.config/log-labor/journal.jsonl`: one line
  per successful add/update — timestamp, date-field, content, hours,
  status, record_id.
- `log-labor ids [--date YYYY-MM-DD]` — list this machine's known
  rows (content + record_id) for a day.
- `log-labor update --last [--date D]` sugar for the newest row.

## Acceptance

- Journal write fails NEVER block the sheet write (best-effort, warn).
- `ids --date` round-trips every row added that day from this machine.
- Tests cover append/rotation (cap size) and `--last` resolution.
