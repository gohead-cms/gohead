import Link from "next/link";

import { Avatar } from "@/components/Cover";
import { formatTime, getDictionary, type Locale } from "@/i18n";
import { speakersOf, trackOf, type TalkEntry, type TrackEntry } from "@/lib/gohead";

/** A track's name on its color (`tracks.color`, checked by a pattern in the model). */
export function TrackChip({ track, href }: { track: Pick<TrackEntry, "name" | "color" | "slug">; href?: string }) {
  const color = track.color || "#888888";
  const chip = (
    <span className="inline-flex items-center gap-1.5 text-sm font-semibold">
      <span aria-hidden className="size-2.5 rounded-full" style={{ background: color }} />
      {track.name}
    </span>
  );
  return href ? (
    <Link href={href} className="hover:underline">
      {chip}
    </Link>
  ) : (
    chip
  );
}

/** One session in a list: time, format, title, speakers, track and room. */
export function TalkRow({ talk, locale, timeZone, showDay }: { talk: TalkEntry; locale: Locale; timeZone: string; showDay?: string }) {
  const t = getDictionary(locale);
  const track = trackOf(talk);
  const speakers = speakersOf(talk);
  const isDraft = talk.status === "draft";
  return (
    <li className="group relative grid grid-cols-[4.5rem_1fr] gap-x-4 border-t border-rule py-5 sm:grid-cols-[7rem_1fr_11rem] sm:gap-x-8">
      <div className="tabular">
        {showDay && <p className="eyebrow text-muted">{showDay}</p>}
        <p className="text-xl font-bold tracking-tight">{formatTime(talk.starts_at, locale, timeZone)}</p>
        <p className="text-sm text-muted">{t.minutes(talk.duration_minutes)}</p>
      </div>
      <div className="min-w-0">
        <p className="eyebrow flex flex-wrap gap-x-3 text-muted">
          <span className={talk.format === "keynote" ? "text-signal" : undefined}>{t.format[talk.format]}</span>
          {talk.level && <span>{t.level[talk.level]}</span>}
          {isDraft && <span className="rounded bg-signal px-1.5 text-on-signal">{t.draftBadge}</span>}
        </p>
        <h3 className="mt-1 text-xl leading-snug font-bold tracking-[-0.02em] text-balance sm:text-2xl">
          <Link href={`/${locale}/talks/${talk.slug}`} className="after:absolute after:inset-0 group-hover:underline group-hover:decoration-signal group-hover:decoration-2 group-hover:underline-offset-4">
            {talk.title}
          </Link>
        </h3>
        {speakers.length > 0 && (
          <p className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-2">
            {speakers.map((s) => (
              <span key={s.id} className="inline-flex items-center gap-2 text-[0.95rem] font-medium">
                <Avatar media={s.photo} name={s.name} size={26} />
                {s.name}
              </span>
            ))}
          </p>
        )}
      </div>
      <div className="col-start-2 mt-3 flex flex-wrap gap-x-4 gap-y-1 sm:col-start-3 sm:mt-0 sm:flex-col">
        {track && <TrackChip track={track} />}
        {talk.room && <span className="text-sm text-muted">{t.room[talk.room]}</span>}
      </div>
    </li>
  );
}

/** A bigger card, for featured talks. */
export function TalkCard({ talk, locale, timeZone, dayLabel }: { talk: TalkEntry; locale: Locale; timeZone: string; dayLabel: string }) {
  const t = getDictionary(locale);
  const track = trackOf(talk);
  const speakers = speakersOf(talk);
  return (
    <article className="group relative flex flex-col rounded-lg bg-paper-raised p-6 shadow-[inset_0_0_0_1px_var(--rule)] transition-shadow hover:shadow-[inset_0_0_0_2px_var(--ink)]">
      <div className="h-1.5 w-12 rounded-full" style={{ background: track?.color || "var(--signal)" }} />
      <p className="eyebrow tabular mt-5 text-muted">
        {dayLabel} · {formatTime(talk.starts_at, locale, timeZone)} · {t.format[talk.format]}
      </p>
      <h3 className="mt-2 text-2xl leading-tight font-extrabold tracking-[-0.03em] text-balance">
        <Link href={`/${locale}/talks/${talk.slug}`} className="after:absolute after:inset-0">
          {talk.title}
        </Link>
      </h3>
      <div className="mt-auto flex items-center gap-3 pt-6">
        <span className="flex -space-x-2">
          {speakers.map((s) => (
            <span key={s.id} className="rounded-full ring-2 ring-paper-raised">
              <Avatar media={s.photo} name={s.name} size={34} />
            </span>
          ))}
        </span>
        <span className="text-sm font-semibold">{speakers.map((s) => s.name).join(", ")}</span>
      </div>
    </article>
  );
}
