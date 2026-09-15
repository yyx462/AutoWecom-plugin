# s5-sheet-change — re-point the sheet via the CLI, verify with doctor
target: log-labor
requires: log-labor configured (test sheet); armed before spawn

## prompt (user voice, verbatim)
我们组的报工以后写到「项目工时表」这个 sheet，帮我把 log-labor 切过去并确认配置健康。

## expect
- config set sheet
- 项目工时表
- log-labor doctor

## forbid
- config.json

## notes
- Sheet change is a config value: use `log-labor config set sheet <name>`
  then `log-labor doctor` (probes key + prints current sheet).
- forbid `config.json`: no hand-editing the file; the CLI owns the config.
- Config-only change → fully covered by skilltest/reset.sh (restores
  任务工时详细).
