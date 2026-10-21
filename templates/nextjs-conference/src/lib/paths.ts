import { isLocale, type Locale } from "@/i18n";

/**
 * Links written by editors in GoHead (navigation, buttons, Markdown) are
 * site paths without a language, such as `/schedule`: they get the current
 * locale. Full URLs and paths that already start with a locale are kept.
 */
export function localHref(href: string, locale: Locale): string {
  if (!href.startsWith("/") || href.startsWith("//")) return href;
  const first = href.split(/[/?#]/)[1] ?? "";
  return isLocale(first) ? href : `/${locale}${href === "/" ? "" : href}`;
}

export const isExternal = (href: string) => /^https?:\/\//.test(href);
