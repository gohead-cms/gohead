import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";

import { TalkRow } from "@/components/Talks";
import type { Talk } from "@/gohead";
import { dayOf, formatDay, getDictionary, isLocale } from "@/i18n";
import { daysBetween, getSite, getTalks, getTracks, setupScreen, type TalkFilters } from "@/lib/gohead";

export async function generateMetadata({ params }: PageProps<"/[locale]/schedule">): Promise<Metadata> {
  const { locale } = await params;
  if (!isLocale(locale)) return {};
  return { title: getDictionary(locale).schedule, alternates: { canonical: `/${locale}/schedule` } };
}

const LEVELS = ["beginner", "intermediate", "advanced"] as const satisfies readonly NonNullable<Talk["level"]>[];
const FORMATS = ["keynote", "talk", "workshop", "lightning", "panel"] as const satisfies readonly Talk["format"][];

const one = (v: string | string[] | undefined) => (Array.isArray(v) ? v[0] : v);

/**
 * The programme. Filters are query parameters (?day=2027-03-11&level=beginner)
 * sent by a plain GET form, so the page works without JavaScript; GoHead does
 * the filtering (see getTalks in src/lib/gohead.ts).
 */
export default async function SchedulePage({ params, searchParams }: PageProps<"/[locale]/schedule">) {
  const { locale } = await params;
  if (!isLocale(locale)) notFound();
  if (await setupScreen()) return null;

  const t = getDictionary(locale);
  const query = await searchParams;
  const site = await getSite(locale);
  const days = daysBetween(site.startsOn, site.endsOn);
  const filters: TalkFilters = {
    day: days.find((d) => d === one(query.day)),
    level: LEVELS.find((l) => l === one(query.level)),
    format: FORMATS.find((f) => f === one(query.format)),
    track: one(query.track) || undefined,
  };
  const [talks, tracks] = await Promise.all([getTalks(locale, filters, site.timeZone), getTracks(locale)]);
  const active = Boolean(filters.level || filters.format || filters.track);

  const byDay = new Map<string, typeof talks>();
  for (const talk of talks) {
    const d = dayOf(talk.starts_at, site.timeZone);
    byDay.set(d, [...(byDay.get(d) ?? []), talk]);
  }
  const href = (day?: string) => {
    const q = new URLSearchParams();
    if (day) q.set("day", day);
    if (filters.level) q.set("level", filters.level);
    if (filters.format) q.set("format", filters.format);
    if (filters.track) q.set("track", filters.track);
    const s = q.toString();
    return `/${locale}/schedule${s ? `?${s}` : ""}`;
  };

  return (
    <main id="main" className="wrap pt-12 sm:pt-16">
      <div className="flex flex-wrap items-end justify-between gap-6">
        <div>
          <h1 className="display">{t.schedule}</h1>
          <p className="mt-4 text-muted">
            {t.scheduleIntro} ({site.timeZone})
          </p>
        </div>
        <a href={`/${locale}/schedule.ics`} className="button-ghost" download>
          <svg aria-hidden viewBox="0 0 24 24" className="size-5" fill="none" stroke="currentColor" strokeWidth="2">
            <rect x="3" y="5" width="18" height="16" rx="2" />
            <path d="M3 10h18M8 3v4M16 3v4" strokeLinecap="round" />
          </svg>
          {t.addToCalendar}
        </a>
      </div>

      <nav aria-label={t.filterBy} className="mt-10 flex flex-wrap gap-2">
        {[undefined, ...days].map((d, i) => {
          const current = filters.day === d;
          return (
            <Link
              key={d ?? "all"}
              href={href(d)}
              aria-current={current ? "page" : undefined}
              className={`tabular rounded-full px-4 py-2 font-semibold ${current ? "bg-ink text-paper" : "shadow-[inset_0_0_0_1.5px_var(--rule)] hover:shadow-[inset_0_0_0_1.5px_var(--ink)]"}`}
            >
              {d ? `${t.day(i)} · ${formatDay(d, locale)}` : t.allDays}
            </Link>
          );
        })}
      </nav>

      <form method="get" className="mt-4 flex flex-wrap items-end gap-3 rounded-lg bg-paper-raised p-4 shadow-[inset_0_0_0_1px_var(--rule)]">
        {filters.day && <input type="hidden" name="day" value={filters.day} />}
        <label className="grid gap-1 text-sm font-semibold">
          {t.track}
          <select name="track" defaultValue={filters.track ?? ""} className="input py-2">
            <option value="">{t.allTracks}</option>
            {tracks.map((tr) => (
              <option key={tr.id} value={tr.slug}>
                {tr.name}
              </option>
            ))}
          </select>
        </label>
        <label className="grid gap-1 text-sm font-semibold">
          {t.levelLabel}
          <select name="level" defaultValue={filters.level ?? ""} className="input py-2">
            <option value="">{t.allLevels}</option>
            {LEVELS.map((l) => (
              <option key={l} value={l}>
                {t.level[l]}
              </option>
            ))}
          </select>
        </label>
        <label className="grid gap-1 text-sm font-semibold">
          {t.formatLabel}
          <select name="format" defaultValue={filters.format ?? ""} className="input py-2">
            <option value="">{t.allFormats}</option>
            {FORMATS.map((f) => (
              <option key={f} value={f}>
                {t.format[f]}
              </option>
            ))}
          </select>
        </label>
        <button type="submit" className="button">
          {t.apply}
        </button>
        {active && (
          <Link href={href(filters.day)} className="px-2 py-2.5 font-semibold text-link hover:underline">
            {t.reset}
          </Link>
        )}
      </form>

      {talks.length === 0 && <p className="mt-12 text-lg text-muted">{t.noTalks}</p>}
      {[...byDay.entries()].map(([day, list]) => (
        <section key={day} aria-labelledby={`day-${day}`} className="mt-14">
          <h2 id={`day-${day}`} className="flex items-baseline gap-4 text-3xl font-extrabold tracking-[-0.04em]">
            <span className="text-signal">{t.day(days.indexOf(day) + 1)}</span>
            <span className="capitalize">{formatDay(day, locale)}</span>
          </h2>
          <ol className="mt-4 border-b border-rule">
            {list.map((talk) => (
              <TalkRow key={talk.id} talk={talk} locale={locale} timeZone={site.timeZone} />
            ))}
          </ol>
        </section>
      ))}
    </main>
  );
}
