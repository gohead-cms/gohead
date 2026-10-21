import Link from "next/link";

import { Cover } from "@/components/Cover";
import type { Locale } from "@/i18n";
import type { SpeakerEntry } from "@/lib/gohead";

export function SpeakerCard({ speaker, locale }: { speaker: SpeakerEntry; locale: Locale }) {
  return (
    <article className="group relative">
      <Cover media={speaker.photo} alt="" ratio="1 / 1" sizes="(min-width: 1024px) 280px, 45vw" className="rounded-lg" />
      <h3 className="mt-3 text-lg leading-tight font-bold tracking-[-0.02em]">
        <Link href={`/${locale}/speakers/${speaker.slug}`} className="after:absolute after:inset-0 group-hover:underline group-hover:decoration-signal group-hover:decoration-2 group-hover:underline-offset-4">
          {speaker.name}
        </Link>
      </h3>
      {speaker.headline && <p className="text-sm text-muted">{speaker.headline}</p>}
      {speaker.company && <p className="text-sm font-medium">{speaker.company}</p>}
    </article>
  );
}
