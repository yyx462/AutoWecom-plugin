# log-labor — 报工 CLI + agent skill (WeCom 智能表格 webhook)

One command to log labor into the team's 任务工时详细 smartsheet — by hand
or by your AI agent. The CLI talks to the sheet's WeDoc webhook
(「接收外部数据」) directly; your agent gets a skill that teaches it when
and how to call the CLI.

    张三$ log-labor add -c "完成登录页联调" -h 3 --status 进行中
    ok  add record_id=rAbC12  sheet=任务工时详细

## Install

```bash
# 1. npm (agent users — Node already on the machine)
npm i -g @yyx462/log-labor      # installs the `log-labor` command

# 2. curl — no Node, no toolchain; prebuilt binary from GitHub Releases
curl -fsSL https://raw.githubusercontent.com/yyx462/AutoWecom-plugin/master/log-labor/install.sh | sh
# Windows PowerShell:  irm https://raw.githubusercontent.com/yyx462/AutoWecom-plugin/master/log-labor/install.ps1 | iex
```

## Troubleshooting

**`npm error EEXIST ... log-labor.ps1` (or `log-labor`, `log-labor.cmd`)
on install** — the pre-rename unscoped `log-labor` package (since renamed
to `@yyx462/log-labor`) is still installed, and npm refuses to overwrite
its shims. Remove it once and reinstall:

```bash
npm rm -g log-labor && npm i -g @yyx462/log-labor
```

(`npm rm -g log-labor` is a harmless no-op when it isn't installed.)

## Init

```bash
# configure — wizard asks for the sheet webhook key + your corp id
#   (key lives at 智能表格 → 右上角文档操作 → 接收外部数据 → Webhook 地址 —
#    pasting the whole URL works too)
log-labor init
#   ? Webhook key / ? 你的企业userid (zhang.san) / ? Sheet name [任务工时详细]
#   → ~/.config/log-labor/config.json (0600; Windows: %AppData%\log-labor)

# different sheet (fields/statuses differ)? paste its 示例数据 and the
#   profile re-derives itself — 智能表格 → 右上角文档操作 → 接收外部数据 → 示例数据
log-labor profile import            # interactive paste; or: profile import <sample.json>

# prove the pipeline — one clearly-marked sample row, asks first
log-labor doctor --write-sample

# give your agent the skill (Claude Code / opencode / Codex / Cursor /
# Trae / AGENTS.md; --all forces everything)
log-labor skill install

# 3rd-party agents: self-register their session store for daily collect
#   (opencode / Claude Code / Codex are built-in, no registration)
log-labor daily sources add --name <harness> --format jsonl-claude --path <dir>
log-labor daily sources test --name <harness>     # verify before trusting
```

Upgrades are one step — reinstall (`npm i -g @yyx462/log-labor` or the
curl one-liner) and the next `log-labor` command silently refreshes any
installed skill to the new version.

## Daily use

```bash
log-labor add -c "完成登录页联调" -h 3                    # today, 进行中
log-labor add -c "修复导出崩溃" -h 2 --status 已完成 --blocker "无" --due 2026-09-20
log-labor update --record-id rAbC12 --status 已完成       # closure = update, not a new row
log-labor doctor                                          # config + key health (no writes)
log-labor upgrade                                         # self-update: npm i -g / install.sh one-liner
log-labor skill upgrade                                   # manual skill refresh (auto: next command after an upgrade refreshes for you)
```

End-of-day batch from parallel agent sessions (Σ lands on exactly the
day's total — 8h by default):

```bash
log-labor daily collect                     # today's sessions: project, span, msgs
log-labor daily fit "联调"=3.2 "修DAG"=2.7   # scales+snaps to Σ=8.0, prints add lines
log-labor daily fit "联调"=4 "修DAG"=6 --total 10h   # user-set cap: "报今天的工,按 10 小时"
log-labor daily fit "联调"=3 --status 已完成 # one 文本 for every row+add line (default 进行中)
```

`--total` accepts `10`, `10h`, `10小时` and defaults to 8 — the user
names the cap in plain language, the agent passes the flag. The fitted
table always shows the 文本 (状态) column — same-day logs default
进行中, backfills of past days usually want `--status 已完成` — so
status is confirmed BEFORE anything is written. Fixing a
logged row later: `log-labor update --record-id R -h <new>` while R is
still known from the add output (the webhook is write-only — no read,
no delete); anything older is edited by hand in the sheet UI.

Second, same-shaped sheet? The config holds ONE sheet today — see the
skill's "Another sheet, same shape (interim)" section for the
backup/jq-swap/doctor procedure; native `profile` support is ticketed
in `docs/tickets/0001-sheet-profiles.md`.

Collect scans every source it knows: built-ins (opencode, Claude Code,
Codex) auto-detect, and any other agent harness registers itself via
`log-labor daily sources add` — see the skill's "Session sources"
section.

Missing `--date` = today (Asia/Shanghai); missing `--status` = 进行中
(or `config set default_status 已完成` team-wide); 预计完成时间
defaults to the row's 日期 (`config set due_mirror off` disables);
unset optionals are omitted from the row, never written empty. Exit codes:
`0` ok · `1` WeCom returned an error · `2` usage/config problem.

## Get the webhook key

The sheet owner enables 接收外部数据 on the sheet (智能表格 → 右上角文档操作
→ 接收外部数据) and shares the Webhook 地址 with you — `init` accepts the
key or the whole URL. The key is a write credential
for that one sheet — treat it like a token; the owner can rotate it in the
console and you update with `log-labor config set key <new>`.

## Configuration

`~/.config/log-labor/config.json` (0600): `key`, `person` (corp id, e.g.
`zhang.san`), `endpoint`, and the sheet `profile` (name + field-id map +
status enum). The default profile targets the team labor sheet; point the
CLI at a different sheet with `log-labor profile import` — paste that
sheet's 接收外部数据 示例数据 and the field map + statuses re-derive
(`--map role=id` fixes any mis-guessed column; unmapped roles just
disable their flags). `config set sheet <name>` renames only.

## For agents

Install the skill (step 4 above) — it teaches your agent the whole
protocol: value shapes, one-op-per-request, atomic 40031 on bad user ids,
rate caps, and exactly how to draft 需求内容 and call `log-labor`. The
rendered skill lives at `log-labor/skill/SKILL.md`; the embedded template
is the source of truth. For the day-to-day experience — what to say to
your agent, where skills land per harness, troubleshooting — see
[docs/USAGE.md](docs/USAGE.md).

## Building from source

```bash
cd log-labor && go build -o dist/log-labor ./cmd/log-labor
```
