import { populated } from "@gohead-cms/client";
import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { Cover } from "@/components/Cover";
import { Markdown } from "@/components/Markdown";
import { TalkRow } from "@/components/Talks";
import { dayOf, getDictionary, isLocale, locales } from "@/i18n";
import { daysBetween, getSite, getSlugs, getSpeaker, getTalksBySpeaker, mediaUrl, setupScreen } from "@/lib/gohead";
import { isExternal } from "@/lib/paths";

export async function generateStaticParams() {
  const slugs = await getSlugs("speakers");
  return locales.flatMap((locale) => slugs.map((slug) => ({ locale, slug })));
}

export async function generateMetadata({ params }: PageProps<"/[locale]/speakers/[slug]">): Promise<Metadata> {
  const { locale, slug } = await params;
  if (!isLocale(locale) || (await setupScreen())) return {};
  const speaker = await getSpeaker(slug, locale);
  if (!speaker) return {};
  const photo = mediaUrl(populated(speaker.photo));
  return {
    title: speaker.name,
    description: speaker.headline || undefined,
    alternates: { canonical: `/${locale}/speakers/${slug}`, languages: Object.fromEntries(locales.map((l) => [l, `/${l}/speakers/${slug}`])) },
    openGraph: { type: "profile", images: photo ? [{ url: photo }] : undefined },
  };
}

export default async function SpeakerPage({ params }: PageProps<"/[locale]/speakers/[slug]">) {
  const { locale, slug } = await params;
  if (!isLocale(locale)) notFound();
  if (await setupScreen()) return null;
  const speaker = await getSpeaker(slug, locale);
  if (!speaker) notFound();

  const t = getDictionary(locale);
  const [site, talks] = await Promise.all([getSite(locale), getTalksBySpeaker(speaker.id, locale)]);
  const days = daysBetween(site.startsOn, site.endsOn);
  return (
    <main id="main" className="wrap pt-12 sm:pt-16">
      <div className="grid gap-10 md:grid-cols-12 md:items-end">
        <div className="md:col-span-4">
          <Cover media={speaker.photo} alt="" ratio="1 / 1" sizes="(min-width: 768px) 360px, 100vw" preload className="rounded-lg" />
        </div>
        <div className="md:col-span-8">
          {speaker.company && <p className="eyebrow text-signal">{[speaker.company, speaker.country].filter(Boolean).join(" · ")}</p>}
          <h1 className="display mt-3">{speaker.name}</h1>
          {speaker.headline && <p className="mt-4 text-xl text-muted">{speaker.headline}</p>}
          {speaker.links && speaker.links.length > 0 && (
            <ul className="mt-6 flex flex-wrap gap-x-5 gap-y-2">
              {speaker.links.map((l) => (
                <li key={l.url}>
                  <a href={l.url} className="link" {...(isExternal(l.url) ? { rel: "noopener noreferrer" } : {})}>
                    {l.label}
                  </a>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
      <div className="mt-12 max-w-[68ch]">
        <Markdown>{speaker.bio}</Markdown>
      </div>
      {talks.length > 0 && (
        <section aria-labelledby="talks" className="mt-16">
          <h2 id="talks" className="text-3xl font-extrabold tracking-[-0.04em]">
            {t.talksBy(speaker.name)}
          </h2>
          <ol className="mt-4 border-b border-rule">
            {talks.map((talk) => (
              <TalkRow key={talk.id} talk={talk} locale={locale} timeZone={site.timeZone} showDay={t.day(days.indexOf(dayOf(talk.starts_at, site.timeZone)) + 1)} />
            ))}
          </ol>
        </section>
      )}
    </main>
  );
}
