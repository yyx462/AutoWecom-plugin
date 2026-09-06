---
name: wecombot-sop
description: Query and update SOP/project schedules (tasks, critical path, at-risk tasks) through the weComBot manager API, which syncs the canonical local schedule to the WeCom SOP sheet. Use when the user asks about project schedule/进度/关键路径/SOP, wants to mark a task done/started, or wants the SOP sheet refreshed.
---

# SOP schedule queries & updates (weComBot manager)

Canonical schedules are YAML files in the weComBot data repo
(`sop/<project>/schedule.yaml`); the WeCom sheet is a one-way
projection. All access goes through the manager API — do NOT edit the
sheet directly with wecom-cli.

## Prerequisites

- `WECOMBOT_API_URL` and the repo helper `apis/wecombot_api.py`.
  One-time login: `python3 <repo>/apis/wecombot_api.py login`
  (WeChat QR; cached 7 days at `~/.config/wecombot/profile.json`).

## Read: schedule + critical path

```bash
python3 <repo>/apis/wecombot_api.py GET /v1/sop/<project>
```

Returns tasks with es/ef/slack/critical flags, `critical_path` ids, and
`project_end`. Use it to answer: 关键路径是什么 / 哪些任务有浮动 /
项目什么时候结束 / 某任务为什么紧急.

## Update: task status

```bash
echo '{"task":"T2","status":"done"}' > /tmp/sop.json
python3 <repo>/apis/wecombot_api.py POST /v1/sop/<project>/status --data @/tmp/sop.json
```

status ∈ todo / doing / done. This edits the canonical file (git
commit) — after a status change, always sync.

## Sync: push to the WeCom sheet

```bash
python3 <repo>/apis/wecombot_api.py POST /v1/sop/<project>/sync
```

Overwrites the project's sheet tab with the computed table and appends
a snapshot row to the Log tab. Response carries the fresh
`critical_path` and `project_end` — surface them to the user.

## Reminders

```bash
python3 <repo>/apis/wecombot_api.py POST /v1/reminders/run
```

Returns per-project digests of at-risk tasks (critical, deadline ≤ 3
days, not done). The manager also runs this daily on its own.

## Rules

- Structural edits (new tasks, deps, durations, deadlines) are NOT done
  via the API — tell the user to edit `sop/<project>/schedule.yaml` (or
  ask an agent with repo access to commit the edit), then sync.
- `dry_run: true` in responses means wecom-cli auth is pending: the
  sheet was not actually written. Tell the user plainly.
- Report `critical_path` as task ids + names, and flag any task whose
  slack is 0 when the user asks "what's urgent".
