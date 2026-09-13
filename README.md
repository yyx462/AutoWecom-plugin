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
log-labor init
#   ? Webhook key / ? 你的企业userid (zhang.san) / ? Sheet name [任务工时详细]
#   → ~/.config/log-labor/config.json (0600; Windows: %AppData%\log-labor)

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

## Daily use

```bash
log-labor add -c "完成登录页联调" -h 3                    # today, 进行中
log-labor add -c "修复导出崩溃" -h 2 --status 已完成 --blocker "无" --due 2026-09-20
log-labor update --record-id rAbC12 --status 已完成       # closure = update, not a new row
log-labor doctor                                          # config + key health (no writes)
log-labor skill upgrade                                   # refresh installed skills after CLI updates
```

End-of-day batch from parallel agent sessions (Σ lands on exactly 8h):

```bash
log-labor daily collect                     # today's sessions: project, span, msgs
log-labor daily fit "联调"=3.2 "修DAG"=2.7   # scales+snaps to Σ=8.0, prints add lines
```

Collect scans every source it knows: built-ins (opencode, Claude Code,
Codex) auto-detect, and any other agent harness registers itself via
`log-labor daily sources add` — see the skill's "Session sources"
section.

Missing `--date` = today (Asia/Shanghai); missing `--status` = 进行中;
unset optionals are omitted from the row, never written empty. Exit codes:
`0` ok · `1` WeCom returned an error · `2` usage/config problem.

## Get the webhook key

The sheet owner enables 接收外部数据 on the sheet (智能表格 → 更多 →
接收外部数据) and shares the key with you. The key is a write credential
for that one sheet — treat it like a token; the owner can rotate it in the
console and you update with `log-labor config set key <new>`.

## Configuration

`~/.config/log-labor/config.json` (0600): `key`, `person` (corp id, e.g.
`zhang.san`), `endpoint`, and the sheet `profile` (name + field-id map +
status enum). The default profile targets the team labor sheet; point the
CLI at a different sheet by editing the profile or `log-labor config set
sheet <name>` after swapping the field ids.

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
