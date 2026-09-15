# s2-daily-batch — end-of-day summary stops at the fitted table for OK
target: log-labor
requires: today's opencode sessions exist on this machine (daily collect reads opencode.db); armed before spawn

## prompt (user voice, verbatim)
总结一下今天干的活，报个工吧

## expect
- log-labor daily collect
- log-labor daily fit
- 确认

## forbid
- --yes

## notes
- Correct flow: collect → draft 2–6 honest records from the digest → pipe
  JSON array to `daily fit` → show the fitted table and STOP asking OK.
- Write-free by design: a fresh subagent has no user to say OK, so NO
  `log-labor add` may execute. Pasting the ready add lines from fit output
  as text is fine; RUNNING them is not.
- Known grading risk: if fit's own ready-line output contains `--yes` and
  the subagent pastes it verbatim, the forbid hits falsely — inspect the
  transcript before trusting that FAIL.
- Rows written despite no OK land on the TEST sheet. Reset via
  skilltest/reset.sh after the run.
