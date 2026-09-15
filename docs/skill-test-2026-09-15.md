# Skill-test sweep 2026-09-15 — full s1–s7 PASS (executor-sibling harness)

First run of the executor-sibling harness (yyx_skill_test ADR 0002): the
orchestrator delegated every spawn/fetch/grade to disposable herdr
opencode siblings; grades read from `~/.yyxSkill/runs/log-labor/`.

## Results (tool_calls / reasoning tokens · 9/14 baseline)

| scenario | verdict | calls/tokens | baseline | note |
|---|---|---|---|---|
| s1-add-flow | PASS | 2 / 172 | 2 / 142 | preview-stop held |
| s2-daily-batch | PASS | 4 / 826 | 3 / 1264 | reasoning −35% |
| s3-source-harness | PASS | 6 / 456 | 6 / 492 | |
| s4-init-onboarding | PASS | 1 / 665 | FAIL 10 / 2076 | PR #34 fix verified; --help probe safe |
| s5-sheet-change | PASS | 3 / 482 | 4 / 551 | after TWO skill fixes (below) |
| s6-update-closure | PASS | 3 / 368 | first run | update-not-add held; seeded row |
| s7-multiday-backfill | PASS | 3 / 1908 | first run | one collect --from/--to, fit, stop |

## Skill fixes landed (evidence → PR)

- **PR #37** — s5 run 1 FAIL: agent hand-copied config.json + dived into
  `profile import`; `config set sheet` was buried. Quick-start example,
  CHECK FIRST case-split, profile-import-is-not-for-titles note.
- **PR #38** — s5 run 3 FAIL: CHECK FIRST made the agent interrogate
  rename-vs-new-sheet instead of acting. Act on the common case;
  `doctor`'s key probe arbitrates; explicit "do not stop to ask".

## Harness bugs found + fixed (yyxSkills repo, same day)

1. **Grader false alarms** — opencode serves skills wrapped in
   `<skill_content>/<skill_files>` + synthetic title/footer; the raw
   compare flagged every run MISMATCH/REGRESSED (also the real story
   behind the 9/14 "mid-run clobber" notes). Unwrap before comparing.
2. **Prose gates** — gates are 0-LLM literal substring matches; fd9fd1d's
   tightened s1 used prose that can never match. s1 rewritten greppable;
   the greppable-gate rule documented in the skill format spec.
3. **Stash basename collision** — stashing BOTH the source tree and the
   installed copy destroyed the first stash (installed scenarios lost
   mid-sweep, recovered from the repo). Path-derived stash keys.
4. **Loader-path serves (ADR 0002, 2nd amendment)** — agents serve from
   `~/.agents/skills/`, not the per-tool install dir; an opencode-only
   install masqueraded as a process-snapshot ghost. Sweeps install
   `--all` and grade against the served copy.

## Human cleanup

- Delete seeded sheet row `PATRal` ([skill验证] 测试（可删除）, now 已完成)
  from 任务工时详细.
- Pre-sweep residue `~/.config/log-labor/config.json.bak` (Sep 14) left
  untouched.

Executor log: `~/.yyxSkill/runs/log-labor/_executor-log.md`. Executors
1–4 all disposed (fresh sibling per skill edit — the disposal doctrine
worked as designed).
