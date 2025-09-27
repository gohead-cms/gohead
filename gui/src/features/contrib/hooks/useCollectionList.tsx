import { useState, useEffect } from "react";
import { apiFetchWithAuth } from "../../../services/api";
import { Schema } from "../../../shared/types";

interface CollectionResponse {
    schema: Schema;
}

export function useCollectionsList() {
    const [collections, setCollections] = useState<CollectionResponse[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        setLoading(true);
        setError(null);
        apiFetchWithAuth("/admin/collections")
            .then(res => {
                if (!res.ok) throw new Error("Failed to fetch collections list");
                return res.json();
            })
            .then(json => {
                setCollections(Array.isArray(json.data) ? json.data : []);
            })
            .catch(err => setError(err.message))
            .finally(() => setLoading(false));
    }, []);

    return { collections, loading, error };
}
