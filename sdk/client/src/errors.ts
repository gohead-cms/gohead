/**
 * An error answered by GoHead (`{"error": {status, name, message, details}}`),
 * or a network failure (status 0).
 */
export class GoHeadError extends Error {
  /** HTTP status; 0 when the server could not be reached. */
  readonly status: number;
  /** The server's error name, e.g. `ValidationError`, `NotFoundError`. */
  readonly code: string;
  /** Extra information from the server (validation details…). */
  readonly details: unknown;
  /** The request, e.g. `GET /api/collections/articles`. */
  readonly request: string;

  constructor(init: { status: number; code: string; message: string; details?: unknown; request: string; cause?: unknown }) {
    super(init.message, init.cause === undefined ? undefined : { cause: init.cause });
    this.name = "GoHeadError";
    this.status = init.status;
    this.code = init.code;
    this.details = init.details ?? null;
    this.request = init.request;
  }

  get isNotFound(): boolean {
    return this.status === 404;
  }

  get isUnauthorized(): boolean {
    return this.status === 401;
  }

  get isForbidden(): boolean {
    return this.status === 403;
  }

  get isValidation(): boolean {
    return this.status === 400;
  }
}

/** Reports whether err is a GoHeadError (also across package copies). */
export function isGoHeadError(err: unknown): err is GoHeadError {
  return err instanceof GoHeadError || (typeof err === "object" && err !== null && (err as { name?: unknown }).name === "GoHeadError");
}
