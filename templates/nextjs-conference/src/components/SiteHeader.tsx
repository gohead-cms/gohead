import Link from "next/link";

import { LanguageSwitcher } from "@/components/LanguageSwitcher";
import { formatDateRange, getDictionary, locales, type Locale } from "@/i18n";
import type { SiteSettings } from "@/lib/gohead";
import { localHref } from "@/lib/paths";

/** Name, dates and the navigation of the `site` singleton (edited in GoHead). */
export function SiteHeader({ site, locale }: { site: SiteSettings; locale: Locale }) {
  const t = getDictionary(locale);
  const names = Object.fromEntries(locales.map((l) => [l, getDictionary(l).languageName])) as Record<Locale, string>;
  const dates = formatDateRange(site.startsOn, site.endsOn, locale);
  const nav = (
    <ul className="flex items-center gap-x-5 text-[0.95rem] font-semibold whitespace-nowrap">
      {site.nav.map((item) => (
        <li key={item.url}>
          <Link href={localHref(item.url, locale)} className="decoration-signal decoration-2 underline-offset-[0.3em] hover:underline">
            {item.label}
          </Link>
        </li>
      ))}
    </ul>
  );
  return (
    <header className="border-b border-rule">
      <div className="wrap flex items-center justify-between gap-x-6 gap-y-3 pt-5 pb-4">
        <Link href={`/${locale}`} className="group flex min-w-0 items-center gap-3">
          <span aria-hidden className="grid size-9 shrink-0 place-items-center rounded-full bg-signal text-on-signal">
            <svg viewBox="0 0 24 24" className="size-5" fill="currentColor">
              <circle cx="12" cy="12" r="4" />
              <path d="M12 2a10 10 0 1 0 10 10h-3a7 7 0 1 1-7-7z" />
            </svg>
          </span>
          <span className="flex min-w-0 flex-col leading-tight">
            <span className="text-lg font-extrabold tracking-[-0.03em]">{site.name}</span>
            {dates && (
              <span className="tabular truncate text-xs font-medium text-muted">
                {dates}
                {site.venue?.city ? ` · ${site.venue.city}` : ""}
              </span>
            )}
          </span>
        </Link>
        <div className="flex items-center gap-x-5">
          <nav aria-label={t.menu} className="hidden lg:block">
            {nav}
          </nav>
          <Link href={`/${locale}/search`} className="rounded-full p-2 hover:bg-rule" aria-label={t.search} title={t.search}>
            <svg aria-hidden viewBox="0 0 24 24" className="size-5" fill="none" stroke="currentColor" strokeWidth="2.2">
              <circle cx="11" cy="11" r="7" />
              <path d="m20 20-3.5-3.5" strokeLinecap="round" />
            </svg>
          </Link>
          <LanguageSwitcher current={locale} label={t.language} names={names} />
        </div>
      </div>
      <nav aria-label={t.menu} className="wrap overflow-x-auto pb-3 lg:hidden">
        {nav}
      </nav>
    </header>
  );
}
