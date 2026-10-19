import { getJSON, useApiQuery } from "./useApiQuery";

/** A content locale (GET /api/locales). Exactly one is the default. */
export interface Locale {
  id: number;
  code: string;
  name: string;
  isDefault: boolean;
  createdAt: string;
}

export const LOCALES_PATH = "/api/locales";

export async function loadLocales(): Promise<Locale[]> {
  const res = await getJSON<Locale[]>(LOCALES_PATH, "Failed to load locales");
  return Array.isArray(res.data) ? res.data : [];
}

/**
 * The instance's locales, the default first (any signed-in user can read
 * them). enabled=false loads nothing.
 */
export function useLocales(enabled = true) {
  const { data, loading, error, refetch } = useApiQuery(enabled ? LOCALES_PATH : null, loadLocales);
  const locales = data ?? [];
  const defaultLocale = locales.find((l) => l.isDefault) ?? locales[0] ?? null;
  return { locales, defaultLocale, loading, error, refetch };
}

/** "Français (fr)", or the code alone when the locale has no name. */
export function localeLabel(l: Pick<Locale, "code" | "name">): string {
  return l.name && l.name !== l.code ? `${l.name} (${l.code})` : l.code;
}
