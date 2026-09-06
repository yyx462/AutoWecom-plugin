# plugin — the model-facing tree (ships on the plugin tag)

Mounted read-only into the brain at `/app/plugin`. Contains everything
the model or the Pi runtime needs **without performing real I/O**:

```
catalog.yaml      the join key: one row per product capability
                  (name + args + description); the broker reads the same
                  file to authorize /invoke names
extensions/       Pi registerTool stubs — POST /invoke only, no keys
skills/           model-facing how-to
  agent/agent.md  the live investigation prompt (hot-read per turn)
  sop/            SOP schedule workflows (via the manager API)
  reporting/      end-of-session labor reporting
  vendor/         vendored wecom-cli agent skills (upstream, MIT)
opencode.json     registers skills/ for agents working on the plugin
```

Rules (HOW_TO_MIGRATE §1/§7): nothing in this tree may hold a key, call
a backend, or name a credential. If a file talks to a ticket/doc/LLM/
WeCom API, it belongs in the broker instead.
