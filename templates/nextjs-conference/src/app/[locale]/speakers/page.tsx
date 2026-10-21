import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { SpeakerCard } from "@/components/Speakers";
import { getDictionary, isLocale } from "@/i18n";
import { getSpeakers, setupScreen } from "@/lib/gohead";

export async function generateMetadata({ params }: PageProps<"/[locale]/speakers">): Promise<Metadata> {
  const { locale } = await params;
  return isLocale(locale) ? { title: getDictionary(locale).speakers } : {};
}

export default async function SpeakersPage({ params }: PageProps<"/[locale]/speakers">) {
  const { locale } = await params;
  if (!isLocale(locale)) notFound();
  if (await setupScreen()) return null;
  const t = getDictionary(locale);
  const speakers = await getSpeakers(locale);
  // `featured` is a boolean attribute: featured speakers come first.
  const sorted = [...speakers].sort((a, b) => Number(Boolean(b.featured)) - Number(Boolean(a.featured)));
  return (
    <main id="main" className="wrap pt-12 sm:pt-16">
      <h1 className="display">{t.speakers}</h1>
      <p className="mt-4 text-lg text-muted">{t.speakersIntro}</p>
      <div className="mt-12 grid grid-cols-2 gap-x-4 gap-y-10 md:grid-cols-3 lg:grid-cols-4">
        {sorted.map((s) => (
          <SpeakerCard key={s.id} speaker={s} locale={locale} />
        ))}
      </div>
    </main>
  );
}
