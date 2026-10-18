import { useState, useEffect, useCallback } from "react";
import { apiFetchWithAuth } from "../../../services/api";
import { errorMessage } from "../../../shared/utils/errors";

export interface ContentItem {
    id: string | number;
    status?: "draft" | "published";
    [key: string]: unknown;
}

interface ContentResponse {
    items: ContentItem[];
}

export function useCollectionContent(collectionName: string | null) {
    const [content, setContent] = useState<ContentResponse>({ items: [] });
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const fetchContent = useCallback(async () => {
        if (!collectionName) {
            setContent({ items: [] });
            return;
        }

        setLoading(true);
        setError(null);
        try {
            // Editors and admins see drafts too; viewers get 403 for
            // status=all, so fall back to published content.
            let res = await apiFetchWithAuth(`/api/collections/${collectionName}?status=all&pageSize=100`);
            if (res.status === 403) {
                res = await apiFetchWithAuth(`/api/collections/${collectionName}?pageSize=100`);
            }
            if (!res.ok) throw new Error(`Failed to fetch content for ${collectionName}`);
            const json = await res.json();
            const items = json.data || [];
            setContent({ items: Array.isArray(items) ? items : [] });
        } catch (err: unknown) {
            setError(errorMessage(err));
        } finally {
            setLoading(false);
        }
    }, [collectionName]);

    useEffect(() => {
        fetchContent();
    }, [fetchContent]);

    // Expose the fetchContent function so it can be called manually
    return { content, loading, error, refetch: fetchContent };
}
