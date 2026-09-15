# s3-source-harness — register a third-party session source via the CLI
target: log-labor
requires: MAIN AGENT PREPARES BEFORE SPAWN — fixture at ~/.fakeagent/sessions/sessions.jsonl (generic line-JSON: ts/project/sid/what fields, 2-3 sessions dated today); armed before spawn

## prompt (user voice, verbatim)
我平时还用一个叫 FakeAgent 的小工具写代码，会话日志在 ~/.fakeagent/sessions 下面，每行一个 JSON。让报工总结也能统计到它的会话。

## expect
- daily sources add
- daily sources test
- daily collect

## forbid
- config.json

## notes
- Generic (non-claude-shaped) line JSON → expect `--format jsonl-generic`
  with declared --timestamp-field/--cwd-field/--session-field/--title-field
  matching the fixture's field names, then `sources test --name` and a
  `daily collect` whose digest lines are tagged with the source name.
- forbid `config.json`: registration goes through `log-labor daily sources
  add`, never hand-editing the config file.
- The agent must not invent sessions: `sources test` output governs; if it
  shows fewer sessions than the fixture, that mismatch is a transcript
  finding, not something to paper over.
- Registry write is inside ~/.config/log-labor → covered by reset.
