import { useEffect, useState } from "react";
import { API_BASE_URL, api, apiFetch } from "../../services/api";

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

const BIND_KEY = "gohead:sso-bind";

/**
 * A random value this browser keeps for one sign-in: the result is only
 * accepted with it, so a crafted link cannot sign this browser into
 * someone else's account.
 */
function newBind(): string {
  const bytes = new Uint8Array(24);
  crypto.getRandomValues(bytes);
  const bind = Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
  try {
    sessionStorage.setItem(BIND_KEY, bind);
  } catch {
    // Without storage, results cannot be checked: they are refused.
  }
  return bind;
}

function takeBind(): string | null {
  try {
    const bind = sessionStorage.getItem(BIND_KEY);
    sessionStorage.removeItem(BIND_KEY);
    return bind;
  } catch {
    return null;
  }
}

/** Sends the browser to a provider to sign in. */
export function startSSO(provider: string, returnPath = "/dashboard") {
  const q = new URLSearchParams({ return: returnPath, bind: newBind() });
  window.location.assign(`${API_BASE_URL}/auth/sso/${encodeURIComponent(provider)}/start?${q}`);
}

/** Links the signed-in account to a provider: the browser signs in there. */
export async function linkSSO(provider: string, returnPath = "/dashboard") {
  const q = new URLSearchParams({ return: returnPath, bind: newBind() });
  const { data } = await api<{ url: string }>("POST", `/auth/sso/${encodeURIComponent(provider)}/link?${q}`);
  window.location.assign(data.url);
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
  // Only the sign-in this browser started counts.
  const bind = takeBind();
  if (!bind || params.get("bind") !== bind) {
    return { error: token || params.get("bind") ? "This sign-in was not started from this browser: sign in again." : error, returnPath: "/dashboard" };
  }
  const ret = params.get("return") ?? "/dashboard";
  return { token, error, returnPath: ret.startsWith("/") && !ret.startsWith("//") ? ret : "/dashboard" };
}
