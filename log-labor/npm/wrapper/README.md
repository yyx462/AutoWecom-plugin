# log-labor

报工 / labor logging on the WeCom 任务工时详细 smartsheet — one command,
by hand or by your AI agent.

This npm package is a thin wrapper: `npm i -g @yyx462/log-labor` puts
the `log-labor` command on your PATH (the command name comes from the
`bin` field — same pattern as `@anthropic-ai/claude-code` → `claude`).
The actual Go CLI arrives as the one `optionalDependencies` package
matching your platform (`os`/`cpu` fields make npm skip the other
four).

## Quick start

```bash
npm install -g @yyx462/log-labor
log-labor init            # webhook key + your corp id (zhang.san form)
log-labor doctor          # verify
log-labor skill install   # give your agents the skill
```

## Upgrading from the old unscoped `log-labor` package

`npm i -g @yyx462/log-labor` fails with `EEXIST` (`log-labor`,
`log-labor.cmd` or `log-labor.ps1` already exists) when the pre-rename
unscoped package is still installed — npm refuses to overwrite its
shims. Remove it once, then install:

```bash
npm rm -g log-labor && npm i -g @yyx462/log-labor
```

No Node? Install straight from GitHub Releases instead:

```bash
curl -fsSL https://raw.githubusercontent.com/yyx462/AutoWecom-plugin/master/log-labor/install.sh | sh
```

(Windows PowerShell: `irm https://raw.githubusercontent.com/yyx462/AutoWecom-plugin/master/log-labor/install.ps1 | iex`)

Docs and source: https://github.com/yyx462/AutoWecom-plugin
