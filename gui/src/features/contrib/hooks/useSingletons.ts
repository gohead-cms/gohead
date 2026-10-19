import { ApiError } from "../../../services/api";
import { getJSON, useApiQuery } from "../../../shared/hooks/useApiQuery";
import type { Attribute } from "../../../shared/types";

/** A singleton (single type) definition, as GET /api/schema/singletons lists it. */
export interface SingletonDef {
  name: string;
  description?: string;
  attributes: Attribute[] | null;
}

/** A singleton's content: its attributes, flat, next to id and timestamps. */
export type SingletonContent = Record<string, unknown> & { id?: number; updatedAt?: string };

/** Lists singletons (enabled=false loads nothing). */
export function useSingletonsList(enabled: boolean) {
  const { data, loading, error } = useApiQuery(enabled ? "/api/schema/singletons" : null, async () => {
    const json = await getJSON<SingletonDef[]>("/api/schema/singletons", "Failed to fetch single types");
    return Array.isArray(json.data) ? json.data : [];
  });
  return { singletons: data ?? [], loading, error };
}

export const singletonPath = (name: string) => `/api/singleton/${encodeURIComponent(name)}`;

/**
 * Loads a singleton's content with relations as stored (level=0), so saving
 * the form sends them back unchanged. Null when it has no content yet (404).
 */
export function useSingletonContent(name: string | null) {
  const path = name ? `${singletonPath(name)}?level=0` : null;
  const { data, loading, error, refetch } = useApiQuery(path, async (): Promise<SingletonContent | null> => {
    try {
      const json = await getJSON<SingletonContent>(path!, "Failed to load the content");
      return json.data ?? null;
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) return null;
      throw err;
    }
  });
  return { content: data, loading, error, refetch };
}
