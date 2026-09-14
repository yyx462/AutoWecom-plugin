# 0001 — native multi-sheet profiles (`log-labor profile`)

- **Status:** open
- **Type:** feature
- **Origin:** live 2026-09-14 — backfilling a second 任务工时详细-shaped
  sheet took manual jq surgery on `~/.config/log-labor/config.json`
  (backup, swap `.key` + nine `.profile.fields.*.id`, doctor, restore).

## Problem

One config = one sheet. Any second same-shaped sheet (same column
titles and statuses, different field ids + webhook key — e.g. a
teammate's or a test copy of 任务工时详细) forces the interim dance
documented in SKILL.md "Another sheet, same shape (interim)":
error-prone, minutes per switch, and the restore step is easy to
forget.

## Ask

- Named profiles in config.json (migrate the single existing profile
  to `default`; file stays 0600, key never printed).
- `log-labor profile add <name>` — interactive: webhook key + sheet
  name + schema-JSON paste (the 接收外部数据 sample) → derive the
  field-id map BY COLUMN TITLE automatically; reject unknown/missing
  titles loudly.
- `log-labor profile list|remove|use <name>` plus a global
  `--profile <name>` on `add|update|doctor|daily` so one-offs never
  mutate the default.
- `doctor` prints the active profile and probes non-default keys.

## Acceptance

- Two profiles configured; `--profile other add …` writes to the
  other sheet without touching the default's rows or becoming the
  default.
- Old single-profile configs work untouched (migration on load).
- Schema import maps the nine roles by title with zero manual ids.
- Go tests cover migration, title-mapping, and flag routing.
