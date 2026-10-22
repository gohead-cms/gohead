/** A user as returned by GET /admin/users (a GORM model: ID is upper-case). */
export interface AdminUser {
  ID: number;
  username: string;
  email: string;
  role?: { id?: number; name?: string } | null;
  created_at?: string;
  CreatedAt?: string;
  /** The identity provider the account signs in with ("google"). */
  ssoProvider?: string;
}

/** An API token (never its secret), GET /admin/tokens. */
export interface ApiToken {
  id: number;
  name: string;
  description: string;
  role: string;
  prefix: string;
  status: "active" | "revoked" | "expired" | string;
  createdBy: string;
  createdAt: string;
  expiresAt: string | null;
  lastUsedAt: string | null;
  revokedAt: string | null;
}

/** POST /admin/tokens response: the token plus its secret, shown once. */
export interface CreatedApiToken extends ApiToken {
  token: string;
}

/** Events an outbound webhook can subscribe to. */
export const WEBHOOK_EVENTS = [
  "item:created",
  "item:updated",
  "item:deleted",
  "item:published",
  "item:unpublished",
  "singleton:updated",
  "singleton:deleted",
] as const;

export type WebhookFormat = "json" | "slack";

/** An outbound webhook; header values and the secret are never returned. */
export interface Webhook {
  id: number;
  name: string;
  url: string;
  events: string[];
  collections: string[];
  format: WebhookFormat | string;
  /** Header names only. */
  headers: string[];
  /** Whether deliveries carry an X-Gohead-Signature. */
  signed: boolean;
  enabled: boolean;
  lastStatus: number;
  lastError: string;
  lastDeliveryAt: string | null;
  createdAt: string;
  updatedAt: string;
  /** Only on create, or when a secret was (re)generated. */
  secret?: string;
}

/** Body of POST/PUT /admin/webhooks. Fields left out keep their value on update. */
export interface WebhookInput {
  name: string;
  url: string;
  events: string[];
  collections: string[];
  format: WebhookFormat;
  enabled: boolean;
  signed: boolean;
  headers?: Record<string, string>;
  rotate_secret?: boolean;
}

/** One delivery attempt, GET /admin/webhooks/{id}/deliveries. */
export interface WebhookDelivery {
  id: number;
  webhookId: number;
  deliveryId: string;
  event: string;
  collection: string;
  itemId: number;
  attempt: number;
  statusCode: number;
  error: string;
  durationMs: number;
  createdAt: string;
}

/** POST /admin/webhooks/{id}/test response. */
export interface WebhookTestResult {
  delivered: boolean;
  statusCode: number;
  durationMs: number;
  error?: string;
}

/** A row of the custom headers editor. */
export interface HeaderRow {
  key: string;
  name: string;
  value: string;
}
