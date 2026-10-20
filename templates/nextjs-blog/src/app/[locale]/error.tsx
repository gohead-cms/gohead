"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect } from "react";

import { defaultLocale, getDictionary, isLocale } from "@/i18n";

/** Shown when a page fails to render, e.g. GoHead answered with an error. */
export default function ErrorPage({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  const params = useParams<{ locale?: string }>();
  const locale = params?.locale && isLocale(params.locale) ? params.locale : defaultLocale;
  const t = getDictionary(locale);

  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <main id="main" className="wrap py-16 sm:py-24">
      <h1 className="text-4xl font-extrabold tracking-[-0.035em] sm:text-5xl">{t.errorTitle}</h1>
      <p className="mt-4 max-w-[48ch] font-serif text-xl leading-relaxed text-muted">{t.errorText}</p>
      {error.digest && <p className="mt-3 text-sm text-muted">Digest: {error.digest}</p>}
      <div className="mt-8 flex flex-wrap gap-3">
        <button type="button" onClick={reset} className="rounded-full bg-ink px-6 py-3 font-semibold text-paper hover:opacity-90">
          {t.retry}
        </button>
        <Link href={`/${locale}`} className="rounded-full border border-ink px-6 py-3 font-semibold hover:bg-ink hover:text-paper">
          {t.backHome}
        </Link>
      </div>
    </main>
  );
}
