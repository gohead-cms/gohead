import { useState } from "react";
import { apiFetchWithAuth } from "../../../services/api";
import { useApiQuery } from "../../../shared/hooks/useApiQuery";

export interface ContentItem {
  id: string | number;
  status?: "draft" | "published";
  [key: string]: unknown;
}

interface ContentResponse {
  items: ContentItem[];
  total: number;
}

const PAGE_SIZE = 100;

/**
 * Lists a collection's entries, newest first, PAGE_SIZE at a time.
 * loadMore() adds the next page; switching collections starts over.
 */
export function useCollectionContent(collectionName: string | null) {
  const [pages, setPages] = useState<{ name: string | null; count: number }>({ name: collectionName, count: 1 });
  const pageCount = pages.name === collectionName ? pages.count : 1;
  // The key stays the collection name, so loading more keeps the current
  // entries on screen until the longer list arrives.
  const { data, loading, error, refetch } = useApiQuery(collectionName, async (): Promise<ContentResponse> => {
    const base = `/api/collections/${encodeURIComponent(collectionName!)}`;
    const items: ContentItem[] = [];
    let total = 0;
    let all = true; // editors and admins see drafts; viewers get 403 for status=all
    for (let page = 1; page <= pageCount; page++) {
      const query = `sort=id:desc&page=${page}&pageSize=${PAGE_SIZE}`;
      let res = await apiFetchWithAuth(`${base}?${all ? "status=all&" : ""}${query}`);
      if (res.status === 403 && all) {
        all = false;
        res = await apiFetchWithAuth(`${base}?${query}`);
      }
      if (!res.ok) throw new Error(`Failed to fetch content for ${collectionName}`);
      const json = await res.json();
      const batch: ContentItem[] = Array.isArray(json.data) ? json.data : [];
      items.push(...batch);
      total = Number(json.meta?.pagination?.total ?? items.length);
      if (batch.length < PAGE_SIZE) break;
    }
    return { items, total };
  });

  const content = data ?? { items: [], total: 0 };
  const loadMore = () => {
    setPages({ name: collectionName, count: pageCount + 1 });
    refetch();
  };
  return { content, loading, error, refetch, hasMore: content.items.length < content.total, loadMore };
}
