import { useState, useEffect, useCallback } from "react";
import { apiFetchWithAuth } from "../../../services/api";

export interface ContentItem {
    id: string | number;
    [key: string]: any;
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
            const res = await apiFetchWithAuth(`/api/collections/${collectionName}`);
            if (!res.ok) throw new Error(`Failed to fetch content for ${collectionName}`);
            const json = await res.json();
            const items = json.data?.items || json.data || [];
            setContent({ items: Array.isArray(items) ? items : [] });
        } catch (err: any) {
            setError(err.message);
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
