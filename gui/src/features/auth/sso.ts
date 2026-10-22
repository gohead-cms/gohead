import { useEffect, useState } from "react";
import { API_BASE_URL, apiFetch } from "../../services/api";

/** An identity provider people can sign in with (GET /auth/sso). */
export interface SSOProvider {
  id: string;
  label: string;
}

/** The enabled providers (none when the API has none, or is older). */
export function useSSOProviders(): SSOProvider[] {
  const [list, setList] = useState<SSOProvider[]>([]);
  useEffect(() => {
    let live = true;
    apiFetch("/auth/sso")
      .then(async (res) => {
        if (!res.ok) return;
        const json = await res.json();
        if (live && Array.isArray(json?.data)) setList(json.data as SSOProvider[]);
      })
      .catch(() => {});
    return () => {
      live = false;
    };
  }, []);
  return list;
}

/** Where the browser goes to sign in with a provider. */
export function ssoStartURL(provider: string, returnPath = "/dashboard"): string {
  return `${API_BASE_URL}/auth/sso/${encodeURIComponent(provider)}/start?${new URLSearchParams({ return: returnPath })}`;
}

/** What a provider sign-in left in the URL fragment: a token, or an error. */
export interface SSOResult {
  token?: string;
  error?: string;
  returnPath: string;
}

/**
 * Reads #sso_token=… or #sso_error=… once, and removes it from the address
 * bar (and history) right away.
 */
export function takeSSOResult(): SSOResult | null {
  const hash = window.location.hash.replace(/^#/, "");
  if (!hash) return null;
  const params = new URLSearchParams(hash);
  const token = params.get("sso_token") ?? undefined;
  const error = params.get("sso_error") ?? undefined;
  if (!token && !error) return null;
  window.history.replaceState(null, "", window.location.pathname + window.location.search);
  const ret = params.get("return") ?? "/dashboard";
  return { token, error, returnPath: ret.startsWith("/") && !ret.startsWith("//") ? ret : "/dashboard" };
}
