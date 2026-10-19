import { useCallback, useEffect, useRef, useState } from "react";
import { api, ApiError, Envelope } from "../../services/api";
import { errorMessage } from "../utils/errors";

interface Settled<T> {
  key: string;
  version: number;
  data: T | null;
  error: string | null;
}

export interface ApiQuery<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
  /** Loads the same key again; the current data stays until it arrives. */
  refetch: () => void;
}

/**
 * Loads data for `key` with `load` and re-runs it whenever the key changes.
 * A null key loads nothing. State is only set when a load settles, and a
 * result for a key that is no longer current is ignored, so switching
 * quickly between keys never shows stale data.
 */
export function useApiQuery<T>(key: string | null, load: () => Promise<T>): ApiQuery<T> {
  const [version, setVersion] = useState(0);
  const [settled, setSettled] = useState<Settled<T> | null>(null);
  const loadRef = useRef(load);
  useEffect(() => {
    loadRef.current = load;
  });

  useEffect(() => {
    if (key === null) return;
    let current = true;
    loadRef.current().then(
      (data) => current && setSettled({ key, version, data, error: null }),
      (err: unknown) => current && setSettled({ key, version, data: null, error: errorMessage(err) }),
    );
    return () => {
      current = false;
    };
  }, [key, version]);

  const refetch = useCallback(() => setVersion((v) => v + 1), []);

  if (key === null) return { data: null, loading: false, error: null, refetch };
  const sameKey = settled?.key === key;
  const done = sameKey && settled.version === version;
  return {
    // While refetching the same key, keep showing what was loaded before.
    data: sameKey ? settled.data : null,
    loading: !done,
    error: done ? settled.error : null,
    refetch,
  };
}

/** GETs an API path and returns its JSON envelope. */
export async function getJSON<T = unknown>(path: string, failure = "Request failed"): Promise<Envelope<T>> {
  try {
    return await api<T>("GET", path);
  } catch (err) {
    if (err instanceof ApiError && err.message.startsWith("Request failed")) {
      throw new ApiError(err.status, `${failure} (${err.status})`, err.details);
    }
    throw err;
  }
}
