import { populated } from "@gohead-cms/client";
import Image from "next/image";
import Link from "next/link";
import { notFound } from "next/navigation";

import { Faq, SectionTitle } from "@/components/Faq";
import { SpeakerCard } from "@/components/Speakers";
import { TalkCard } from "@/components/Talks";
import { dayOf, formatDateRange, getDictionary, isLocale } from "@/i18n";
import { daysBetween, getHomepage, getPosts, getSite, getSponsors, getStats, mediaUrl, setupScreen, type SpeakerEntry, type TalkEntry } from "@/lib/gohead";
import { localHref } from "@/lib/paths";

export default async function Home({ params }: PageProps<"/[locale]">) {
  const { locale } = await params;
  if (!isLocale(locale)) notFound();
  if (await setupScreen()) return null;

  const t = getDictionary(locale);
  const [site, home, stats, news, sponsors] = await Promise.all([
    getSite(locale),
    getHomepage(locale),
    getStats(),
    getPosts(locale, 1, 3),
    getSponsors(locale),
  ]);
  const days = daysBetween(site.startsOn, site.endsOn);
  const dayLabel = (iso: string) => t.day(days.indexOf(dayOf(iso, site.timeZone)) + 1);
  // Featured entries are relations of the singleton, populated two levels
  // deep (level: 2) so a talk comes with its speakers.
  const talks = (home?.featured_talks ?? []).map((x) => populated(x)).filter((x): x is TalkEntry => Boolean(x));
  const speakers = (home?.featured_speakers ?? []).map((x) => populated(x)).filter((x): x is SpeakerEntry => Boolean(x));
  const hero = home?.hero;

  return (
    <main id="main">
      <section className="wrap grid gap-10 pt-14 pb-16 sm:pt-20 lg:grid-cols-12 lg:items-end">
        <div className="lg:col-span-8">
          <p className="eyebrow tabular text-signal">
            {formatDateRange(site.startsOn, site.endsOn, locale)}
            {site.venue ? ` · ${site.venue.name}, ${site.venue.city ?? ""}` : ""}
          </p>
          <h1 className="display mt-5">{hero?.title || site.tagline || site.name}</h1>
          {hero?.text && <p className="mt-7 max-w-[52ch] font-serif text-xl leading-relaxed text-muted sm:text-2xl">{hero.text}</p>}
          <div className="mt-9 flex flex-wrap gap-3">
            {hero?.primary && (
              <Link href={localHref(hero.primary.url, locale)} className="button text-lg">
                {hero.primary.label}
              </Link>
            )}
            {hero?.secondary && (
              <Link href={localHref(hero.secondary.url, locale)} className="button-ghost text-lg">
                {hero.secondary.label}
              </Link>
            )}
          </div>
        </div>
        <dl className="tabular grid grid-cols-3 gap-4 border-t-2 border-ink pt-5 lg:col-span-4 lg:grid-cols-1 lg:border-t-0 lg:border-l-2 lg:pt-0 lg:pl-8">
          {[
            [stats.talks, t.stats.talks],
            [stats.speakers, t.stats.speakers],
            [stats.tracks, t.stats.tracks],
          ].map(([n, label]) => (
            <div key={String(label)}>
              <dt className="sr-only">{label}</dt>
              <dd className="text-5xl font-extrabold tracking-[-0.05em] sm:text-6xl">{n}</dd>
              <dd className="text-sm font-semibold text-muted">{label}</dd>
            </div>
          ))}
        </dl>
      </section>

      {talks.length > 0 && (
        <section aria-labelledby="featured" className="wrap">
          <SectionTitle
            id="featured"
            action={
              <Link href={`/${locale}/schedule`} className="link">
                {t.fullSchedule}
              </Link>
            }
          >
            {t.featuredTalks}
          </SectionTitle>
          <div className="grid gap-4 md:grid-cols-3">
            {talks.map((talk) => (
              <TalkCard key={talk.id} talk={talk} locale={locale} timeZone={site.timeZone} dayLabel={dayLabel(talk.starts_at)} />
            ))}
          </div>
        </section>
      )}

      {speakers.length > 0 && (
        <section aria-labelledby="speakers" className="wrap mt-24">
          <SectionTitle
            id="speakers"
            action={
              <Link href={`/${locale}/speakers`} className="link">
                {t.allSpeakers}
              </Link>
            }
          >
            {t.featuredSpeakers}
          </SectionTitle>
          <div className="grid grid-cols-2 gap-x-4 gap-y-8 md:grid-cols-4">
            {speakers.map((s) => (
              <SpeakerCard key={s.id} speaker={s} locale={locale} />
            ))}
          </div>
        </section>
      )}

      <div className="wrap mt-24 grid gap-16 lg:grid-cols-12">
        {news.data.length > 0 && (
          <section aria-labelledby="news" className="lg:col-span-7">
            <SectionTitle
              id="news"
              action={
                <Link href={`/${locale}/news`} className="link">
                  {t.allNews}
                </Link>
              }
            >
              {t.latestNews}
            </SectionTitle>
            <ul className="border-b border-rule">
              {news.data.map((post) => (
                <li key={post.id} className="group relative border-t border-rule py-5">
                  {post.kind && <p className="eyebrow text-muted">{t.kind[post.kind]}</p>}
                  <h3 className="mt-1 text-xl font-bold tracking-[-0.02em]">
                    <Link href={`/${locale}/news/${post.slug}`} className="after:absolute after:inset-0 group-hover:underline group-hover:decoration-signal group-hover:decoration-2 group-hover:underline-offset-4">
                      {post.title}
                    </Link>
                  </h3>
                  {post.excerpt && <p className="mt-1 text-muted">{post.excerpt}</p>}
                </li>
              ))}
            </ul>
          </section>
        )}
        {home?.faq && home.faq.length > 0 && (
          <section aria-labelledby="faq" className="lg:col-span-5">
            <SectionTitle id="faq">{t.faq}</SectionTitle>
            <Faq items={home.faq} />
          </section>
        )}
      </div>

      {sponsors.length > 0 && (
        <section aria-labelledby="sponsors" className="wrap mt-24">
          <SectionTitle
            id="sponsors"
            action={
              <Link href={`/${locale}/sponsors`} className="link">
                {t.sponsors}
              </Link>
            }
          >
            {t.sponsors}
          </SectionTitle>
          <ul className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
            {sponsors.map((s) => {
              const logo = mediaUrl(s.logo);
              return (
                <li key={s.id} className="grid aspect-[3/1] place-items-center rounded-lg bg-white p-4 shadow-[inset_0_0_0_1px_var(--rule)]">
                  {logo ? <Image src={logo} alt={s.name} width={s.logo?.width ?? 360} height={s.logo?.height ?? 120} className="h-auto max-h-full w-full object-contain" /> : <span className="font-bold text-[#181614]">{s.name}</span>}
                </li>
              );
            })}
          </ul>
        </section>
      )}
    </main>
  );
}
