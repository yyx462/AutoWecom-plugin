# Using log-labor with your AI agent

You installed the CLI, you have an AI agent (Claude Code, opencode,
Codex CLI, Cursor, Trae — any of them). This guide is the part after
`log-labor skill install`: what to say to your agent, what it will do,
and what to do when something looks wrong. For one-time setup see the
repo README (张三's walkthrough).

## What `skill install` gave your agent

`log-labor skill install` detects which agents live on your machine and
wires a SKILL.md into each:

| Harness      | Global install (~)                    | Project install (repo)           |
| ------------ | ------------------------------------- | -------------------------------- |
| Claude Code  | `~/.claude/skills/log-labor/`         | `.claude/skills/log-labor/`      |
| opencode     | `~/.config/opencode/skills/log-labor/`| `.opencode/skills/log-labor/`    |
| Codex CLI    | `~/.codex/skills/log-labor/`          | `.codex/skills/log-labor/`       |
| generic      | `~/.agents/skills/log-labor/`         | `.agents/skills/log-labor/`      |
| Cursor       | —                                     | `.cursor/rules/log-labor.mdc`    |
| Trae         | —                                     | `.trae/rules/log-labor.md`       |

Cursor/Trae are project-only rules files with a marked stanza; the
others are real Agent-Skills folders. Install everywhere at once with
`--all`, into the current repo only with `--project`:

```bash
log-labor skill install --all           # every detected harness, global
log-labor skill install --agent cursor --project
log-labor skill uninstall --agent trae  # clean removal (stanza included)
```

## Talking to your agent (the normal loop)

The skill triggers on 记工时 / 报工 / logging labor / adding a row to
任务工时详细. Plain language is enough:

```text
你: 记一下工时 — 今天花 2.5 小时调通了登录页接口
agent: 起草需求内容「完成登录页接口联调」, shows a preview:
       | 人员    | 日期         | 状态   | 需求内容           | 工时 |
       | zhang.san | 2026年9月11日 | 进行中 | 完成登录页接口联调 | 2.5 |
       proceed? [y/N]
你: y
agent: ok add record_id=rAbC12 — 已记入 任务工时详细
```

```text
你: 把 rAbC12 那条报工改成已完成
agent: log-labor update --record-id rAbC12 --status 已完成
```

```text
你: 昨天忘了报工,补一条 3 小时,写"修复导出崩溃",卡点是等后端字段
agent: log-labor add -c "修复导出崩溃" -h 3 --date 2026-09-10 \
         --blocker "等后端字段"
```

What your agent is trained to do (from the skill):

1. **Draft, don't guess.** 需求内容 is ONE concise sentence on the
   actual work; if the hours are ambiguous it asks you instead of
   inventing a number.
2. **Preview before writing.** First run omits `--yes`, so you see the
   exact row — 日期, 文本 (状态), 需求内容, 工时 — creative summaries
   AND the status column get confirmed with you.
3. **Report the record_id.** The `rAbC12`-style id is how you reference
   the row later for updates.
4. **Close, don't duplicate.** Rework/closure is `update --record-id`,
   never a second row.
5. **Defaults for speed.** 预计完成时间 mirrors 日期 on add; pin
   `config set default_status 已完成` / `config set default_proposer
   <corp-id>` once and the agent stops passing those flags.

## End-of-day: 总结今天 (daily batch)

Ran agent sessions all day? Close out in one line — the skill digests
the day's sessions into 2–6 records scaled to one working day:

```text
你: 帮我总结今天并报工
agent: log-labor daily collect        # today's sessions, every source
       drafts records per theme → log-labor daily fit → shows the
       fitted table (Σ = 8.0h, 文本 column visible) → you OK it →
       one log-labor add per record
```

The fitted table always carries the 文本 (状态) column — same-day
batches default 进行中; backfilling a past day usually wants them all
已完成, so the agent regenerates with `daily fit --status 已完成`
before writing. Backfilling several days in one sitting: collect
windows overlap at day boundaries, so the agent excludes sessions
already logged under another date.

Overtime days have their own cap — say the number and the agent fits to
it instead of 8 ("help me log today to 10 hours"):

```text
你: 今天加班了,帮我把今天报成 10 小时
agent: same flow, but `log-labor daily fit --total 10` — Σ = 10.0h exactly
```

`--total` accepts `10`, `10h`, `10小时`; default `8`. Correcting hours
after insert: `log-labor update --record-id R -h <new>` works while R
is still known from the add output — the webhook is write-only (no
read, no delete), so rows you can't reference anymore are edited by
hand in the WeCom sheet UI.

## The CLI, without an agent

```bash
log-labor add -c "完成登录页联调" -h 3                  # today, 进行中
log-labor add -c "评审" -h 1 --status 已完成 --link rAbC12
log-labor update --record-id rAbC12 --status 已完成
log-labor doctor                      # config + key health, no writes
log-labor config list                 # see the stored config (key masked)
```

Flags: `-c/--content` (需求内容) · `-h/--hours` (accepts `2.5`, `1h`,
`2小时`) · `--date YYYY-MM-DD` (default today Asia/Shanghai) ·
`--status` (已完成, 调休, 进行中, 待排期, 已合并, 已关闭; default
进行中) · `--person` (corp id, default from config) · `--proposer` ·
`--blocker` · `--due` · `--link R1[,R2]` · `--yes` · `--dry-run`.
Unset optionals are omitted from the row — never written empty.
Exit codes: `0` ok · `1` WeCom error · `2` usage/config problem.

## Keeping it current

Mostly nothing: upgrading the CLI is one step (`npm i -g
@yyx462/log-labor`, or the curl one-liner again) and the next
`add`/`update`/`daily`/`doctor` run silently re-renders any installed
skill whose stamp is older than the binary — you'll see one line,
`skills refreshed → v0.1.3 (opencode, …)`. Manual, when you want it:

```bash
log-labor skill install     # or: log-labor skill upgrade

# webhook key rotated by the sheet owner? update in place
log-labor config set key <new-key>

# not using it with an agent anymore
log-labor skill uninstall   # removes every installed copy
```

The skill itself is versioned with the CLI (a trailing version marker
in each installed copy; `log-labor doctor` reports `skills STALE …`
when one lags the binary); re-running install is always safe and
idempotent. Skills you uninstalled stay uninstalled — the refresh only
touches copies that are still there.

## Troubleshooting

| Symptom | Fix |
| --- | --- |
| `exit 2: no config yet` | run `log-labor init` |
| `exit 1: errcode 840001` | key invalid/expired — sheet owner rotates it, then `log-labor config set key <new>` |
| `exit 1: errcode 40031` | bad 人员 value — needs the corp id (`zhang.san` form), not `woa-…` or numeric ids; the whole request is rejected atomically, just retry with the right id |
| `exit 1: errcode 2022004` | unknown field/status — status must be one of the 状态 single-select options |
| agent never offers to 报工 | `log-labor skill install` didn't reach that harness — install with `--all`, restart the agent session |
| sample row in the sheet | the `[skill验证] 测试（可删除）` row is a marked test row — delete it in the WeCom sheet UI (the webhook is write-only; no deletes) |

`log-labor doctor` is always the first move: it checks config shape,
key validity (a zero-write probe), and prints the sheet profile it
would target. `doctor --write-sample` inserts one clearly-marked row to
prove the whole pipeline.
