# agent skills on the plugin tag

Loaded by opencode via `skills.paths` in `opencode.json`, and mounted
into the brain with the rest of `plugin/`.

## ours (manager-bot workflows)

| Skill | Trigger |
|---|---|
| `agent/` | the brain's investigation prompt (`agent.md`, hot-read per turn — NOT an opencode skill; no SKILL.md frontmatter) |
| `reporting/` (was wecombot-reporting) | end of any work session on a tracked project → POST labor summary to the manager API |
| `sop/` (was wecombot-sop) | query or update SOP/project schedules (critical path, at-risk tasks) via the manager API |

Installation as **global** skills on web01 agents:

```bash
./skills/install-global.sh          # symlinks the ours-skills into ~/.agents/skills/
```

Agents need `WECOMBOT_API_URL` and `WECOMBOT_API_TOKEN` in their env.

## `vendor/wecom-cli` — vendored upstream skills

From <https://github.com/WecomTeam/wecom-cli> (MIT), pinned upstream
commit `59604ae`. These document the wecom-cli tool surface itself
(message push, docs, sheets, smartsheets, mail, todo, calendar,
meetings, drive, contacts). The manager service is the only process
that *executes* wecom-cli writes (ADR-0002); agents normally go through
the manager API instead.

Update procedure:

```bash
git clone --depth 1 https://github.com/WecomTeam/wecom-cli /tmp/wecom-cli
rsync -a --delete /tmp/wecom-cli/skills/ skills/vendor/
# then strip nested `metadata:` frontmatter blocks from vendored SKILL.md
```

Credentials never enter this repo: wecom-cli keeps them in
`WECOM_CLI_CONFIG_DIR` (`credentials.enc`, encrypted).
