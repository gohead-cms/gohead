#!/usr/bin/env node
/**
 * npm create @gohead-cms@latest [directory] -- [--template nextjs-blog|nextjs-conference] [--url http://localhost:8080] [--no-install] [--yes]
 */
import { spawnSync } from "node:child_process";
import { existsSync } from "node:fs";
import { readFile } from "node:fs/promises";
import { join, resolve } from "node:path";
import { createInterface } from "node:readline/promises";
import { parseArgs } from "node:util";

import { findTemplates, here, scaffold } from "./scaffold.js";

function packageManager(): "npm" | "pnpm" | "yarn" | "bun" {
  const ua = process.env.npm_config_user_agent ?? "";
  if (ua.startsWith("pnpm")) return "pnpm";
  if (ua.startsWith("yarn")) return "yarn";
  if (ua.startsWith("bun")) return "bun";
  return "npm";
}

const bold = (s: string) => (process.stdout.isTTY ? `\x1b[1m${s}\x1b[22m` : s);
const dim = (s: string) => (process.stdout.isTTY ? `\x1b[2m${s}\x1b[22m` : s);

async function main(): Promise<number> {
  const { values, positionals } = parseArgs({
    allowPositionals: true,
    options: {
      template: { type: "string", short: "t" },
      url: { type: "string" },
      install: { type: "boolean", default: true },
      "no-install": { type: "boolean" },
      yes: { type: "boolean", short: "y" },
      help: { type: "boolean", short: "h" },
    },
  });
  const templates = await findTemplates();
  if (values.help) {
    console.log(`Create a website on GoHead.

  npm create @gohead-cms@latest [directory] -- [options]

Options:
  -t, --template <name>  ${templates.map((t) => t.name).join(", ")}
      --url <url>        your GoHead (default http://localhost:8080)
      --no-install       do not install dependencies
  -y, --yes              accept the defaults
`);
    return 0;
  }
  if (templates.length === 0) throw new Error("no templates found in this package");

  const interactive = process.stdin.isTTY && !values.yes;
  const rl = interactive ? createInterface({ input: process.stdin, output: process.stdout }) : null;
  const ask = async (q: string, def: string) => (rl ? (await rl.question(`${q} ${dim(`(${def})`)} `)).trim() || def : def);

  console.log(`\n${bold("GoHead")} — create a website\n`);
  const directory = positionals[0] ?? (await ask("Project directory?", "my-gohead-site"));
  let template = templates.find((t) => t.name === values.template);
  if (values.template && !template) throw new Error(`unknown template "${values.template}" (${templates.map((t) => t.name).join(", ")})`);
  if (!template) {
    if (templates.length === 1 || !rl) template = templates[0]!;
    else {
      templates.forEach((t, i) => console.log(`  ${i + 1}. ${bold(t.name)} ${dim(t.description)}`));
      const pick = Number(await ask("Template?", "1"));
      template = templates[pick - 1] ?? templates[0]!;
    }
  }
  const url = values.url ?? (await ask("GoHead URL?", "http://localhost:8080"));
  rl?.close();

  const ownVersion = (JSON.parse(await readFile(resolve(here, "../package.json"), "utf8")) as { version: string }).version;
  await scaffold({ directory, template, url, clientRange: `^${ownVersion}` });
  console.log(`\nCreated ${bold(directory)} from ${bold(template.name)}.`);

  const pm = packageManager();
  const install = values.install && !values["no-install"];
  if (install) {
    console.log(dim(`\nInstalling dependencies with ${pm}…`));
    const res = spawnSync(pm, ["install"], { cwd: resolve(directory), stdio: "inherit", shell: process.platform === "win32" });
    if (res.status !== 0) console.log(`\nInstalling failed: run ${bold(`${pm} install`)} in ${directory}.`);
  }
  const run = pm === "npm" ? "npm run" : pm;
  const hasCompose = existsSync(join(resolve(directory), "docker-compose.yml"));
  const cd = /^[\w./-]+$/.test(directory) ? directory : JSON.stringify(directory);
  const steps = [
    `cd ${cd}`,
    ...(install ? [] : [`${pm} install`]),
    ...(hasCompose && /^http:\/\/(localhost|127\.0\.0\.1):8080\/?$/.test(url) ? ["docker compose up -d        # GoHead on http://localhost:8080"] : []),
    `${run} gohead:setup     # content model, sample content, types (admin username/password in .env.local)`,
    `${run} dev`,
  ];
  console.log(`\nNext steps:\n\n${steps.map((s) => `  ${s}`).join("\n")}\n\nSee README.md for webhooks and deployment.\n`);
  return 0;
}

main().then(
  (code) => process.exit(code),
  (err: unknown) => {
    console.error(`\nError: ${err instanceof Error ? err.message : String(err)}`);
    process.exit(1);
  },
);
