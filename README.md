# log-labor — 报工 CLI + agent skill (WeCom 智能表格 webhook)

One command to log labor into the team's 任务工时详细 smartsheet — by hand
or by your AI agent. The CLI talks to the sheet's WeDoc webhook
(「接收外部数据」) directly; your agent gets a skill that teaches it when
and how to call the CLI.

    张三$ log-labor add -c "完成登录页联调" -h 3 --status 进行中
    ok  add record_id=rAbC12  sheet=任务工时详细

## Install (张三's walkthrough)

```bash
# 1a. easy path — anonymous GitHub mirror (needs `go` to build):
curl -fsSL https://raw.githubusercontent.com/yyx462/AutoWecom-plugin/master/log-labor/install.sh | sh

# 1b. corp path — clone with your Gitea login, then run the installer
git clone https://git.sh.nint.com/ying.yuxiang/AutoWecom-plugin
./AutoWecom-plugin/log-labor/install.sh
#   → builds from the checkout, installs to ~/.local/bin/log-labor
#   (git.sh.nint.com signs everyone in, so a piped curl gets the login
#   page — clone first, then run)

# Windows: run either path inside Git Bash (https://git-scm.com) with `go`
# installed — the installer detects MSYS/MINGW, builds log-labor.exe, and
# installs to %USERPROFILE%\.local\bin. Add that dir to PATH (~/.bashrc for
# bash, or Windows Settings > Environment Variables for cmd/PowerShell).
# Manual fallback: cd log-labor && go build -o log-labor.exe ./cmd/log-labor

# 2. configure — wizard asks for the sheet webhook key + your corp id
log-labor init
#   ? Webhook key: nkVN…********************************…X9z
#   ? 你的企业userid (corp id): zhang.san
#   ? Sheet name [任务工时详细]: ⏎
#   config written to ~/.config/log-labor/config.json (0600)
#   (Windows/Git Bash: %AppData%\log-labor\config.json)

# 3. prove the pipeline with one clearly-marked sample row (asks first)
log-labor doctor --write-sample
#   you will insert 1 row into sheet 任务工时详细:
#   | 人员       | 日期        | 状态   | 需求内容                        | 预计花费工时 | 提出人 | 卡点 |
#   | zhang.san | 2026年9月11日 | 已完成 | [skill验证] 测试（可删除） | 0.5         | 张三   | 无   |
#   proceed? [y/N] y
#   ok add record_id=rXyZ98 — delete this row in the sheet UI when done

# 4. give your agent the skill (detects Claude Code / opencode / Codex /
#    Cursor / Trae / AGENTS.md; --all forces everything)
log-labor skill install
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
