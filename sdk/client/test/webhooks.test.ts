import { createHmac } from "node:crypto";

import { describe, expect, it } from "vitest";

import { parseWebhook, verifyWebhook } from "../src/index.js";

const secret = "whsec_test";
const body = JSON.stringify({ id: "d1", event: "item:published", collection: "articles", occurredAt: "2026-10-20T00:00:00Z", data: { id: 1, title: "Hi" } });
const now = 1_760_000_000;
const sign = (ts: number, b = body, s = secret) => "sha256=" + createHmac("sha256", s).update(`${ts}.${b}`).digest("hex");

describe("webhooks", () => {
  it("accepts GoHead's signature (same algorithm as the server)", async () => {
    expect(await verifyWebhook({ body, secret, signature: sign(now), timestamp: String(now), now: () => now * 1000 })).toBe(true);
  });

  it("refuses wrong signatures, other secrets, tampered bodies and old deliveries", async () => {
    const base = { body, secret, timestamp: String(now), now: () => now * 1000 };
    expect(await verifyWebhook({ ...base, signature: sign(now, body, "other") })).toBe(false);
    expect(await verifyWebhook({ ...base, signature: sign(now), body: body + " " })).toBe(false);
    expect(await verifyWebhook({ ...base, signature: null })).toBe(false);
    expect(await verifyWebhook({ ...base, signature: sign(now - 600), timestamp: String(now - 600) })).toBe(false);
    expect(await verifyWebhook({ ...base, signature: sign(now - 600), timestamp: String(now - 600), toleranceSeconds: 0 })).toBe(true);
  });

  it("parses verified requests", async () => {
    const req = (sig: string) =>
      new Request("http://site/api/revalidate", { method: "POST", body, headers: { "x-gohead-signature": sig, "x-gohead-timestamp": String(now) } });
    const ev = await parseWebhook(req(sign(now)), secret, { now: () => now * 1000 });
    expect(ev?.event).toBe("item:published");
    expect(ev?.data.id).toBe(1);
    expect(await parseWebhook(req("sha256=00"), secret, { now: () => now * 1000 })).toBeNull();
  });
});
