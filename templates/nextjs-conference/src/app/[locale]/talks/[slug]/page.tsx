import { populated } from "@gohead-cms/client";
import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";

import { Avatar, Cover } from "@/components/Cover";
import { QuestionForm } from "@/components/forms/QuestionForm";
import { Markdown } from "@/components/Markdown";
import { TrackChip } from "@/components/Talks";
import { dayOf, defaultLocale, formatDay, formatTime, getDictionary, isLocale, locales } from "@/i18n";
import { formsConfigured } from "@/lib/forms";
import { daysBetween, getQuestions, getSite, getSlugs, getTalk, mediaUrl, setupScreen, speakersOf, stringsOf, trackOf } from "@/lib/gohead";

export async function generateStaticParams() {
  const slugs = await getSlugs("talks");
  return locales.flatMap((locale) => slugs.map((slug) => ({ locale, slug })));
}

export async function generateMetadata({ params }: PageProps<"/[locale]/talks/[slug]">): Promise<Metadata> {
  const { locale, slug } = await params;
  if (!isLocale(locale) || (await setupScreen())) return {};
  const talk = await getTalk(slug, locale);
  if (!talk) return {};
  // `seo` is a shared (not localized) component: its texts are in the default locale.
  const seo = locale === defaultLocale ? talk.seo : null;
  const title = seo?.meta_title || talk.title;
  const description = seo?.meta_description || talk.summary || undefined;
  const image = mediaUrl(talk.seo?.share_image) ?? mediaUrl(populated(talk.gallery?.[0]));
  return {
    title: { absolute: title },
    description,
    alternates: { canonical: `/${locale}/talks/${slug}`, languages: Object.fromEntries(locales.map((l) => [l, `/${l}/talks/${slug}`])) },
    openGraph: { type: "article", title, description, images: image ? [{ url: image }] : undefined },
    // Drafts are only reachable in draft mode; keep them out of search engines anyway.
    robots: talk.status === "draft" ? { index: false } : undefined,
  };
}

export default async function TalkPage({ params }: PageProps<"/[locale]/talks/[slug]">) {
  const { locale, slug } = await params;
  if (!isLocale(locale)) notFound();
  if (await setupScreen()) return null;

  const talk = await getTalk(slug, locale);
  if (!talk) notFound();

  const t = getDictionary(locale);
  const [site, questions] = await Promise.all([getSite(locale), getQuestions(talk.slug)]);
  const track = trackOf(talk);
  const speakers = speakersOf(talk);
  const tags = stringsOf(talk.tags);
  const slides = populated(talk.slides);
  const gallery = (talk.gallery ?? []).map((m) => populated(m)).filter((m) => m !== undefined);
  const days = daysBetween(site.startsOn, site.endsOn);
  const day = dayOf(talk.starts_at, site.timeZone);
  const end = new Date(new Date(talk.starts_at).getTime() + talk.duration_minutes * 60_000).toISOString();

  return (
    <main id="main">
      <article>
        <header className="border-b border-rule">
          <div className="wrap pt-12 pb-10 sm:pt-16">
            <p className="eyebrow flex flex-wrap gap-x-3 gap-y-1">
              <span className="text-signal">{t.format[talk.format]}</span>
              {talk.level && <span className="text-muted">{t.level[talk.level]}</span>}
              {talk.status === "draft" && <span className="rounded bg-signal px-1.5 text-on-signal">{t.draftBadge}</span>}
            </p>
            <h1 className="mt-4 max-w-[22ch] text-[clamp(2.3rem,5.5vw,4.5rem)] leading-[0.95] font-extrabold tracking-[-0.045em] text-balance">
              {talk.title}
            </h1>
            <dl className="tabular mt-10 grid gap-x-10 gap-y-5 sm:grid-cols-2 lg:grid-cols-4">
              <div>
                <dt className="text-sm text-muted">{t.when}</dt>
                <dd className="font-semibold">
                  <span className="capitalize">
                    {days.includes(day) ? `${t.day(days.indexOf(day) + 1)} · ` : ""}
                    {formatDay(day, locale)}
                  </span>
                  <br />
                  <time dateTime={talk.starts_at}>{formatTime(talk.starts_at, locale, site.timeZone)}</time>–
                  <time dateTime={end}>{formatTime(end, locale, site.timeZone)}</time>
                </dd>
              </div>
              {talk.room && (
                <div>
                  <dt className="text-sm text-muted">{t.where}</dt>
                  <dd className="font-semibold">
                    {t.room[talk.room]}
                    {site.venue && (
                      <>
                        <br />
                        <span className="font-normal text-muted">{site.venue.name}</span>
                      </>
                    )}
                  </dd>
                </div>
              )}
              {track && (
                <div>
                  <dt className="text-sm text-muted">{t.track}</dt>
                  <dd>
                    <TrackChip track={track} href={`/${locale}/schedule?track=${track.slug}`} />
                  </dd>
                </div>
              )}
              {(slides || talk.recording_url) && (
                <div className="flex flex-col gap-1.5">
                  {slides && (
                    <a href={mediaUrl(slides)} className="link" download>
                      {t.slides}
                    </a>
                  )}
                  {talk.recording_url && (
                    <a href={talk.recording_url} className="link" rel="noopener noreferrer">
                      {t.recording}
                    </a>
                  )}
                </div>
              )}
            </dl>
          </div>
        </header>

        <div className="wrap mt-12 grid gap-14 lg:grid-cols-12">
          <div className="min-w-0 lg:col-span-7">
            {talk.summary && (
              <aside className="mb-10 rounded-lg bg-paper-raised p-5 shadow-[inset_0_0_0_1px_var(--rule)]">
                <p className="eyebrow text-signal">{t.inShort}</p>
                <p className="mt-2 font-serif text-xl leading-relaxed">{talk.summary}</p>
                <p className="hint mt-2">{t.inShortNote}</p>
              </aside>
            )}
            <Markdown>{talk.abstract}</Markdown>
            {tags.length > 0 && (
              <ul className="mt-10 flex flex-wrap gap-2" aria-label="Tags">
                {tags.map((tag) => (
                  <li key={tag} className="rounded-full px-3 py-1 text-sm text-muted shadow-[inset_0_0_0_1px_var(--rule)]">
                    #{tag}
                  </li>
                ))}
              </ul>
            )}
          </div>

          <aside className="lg:col-span-4 lg:col-start-9">
            <h2 className="eyebrow text-muted">{t.speakers}</h2>
            <ul className="mt-4 grid gap-6">
              {speakers.map((s) => (
                <li key={s.id} className="group relative flex gap-4">
                  <Avatar media={s.photo} name={s.name} size={64} />
                  <div>
                    <p className="text-lg font-bold">
                      <Link href={`/${locale}/speakers/${s.slug}`} className="after:absolute after:inset-0 group-hover:underline">
                        {s.name}
                      </Link>
                    </p>
                    {s.headline && <p className="text-sm text-muted">{s.headline}</p>}
                    {s.company && <p className="text-sm font-medium">{s.company}</p>}
                  </div>
                </li>
              ))}
            </ul>
          </aside>
        </div>

        {gallery.length > 0 && (
          <section aria-labelledby="photos" className="wrap mt-16">
            <h2 id="photos" className="eyebrow text-muted">
              {t.photos}
            </h2>
            <div className="mt-4 grid grid-cols-2 gap-3 md:grid-cols-3">
              {gallery.map((m) => (
                <Cover key={m.id} media={m} ratio="16 / 10" sizes="(min-width: 768px) 33vw, 50vw" />
              ))}
            </div>
          </section>
        )}
      </article>

      <section aria-labelledby="questions" className="wrap mt-20 grid gap-12 lg:grid-cols-12">
        <div className="lg:col-span-7">
          <h2 id="questions" className="text-3xl font-extrabold tracking-[-0.04em]">
            {t.questions}
          </h2>
          {questions.length === 0 && <p className="mt-4 text-muted">{t.noQuestions}</p>}
          <ul className="mt-6 grid gap-4">
            {questions.map((q) => (
              <li key={q.id} className="rounded-lg bg-paper-raised p-5 shadow-[inset_0_0_0_1px_var(--rule)]">
                <p className="font-serif text-lg leading-relaxed">{q.body}</p>
                <p className="mt-2 text-sm font-semibold text-muted">— {q.name}</p>
                {q.answer && (
                  <div className="mt-4 border-l-4 border-signal pl-4">
                    <p className="eyebrow text-signal">{t.answer}</p>
                    <p className="mt-1 leading-relaxed">{q.answer}</p>
                  </div>
                )}
              </li>
            ))}
          </ul>
        </div>
        {formsConfigured && talk.status !== "draft" && (
          <div className="lg:col-span-4 lg:col-start-9">
            <h2 className="text-xl font-bold">{t.askTitle}</h2>
            <div className="mt-4">
              <QuestionForm talkSlug={talk.slug} locale={locale} />
            </div>
          </div>
        )}
      </section>
    </main>
  );
}

