import { existsSync } from "node:fs";
import { mkdtemp, readFile, rm, writeFile, mkdir } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { afterEach, describe, expect, it } from "vitest";

import { findTemplates, scaffold } from "../src/scaffold.js";

let dirs: string[] = [];
afterEach(async () => {
  await Promise.all(dirs.map((d) => rm(d, { recursive: true, force: true })));
  dirs = [];
});

describe("create", () => {
  it("finds the repository's templates", async () => {
    const t = await findTemplates();
    expect(t.map((x) => x.name)).toContain("nextjs-blog");
  });

  it("scaffolds a configured project", async () => {
    const root = await mkdtemp(join(tmpdir(), "create-gohead-"));
    dirs.push(root);
    const [template] = (await findTemplates()).filter((t) => t.name === "nextjs-blog");
    const target = join(root, "My Blog");
    await scaffold({ directory: target, template: template!, url: "https://cms.example.com", clientRange: "^9.9.9" });

    const pkg = JSON.parse(await readFile(join(target, "package.json"), "utf8"));
    expect(pkg.name).toBe("my-blog");
    expect(pkg.dependencies["@gohead-cms/client"]).toBe("^9.9.9");
    const env = await readFile(join(target, ".env.local"), "utf8");
    expect(env).toMatch(/^GOHEAD_URL=https:\/\/cms\.example\.com$/m);
    expect(env).toMatch(/^GOHEAD_WEBHOOK_SECRET=$/m);
    expect(existsSync(join(target, "gohead", "model.json"))).toBe(true);
    expect(existsSync(join(target, "node_modules"))).toBe(false);
    expect(existsSync(join(target, ".next"))).toBe(false);
  });

  it("refuses a directory that is not empty", async () => {
    const root = await mkdtemp(join(tmpdir(), "create-gohead-"));
    dirs.push(root);
    await mkdir(join(root, "site"));
    await writeFile(join(root, "site", "x"), "");
    const [template] = await findTemplates();
    await expect(scaffold({ directory: join(root, "site"), template: template!, url: "http://localhost:8080" })).rejects.toThrow(/not empty/);
  });
});
