# AutoWecom-plugin

The model-facing tree for AutoWecom (repo split 2026-09): `catalog.yaml`,
`extensions/` (Pi registerTool stubs), `skills/` (model how-to + agent
prompt), `opencode.json`. The core (brain + broker + docs) lives in
`AutoWecom-core`; the live data repo stays `ying.yuxiang/weComBot-dat`.

## Rules (HOW_TO_MIGRATE §1/§7)

- Nothing in this tree may hold a key, call a backend, or name a
  credential. If a file performs real I/O, it belongs in the broker
  (AutoWecom-core `broker/handlers/`), not here.
- The catalog is the join key: one row = one stub here + one handler in
  AutoWecom-core. **Land the core handler first, then the catalog row.**
  `/invoke` fails closed — a row without a handler is a clean 404 on that
  tool, never a crash.
- `send_*`, raw `doc.read`/`doc.edit`, admin bash/write/edit: never rows.

## Deploy

Mounted read-only at `/app/plugin` into BOTH broker (catalog
authorization) and brain (stubs + skills). Same clone serves both mounts
(docker-compose `PLUGIN_DIR`). Pin scheme: tag `plugin-x.y.z` here;
`deploy.sh` checks out `PLUGIN_REF` (default `origin/master`).

Dev layout (gwt): this repo at `../AutoWecom-plugin/master`, core at
`../AutoWecom/master` — run compose from the core worktree with
`PLUGIN_DIR=../AutoWecom-plugin/master`.
