// toMediaRef converts a media form value to what the API accepts: IDs for
// uploaded files, strings for external URLs.
export function toMediaRef(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(toMediaRef);
  if (value && typeof value === "object" && "id" in value) return (value as { id: number }).id;
  return value;
}
