import type { MetadataRoute } from "next";

import { locales } from "@/i18n";
import { getSetupIssue, getSlugs } from "@/lib/gohead";
import { absoluteUrl } from "@/lib/site-url";

export const revalidate = 3600;

/** Every page in every locale, each listing its translations. */
export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const entry = (path: string): MetadataRoute.Sitemap[number] => ({
    url: absoluteUrl(`/${locales[0]}${path}`),
    alternates: { languages: Object.fromEntries(locales.map((l) => [l, absoluteUrl(`/${l}${path}`)])) },
  });
  if (await getSetupIssue()) return [entry("")];

  const [talks, speakers, posts, pages] = await Promise.all([getSlugs("talks"), getSlugs("speakers"), getSlugs("posts"), getSlugs("pages")]);
  return [
    ...["", "/schedule", "/speakers", "/news", "/sponsors", "/tickets", "/cfp", "/feedback"].map(entry),
    ...talks.map((s) => entry(`/talks/${s}`)),
    ...speakers.map((s) => entry(`/speakers/${s}`)),
    ...posts.map((s) => entry(`/news/${s}`)),
    ...pages.map((s) => entry(`/${s}`)),
  ];
}
