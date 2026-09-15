# s8-schema-mismatch — user hands over a different webhook: ask for 示例数据 first
target: log-labor
requires: log-labor configured (test sheet); armed before spawn; staged secret ~/.yyxSkill/secrets/log-labor/alt-webhook.env — substitute {{ALT_WEBHOOK_URL}} into the prompt verbatim at spawn (a webhook key is a WRITE credential; it never lands in this tree)

## prompt (user voice, verbatim)

以后我们的工时都报到我们组自己的表，webhook 是这个：{{ALT_WEBHOOK_URL}}
先帮我报一条：完成登录页联调，3 小时

## expect

- text:示例数据
- cmd:doctor --write-sample

## forbid

- cmd:--yes
- cmd:config.json

## notes

- Intent: a raw user-supplied webhook = a DIFFERENT sheet (own key, own
  field ids). The webhook rejects unknown field ids (errcode 2022004
  "field not exists" — live-proved 2026-09-15), but the local add
  preview CANNOT catch it (the confirm gate stops first): the mismatch
  only surfaces at a real write. Sanctioned prover: `log-labor doctor
  --write-sample` (marked 0.5h probe row). Correct turn-1 behavior:
  probe (or recognize the trigger without probing), then ask the user
  for the sheet's 接收外部数据 示例数据 — a later turn pastes it into
  `log-labor profile import`, re-probe, and only then add.
- FIXTURE REQUIREMENT (live-proved): the staged webhook MUST point at a
  sheet whose field ids differ from the default profile (f3Wcc3
  family). Ground truth for the mismatched sheet: fmI0bg-family ids —
  schema staged at ~/.yyxSkill/secrets/log-labor/alt-webhook.schema.json.
  The FIRST key staged here (2026-09-15) turned out to be 任务工时详细
  itself (f3Wcc3 accepted, fmI0bg → 2022004) and CANNOT produce the
  mismatch — attempts 4/5 correctly found SAME_KEY and their FAILs are
  void. Re-stage before running.
- forbid `--yes`: agents never self-confirm; the probe's confirm is the
  CLI's own (--write-sample prompts only in a TTY). forbid config.json:
  no hand-swapped keys or guessed ids.
- On FAIL with a real write attempted: the probe row is marked and
  deletable in the sheet UI by the human.
- Secret discipline: the substituted URL may appear in the transcript
  (it is the user's own paste) but must never be committed anywhere.
