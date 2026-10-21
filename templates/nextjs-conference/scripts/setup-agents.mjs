/**
 * npm run gohead:agents
 *
 * Creates the conference's seven agents. They use the LLM configured on
 * GoHead (GOHEAD_LLM_PROVIDER, GOHEAD_LLM_MODEL, GOHEAD_LLM_API_KEY…: see
 * docker-compose.yml), so set one before they can run. Agents that already
 * exist are left as they are: edit them in the admin UI (Agents).
 *
 * From templates (rendered by GoHead from a few parameters):
 *   talk-summarizer     talks: abstract → summary          collection events
 *   talk-tagger         talks: title, abstract → tags       collection events
 *   question-moderator  questions: body → flagged, reason   collection events
 *   post-translator     posts → French, needs approval      collection events
 *
 * Written by hand:
 *   proposal-triage     scores new proposals (llm.* helpers, scoped write, limits)
 *   feedback-desk       sorts messages of the feedback form  webhook
 *   weekly-digest       drafts a news post every Monday      cron
 */
import { adminSession, log, randomSecret, writeEnv } from "./lib.mjs";

const { url, env, api } = await adminSession();
console.log(`Setting up agents on ${url}:`);

const exists = async (name) => (await api.find(`/admin/agents/${encodeURIComponent(name)}`)) !== null;

/** Renders a template into an agent definition, then creates it. */
async function fromTemplate(template, name, params) {
  if (await exists(name)) return log(`keep agent ${name}`);
  const { data } = await api("POST", `/admin/agent-templates/${template}/render`, { params: { ...params, name } });
  await api("POST", "/admin/agents", { ...data, name });
  log(`create agent ${name} (template ${template}${data.require_approval ? ", needs approval" : ""})`);
}

async function create(agent) {
  if (await exists(agent.name)) return null;
  const { data } = await api("POST", "/admin/agents", agent);
  log(`create agent ${agent.name} (${agent.trigger.type})`);
  return data;
}

// Hand-written agents inherit the instance's model, key and base URL by
// naming its provider and nothing else (see "LLM providers" in the docs).
const llm = { provider: env.GOHEAD_LLM_PROVIDER || "openai" };
const memory = { type: "in-memory", session_scope: "user" };

// ---- From templates ----

await fromTemplate("summarizer", "talk-summarizer", { collection: "talks", source: "abstract", target: "summary", words: 30 });
await fromTemplate("tagger", "talk-tagger", {
  collection: "talks",
  sources: ["title", "abstract"],
  target: "tags",
  tags: "content modeling, architecture, ci, typescript, design, ux, nextjs, workshop, kubernetes, postgres, operations, webhooks, sqlite, self-hosting, agents, llm, governance, journalism, ai, localization, go, performance, caching, opentelemetry, accessibility, forms, community",
  max: 4,
});
await fromTemplate("moderator", "question-moderator", { collection: "questions", sources: ["name", "body"], flag: "flagged", reason: "flag_reason" });
// The translator proposes its translations: editors approve them in Approvals.
await fromTemplate("translator", "post-translator", { collection: "posts", fields: ["title", "excerpt", "body"], locales: "fr" });

// ---- proposal-triage: collection event, AI helpers, a scoped write, limits ----
//
// Tools name what the agent may do ("use") and where ("on"); GoHead tells the
// model what each one does and takes. "name" is the name the prompt uses.

await create({
  name: "proposal-triage",
  description: "Scores new talk proposals for the programme committee",
  system_prompt: [
    "A talk proposal was just sent through the call for papers. The message gives its data.",
    "1. Call score_quality with the proposal's title and abstract as text, and these criteria: clear audience, concrete content, fits a conference about headless CMSs.",
    "2. Call save_review once: collection_name \"proposals\", item_id the proposal's ID, data {\"review_score\": <0-10>, \"review_notes\": \"<two short sentences for the committee>\", \"decision\": \"pending\"}.",
    "Never change any other field and never contact the speaker.",
  ].join("\n"),
  max_turns: 4,
  llm_config: llm,
  memory,
  trigger: { type: "collection_event", event_trigger: { collection: "proposals", events: ["item:created"] } },
  limits: { max_runs_per_hour: 30, max_runs_per_day: 200, max_tokens_per_run: 20000 },
  tools: [
    { use: "ai.score" },
    // Enforced by GoHead, whatever the prompt says: this tool can only touch proposals.
    { use: "content.update", on: ["proposals"], name: "save_review", description: "Saves the committee notes on the proposal" },
  ],
});

// ---- feedback-desk: webhook trigger ----

const feedbackToken = env.GOHEAD_FEEDBACK_AGENT_TOKEN || randomSecret(32);
const desk = await create({
  name: "feedback-desk",
  description: "Sorts the messages of the site's feedback form",
  system_prompt: [
    "The message holds a visitor's feedback, sent by the conference website: {\"message\": …, \"email\": …}.",
    "1. Call sentiment with the message.",
    "2. Call classify with the message and the categories venue, talks, food, organisation, other.",
    "3. Call save_feedback once: collection_name \"feedback\", data {\"message\": <the message, unchanged>, \"email\": <the email, or leave it out>, \"sentiment\": positive|neutral|negative, \"topic\": <the category>, \"summary\": <one sentence>}.",
  ].join("\n"),
  max_turns: 5,
  llm_config: llm,
  memory,
  trigger: { type: "webhook", webhook_token: feedbackToken },
  limits: { max_runs_per_hour: 120 },
  tools: [
    { use: "ai.sentiment", name: "sentiment" },
    { use: "ai.classify", name: "classify" },
    { use: "content.save", on: ["feedback"], name: "save_feedback", description: "Stores the sorted feedback" },
  ],
});
if (desk) {
  // The site posts the form to /agents/webhook/<id> with this token (src/app/[locale]/feedback/actions.ts).
  writeEnv("GOHEAD_FEEDBACK_AGENT_ID", String(desk.id ?? desk.ID));
  writeEnv("GOHEAD_FEEDBACK_AGENT_TOKEN", feedbackToken);
  log("GOHEAD_FEEDBACK_AGENT_ID and GOHEAD_FEEDBACK_AGENT_TOKEN in .env.local");
} else {
  log("keep agent feedback-desk");
}

// ---- weekly-digest: cron trigger ----

const digest = await create({
  name: "weekly-digest",
  description: "Drafts a news post about the programme every Monday morning",
  system_prompt: [
    "It is Monday morning. Write this week's news post about the conference programme.",
    "1. Call list_talks (collection_name \"talks\", page 1, pageSize 50) and pick two or three sessions worth highlighting.",
    "2. Call save_post once: collection_name \"posts\", data {\"title\": …, \"slug\": \"week-<ISO week number>-highlights\", \"kind\": \"programme\", \"excerpt\": <one sentence>, \"body\": <Markdown, under 150 words, linking each talk as /en/talks/<slug>>}.",
    "The post is saved as a draft: an editor reviews and publishes it. If save_post says the slug is taken, stop.",
  ].join("\n"),
  max_turns: 4,
  llm_config: llm,
  memory,
  // Standard cron, in UTC: 07:00 every Monday.
  trigger: { type: "cron", cron: "0 7 * * 1" },
  limits: { max_runs_per_day: 3 },
  tools: [
    { use: "content.list", on: ["talks"], name: "list_talks" },
    { use: "content.save", on: ["posts"], name: "save_post", description: "Saves a news post (as a draft)" },
  ],
});
if (!digest) log("keep agent weekly-digest");

console.log(
  "Done. Agents run in GoHead's worker with the LLM configured there; try one first with a test run (Agents → an agent → Test run).",
);
