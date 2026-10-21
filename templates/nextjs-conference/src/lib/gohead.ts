/**
 * The GoHead clients and every read the pages make.
 *
 * - `gohead` reads published content anonymously: gohead/model.json opens the
 *   site's types to the public role. GOHEAD_TOKEN is only needed for a
 *   private instance.
 * - In draft mode (/api/preview), reads go through `preview` instead: an API
 *   token whose role may update talks and posts, so it sees their drafts.
 * - Writes (the site's forms) live in src/lib/forms.ts, with their own token.
 *
 * Every read is cached by Next.js for GOHEAD_REVALIDATE seconds (60 by
 * default) and tagged `gohead` plus `gohead:<type>` for each content type it
 * returns, so /api/revalidate can refresh exactly what a webhook says changed.
 * Types come from src/gohead.d.ts (`npm run gohead:types`).
 */
import "server-only";

import { PHASE_PRODUCTION_BUILD } from "next/constants";
import { draftMode } from "next/headers";
import { cache } from "react";
import { createClient, isGoHeadError, populated, type Entry, type Filters, type GoHeadClient, type Media } from "@gohead-cms/client";

import type { Page, Post, Speaker, Sponsor, Talk, Track } from "@/gohead";
import type { Locale } from "@/i18n";

export const GOHEAD_URL = process.env.GOHEAD_URL || "http://localhost:8080";

/** Seconds a read stays fresh; `false` caches until a webhook revalidates it. */
function revalidateSeconds(value: string | undefined): number | false {
  if (value === undefined || value.trim() === "") return 60;
  if (value === "false") return false;
  const n = Number(value);
  return Number.isFinite(n) && n >= 0 ? n : 60;
}

const revalidate = revalidateSeconds(process.env.GOHEAD_REVALIDATE);

export const gohead = createClient({
  url: GOHEAD_URL,
  token: process.env.GOHEAD_TOKEN || undefined,
  // Populate relations one level deep: a talk's track and speakers.
  level: 1,
  timeoutMs: 15_000,
  init: { next: { revalidate } },
});

/** Reads drafts. It is only used in draft mode, which skips Next.js caches. */
const preview = process.env.GOHEAD_PREVIEW_TOKEN ? gohead.with({ token: process.env.GOHEAD_PREVIEW_TOKEN }) : null;

export const previewConfigured = preview !== null && Boolean(process.env.GOHEAD_PREVIEW_SECRET);

/**
 * The client of this request and the `status` to read draft-and-publish
 * collections with: drafts too in draft mode, published content otherwise.
 */
async function reader(): Promise<{ client: GoHeadClient; status: "all" | "published"; preview: boolean }> {
  const { isEnabled } = await draftMode();
  if (isEnabled && preview) return { client: preview, status: "all", preview: true };
  return { client: gohead, status: "published", preview: false };
}

/**
 * Content types this site reads; each is a cache tag `gohead:<type>`, named
 * like the `collection` of GoHead's webhook events (singletons are
 * `singleton:<name>`).
 */
export type ContentType =
  | "talks"
  | "speakers"
  | "tracks"
  | "posts"
  | "sponsors"
  | "pages"
  | "questions"
  | "singleton:site"
  | "singleton:homepage"
  | "singleton:tickets";

export const CACHE_TAG = "gohead";
export const cacheTag = (type: string) => `${CACHE_TAG}:${type}`;

/** fetch options of a read returning these content types. */
function cached(...types: ContentType[]): RequestInit {
  return { next: { revalidate, tags: [CACHE_TAG, ...types.map(cacheTag)] } };
}

// Talks embed their track and speakers, so they are tagged with all three.
const TALK_TYPES: ContentType[] = ["talks", "tracks", "speakers"];

export type TalkEntry = Entry<Talk>;
export type SpeakerEntry = Entry<Speaker>;
export type TrackEntry = Entry<Track>;
export type PostEntry = Entry<Post>;
export type PageEntry = Entry<Page>;

// ---- Setup ----

export interface SetupIssue {
  kind: "unreachable" | "model" | "access";
  detail: string;
  url: string;
}

/** Whether GoHead can serve this site: reachable, with the content model pushed and readable. */
export const getSetupIssue = cache(async (): Promise<SetupIssue | null> => {
  try {
    await gohead.collection("talks").count({}, cached("talks"));
    return null;
  } catch (err) {
    if (!isGoHeadError(err)) throw err;
    const issue = (kind: SetupIssue["kind"]): SetupIssue => ({ kind, detail: err.message, url: gohead.url });
    if (err.status === 0) return issue("unreachable");
    if (err.status === 404) return issue("model");
    if (err.status === 401 || err.status === 403) return issue("access");
    throw err;
  }
});

/**
 * The setup issue to show instead of a page, during development and builds.
 * A production server throws instead: a page being revalidated then keeps
 * serving its last good version while GoHead is down.
 */
export async function setupScreen(): Promise<SetupIssue | null> {
  const issue = await getSetupIssue();
  if (!issue) return null;
  if (process.env.NODE_ENV === "production" && process.env.NEXT_PHASE !== PHASE_PRODUCTION_BUILD) {
    throw new Error(`GoHead cannot serve this site (${issue.kind}) at ${issue.url}: ${issue.detail}. See the README.`);
  }
  return issue;
}

// ---- Singletons ----

/** The `site` singleton (name, dates, venue, navigation), with defaults while it is empty. */
export const getSite = cache(async (locale: Locale) => {
  const site = await gohead.singleton("site").find({ locale }, cached("singleton:site"));
  return {
    name: site?.name || "GoHead Conf",
    tagline: site?.tagline ?? null,
    description: site?.description ?? null,
    startsOn: site?.starts_on ?? null,
    endsOn: site?.ends_on ?? null,
    venue: site?.venue ?? null,
    // The venue's time zone: talk times are stored in UTC and shown in it.
    timeZone: site?.venue?.timezone || "UTC",
    nav: site?.nav ?? [],
    footer: site?.footer ?? null,
    social: linksOf(site?.social),
    seo: site?.seo ?? null,
  };
});
export type SiteSettings = Awaited<ReturnType<typeof getSite>>;

/** The home page: hero, featured talks (with their speakers: level 2) and featured speakers. */
export const getHomepage = cache(async (locale: Locale) => {
  return gohead.singleton("homepage").find({ locale, level: 2 }, cached("singleton:homepage", ...TALK_TYPES));
});

export const getTickets = cache(async (locale: Locale) => {
  return gohead.singleton("tickets").find({ locale }, cached("singleton:tickets"));
});

// ---- Talks ----

export interface TalkFilters {
  /** A day of the conference, YYYY-MM-DD in the venue's time zone. */
  day?: string;
  level?: Talk["level"];
  format?: Talk["format"];
  /** A track slug. */
  track?: string;
}

/**
 * The programme, in time order. `day`, `level` and `format` are GoHead
 * filters (a datetime range, and enum equality); `track` is a relation, which
 * GoHead filters can't match yet, so it is applied here.
 */
export async function getTalks(locale: Locale, filters: TalkFilters = {}, timeZone = "UTC"): Promise<TalkEntry[]> {
  const { client, status } = await reader();
  const where: Filters<Talk> = {};
  if (filters.day) {
    where.starts_at = { $gte: zonedMidnight(filters.day, timeZone, 0), $lt: zonedMidnight(filters.day, timeZone, 1) };
  }
  if (filters.level) where.level = filters.level;
  if (filters.format) where.format = filters.format;
  const talks = await client
    .collection("talks")
    .findAll({ locale, status, filters: where, sort: ["starts_at", "room"] }, cached(...TALK_TYPES));
  return filters.track ? talks.filter((t) => trackOf(t)?.slug === filters.track) : talks;
}

export const getTalk = cache(async (slug: string, locale: Locale): Promise<TalkEntry | null> => {
  const { client, status } = await reader();
  return client.collection("talks").findFirst({ locale, status, filters: { slug } }, cached(...TALK_TYPES));
});

/** Talks a speaker gives: `speakers` is a many-to-many relation, filtered here. */
export async function getTalksBySpeaker(speakerId: number, locale: Locale): Promise<TalkEntry[]> {
  return (await getTalks(locale)).filter((t) => speakersOf(t).some((s) => s.id === speakerId));
}

/** How many published talks, speakers and tracks, for the home page. */
export const getStats = cache(async () => {
  const [talks, speakers, tracks] = await Promise.all([
    gohead.collection("talks").count({}, cached("talks")),
    gohead.collection("speakers").count({}, cached("speakers")),
    gohead.collection("tracks").count({}, cached("tracks")),
  ]);
  return { talks, speakers, tracks };
});

// ---- Speakers and tracks ----

export const getSpeakers = cache(async (locale: Locale): Promise<SpeakerEntry[]> => {
  return gohead.collection("speakers").findAll({ locale, sort: "name" }, cached("speakers"));
});

export const getSpeaker = cache(async (slug: string, locale: Locale): Promise<SpeakerEntry | null> => {
  return gohead.collection("speakers").findFirst({ locale, filters: { slug } }, cached("speakers"));
});

export const getTracks = cache(async (locale: Locale): Promise<TrackEntry[]> => {
  return gohead.collection("tracks").findAll({ locale, sort: "name" }, cached("tracks"));
});

// ---- News ----

/** A page of posts, newest first. */
export async function getPosts(locale: Locale, page: number, pageSize = 6) {
  const { client, status } = await reader();
  return client.collection("posts").find({ locale, status, page, pageSize, sort: "publishedAt:desc" }, cached("posts"));
}

/** The 20 latest published posts, for the RSS feed (never drafts, even in draft mode). */
export const getFeedPosts = cache(async (locale: Locale) => {
  return gohead.collection("posts").find({ locale, pageSize: 20, sort: "publishedAt:desc" }, cached("posts"));
});

export const getPost = cache(async (slug: string, locale: Locale): Promise<PostEntry | null> => {
  const { client, status } = await reader();
  return client.collection("posts").findFirst({ locale, status, filters: { slug } }, cached("posts"));
});

// ---- Pages ----

export const getPage = cache(async (slug: string, locale: Locale): Promise<PageEntry | null> => {
  return gohead.collection("pages").findFirst({ locale, filters: { slug } }, cached("pages"));
});

// ---- Questions ----

/**
 * Published questions of a talk. They are filtered on `talk_slug`, a plain
 * copy of the talk's slug written by the form, because filters can't match
 * on the `talk` relation. Drafts (not moderated yet) are never returned.
 */
export const getQuestions = cache(async (talkSlug: string) => {
  return gohead
    .collection("questions")
    .findAll({ filters: { talk_slug: talkSlug }, sort: "publishedAt", level: 0 }, cached("questions"));
});

// ---- Sponsors, through GraphQL ----

export interface SponsorNode {
  id: string;
  name: string;
  slug: string;
  tier: NonNullable<Sponsor["tier"]>;
  website: string | null;
  blurb: string | null;
  rank: number | null;
  logo: Pick<Media, "url" | "width" | "height" | "alt"> | null;
}

/**
 * Sponsors come from GraphQL, to show the other API: one query, only the
 * fields the page needs, localized. Anonymous queries can read public types.
 */
export const getSponsors = cache(async (locale: Locale): Promise<SponsorNode[]> => {
  const data = await gohead.graphql<{ sponsors: SponsorNode[] }>(
    `query Sponsors($locale: String) {
      sponsors(limit: 100, locale: $locale) {
        id name slug tier website blurb rank
        logo { url width height alt }
      }
    }`,
    { locale },
    { operationName: "Sponsors", init: cached("sponsors") },
  );
  const order = ["platinum", "gold", "silver", "community"];
  return [...data.sponsors].sort(
    (a, b) => order.indexOf(a.tier) - order.indexOf(b.tier) || (a.rank ?? 99) - (b.rank ?? 99) || a.name.localeCompare(b.name),
  );
});

// ---- Search ----

/**
 * Talks, speakers and posts whose title or name contains the words, in this
 * locale. `$contains` is a SQL LIKE: case-insensitive on SQLite and MySQL,
 * case-sensitive on Postgres, so the query is also tried lowercase and
 * capitalized.
 */
export async function search(query: string, locale: Locale) {
  const q = query.trim().slice(0, 80);
  if (q.length < 2) return { talks: [], speakers: [], posts: [] };
  const variants = [...new Set([q, q.toLowerCase(), q.charAt(0).toUpperCase() + q.slice(1).toLowerCase()])];
  const merge = <T extends { id: number }>(lists: T[][]) => [...new Map(lists.flat().map((e) => [e.id, e])).values()];
  const [talks, speakers, posts] = await Promise.all([
    Promise.all(variants.map((v) => gohead.collection("talks").find({ locale, filters: { title: { $contains: v } }, pageSize: 20 }, cached(...TALK_TYPES)))),
    Promise.all(variants.map((v) => gohead.collection("speakers").find({ locale, filters: { name: { $contains: v } }, pageSize: 20 }, cached("speakers")))),
    Promise.all(variants.map((v) => gohead.collection("posts").find({ locale, filters: { title: { $contains: v } }, pageSize: 20 }, cached("posts")))),
  ]);
  return {
    talks: merge(talks.map((r) => r.data)).sort((a, b) => a.starts_at.localeCompare(b.starts_at)),
    speakers: merge(speakers.map((r) => r.data)),
    posts: merge(posts.map((r) => r.data)),
  };
}

// ---- Build time ----

/**
 * Slugs to prerender. When GoHead cannot be reached the build still
 * succeeds: pages are then rendered on their first visit.
 */
export async function getSlugs(type: "talks" | "speakers" | "posts" | "pages"): Promise<string[]> {
  try {
    const entries = await gohead.collection(type).findAll({ level: 0 }, cached(type));
    return entries.map((e) => e.slug).filter(Boolean);
  } catch (err) {
    if (!isGoHeadError(err)) throw err;
    console.warn(`[gohead] not prerendering ${type}: ${err.message}`);
    return [];
  }
}

// ---- Helpers ----

/**
 * Links to files use GoHead's public address (GOHEAD_PUBLIC_URL) when the
 * site reaches it through an internal one (a Docker service name…).
 */
const publicMedia = process.env.GOHEAD_PUBLIC_URL ? gohead.with({ url: process.env.GOHEAD_PUBLIC_URL }) : gohead;

/** An absolute URL for a GoHead file (local uploads are served by GoHead). */
export function mediaUrl(media: Pick<Media, "url"> | string | number | null | undefined): string | undefined {
  return publicMedia.media.url(media);
}

/** Populated relations (relations are plain IDs below the read's `level`). */
export const trackOf = (t: TalkEntry): TrackEntry | null => populated(t.track) ?? null;
export const speakersOf = (t: TalkEntry): SpeakerEntry[] =>
  (t.speakers ?? []).map((s) => populated(s)).filter((s): s is SpeakerEntry => Boolean(s));

/** A JSON attribute holding a list of strings, such as `tags`. */
export function stringsOf(value: unknown): string[] {
  return Array.isArray(value) ? value.filter((t): t is string => typeof t === "string" && t.trim() !== "") : [];
}

export interface LinkItem {
  label: string;
  url: string;
}

function linksOf(value: unknown): LinkItem[] {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) => {
    if (item === null || typeof item !== "object") return [];
    const { label, url } = item as Record<string, unknown>;
    return typeof label === "string" && typeof url === "string" ? [{ label, url }] : [];
  });
}

/** The UTC instant of midnight, `addDays` after `day` (YYYY-MM-DD), in a time zone. */
export function zonedMidnight(day: string, timeZone: string, addDays = 0): string {
  const [y, m, d] = day.split("-").map(Number) as [number, number, number];
  const utc = Date.UTC(y, m - 1, d + addDays);
  // The zone's offset at that moment: format the instant in the zone, read it back as UTC.
  const parts = Object.fromEntries(
    new Intl.DateTimeFormat("en-US", { timeZone, hourCycle: "h23", year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit", second: "2-digit" })
      .formatToParts(new Date(utc))
      .map((p) => [p.type, p.value]),
  );
  const asUtc = Date.UTC(+parts.year!, +parts.month! - 1, +parts.day!, +parts.hour!, +parts.minute!, +parts.second!);
  return new Date(utc - (asUtc - utc)).toISOString();
}

/** The conference days between two dates (inclusive), as YYYY-MM-DD. */
export function daysBetween(start: string | null, end: string | null): string[] {
  if (!start) return [];
  const out: string[] = [];
  const last = new Date(`${end || start}T00:00:00Z`).getTime();
  for (let t = new Date(`${start}T00:00:00Z`).getTime(); t <= last && out.length < 14; t += 86_400_000) {
    out.push(new Date(t).toISOString().slice(0, 10));
  }
  return out;
}
