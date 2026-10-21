/**
 * npm run gohead:access
 *
 * Who may do what, beyond the public read access of gohead/model.json:
 *
 * - role `site-forms` + API token GOHEAD_FORMS_TOKEN: the site's server
 *   creates questions, proposals, subscribers and uploads, and nothing else;
 * - role `site-preview` + API token GOHEAD_PREVIEW_TOKEN: the site's draft
 *   mode reads unpublished talks and posts;
 * - role `program-committee` + user `committee`: people who review proposals,
 *   draft talks (without publishing them) and moderate questions;
 * - webhook `website revalidation`: GoHead tells the site what changed
 *   (GOHEAD_WEBHOOK_SECRET), plus a disabled Slack-formatted example;
 * - GOHEAD_PREVIEW_SECRET: the shared secret of /api/preview.
 *
 * Generated tokens and secrets are written to .env.local. Running it again
 * updates the roles and keeps everything that already exists.
 */
import { adminSession, log, randomSecret, writeEnv } from "./lib.mjs";

const ROLES = [
  {
    name: "site-forms",
    description: "The website's forms: questions, talk proposals, newsletter sign-ups, uploads",
    permissions: {
      // Every action implies `read` of its type, so this token can also list
      // proposals and subscribers: it stays on the site's server, and it can't
      // reach any other type (feedback, talks…: 403).
      content: { questions: ["create"], proposals: ["create"], subscribers: ["create"] },
      media: ["create"],
    },
  },
  {
    name: "site-preview",
    description: "The website's draft mode: reads unpublished talks and posts",
    permissions: {
      // Drafts are visible to roles that may update a type. The token stays on
      // the site's server; it never reaches a browser.
      content: {
        talks: ["update"],
        posts: ["update"],
        tracks: ["read"],
        speakers: ["read"],
        sponsors: ["read"],
        pages: ["read"],
        questions: ["read"],
        "singleton:site": ["read"],
        "singleton:homepage": ["read"],
        "singleton:tickets": ["read"],
      },
    },
  },
  {
    name: "program-committee",
    description: "Reviews proposals, drafts talks and moderates questions; publishing stays with editors",
    permissions: {
      content: {
        proposals: ["read", "update"],
        // No `publish`: talks they create or edit stay drafts until an editor publishes them.
        talks: ["create", "update"],
        speakers: ["create", "update"],
        questions: ["update", "publish", "delete"],
        "*": ["read"],
      },
      media: ["read", "create"],
    },
  },
];

const { url, env, api } = await adminSession();
console.log(`Setting up access on ${url}:`);

// ---- Roles ----
for (const role of ROLES) {
  const existing = await api.find(`/admin/roles/${role.name}`);
  if (existing) {
    await api("PUT", `/admin/roles/${role.name}`, { description: role.description, permissions: role.permissions });
    log(`update role ${role.name}`);
  } else {
    await api("POST", "/admin/roles", role);
    log(`create role ${role.name}`);
  }
}

// ---- API tokens for the site's server ----
async function token(envKey, name, role, description) {
  if (env[envKey]) {
    log(`keep ${envKey} (already set)`);
    return;
  }
  const res = await api("POST", "/admin/tokens", { name, role, description });
  writeEnv(envKey, res.data.token);
  log(`create token ${name} (role ${role}) → ${envKey} in .env.local`);
}
await token("GOHEAD_FORMS_TOKEN", "website-forms", "site-forms", "Forms of the conference site");
await token("GOHEAD_PREVIEW_TOKEN", "website-preview", "site-preview", "Draft mode of the conference site");

// ---- A person with a custom role ----
// GET /admin/users answers { data: { users: [...] } }.
const users = (await api("GET", "/admin/users")).data?.users ?? [];
if (users.some((u) => u.username === "committee")) {
  log("keep user committee");
} else {
  const password = randomSecret(9);
  await api("POST", "/admin/users", {
    username: "committee",
    email: "committee@example.com",
    password,
    role_name: "program-committee",
  });
  log(`create user committee (role program-committee), password: ${password}`);
}

// ---- Draft mode secret ----
if (!env.GOHEAD_PREVIEW_SECRET) {
  writeEnv("GOHEAD_PREVIEW_SECRET", randomSecret());
  log("create GOHEAD_PREVIEW_SECRET in .env.local");
}

// ---- Webhooks ----
const hooks = (await api("GET", "/admin/webhooks")).data ?? [];
// GoHead calls the site, so the URL is the site as seen from GoHead: from the
// containers of docker-compose.yml, your machine is host.docker.internal.
const target = env.GOHEAD_REVALIDATE_URL || "http://host.docker.internal:3000/api/revalidate";
const revalidation = hooks.find((h) => h.name === "website revalidation");
if (revalidation && env.GOHEAD_WEBHOOK_SECRET) {
  log(`keep webhook "website revalidation" → ${revalidation.url}`);
} else {
  const body = {
    name: "website revalidation",
    url: target,
    events: ["item:published", "item:unpublished", "item:updated", "item:deleted", "singleton:updated"],
    collections: [],
    signed: true,
  };
  // The secret is only shown when it is issued: rotate it if we lost it.
  const res = revalidation
    ? await api("PUT", `/admin/webhooks/${revalidation.id}`, { ...body, rotate_secret: true })
    : await api("POST", "/admin/webhooks", body);
  writeEnv("GOHEAD_WEBHOOK_SECRET", res.data.secret ?? res.meta?.secret);
  log(`${revalidation ? "update" : "create"} webhook "website revalidation" → ${target}, GOHEAD_WEBHOOK_SECRET in .env.local`);
}
if (!hooks.some((h) => h.name === "slack: new talks")) {
  await api("POST", "/admin/webhooks", {
    name: "slack: new talks",
    // Replace with your Slack incoming webhook URL, then enable it.
    url: "https://hooks.slack.com/services/T00000000/B00000000/REPLACE_ME",
    events: ["item:published"],
    collections: ["talks"],
    format: "slack",
    enabled: false,
  });
  log(`create webhook "slack: new talks" (disabled: set its URL, then enable it)`);
}

console.log("Done. Restart `npm run dev` so the site reads the new values of .env.local.");
