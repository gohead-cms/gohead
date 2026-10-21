"use client";

import Link from "next/link";
import { useParams } from "next/navigation";

import { defaultLocale, getDictionary, isLocale } from "@/i18n";

/** The 404 page, inside the site's header and footer. */
export default function NotFound() {
  const params = useParams<{ locale?: string }>();
  const locale = params?.locale && isLocale(params.locale) ? params.locale : defaultLocale;
  const t = getDictionary(locale);
  return (
    <main id="main" className="wrap py-16 sm:py-24">
      <p aria-hidden className="text-[clamp(5rem,18vw,12rem)] leading-none font-extrabold tracking-[-0.06em] text-rule">
        404
      </p>
      <h1 className="mt-2 text-4xl font-extrabold tracking-[-0.035em] sm:text-5xl">{t.notFoundTitle}</h1>
      <p className="mt-4 max-w-[48ch] font-serif text-xl leading-relaxed text-muted">{t.notFoundText}</p>
      <Link href={`/${locale}`} className="mt-8 inline-block rounded-full bg-ink px-6 py-3 font-semibold text-paper hover:opacity-90">
        {t.backHome}
      </Link>
    </main>
  );
}
