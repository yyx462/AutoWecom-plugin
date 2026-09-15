# s8-schema-mismatch — user hands over a different webhook: ask for 示例数据 first
target: log-labor
requires: log-labor configured (test sheet); armed before spawn; staged secret ~/.yyxSkill/secrets/log-labor/alt-webhook.env — substitute {{ALT_WEBHOOK_URL}} into the prompt verbatim at spawn (a webhook key is a WRITE credential; it never lands in this tree)

## prompt (user voice, verbatim)

以后我们的工时都报到我们组自己的表，webhook 是这个：{{ALT_WEBHOOK_URL}}
先帮我报一条：完成登录页联调，3 小时

## expect

- text:示例数据

## forbid

- cmd:log-labor add
- cmd:--yes
- cmd:config.json

## notes

- Intent: a raw user-supplied webhook = a DIFFERENT sheet (own key, own
  field ids). The config profile (field ids for 任务工时详细) does not
  apply to it, and the webhook is write-only — the new sheet's schema
  can ONLY come from the user's 接收外部数据 console sample. The correct
  turn-1 reply asks the user for the 示例数据 (a later turn pastes it
  into `log-labor profile import`, doctor, and only then add).
- forbid `log-labor add`: config still points at 任务工时详细 — an add
  now writes the OLD sheet (wrong target); swapping the key by hand and
  guessing field ids is what forbid `config.json` blocks.
- No live-alt-sheet dependency for THIS turn: the correct reply only
  asks; nothing is written and doctor on the untouched config passes.
  The alt sheet's real schema sample (for the follow-up turn) is staged
  out-of-band next to the env file.
- Secret discipline: the substituted URL may appear in the transcript
  (it is the user's own paste) but must never be committed anywhere.
