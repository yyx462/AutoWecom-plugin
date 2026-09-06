# extensions — Pi registerTool stubs (Design 1, handwritten)

This is the plugin's single extension bundle. The brain loads it into
every pi run (`--extension <this dir> --no-default-extensions`). It
registers exactly three stubs, one per catalog row:

- `kb_search` — trusted knowledge base
- `repo_search` — answerable internal repos (repos.yaml)
- `list_repos` — answerable repos overview

A stub is name + args + description ONLY. Execution is a `POST
$BROKER_BASE_URL/invoke` with `{name, args, thread, principal}` — no
handler logic, no keys, no backend SDKs. The implementations live in
`broker/broker/handlers/search/ptools.py`.

The brain injects `BROKER_BASE_URL`, `WECOM_THREAD` (chat id) and
`WECOM_PRINCIPAL` (userid) into the pi environment per turn.
