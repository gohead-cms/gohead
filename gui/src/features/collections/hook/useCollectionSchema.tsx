import type { Schema } from "../../../shared/types";
import { getJSON, useApiQuery } from "../../../shared/hooks/useApiQuery";

export function useCollectionSchema(collectionName: string | null | undefined) {
  const path = collectionName ? `/admin/collections/${encodeURIComponent(collectionName)}` : null;
  const { data, loading, error } = useApiQuery(path, async () => {
    const json = await getJSON<{ schema: Schema }>(path!, "Failed to fetch schema");
    return json.data?.schema ?? null;
  });
  return { schema: data, loading, error };
}
