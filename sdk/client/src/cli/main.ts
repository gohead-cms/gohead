#!/usr/bin/env node
/**
 * gohead-client: type generation, content models as code and sample content.
 *
 *   npx @gohead-cms/client types --out src/gohead.d.ts
 *   npx @gohead-cms/client model pull --out gohead/model.json
 *   npx @gohead-cms/client model push gohead/model.json
 *   npx @gohead-cms/client seed gohead/seed.json
 *   npx @gohead-cms/client token create --name website --role viewer
 */
import { existsSync } from "node:fs";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { parseArgs } from "node:util";

import { createClient, type GoHeadClient } from "../client.js";
import { GoHeadError } from "../errors.js";
import { pullModel, pushModel, validateModel, type Model } from "./model.js";
import { readSeed, seed, seedBaseDir } from "./seed.js";
import { generateTypes } from "./typegen.js";

const HELP = `gohead-client — GoHead from the command line

Usage:
  gohead-client types [--out <file>] [--model <model.json>]
      Generate TypeScript types from an instance's content model (or a model file).
      Default output: gohead.d.ts
  gohead-client model pull [--out <file>]
      Write the instance's content model to a file (default: gohead/model.json).
  gohead-client model push [<file>] [--dry-run] [--allow-removals] [--allow-publish-drafts]
      Create or update locales, components, collections, singletons and public
      access from a model file (default: gohead/model.json). Never deletes types.
      Removing attributes or public access needs --allow-removals; turning draft
      and publish off (which publishes drafts) needs --allow-publish-drafts.
  gohead-client seed [<file>] [--force]
      Upload media and create sample entries (default: gohead/seed.json).
      Collections that already have entries are skipped unless --force.
  gohead-client token create --name <name> [--role viewer] [--days <n>]
      Create an API token and print it (needs a user session).

Connection (flags or environment; .env.local and .env are read):
  --url        GOHEAD_URL          default http://localhost:8080
  --token      GOHEAD_TOKEN        an API token (ght_…) or session token
  --username   GOHEAD_USERNAME     sign in instead of a token
  --password   GOHEAD_PASSWORD     (GOHEAD_ADMIN_USERNAME / _PASSWORD also work)

model push, the public access it sets, and token create need an admin
user (username and password): API tokens cannot manage roles or tokens.
`;

function loadEnvFiles(): void {
  const load = (process as unknown as { loadEnvFile?: (p: string) => void }).loadEnvFile;
  if (!load) return;
  for (const f of [".env.local", ".env"]) {
    if (existsSync(f)) {
      try {
        load(f);
      } catch {
        // ignore malformed files
      }
    }
  }
}

interface Conn {
  url?: string;
  token?: string;
  username?: string;
  password?: string;
}

async function connect(c: Conn, needUser = false): Promise<GoHeadClient> {
  const url = c.url ?? process.env.GOHEAD_URL ?? "http://localhost:8080";
  const username = c.username ?? process.env.GOHEAD_USERNAME ?? process.env.GOHEAD_ADMIN_USERNAME;
  const password = c.password ?? process.env.GOHEAD_PASSWORD ?? process.env.GOHEAD_ADMIN_PASSWORD;
  let token = c.token ?? process.env.GOHEAD_TOKEN;
  const client = createClient({ url, timeoutMs: 60_000 });
  if ((needUser || !token) && username && password) {
    token = await client.login(username, password);
  } else if (needUser && (!username || !password)) {
    throw new Error("this command needs an admin user: set --username and --password (or GOHEAD_USERNAME / GOHEAD_PASSWORD)");
  }
  if (!token) throw new Error("no credentials: set --token (GOHEAD_TOKEN), or --username and --password");
  return client.with({ token });
}

async function writeOut(path: string, content: string): Promise<void> {
  await mkdir(dirname(resolve(path)), { recursive: true });
  await writeFile(path, content);
}

async function readModel(path: string): Promise<Model> {
  const model = JSON.parse(await readFile(path, "utf8")) as Model;
  validateModel(model);
  return model;
}

const log = (line: string) => console.log(`  ${line}`);

async function main(argv: string[]): Promise<number> {
  loadEnvFiles();
  const { values, positionals } = parseArgs({
    args: argv,
    allowPositionals: true,
    options: {
      url: { type: "string" },
      token: { type: "string" },
      username: { type: "string" },
      password: { type: "string" },
      out: { type: "string", short: "o" },
      model: { type: "string" },
      "dry-run": { type: "boolean" },
      "allow-removals": { type: "boolean" },
      "allow-publish-drafts": { type: "boolean" },
      force: { type: "boolean" },
      name: { type: "string" },
      role: { type: "string" },
      days: { type: "string" },
      help: { type: "boolean", short: "h" },
    },
  });
  const conn: Conn = { url: values.url, token: values.token, username: values.username, password: values.password };
  const [cmd, sub, arg] = positionals;

  if (values.help || !cmd || cmd === "help") {
    console.log(HELP);
    return 0;
  }

  switch (cmd) {
    case "types": {
      const out = values.out ?? "gohead.d.ts";
      let model: Model;
      let source: string;
      if (values.model) {
        model = await readModel(values.model);
        source = values.model;
      } else {
        const client = await connect(conn);
        model = await pullModel(client);
        source = client.url;
      }
      await writeOut(out, generateTypes(model, { source }));
      const n = Object.keys(model.collections ?? {}).length + Object.keys(model.singletons ?? {}).length;
      console.log(`Wrote ${out} (${n} content types).`);
      return 0;
    }
    case "model": {
      if (sub === "pull") {
        const out = values.out ?? "gohead/model.json";
        const model = await pullModel(await connect(conn));
        await writeOut(out, JSON.stringify({ $schema: "https://gohead-cms.github.io/gohead/schemas/model.json", ...model }, null, 2) + "\n");
        console.log(`Wrote ${out}.`);
        return 0;
      }
      if (sub === "push") {
        const file = arg ?? "gohead/model.json";
        const model = await readModel(file);
        const client = await connect(conn, true);
        console.log(`${values["dry-run"] ? "Would apply" : "Applying"} ${file} to ${client.url}:`);
        const plan = await pushModel(client, model, {
          dryRun: values["dry-run"],
          allowRemovals: values["allow-removals"],
          allowPublishDrafts: values["allow-publish-drafts"],
          log,
        });
        const changes = plan.filter((s) => s.action !== "unchanged").length;
        console.log(changes ? `${values["dry-run"] ? "Nothing applied" : "Done"} (${changes} change${changes > 1 ? "s" : ""}).` : "  nothing to change.");
        return 0;
      }
      break;
    }
    case "seed": {
      const file = sub ?? "gohead/seed.json";
      const data = await readSeed(file);
      const client = await connect(conn);
      console.log(`Seeding ${client.url} from ${file}:`);
      const r = await seed(client, data, { baseDir: seedBaseDir(file), force: values.force, log });
      console.log(`Done: ${r.entries} entries, ${r.singletons} singletons, ${r.media} files.`);
      return 0;
    }
    case "token": {
      if (sub !== "create") break;
      if (!values.name) throw new Error("token create needs --name <name>");
      let days: number | undefined;
      if (values.days !== undefined) {
        days = Number(values.days);
        if (!Number.isInteger(days) || days < 1 || days > 3650) throw new Error("--days must be a whole number of days between 1 and 3650");
      }
      const client = await connect(conn, true);
      const body: Record<string, unknown> = { name: values.name, role: values.role ?? "viewer" };
      if (days !== undefined) body.expires_in_days = days;
      const res = await client.request<{ token: string }>("POST", "/admin/tokens", { body });
      console.log(res.data.token);
      return 0;
    }
  }
  console.error(`Unknown command: ${positionals.join(" ")}\n`);
  console.error(HELP);
  return 2;
}

main(process.argv.slice(2)).then(
  (code) => process.exit(code),
  (err: unknown) => {
    if (err instanceof GoHeadError) {
      console.error(`Error: ${err.message}${err.details ? ` (${typeof err.details === "string" ? err.details : JSON.stringify(err.details)})` : ""}`);
    } else {
      console.error(`Error: ${err instanceof Error ? err.message : String(err)}`);
    }
    process.exit(1);
  },
);
