/**
 * Outbound webhooks: verify that a request comes from GoHead.
 *
 * GoHead signs each delivery with the webhook's secret:
 * `X-Gohead-Signature: sha256=<hex HMAC-SHA256 of "<X-Gohead-Timestamp>.<raw body>">`.
 */

/** A webhook delivery body (JSON format). */
export interface WebhookEvent<T = Record<string, unknown>> {
  /** Delivery ID (also X-Gohead-Delivery), stable across retries. */
  id: string;
  event: "item:created" | "item:updated" | "item:deleted" | "item:published" | "item:unpublished" | "webhook:test" | (string & {});
  collection: string;
  occurredAt: string;
  data: T & { id?: number };
  translations?: Record<string, Record<string, unknown>>;
}

export interface VerifyOptions {
  /** The raw request body, exactly as received. */
  body: string | Uint8Array;
  /** The webhook's signing secret (whsec_…). */
  secret: string;
  /** X-Gohead-Signature. */
  signature: string | null | undefined;
  /** X-Gohead-Timestamp (Unix seconds). */
  timestamp: string | null | undefined;
  /** Reject deliveries older (or newer) than this many seconds; default 300, 0 disables the check. */
  toleranceSeconds?: number;
  /** For tests. */
  now?: () => number;
}

const encoder = new TextEncoder();

function toHex(buf: ArrayBuffer): string {
  return Array.from(new Uint8Array(buf), (b) => b.toString(16).padStart(2, "0")).join("");
}

function timingSafeEqual(a: string, b: string): boolean {
  if (a.length !== b.length) return false;
  let diff = 0;
  for (let i = 0; i < a.length; i++) diff |= a.charCodeAt(i) ^ b.charCodeAt(i);
  return diff === 0;
}

/**
 * Verifies a delivery's signature and age. Works wherever Web Crypto is
 * available (Node 18+, Deno, Bun, edge runtimes, browsers).
 */
export async function verifyWebhook(opts: VerifyOptions): Promise<boolean> {
  const { signature, timestamp, secret } = opts;
  if (!signature || !timestamp || !secret || !/^\d+$/.test(timestamp)) return false;
  const tolerance = opts.toleranceSeconds ?? 300;
  if (tolerance > 0) {
    const now = Math.floor((opts.now ?? Date.now)() / 1000);
    if (Math.abs(now - Number(timestamp)) > tolerance) return false;
  }
  const body = typeof opts.body === "string" ? encoder.encode(opts.body) : opts.body;
  const message = new Uint8Array(timestamp.length + 1 + body.length);
  message.set(encoder.encode(timestamp + "."), 0);
  message.set(body, timestamp.length + 1);
  const key = await crypto.subtle.importKey("raw", encoder.encode(secret), { name: "HMAC", hash: "SHA-256" }, false, ["sign"]);
  const expected = "sha256=" + toHex(await crypto.subtle.sign("HMAC", key, message));
  return timingSafeEqual(expected, signature.trim());
}

/**
 * Verifies a Fetch API Request (Next.js route handlers, Remix, Hono, Workers…)
 * and returns its parsed event, or null when the signature is missing or wrong.
 */
export async function parseWebhook<T = Record<string, unknown>>(
  request: Request,
  secret: string,
  options: Pick<VerifyOptions, "toleranceSeconds" | "now"> = {},
): Promise<WebhookEvent<T> | null> {
  const body = await request.text();
  const ok = await verifyWebhook({
    body,
    secret,
    signature: request.headers.get("x-gohead-signature"),
    timestamp: request.headers.get("x-gohead-timestamp"),
    ...options,
  });
  if (!ok) return null;
  try {
    return JSON.parse(body) as WebhookEvent<T>;
  } catch {
    return null;
  }
}
