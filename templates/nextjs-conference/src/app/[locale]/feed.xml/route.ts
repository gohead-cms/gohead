import { isLocale, locales } from "@/i18n";
import { getFeedPosts, getSetupIssue, getSite } from "@/lib/gohead";
import { absoluteUrl } from "@/lib/site-url";

export function generateStaticParams() {
  return locales.map((locale) => ({ locale }));
}

const escape = (s: string) =>
  s.replace(/[<>&'"]/g, (c) => ({ "<": "&lt;", ">": "&gt;", "&": "&amp;", "'": "&apos;", '"': "&quot;" })[c]!);

/** RSS 2.0 feed of the 20 latest posts in a locale: /en/feed.xml. Published posts only, even in draft mode. */
export async function GET(_request: Request, { params }: RouteContext<"/[locale]/feed.xml">) {
  const { locale } = await params;
  if (!isLocale(locale)) return new Response("Not found", { status: 404 });
  const issue = await getSetupIssue();
  if (issue) return new Response(`GoHead is not ready: ${issue.detail}`, { status: 503 });

  const [site, latest] = await Promise.all([getSite(locale), getFeedPosts(locale)]);
  const home = absoluteUrl(`/${locale}`);
  const items = latest.data.map((p) => {
    const url = absoluteUrl(`/${locale}/news/${p.slug}`);
    return [
      "<item>",
      `<title>${escape(p.title)}</title>`,
      `<link>${url}</link>`,
      `<guid isPermaLink="true">${url}</guid>`,
      p.publishedAt ? `<pubDate>${new Date(p.publishedAt).toUTCString()}</pubDate>` : "",
      p.excerpt ? `<description>${escape(p.excerpt)}</description>` : "",
      p.kind ? `<category>${escape(p.kind)}</category>` : "",
      "</item>",
    ].join("");
  });

  const xml = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
<channel>
<title>${escape(site.name)}</title>
<link>${home}</link>
<description>${escape(site.description || site.tagline || site.name)}</description>
<language>${locale}</language>
<atom:link href="${absoluteUrl(`/${locale}/feed.xml`)}" rel="self" type="application/rss+xml"/>
${items.join("\n")}
</channel>
</rss>
`;
  return new Response(xml, { headers: { "Content-Type": "application/rss+xml; charset=utf-8" } });
}
