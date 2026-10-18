import { request } from "@playwright/test";
import { admin, apiURL } from "./fixtures";

// Seeds the API: the first registered user becomes admin, then a collection
// with one entry and an agent for the pages under test.
export default async function globalSetup() {
  const api = await request.newContext({ baseURL: apiURL });

  for (let i = 0; ; i++) {
    const res = await api.get("/health").catch(() => null);
    if (res?.ok()) break;
    if (i > 60) throw new Error(`API not reachable at ${apiURL}`);
    await new Promise((r) => setTimeout(r, 1000));
  }

  const reg = await api.post("/auth/register", { data: admin });
  if (!reg.ok() && reg.status() !== 409) {
    throw new Error(`register failed: ${reg.status()} ${await reg.text()}`);
  }
  const loginRes = await api.post("/auth/login", { data: { username: admin.username, password: admin.password } });
  if (!loginRes.ok()) throw new Error(`login failed: ${loginRes.status()} ${await loginRes.text()}`);
  const token = (await loginRes.json()).data.token as string;
  const headers = { Authorization: `Bearer ${token}` };

  const ensure = async (path: string, data: unknown) => {
    const res = await api.post(path, { data, headers });
    if (!res.ok() && res.status() !== 409) throw new Error(`POST ${path}: ${res.status()} ${await res.text()}`);
  };

  await ensure("/admin/collections", {
    name: "articles",
    kind: "collection",
    attributes: {
      title: { type: "string", required: true },
      body: { type: "text" },
      views: { type: "int" },
    },
  });
  await ensure("/api/collections/articles", { data: { title: "Seeded article", body: "Hello", views: 1 } });
  await ensure("/admin/agents", {
    name: "e2e-tagger",
    system_prompt: "Tag new articles.",
    max_turns: 3,
    llm_config: { provider: "openai", model: "gpt-4o-mini", api_key: "sk-e2e" },
    memory: { type: "in-memory", session_scope: "agent" },
    trigger: { type: "collection_event", event_trigger: { collection: "articles", events: ["item:created"] } },
    limits: { max_runs_per_hour: 10 },
    functions: [
      {
        name: "save",
        description: "Save an article",
        impl_key: "collections.upsert_item",
        collections: ["articles"],
        parameters: { type: "object", properties: {} },
      },
    ],
  });
  await api.dispose();
}
