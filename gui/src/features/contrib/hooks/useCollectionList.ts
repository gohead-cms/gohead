import type { Schema } from "../../../shared/types";
import { getJSON, useApiQuery } from "../../../shared/hooks/useApiQuery";

interface CollectionResponse {
  schema: Schema;
}

export function useCollectionsList() {
  const { data, loading, error } = useApiQuery("/api/schema/collections", async () => {
    const json = await getJSON<CollectionResponse[]>("/api/schema/collections", "Failed to fetch collections list");
    return Array.isArray(json.data) ? json.data : [];
  });
  return { collections: data ?? [], loading, error };
}
