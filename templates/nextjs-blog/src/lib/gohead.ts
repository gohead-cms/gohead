/**
 * The GoHead client and the reads the pages use.
 *
 * Every read is cached by Next.js for GOHEAD_REVALIDATE seconds (60 by
 * default) and tagged `gohead` plus `gohead:<type>` for each content type it
 * returns, so /api/revalidate can refresh exactly what a GoHead webhook says
 * changed. Types come from src/gohead.d.ts (`npm run gohead:types`).
 */
import "server-only";

import { PHASE_PRODUCTION_BUILD } from "next/constants";
import { cache } from "react";
import { createClient, isGoHeadError, populated, type Entry, type JsonValue, type Media } from "@gohead-cms/client";

import type { Article, Author, Category, Homepage, Seo } from "@/gohead";
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
  // Optional: the site reads what GoHead's public role allows without one.
  token: process.env.GOHEAD_TOKEN || undefined,
  // Populate relations one level deep (an article's author and category).
  level: 1,
  timeoutMs: 15_000,
  init: { next: { revalidate } },
});

/**
 * Content types this site reads; each is a cache tag `gohead:<type>`, named
 * like the `collection` of GoHead's webhook events (singletons are
 * `singleton:<name>`).
 */
export type ContentType = "articles" | "authors" | "categories" | "singleton:site" | "singleton:homepage";

export const CACHE_TAG = "gohead";
export const cacheTag = (type: string) => `${CACHE_TAG}:${type}`;

/** fetch options of a read returning these content types. */
function cached(...types: ContentType[]): RequestInit {
  return { next: { revalidate, tags: [CACHE_TAG, ...types.map(cacheTag)] } };
}

// Articles embed their author and category, so they are tagged with all three.
const ARTICLE_TYPES: ContentType[] = ["articles", "authors", "categories"];

export type ArticleEntry = Entry<Article>;
export type AuthorEntry = Entry<Author>;
export type CategoryEntry = Entry<Category>;

// ---- Setup ----

export interface SetupIssue {
  kind: "unreachable" | "model" | "access";
  detail: string;
  url: string;
}

/**
 * Whether GoHead can serve this site: reachable, with the content model
 * pushed and readable. Pages show the setup screen when it cannot.
 */
export const getSetupIssue = cache(async (): Promise<SetupIssue | null> => {
  try {
    await gohead.collection("articles").count({}, cached("articles"));
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
 * serving its last good version while GoHead is down, and a page never
 * rendered yet shows the error page (the server log says what is wrong).
 */
export async function setupScreen(): Promise<SetupIssue | null> {
  const issue = await getSetupIssue();
  if (!issue) return null;
  if (process.env.NODE_ENV === "production" && process.env.NEXT_PHASE !== PHASE_PRODUCTION_BUILD) {
    throw new Error(`GoHead cannot serve this site (${issue.kind}) at ${issue.url}: ${issue.detail}. See the README.`);
  }
  return issue;
}

// ---- Site and home page ----

export interface SiteLink {
  label: string;
  url: string;
}

export interface SiteSettings {
  name: string;
  tagline: string | null;
  description: string | null;
  footer: string | null;
  links: SiteLink[];
  seo: Seo | null;
}

/** The `site` singleton, with defaults while it has no content. */
export const getSite = cache(async (locale: Locale): Promise<SiteSettings> => {
  const site = await gohead.singleton("site").find({ locale }, cached("singleton:site"));
  return {
    name: site?.name || "GoHead Blog",
    tagline: site?.tagline || null,
    description: site?.description || null,
    footer: site?.footer || null,
    links: linksOf(site?.links),
    seo: site?.seo ?? null,
  };
});

export type HomepageEntry = Entry<Homepage>;

/** The `homepage` singleton, its featured articles populated with their authors. */
export const getHomepage = cache(async (locale: Locale): Promise<HomepageEntry | null> => {
  return gohead.singleton("homepage").find({ locale, level: 2 }, cached("singleton:homepage", ...ARTICLE_TYPES));
});

// ---- Articles ----

/** A page of published articles, newest first. */
export const getLatestArticles = cache(async (locale: Locale, page: number, pageSize = 6) => {
  return gohead
    .collection("articles")
    .find({ locale, page, pageSize, sort: "publishedAt:desc" }, cached(...ARTICLE_TYPES));
});

/**
 * Every published article, newest first. GoHead filters cannot match on
 * relations yet, so category and author pages filter this list; fine for a
 * blog of a few hundred articles.
 */
export const getAllArticles = cache(async (locale: Locale): Promise<ArticleEntry[]> => {
  return gohead.collection("articles").findAll({ locale, sort: "publishedAt:desc" }, cached(...ARTICLE_TYPES));
});

export const getArticle = cache(async (slug: string, locale: Locale): Promise<ArticleEntry | null> => {
  return gohead.collection("articles").findFirst({ locale, filters: { slug } }, cached(...ARTICLE_TYPES));
});

export async function getArticlesInCategory(categoryId: number, locale: Locale): Promise<ArticleEntry[]> {
  return (await getAllArticles(locale)).filter((a) => categoryOf(a)?.id === categoryId);
}

export async function getArticlesByAuthor(authorId: number, locale: Locale): Promise<ArticleEntry[]> {
  return (await getAllArticles(locale)).filter((a) => authorOf(a)?.id === authorId);
}

// ---- Categories and authors ----

export const getCategories = cache(async (locale: Locale): Promise<CategoryEntry[]> => {
  return gohead.collection("categories").findAll({ locale, sort: "name" }, cached("categories"));
});

export const getCategory = cache(async (slug: string, locale: Locale): Promise<CategoryEntry | null> => {
  return gohead.collection("categories").findFirst({ locale, filters: { slug } }, cached("categories"));
});

export const getAuthors = cache(async (locale: Locale): Promise<AuthorEntry[]> => {
  return gohead.collection("authors").findAll({ locale, sort: "name" }, cached("authors"));
});

export const getAuthor = cache(async (slug: string, locale: Locale): Promise<AuthorEntry | null> => {
  return gohead.collection("authors").findFirst({ locale, filters: { slug } }, cached("authors"));
});

/**
 * Slugs to prerender at build time. When GoHead cannot be reached the build
 * still succeeds: pages are then rendered on their first visit.
 */
export async function getSlugs(type: "articles" | "categories" | "authors"): Promise<string[]> {
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

/** An absolute URL for a GoHead file (local uploads are served by GoHead). */
export function mediaUrl(media: Pick<Media, "url"> | string | number | null | undefined): string | undefined {
  return publicMedia.media.url(media);
}

/**
 * Links to files use GoHead's public address (GOHEAD_PUBLIC_URL) when the
 * site reaches it through an internal one (a Docker service name…), so
 * Open Graph images and images in articles work for visitors.
 */
const publicMedia = process.env.GOHEAD_PUBLIC_URL ? gohead.with({ url: process.env.GOHEAD_PUBLIC_URL }) : gohead;

/** A populated relation, or null (relations are plain IDs below `level`). */
export const authorOf = (a: ArticleEntry): AuthorEntry | null => populated(a.author) ?? null;
export const categoryOf = (a: ArticleEntry): CategoryEntry | null => populated(a.category) ?? null;

/** The `tags` JSON attribute as a list of strings. */
export function tagsOf(a: ArticleEntry): string[] {
  return Array.isArray(a.tags) ? a.tags.filter((t): t is string => typeof t === "string" && t.trim() !== "") : [];
}

/** `reading_minutes`, or an estimate from the body (about 220 words a minute). */
export function readingMinutes(a: ArticleEntry): number {
  if (a.reading_minutes && a.reading_minutes > 0) return a.reading_minutes;
  const words = a.body?.trim().split(/\s+/).length ?? 0;
  return Math.max(1, Math.round(words / 220));
}

function linksOf(value: JsonValue | null | undefined): SiteLink[] {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) => {
    if (item === null || typeof item !== "object" || Array.isArray(item)) return [];
    const { label, url } = item;
    return typeof label === "string" && typeof url === "string" ? [{ label, url }] : [];
  });
}
