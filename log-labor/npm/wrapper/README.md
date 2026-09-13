# log-labor

报工 / labor logging on the WeCom 任务工时详细 smartsheet — one command,
by hand or by your AI agent.

This npm package is a thin wrapper: `npm i -g log-labor` puts the
`log-labor` command on your PATH. The actual Go CLI arrives as the one
`optionalDependencies` package matching your platform (`os`/`cpu`
fields make npm skip the other four).

## Quick start

```bash
npm install -g log-labor
log-labor init            # webhook key + your corp id (zhang.san form)
log-labor doctor          # verify
log-labor skill install   # give your agents the skill
```

No Node? Install straight from GitHub Releases instead:

```bash
curl -fsSL https://raw.githubusercontent.com/yyx462/AutoWecom-plugin/master/log-labor/install.sh | sh
```

(Windows PowerShell: `irm https://raw.githubusercontent.com/yyx462/AutoWecom-plugin/master/log-labor/install.ps1 | iex`)

Docs and source: https://github.com/yyx462/AutoWecom-plugin
