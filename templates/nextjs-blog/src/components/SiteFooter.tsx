import Link from "next/link";

import { getDictionary, type Locale } from "@/i18n";
import type { SiteSettings } from "@/lib/gohead";

export function SiteFooter({ site, locale }: { site: SiteSettings; locale: Locale }) {
  const t = getDictionary(locale);
  const year = new Date().getFullYear();
  return (
    <footer className="mt-24 bg-footer text-on-footer">
      <div className="wrap grid gap-10 py-14 md:grid-cols-12">
        <div className="md:col-span-6">
          <Link href={`/${locale}`} className="text-3xl font-extrabold tracking-[-0.035em]">
            {site.name}
          </Link>
          {site.description && <p className="mt-3 max-w-[44ch] font-serif text-lg leading-relaxed opacity-80">{site.description}</p>}
        </div>
        <nav aria-label={t.links} className="md:col-span-3">
          <ul className="flex flex-col gap-2">
            {site.links.map((link) => (
              <li key={link.url}>
                <a href={link.url} className="font-medium underline-offset-4 hover:underline">
                  {link.label}
                </a>
              </li>
            ))}
            <li>
              <a href={`/${locale}/feed.xml`} className="font-medium underline-offset-4 hover:underline">
                {t.rss}
              </a>
            </li>
          </ul>
        </nav>
        <div className="flex flex-col gap-2 text-sm opacity-70 md:col-span-3">
          {site.footer && <p>{site.footer}</p>}
          <p>
            © {year} {site.name}
          </p>
        </div>
      </div>
    </footer>
  );
}
