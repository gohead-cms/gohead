import Link from "next/link";

import { LanguageSwitcher } from "@/components/LanguageSwitcher";
import { getDictionary, locales, type Locale } from "@/i18n";
import type { CategoryEntry, SiteSettings } from "@/lib/gohead";

export function SiteHeader({ site, categories, locale }: { site: SiteSettings; categories: CategoryEntry[]; locale: Locale }) {
  const t = getDictionary(locale);
  const names = Object.fromEntries(locales.map((l) => [l, getDictionary(l).languageName])) as Record<Locale, string>;
  return (
    <header className="wrap flex flex-wrap items-center justify-between gap-x-8 gap-y-3 pt-6 pb-5 sm:pt-8">
      <div className="flex min-w-0 items-baseline gap-x-4">
        <Link href={`/${locale}`} className="text-xl font-extrabold tracking-[-0.035em] whitespace-nowrap sm:text-2xl">
          {site.name}
        </Link>
        {site.tagline && <p className="hidden truncate text-sm text-muted md:block">{site.tagline}</p>}
      </div>
      <div className="flex items-center gap-x-6">
        {categories.length > 0 && (
          <nav aria-label={t.sections} className="hidden sm:block">
            <ul className="flex items-center gap-x-5 text-[0.95rem] font-medium">
              {categories.map((c) => (
                <li key={c.id}>
                  <Link href={`/${locale}/categories/${c.slug}`} className="hover:text-link hover:underline">
                    {c.name}
                  </Link>
                </li>
              ))}
            </ul>
          </nav>
        )}
        <LanguageSwitcher current={locale} label={t.language} names={names} />
      </div>
      {categories.length > 0 && (
        <nav aria-label={t.sections} className="-mx-5 w-[calc(100%+2.5rem)] overflow-x-auto px-5 sm:hidden">
          <ul className="flex gap-x-5 text-[0.95rem] font-medium whitespace-nowrap">
            {categories.map((c) => (
              <li key={c.id}>
                <Link href={`/${locale}/categories/${c.slug}`} className="hover:text-link hover:underline">
                  {c.name}
                </Link>
              </li>
            ))}
          </ul>
        </nav>
      )}
    </header>
  );
}
