---
name: log-labor
description: >-
  报工 / labor logging on the 任务工时详细 smartsheet through the log-labor
  CLI (WeCom 智能表格 webhook). Use when the user asks to 记工时/报工/log
  labor/add a row to 任务工时详细, or asks to set this CLI up. Live-proved
  webhook facts inside; the CLI enforces them — never hand-craft webhook
  requests when the CLI is installed.
---

# log-labor — 报工 via the 任务工时详细 smartsheet webhook (dev)

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
- 人员 takes the CORP userid (`ying.yuxiang` pinyin form). `woa-…`
  bot-namespace ids and numeric ids are rejected **atomically** (40031) —
  a bad user value kills the whole request, nothing is written.
- ONE op per request (mixing add+update → 40058). Rate caps: 3000 rows/min
  per sheet, 10000/min per doc.
- The webhook is write-only: no read, no delete. Deleting the 可删除 test
  rows happens in the sheet UI by the human.
- Value shapes: text = plain string · number = bare · date_time =
  ms-epoch STRING (00:00 +08) · single_select = option text · user =
  corp id · link = record ids.

## End-of-day batch (Σ = 8h default, any --total)

When the user asks to 总结今天 / 报工总结 / close out the day — or runs
parallel agent sessions all day and reports at once — or asks "help me
log today to 10 hours" (帮我把今天报成 10 小时):

1. `log-labor daily collect [--date YYYY-MM-DD]` — digest of that day's
   agent sessions: project, time span, message count, title.
2. Draft 2–6 records grouping the digest into real work themes with
   HONEST raw hours; ask the user when the digest is ambiguous. Never
   invent work that is not in the digest (or that the user rejects).
3. Fit them to the day's total — pipe a JSON array to
   `log-labor daily fit --date <day>`:
   `echo '[{"content":"…","hours":2.5},…]' | log-labor daily fit --date 2026-09-11`
   (or bare args: `log-labor daily fit "内容"=2.5 …`). The fit scales
   proportions, snaps to 0.5h, and lands Σ on EXACTLY the total; it
   prints the fitted table plus ready `log-labor add` lines (with
   `--date`). Total = 8.0h unless the user set the day's cap:
   `--total 10` (accepts 10 / 10h / 10小时) — "log today to 10 hours"
   is collect → draft → `daily fit --total 10` → add, nothing else.
4. Show the fitted table to the user; after OK, run each add line.
   Cite which digest session each record came from.

Hours corrections after insert: the webhook only WRITES — no read, no
delete. `log-labor update --record-id R -h <h>` fixes a row while R is
still known from its add output; without R the user edits the row by
hand in the sheet UI. Mention this ONLY when the user asks to change
an already-logged row — never repeat it on every 报工.

## Session sources — setup for any agent

`daily collect` auto-detects the verified built-ins: opencode
(~/.local/share/opencode/opencode.db), Claude Code (~/.claude/projects),
and Codex (~/.codex/sessions). Any OTHER harness with durable line-JSON
session logs needs a one-time registration — the agent does this itself:

1. Probe: does this harness write session files anywhere under
   `~/.<agent>/`? Look for sqlite stores or `*.jsonl` with per-line
   timestamps (and ideally a session id + project/cwd).
2. Claude-shaped JSONL (timestamp/cwd/sessionId per line):
   `log-labor daily sources add --name <harness> --format jsonl-claude \
      --path <dir>`
   Any other line-JSON shape — declare the field names:
   `log-labor daily sources add --name <harness> --format jsonl-generic \
      --path <dir> --timestamp-field ts --cwd-field project \
      --session-field sid --title-field what`
   (opencode, Claude Code and Codex are built-in; no registration
   needed. The sqlite family is opencode-only — other sqlite stores are
   NOT supported yet.)
3. Verify BEFORE trusting it:
   `log-labor daily sources test --name <harness> [--date YYYY-MM-DD]`
   → session count + sample rows. Then `daily collect` — every digest
   line is tagged with the source name.
4. Harnesses with NO durable transcripts (Cursor, Trae): those sessions
   can't be collected — ask the user what they did there and merge it
   into the same fit.

Never invent a source's contents; if `test` returns 0 sessions for a
day the user says they worked, say so and ask.

## Onboarding a coworker (张三)

```bash
curl -fsSL https://raw.githubusercontent.com/yyx462/AutoWecom-plugin/master/log-labor/install.sh | sh
# Windows PowerShell: irm https://raw.githubusercontent.com/yyx462/AutoWecom-plugin/master/log-labor/install.ps1 | iex
log-labor init            # key + corp id (zhang.san form) + sheet name
log-labor doctor --write-sample   # one marked row proves the pipeline
log-labor skill install   # this skill, for every agent on the machine
```

If `log-labor` is missing: re-run the one-liner (fetches the release
binary) or build from source (`cd log-labor && go build ./cmd/log-labor`).
