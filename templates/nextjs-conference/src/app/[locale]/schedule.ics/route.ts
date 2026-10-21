import { isLocale, locales } from "@/i18n";
import { getSetupIssue, getSite, getTalks, speakersOf } from "@/lib/gohead";
import { absoluteUrl } from "@/lib/site-url";

export function generateStaticParams() {
  return locales.map((locale) => ({ locale }));
}

/** iCalendar text: escape , ; \ and newlines, and fold lines at 75 octets. */
const text = (s: string) => s.replace(/\\/g, "\\\\").replace(/([,;])/g, "\\$1").replace(/\r?\n/g, "\\n");
const stamp = (iso: string) => iso.replace(/[-:]/g, "").replace(/\.\d+/, "");
function fold(line: string): string {
  const out: string[] = [];
  let rest = line;
  while (new TextEncoder().encode(rest).length > 75) {
    let cut = 75;
    while (new TextEncoder().encode(rest.slice(0, cut)).length > 75) cut--;
    out.push(rest.slice(0, cut));
    rest = ` ${rest.slice(cut)}`;
  }
  out.push(rest);
  return out.join("\r\n");
}

/**
 * The whole programme as a calendar file (/en/schedule.ics), built from the
 * same GoHead read as the schedule page: `starts_at` (a datetime, UTC) and
 * `duration_minutes` (an int) make each event.
 */
export async function GET(_request: Request, { params }: RouteContext<"/[locale]/schedule.ics">) {
  const { locale } = await params;
  if (!isLocale(locale)) return new Response("Not found", { status: 404 });
  const issue = await getSetupIssue();
  if (issue) return new Response(`GoHead is not ready: ${issue.detail}`, { status: 503 });

  const [site, talks] = await Promise.all([getSite(locale), getTalks(locale)]);
  const host = new URL(absoluteUrl("/")).host;
  const now = stamp(new Date().toISOString());
  const events = talks
    .filter((talk) => talk.status !== "draft")
    .map((talk) => {
      const start = new Date(talk.starts_at);
      const end = new Date(start.getTime() + talk.duration_minutes * 60_000);
      const speakers = speakersOf(talk).map((s) => s.name).join(", ");
      return [
        "BEGIN:VEVENT",
        `UID:talk-${talk.id}@${host}`,
        `DTSTAMP:${now}`,
        `DTSTART:${stamp(start.toISOString())}`,
        `DTEND:${stamp(end.toISOString())}`,
        `SUMMARY:${text(talk.title)}`,
        speakers ? `DESCRIPTION:${text(speakers)}` : "",
        site.venue ? `LOCATION:${text([site.venue.name, site.venue.city].filter(Boolean).join(", "))}` : "",
        `URL:${absoluteUrl(`/${locale}/talks/${talk.slug}`)}`,
        "END:VEVENT",
      ].filter(Boolean);
    });
  const lines = [
    "BEGIN:VCALENDAR",
    "VERSION:2.0",
    `PRODID:-//${text(site.name)}//GoHead conference starter//${locale.toUpperCase()}`,
    "CALSCALE:GREGORIAN",
    `X-WR-CALNAME:${text(site.name)}`,
    ...events.flat(),
    "END:VCALENDAR",
  ];
  return new Response(lines.map(fold).join("\r\n") + "\r\n", {
    headers: {
      "Content-Type": "text/calendar; charset=utf-8",
      "Content-Disposition": `attachment; filename="${site.name.toLowerCase().replace(/[^a-z0-9]+/g, "-")}-${locale}.ics"`,
    },
  });
}
