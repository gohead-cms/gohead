import { useState, useEffect } from "react";
import { apiFetchWithAuth } from "../../../services/api";
import { Schema } from "../../../shared/types/schema"

export function useCollectionSchema(collectionName: string) {
  const [schema, setSchema] = useState<Schema | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!collectionName) {
      setSchema(null);
      return;
    };
    
    setLoading(true);
    setError(null);

    apiFetchWithAuth(`/admin/collections/${collectionName}`)
      .then(async (res) => {
        if (!res.ok) throw new Error("Failed to fetch schema");
        const json = await res.json();
        setSchema(json.data.schema);
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [collectionName]);

  return { schema, loading, error };
}
