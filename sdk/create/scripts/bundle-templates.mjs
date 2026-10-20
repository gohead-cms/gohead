// Copies the repository's templates/ into the package (npm pack), renaming
// dotfiles npm would drop or rewrite (.gitignore → _gitignore).
import { cp, mkdir, readdir, rename, rm, stat } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const source = resolve(here, "../../../templates");
const target = resolve(here, "../templates");
const skip = new Set(["node_modules", ".next", ".env.local", "next-env.d.ts", ".turbo", "dist", "out"]);

await rm(target, { recursive: true, force: true });
await mkdir(target, { recursive: true });
for (const name of await readdir(source)) {
  if (!(await stat(join(source, name))).isDirectory()) continue;
  await cp(join(source, name), join(target, name), {
    recursive: true,
    filter: (src) => !skip.has(src.split(/[\\/]/).pop() ?? "") && !/\.tsbuildinfo$/.test(src),
  });
  const gi = join(target, name, ".gitignore");
  try {
    await rename(gi, join(target, name, "_gitignore"));
  } catch {
    // no .gitignore
  }
  console.log(`bundled template ${name}`);
}
