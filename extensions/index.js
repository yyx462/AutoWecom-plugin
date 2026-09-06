// plugin/extensions — Pi registerTool stubs (HOW_TO_MIGRATE Design 1,
// handwritten). Each stub is name + args + description ONLY: execution
// is a POST to the broker's /invoke with {name, args, thread, principal}.
// No handler logic, no keys, no backend SDKs — those live broker-side.
//
// Env (injected by the brain around each pi run):
//   BROKER_BASE_URL  where /invoke lives (default http://broker:9990)
//   WECOM_THREAD     chat id of the current turn (thread)
//   WECOM_PRINCIPAL  userid of the human (principal)

const BROKER_BASE_URL = (process.env.BROKER_BASE_URL || "http://broker:9990").replace(/\/$/, "");
const THREAD = process.env.WECOM_THREAD || null;
const PRINCIPAL = process.env.WECOM_PRINCIPAL || null;

async function invoke(name, args, timeoutMs = 60000) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    const resp = await fetch(`${BROKER_BASE_URL}/invoke`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, args: args || {}, thread: THREAD, principal: PRINCIPAL }),
      signal: controller.signal,
    });
    const body = await resp.json().catch(() => ({}));
    if (!resp.ok) {
      throw new Error(`invoke ${name} failed: ${body.error || resp.status}`);
    }
    return body.result;
  } finally {
    clearTimeout(timer);
  }
}

const STRING_QUERY = {
  type: "object",
  properties: { query: { type: "string" } },
  required: ["query"],
};

export default function register(ctx) {
  ctx.registerTool({
    name: "kb_search",
    description:
      "Search the bot's trusted knowledge base (human-approved entries). " +
      "Input: {query: string}. Returns matched entries with source file and section.",
    parameters: STRING_QUERY,
    async execute(args) {
      return invoke("kb_search", { query: String(args.query ?? "") });
    },
  });

  ctx.registerTool({
    name: "repo_search",
    description:
      "Full-text search over the mounted internal repositories selected by repos.yaml. " +
      "Input: {query: string}. Returns per-repo file hits with line numbers.",
    parameters: STRING_QUERY,
    async execute(args) {
      return invoke("repo_search", { query: String(args.query ?? "") });
    },
  });

  ctx.registerTool({
    name: "list_repos",
    description: "List the answerable repositories with their indexed file counts.",
    parameters: { type: "object", properties: {} },
    async execute() {
      return invoke("list_repos", {});
    },
  });
}
