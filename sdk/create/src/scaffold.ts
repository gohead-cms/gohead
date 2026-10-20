/** Copies starter templates into new projects. */
import { randomBytes } from "node:crypto";
import { existsSync } from "node:fs";
import { cp, readdir, readFile, rename, writeFile } from "node:fs/promises";
import { basename, dirname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

export const here = dirname(fileURLToPath(import.meta.url));
const SKIP = new Set(["node_modules", ".next", ".env.local", "next-env.d.ts", ".turbo", "dist", "out"]);

export interface TemplateInfo {
  name: string;
  description: string;
  dir: string;
}

/** Templates bundled in the package, or the repository's when run from a checkout. */
export async function findTemplates(): Promise<TemplateInfo[]> {
  for (const dir of [resolve(here, "../templates"), resolve(here, "../../../templates")]) {
    if (!existsSync(dir)) continue;
    const out: TemplateInfo[] = [];
    for (const name of (await readdir(dir)).sort()) {
      const pkgPath = join(dir, name, "package.json");
      if (!existsSync(pkgPath)) continue;
      const pkg = JSON.parse(await readFile(pkgPath, "utf8")) as { description?: string };
      out.push({ name, description: pkg.description ?? "", dir: join(dir, name) });
    }
    if (out.length) return out;
  }
  return [];
}

export interface CreateOptions {
  directory: string;
  template: TemplateInfo;
  url: string;
  /** Version range of @gohead-cms/client to depend on. */
  clientRange?: string;
}

/** Copies a template into a new directory and configures it. Returns the files written. */
export async function scaffold(opts: CreateOptions): Promise<void> {
  const target = resolve(opts.directory);
  if (existsSync(target) && (await readdir(target)).length > 0) {
    throw new Error(`${relative(process.cwd(), target) || "."} is not empty: choose another directory`);
  }
  await cp(opts.template.dir, target, {
    recursive: true,
    filter: (src) => !SKIP.has(basename(src)) && !src.endsWith(".tsbuildinfo"),
  });
  if (existsSync(join(target, "_gitignore"))) await rename(join(target, "_gitignore"), join(target, ".gitignore"));

  // package.json: the project's name, a published client version.
  const pkgPath = join(target, "package.json");
  const pkg = JSON.parse(await readFile(pkgPath, "utf8")) as Record<string, unknown> & { dependencies?: Record<string, string> };
  pkg.name = basename(target).toLowerCase().replace(/[^a-z0-9-._~]/g, "-").replace(/^[-._]+/, "") || "gohead-site";
  pkg.version = "0.1.0";
  if (pkg.dependencies?.["@gohead-cms/client"] && opts.clientRange) pkg.dependencies["@gohead-cms/client"] = opts.clientRange;
  await writeFile(pkgPath, JSON.stringify(pkg, null, 2) + "\n");

  // .env.local from .env.example: the GoHead URL and a webhook secret.
  const example = join(target, ".env.example");
  if (existsSync(example)) {
    let env = await readFile(example, "utf8");
    const secret = "whsec_" + randomBytes(24).toString("hex");
    env = setEnv(env, "GOHEAD_URL", opts.url);
    env = setEnv(env, "GOHEAD_WEBHOOK_SECRET", secret);
    await writeFile(join(target, ".env.local"), env);
  }
}

function setEnv(content: string, key: string, value: string): string {
  const re = new RegExp(`^#?\\s*${key}=.*$`, "m");
  return re.test(content) ? content.replace(re, `${key}=${value}`) : `${content.trimEnd()}\n${key}=${value}\n`;
}

