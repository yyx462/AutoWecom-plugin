# s6-update-closure — close an existing row via update, never a second add
target: log-labor
requires: MAIN AGENT SEEDS BEFORE SPAWN — `log-labor doctor --write-sample` writes one marked 可删除 row; capture its record_id from the output and substitute it for <RID> in the prompt. Armed before spawn.

## prompt (user voice, verbatim)
报工表里那条 [skill验证] 测试（可删除） 的记录（record_id <RID>）其实已经弄完了，帮我把它改成已完成。

## expect
- log-labor update --record-id
- --status 已完成

## forbid
- cmd:log-labor add

## notes
- Closure/rework = UPDATE on the existing row, NEVER a second row — the
  webhook cannot delete, so a wrong extra row needs human cleanup in the
  sheet UI.
- The user's prompt IS the explicit instruction for this exact change:
  whether the agent re-runs with `--yes` after a non-interactive confirm
  (exit 2) or stops to confirm is acceptable variance this iteration —
  the contract under test is update-not-add. Tighten to a hard gate once
  baseline behavior is known.
- The seeded row lives on the TEST sheet (marked 可删除) — human deletes
  sample rows in the sheet UI; nothing to clean via webhook.
