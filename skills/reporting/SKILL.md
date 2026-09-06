---
name: wecombot-reporting
description: End-of-session labor reporting to the weComBot manager API. Use when a work session on a tracked project is wrapping up (user says 收工/下班/report/report labor/log hours, or the session did substantive tracked-project work), to POST a one-line labor summary that lands in the team WeCom 工时 sheet.
---

# Labor reporting (weComBot manager)

At the end of a work session, POST one labor report to the weComBot
manager API. The API appends it to the team labor sheet (企业微信 在线表格).

## When

- The user asks to "report labor" / "报工时" / "log hours", OR
- a session is ending and substantive work on a tracked project happened
  (the user will usually say 收工 / that's all / 汇报一下).

Do NOT report for: pure reading/questions, failed attempts with zero
progress, or sessions the user marks as "不用报".

## Prerequisites

- `WECOMBOT_API_URL` (e.g. `https://nwi-amazondb.nint.hk/bot`) and the
  repo helper `apis/wecombot_api.py`. One-time login:
  `python3 <repo>/apis/wecombot_api.py login` (browser WeChat QR;
  session cached at `~/.config/wecombot/profile.json`, 7 days). If the
  session is missing/expired, tell the user to run login — never guess.

## What to send

Summarize the session yourself; confirm with the user only if the
project name or hours are ambiguous. One call:

```bash
WECOMBOT_REPORT='{"user":"'"$WECOMBOT_USER"'","date":"'"$(date +%F)"'",
  "project":"<tracked project>","hours":<h>,"summary":"<1-3 句中文：做了什么，产物（PR/issue/commit）>",
  "session":"<session id or uuid4>","link":"<PR/issue URL if any>"}'
echo "$WECOMBOT_REPORT" > /tmp/report.json
python3 <repo>/apis/wecombot_api.py POST /v1/report --data @/tmp/report.json
```

(requests are Ed25519-signed by the helper; `session` is the
idempotency key — resending the same id is a safe no-op)

## Rules

- One report per session (`session` is the idempotency key — re-sending
  the same session id is a no-op, safe on retry).
- `hours` is the session's focused time, 0 < hours ≤ 24.
- `summary` states outcomes, not process ("合并 PR #12，修复 X"； not
  "我思考了很久然后…").
- Response `{"ok": true}` → tell the user one line: 已报工时（project,
  hours h）。 `duplicate: true` → 已报过，无需重复。
- Non-2xx: show the error body verbatim to the user; do not retry more
  than once; do not fall back to writing files or sheets directly.
- Never copy the key file contents anywhere; the seed stays at
  `WECOMBOT_API_KEY` unread.
