import type { MetadataRoute } from "next";

import { locales } from "@/i18n";
import { getAllArticles, getAuthors, getCategories, getSetupIssue } from "@/lib/gohead";
import { absoluteUrl } from "@/lib/site-url";

export const revalidate = 3600;

/** Every page in every locale, each listing its translations. */
export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const entry = (path: string, lastModified?: string): MetadataRoute.Sitemap[number] => ({
    url: absoluteUrl(`/${locales[0]}${path}`),
    lastModified,
    alternates: { languages: Object.fromEntries(locales.map((l) => [l, absoluteUrl(`/${l}${path}`)])) },
  });
  if (await getSetupIssue()) return [entry("")];

  const locale = locales[0];
  const [articles, categories, authors] = await Promise.all([getAllArticles(locale), getCategories(locale), getAuthors(locale)]);
  return [
    entry("", articles[0]?.publishedAt ?? undefined),
    ...articles.map((a) => entry(`/articles/${a.slug}`, a.updatedAt)),
    ...categories.map((c) => entry(`/categories/${c.slug}`)),
    ...authors.map((a) => entry(`/authors/${a.slug}`)),
  ];
}
