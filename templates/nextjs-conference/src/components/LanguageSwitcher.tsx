"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

import { locales, type Locale } from "@/i18n";

/** Links to the current page in every locale (slugs are shared between locales). */
export function LanguageSwitcher({ current, label, names }: { current: Locale; label: string; names: Record<Locale, string> }) {
  const pathname = usePathname() ?? `/${current}`;
  const rest = pathname.split("/").slice(2).join("/");
  return (
    <nav aria-label={label} className="flex items-center gap-1 text-sm">
      {locales.map((locale) => {
        const active = locale === current;
        return (
          <Link
            key={locale}
            href={`/${locale}${rest ? `/${rest}` : ""}`}
            hrefLang={locale}
            lang={locale}
            aria-current={active ? "true" : undefined}
            title={names[locale]}
            className={
              active
                ? "rounded-full bg-ink px-2.5 py-1 font-semibold text-paper"
                : "rounded-full px-2.5 py-1 font-medium text-muted hover:bg-rule hover:text-ink"
            }
          >
            {locale.toUpperCase()}
            <span className="sr-only"> ({names[locale]})</span>
          </Link>
        );
      })}
    </nav>
  );
}
