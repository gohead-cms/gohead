// toMediaRef converts a media form value to what the API accepts: IDs for
// uploaded files, strings for external URLs.
export function toMediaRef(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(toMediaRef);
  if (value && typeof value === "object" && "id" in value) return (value as { id: number }).id;
  return value;
}

/** Whether a MIME type matches one of the attribute's patterns (image/*, application/pdf). */
export function mimeAllowed(mime: string, accept?: string[]): boolean {
  if (!accept || accept.length === 0) return true;
  return accept.some((p) => (p.endsWith("/*") ? mime.startsWith(p.slice(0, -1)) : mime === p));
}
