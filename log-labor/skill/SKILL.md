---
name: log-labor
description: >-
  报工 / labor logging on the 任务工时详细 smartsheet through the log-labor
  CLI (WeCom 智能表格 webhook). Use when the user asks to 记工时/报工/log
  labor/add a row to 任务工时详细, or asks to set this CLI up. Live-proved
  webhook facts inside; the CLI enforces them — never hand-craft webhook
  requests when the CLI is installed.
---

# log-labor — 报工 via the 任务工时详细 smartsheet webhook (vdev)

## Commands

```bash
log-labor add -c "完成登录页联调" -h 3                  # today, 默认状态 进行中
log-labor add -c "修复导出崩溃" -h 2 --status 已完成 --due 2026-09-20 --blocker "无"
log-labor update --record-id R --status 已完成          # closure = UPDATE, never a new row
log-labor doctor                                       # config + key health, no writes
log-labor doctor --write-sample                        # one marked sample row (asks nothing)
log-labor skill install                                # (re)install this skill for detected agents
```

Flags: `-c/--content` · `-h/--hours` (2.5 / 1h / 2小时) · `--date` (YYYY-MM-DD,
default today +08) · `--status` (状态, 已完成, 调休, 进行中, 待排期, 已合并, 已关闭) · `--person` (corp id)
· `--proposer` · `--blocker` · `--due` · `--link R1[,R2]` · `--yes`
(skip preview confirm) · `--dry-run`.

## The agent flow (报工 on demand)

1. Derive facts from session evidence: 需求内容 = ONE concise sentence on
   the actual work; hours from the user or a stated estimate — if hours
   are ambiguous, ASK, don't guess.
2. Run WITHOUT `--yes` first and show the user the preview table; confirm
   creative summaries. Then re-run with `--yes` (or let the confirm run).
3. Check exit 0 and report the `record_id` back with the summary.
4. Rework/closure later = `update --record-id`, never a second row.

## Hard facts (live-proved 2026-09-11, enforced by the CLI)

- Sheet: **任务工时详细**; keys are WRITE credentials — never print the key,
  never commit it (`~/.config/log-labor/config.json`, 0600).
- 人员 takes the CORP userid (`zhang.san` pinyin form). `woa-…`
  bot-namespace ids and numeric ids are rejected **atomically** (40031) —
  a bad user value kills the whole request, nothing is written.
- ONE op per request (mixing add+update → 40058). Rate caps: 3000 rows/min
  per sheet, 10000/min per doc.
- The webhook is write-only: no read, no delete. Deleting the 可删除 test
  rows happens in the sheet UI by the human.
- Value shapes: text = plain string · number = bare · date_time =
  ms-epoch STRING (00:00 +08) · single_select = option text · user =
  corp id · link = record ids.

## Onboarding a coworker (张三)

```bash
curl -fsSL 'https://git.sh.nint.com/ying.yuxiang/AutoWecom-plugin/raw/branch/master/log-labor/install.sh' | sh
log-labor init            # key + corp id (zhang.san form) + sheet name
log-labor doctor --write-sample   # one marked row proves the pipeline
log-labor skill install   # this skill, for every agent on the machine
```

If `log-labor` is missing: build from source (`cd log-labor && go build
./cmd/log-labor`) or point the coworker at the repo README.
