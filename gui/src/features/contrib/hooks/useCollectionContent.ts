import { apiFetchWithAuth } from "../../../services/api";
import { useApiQuery } from "../../../shared/hooks/useApiQuery";

export interface ContentItem {
  id: string | number;
  status?: "draft" | "published";
  [key: string]: unknown;
}

interface ContentResponse {
  items: ContentItem[];
}

export function useCollectionContent(collectionName: string | null) {
  const { data, loading, error, refetch } = useApiQuery(collectionName, async (): Promise<ContentResponse> => {
    const base = `/api/collections/${encodeURIComponent(collectionName!)}`;
    // Editors and admins see drafts too; viewers get 403 for status=all, so
    // fall back to published content.
    let res = await apiFetchWithAuth(`${base}?status=all&pageSize=100`);
    if (res.status === 403) res = await apiFetchWithAuth(`${base}?pageSize=100`);
    if (!res.ok) throw new Error(`Failed to fetch content for ${collectionName}`);
    const json = await res.json();
    return { items: Array.isArray(json.data) ? json.data : [] };
  });
  return { content: data ?? { items: [] }, loading, error, refetch };
}
