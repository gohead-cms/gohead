import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";

import { SpeakerCard } from "@/components/Speakers";
import { TalkRow } from "@/components/Talks";
import { dayOf, getDictionary, isLocale } from "@/i18n";
import { daysBetween, getSite, search, setupScreen } from "@/lib/gohead";

export async function generateMetadata({ params }: PageProps<"/[locale]/search">): Promise<Metadata> {
  const { locale } = await params;
  return isLocale(locale) ? { title: getDictionary(locale).search, robots: { index: false } } : {};
}

/** Search with GoHead's `$contains` filter (see search() in src/lib/gohead.ts). */
export default async function SearchPage({ params, searchParams }: PageProps<"/[locale]/search">) {
  const { locale } = await params;
  if (!isLocale(locale)) notFound();
  if (await setupScreen()) return null;
  const t = getDictionary(locale);
  const raw = (await searchParams).q;
  const q = (Array.isArray(raw) ? raw[0] : raw)?.trim() ?? "";
  const [site, results] = await Promise.all([getSite(locale), search(q, locale)]);
  const days = daysBetween(site.startsOn, site.endsOn);
  const total = results.talks.length + results.speakers.length + results.posts.length;

  return (
    <main id="main" className="wrap pt-12 sm:pt-16">
      <h1 className="display">{t.search}</h1>
      <form method="get" role="search" className="mt-8 flex max-w-2xl gap-2">
        <label htmlFor="q" className="sr-only">
          {t.search}
        </label>
        <input id="q" name="q" type="search" defaultValue={q} placeholder={t.searchPlaceholder} className="input text-lg" minLength={2} maxLength={80} />
        <button type="submit" className="button">
          {t.search}
        </button>
      </form>
      <p className="hint mt-2">{t.searchHint}</p>

      {q.length >= 2 && (
        <div className="mt-12">
          <h2 className="text-2xl font-bold" aria-live="polite">
            {t.searchResults(q)}
          </h2>
          {total === 0 && <p className="mt-4 text-muted">{t.searchEmpty}</p>}
          {results.talks.length > 0 && (
            <section aria-labelledby="r-talks" className="mt-8">
              <h3 id="r-talks" className="eyebrow text-signal">
                {t.schedule}
              </h3>
              <ol className="mt-2 border-b border-rule">
                {results.talks.map((talk) => (
                  <TalkRow key={talk.id} talk={talk} locale={locale} timeZone={site.timeZone} showDay={t.day(days.indexOf(dayOf(talk.starts_at, site.timeZone)) + 1)} />
                ))}
              </ol>
            </section>
          )}
          {results.speakers.length > 0 && (
            <section aria-labelledby="r-speakers" className="mt-10">
              <h3 id="r-speakers" className="eyebrow text-signal">
                {t.speakers}
              </h3>
              <div className="mt-4 grid grid-cols-2 gap-4 md:grid-cols-4 lg:grid-cols-6">
                {results.speakers.map((s) => (
                  <SpeakerCard key={s.id} speaker={s} locale={locale} />
                ))}
              </div>
            </section>
          )}
          {results.posts.length > 0 && (
            <section aria-labelledby="r-posts" className="mt-10">
              <h3 id="r-posts" className="eyebrow text-signal">
                {t.news}
              </h3>
              <ul className="mt-2 border-b border-rule">
                {results.posts.map((p) => (
                  <li key={p.id} className="border-t border-rule py-4">
                    <Link href={`/${locale}/news/${p.slug}`} className="text-lg font-bold hover:underline">
                      {p.title}
                    </Link>
                    {p.excerpt && <p className="text-muted">{p.excerpt}</p>}
                  </li>
                ))}
              </ul>
            </section>
          )}
        </div>
      )}
    </main>
  );
}
