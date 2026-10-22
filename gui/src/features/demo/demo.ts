import { useEffect, useState } from "react";
import { apiFetch } from "../../services/api";

/** GET /auth/demo: how to sign in to a public demo. */
export interface DemoInfo {
  username: string;
  password: string;
  /** Daily reset, "HH:MM" UTC. */
  resetAt?: string;
  nextReset?: string;
}

/** The demo's sign-in details, or null when the instance is not a demo. */
export function useDemoInfo(): DemoInfo | null {
  const [info, setInfo] = useState<DemoInfo | null>(null);
  useEffect(() => {
    let live = true;
    apiFetch("/auth/demo")
      .then(async (res) => {
        if (!res.ok) return;
        const json = await res.json();
        if (live && json?.data?.username) setInfo(json.data as DemoInfo);
      })
      .catch(() => {});
    return () => {
      live = false;
    };
  }, []);
  return info;
}

/** "03:00 UTC (in 5 h)": when the demo resets next. */
export function resetLabel(info: DemoInfo, now = Date.now()): string {
  if (!info.resetAt) return "";
  if (!info.nextReset) return `${info.resetAt} UTC`;
  const hours = Math.max(0, Math.round((new Date(info.nextReset).getTime() - now) / 3_600_000));
  return `${info.resetAt} UTC (${hours < 1 ? "within the hour" : `in ${hours} h`})`;
}
