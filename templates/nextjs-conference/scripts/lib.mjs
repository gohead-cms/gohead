/**
 * Helpers of the setup scripts: settings from .env.local / .env, an admin
 * session, calls to GoHead's admin API, and writing generated secrets back
 * to .env.local.
 *
 * The content model and sample content go through the `gohead-client` CLI
 * (npm run gohead:push, gohead:seed). Roles, API tokens, webhooks, users and
 * agents are not part of a model file, so these scripts create them with the
 * REST admin API, the same calls the admin UI makes.
 */
import { randomBytes } from "node:crypto";
import { existsSync, readFileSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";

const ENV_FILES = [".env.local", ".env"];
export const ENV_LOCAL = resolve(".env.local");

/** Values of .env.local, then .env; the process environment wins. */
export function loadEnv() {
  const env = {};
  for (const file of [...ENV_FILES].reverse()) {
    if (!existsSync(file)) continue;
    for (const line of readFileSync(file, "utf8").split(/\r?\n/)) {
      const m = /^\s*([A-Z0-9_]+)\s*=\s*(.*)\s*$/.exec(line);
      if (m) env[m[1]] = m[2].replace(/^(['"])(.*)\1$/, "$2");
    }
  }
  return { ...env, ...Object.fromEntries(Object.entries(process.env).filter(([, v]) => v !== undefined && v !== "")) };
}

/** Sets KEY=value in .env.local, replacing the line (commented or not) or appending it. */
export function writeEnv(key, value) {
  let content = existsSync(ENV_LOCAL) ? readFileSync(ENV_LOCAL, "utf8") : "";
  const re = new RegExp(`^#?\\s*${key}=.*$`, "m");
  content = re.test(content) ? content.replace(re, `${key}=${value}`) : `${content.trimEnd()}\n${key}=${value}\n`;
  writeFileSync(ENV_LOCAL, content.startsWith("\n") ? content.slice(1) : content);
}

export const randomSecret = (bytes = 24) => randomBytes(bytes).toString("hex");

export class ApiError extends Error {
  constructor(status, body, request) {
    super(`${request}: ${status} ${body?.error?.message ?? JSON.stringify(body)}`);
    this.status = status;
    this.body = body;
  }
}

/**
 * An admin session: logs in with GOHEAD_USERNAME / GOHEAD_PASSWORD. Token and
 * user management need a user session (an API token can't create other
 * tokens), which is why these scripts sign in rather than take a token.
 */
export async function adminSession() {
  const env = loadEnv();
  const url = (env.GOHEAD_URL || "http://localhost:8080").replace(/\/+$/, "");
  if (!env.GOHEAD_USERNAME || !env.GOHEAD_PASSWORD) {
    throw new Error("set GOHEAD_USERNAME and GOHEAD_PASSWORD (an admin account) in .env.local");
  }
  const call = async (method, path, body, token) => {
    let res;
    try {
      res = await fetch(url + path, {
        method,
        headers: { "Content-Type": "application/json", ...(token ? { Authorization: `Bearer ${token}` } : {}) },
        body: body === undefined ? undefined : JSON.stringify(body),
      });
    } catch (err) {
      throw new Error(`cannot reach GoHead at ${url} (${err.cause?.code ?? err.message}): is it running?`);
    }
    const text = await res.text();
    const json = text ? JSON.parse(text) : null;
    if (!res.ok) throw new ApiError(res.status, json, `${method} ${path}`);
    return json;
  };
  const login = await call("POST", "/auth/login", { username: env.GOHEAD_USERNAME, password: env.GOHEAD_PASSWORD });
  const token = login.data.token;
  const api = (method, path, body) => call(method, path, body, token);
  /** GET that returns null on 404. */
  api.find = async (path) => {
    try {
      return (await api("GET", path)).data;
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) return null;
      throw err;
    }
  };
  return { url, env, api };
}

export const log = (line) => console.log(`  ${line}`);
