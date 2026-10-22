/**
 * Finishes the public demo's setup (see render.yaml in this folder), after
 * the conference starter's own setup. Run it from templates/nextjs-conference:
 *
 *   npm run gohead:setup && npm run gohead:agents && node ../../deploy/demo/setup.mjs
 *
 * 1. With GOHEAD_DEMO_GITHUB_TOKEN and GOHEAD_DEMO_GITHUB_REPO (owner/repo) in
 *    .env.local: connects GitHub and adds an agent that opens an issue for
 *    each new talk proposal. Its calls wait for your approval.
 * 2. Takes the demo's snapshot: GoHead goes back to it every night.
 * 3. Prints the settings to paste into the site's environment on Render.
 */
import { adminSession, log } from "../../templates/nextjs-conference/scripts/lib.mjs";

const { url, env, api } = await adminSession();
console.log(`Finishing the demo on ${url}:`);

const status = await api.find("/admin/demo");
if (!status) {
  console.error("GoHead is not in demo mode: set GOHEAD_DEMO_ENABLED=true on it (render.yaml does).");
  process.exit(1);
}

// ---- 1. GitHub: an agent that opens issues, with approval.

const token = env.GOHEAD_DEMO_GITHUB_TOKEN;
const repo = env.GOHEAD_DEMO_GITHUB_REPO;
if (token && repo) {
  const [owner, name] = repo.split("/");
  let conn = await api.find("/admin/connections/github");
  if (!conn) {
    conn = (await api("POST", "/admin/connections", { preset: "github", token })).data;
    log(`connect GitHub (${conn.tools.length} tools)`);
  } else {
    log("keep the GitHub connection");
  }
  // GitHub's MCP server calls it issue_write (method "create") in recent
  // versions, create_issue in earlier ones.
  const tools = new Set(conn.tools.map((t) => t.name));
  const tool = tools.has("issue_write") ? "issue_write" : tools.has("create_issue") ? "create_issue" : null;
  if (conn.lastError || !tool) {
    log(`skip the GitHub agent: ${conn.lastError || "no tool to create issues (check the token's permissions)"}`);
  } else if (await api.find("/admin/agents/cfp-to-github")) {
    log("keep agent cfp-to-github");
  } else {
    await api("POST", "/admin/agents", {
      name: "cfp-to-github",
      description: "Opens a GitHub issue for each new talk proposal (each call waits for approval)",
      system_prompt: [
        "A talk proposal was just sent through the call for papers. The message gives its data.",
        `Call open_issue once to open an issue in the GitHub repository ${owner}/${name}${tool === "issue_write" ? ' (method "create")' : ""}:`,
        '- title: "Talk proposal: <the proposal\'s title>"',
        "- body: Markdown with the speaker's name, the format, and the abstract (at most 600 characters), then the line \"Opened by the GoHead demo's cfp-to-github agent.\"",
        "Use only the proposal's data. Never include an email address.",
      ].join("\n"),
      max_turns: 3,
      llm_config: { provider: env.GOHEAD_LLM_PROVIDER || "openai" },
      memory: { type: "in-memory", session_scope: "agent" },
      trigger: { type: "collection_event", event_trigger: { collection: "proposals", events: ["item:created"] } },
      limits: { max_runs_per_hour: 10, max_runs_per_day: 30, max_tokens_per_run: 20000 },
      tools: [{ use: `github.${tool}`, name: "open_issue" }],
    });
    log(`create agent cfp-to-github (github.${tool}, waits for approval)`);
  }
} else {
  log("no GitHub agent (set GOHEAD_DEMO_GITHUB_TOKEN and GOHEAD_DEMO_GITHUB_REPO to add one)");
}

// ---- 2. The snapshot the nightly reset restores.

const snap = (await api("POST", "/admin/demo/snapshot")).data;
log(`snapshot taken (${snap.snapshotAt}): the demo resets to it every day at ${status.demo.resetAt || "never"} UTC`);

// ---- 3. The site's settings.

const keys = [
  "GOHEAD_FORMS_TOKEN",
  "GOHEAD_PREVIEW_TOKEN",
  "GOHEAD_PREVIEW_SECRET",
  "GOHEAD_WEBHOOK_SECRET",
  "GOHEAD_FEEDBACK_AGENT_ID",
  "GOHEAD_FEEDBACK_AGENT_TOKEN",
];
console.log("\nPaste these into the gohead-demo-site service's Environment on Render, then deploy it:\n");
for (const key of keys) console.log(`${key}=${env[key] ?? ""}`);
console.log("\nTake a new snapshot after changing the demo's content: node ../../deploy/demo/setup.mjs");
