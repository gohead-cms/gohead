import Link from "next/link";

import { NewsletterForm } from "@/components/forms/NewsletterForm";
import { formatDateRange, getDictionary, type Locale } from "@/i18n";
import type { SiteSettings } from "@/lib/gohead";

export function SiteFooter({ site, locale }: { site: SiteSettings; locale: Locale }) {
  const t = getDictionary(locale);
  const dates = formatDateRange(site.startsOn, site.endsOn, locale);
  const links = [
    { href: `/${locale}/schedule`, label: t.schedule },
    { href: `/${locale}/speakers`, label: t.speakers },
    { href: `/${locale}/news`, label: t.news },
    { href: `/${locale}/practical-information`, label: locale === "fr" ? "Infos pratiques" : "Practical information" },
    { href: `/${locale}/code-of-conduct`, label: locale === "fr" ? "Code de conduite" : "Code of conduct" },
    { href: `/${locale}/feedback`, label: t.feedback },
  ];
  return (
    <footer className="mt-28 bg-footer text-on-footer">
      <div className="wrap grid gap-12 py-14 md:grid-cols-12">
        <div className="md:col-span-5">
          <p className="text-3xl font-extrabold tracking-[-0.04em]">{site.name}</p>
          {dates && <p className="tabular mt-2 font-semibold">{dates}</p>}
          {site.venue && (
            <address className="mt-4 not-italic opacity-80">
              {site.venue.name}
              {site.venue.address && <br />}
              {site.venue.address}
              {site.venue.city && `, ${site.venue.city}`}
              {site.venue.map_url && (
                <>
                  <br />
                  <a href={site.venue.map_url} className="underline underline-offset-4" rel="noopener noreferrer">
                    {locale === "fr" ? "Voir sur la carte" : "See on the map"}
                  </a>
                </>
              )}
            </address>
          )}
        </div>
        <nav aria-label={t.menu} className="md:col-span-3">
          <ul className="grid gap-2 font-medium">
            {links.map((l) => (
              <li key={l.href}>
                <Link href={l.href} className="underline-offset-4 hover:underline">
                  {l.label}
                </Link>
              </li>
            ))}
            <li>
              <a href={`/${locale}/feed.xml`} className="underline-offset-4 hover:underline">
                {t.rss}
              </a>
            </li>
          </ul>
        </nav>
        <div className="md:col-span-4">
          <p className="text-lg font-bold">{t.newsletter}</p>
          <p className="mt-1 mb-4 opacity-75">{t.newsletterIntro}</p>
          <NewsletterForm locale={locale} />
          {site.social.length > 0 && (
            <ul className="mt-6 flex flex-wrap gap-x-5 gap-y-2 text-sm font-semibold">
              {site.social.map((s) => (
                <li key={s.url}>
                  <a href={s.url} rel="noopener noreferrer" className="underline-offset-4 hover:underline">
                    {s.label}
                  </a>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
      <div className="wrap flex flex-wrap justify-between gap-4 border-t border-white/10 py-6 text-sm opacity-60">
        {site.footer && <p className="max-w-[60ch]">{site.footer}</p>}
        <a href="https://github.com/gohead-cms/gohead" className="font-semibold hover:underline" rel="noopener noreferrer">
          {t.poweredBy}
        </a>
      </div>
    </footer>
  );
}
