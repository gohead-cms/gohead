import { Schema } from "../../../shared/types";
import { getJSON, useApiQuery } from "../../../shared/hooks/useApiQuery";

interface CollectionResponse {
  schema: Schema;
}

export function useCollectionsList() {
  const { data, loading, error } = useApiQuery("/admin/collections", async () => {
    const json = await getJSON<CollectionResponse[]>("/admin/collections", "Failed to fetch collections list");
    return Array.isArray(json.data) ? json.data : [];
  });
  return { collections: data ?? [], loading, error };
}
