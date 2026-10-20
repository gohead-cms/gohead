import { isLocale, locales } from "@/i18n";
import { authorOf, categoryOf, getLatestArticles, getSetupIssue, getSite } from "@/lib/gohead";
import { absoluteUrl } from "@/lib/site-url";

export function generateStaticParams() {
  return locales.map((locale) => ({ locale }));
}

const escape = (s: string) =>
  s.replace(/[<>&'"]/g, (c) => ({ "<": "&lt;", ">": "&gt;", "&": "&amp;", "'": "&apos;", '"': "&quot;" })[c]!);

/** RSS 2.0 feed of the 20 latest articles in a locale: /en/feed.xml. */
export async function GET(_request: Request, { params }: RouteContext<"/[locale]/feed.xml">) {
  const { locale } = await params;
  if (!isLocale(locale)) return new Response("Not found", { status: 404 });
  const issue = await getSetupIssue();
  if (issue) return new Response(`GoHead is not ready: ${issue.detail}`, { status: 503 });

  const [site, latest] = await Promise.all([getSite(locale), getLatestArticles(locale, 1, 20)]);
  const home = absoluteUrl(`/${locale}`);
  const items = latest.data.map((a) => {
    const url = absoluteUrl(`/${locale}/articles/${a.slug}`);
    const author = authorOf(a);
    const category = categoryOf(a);
    return [
      "<item>",
      `<title>${escape(a.title)}</title>`,
      `<link>${url}</link>`,
      `<guid isPermaLink="true">${url}</guid>`,
      a.publishedAt ? `<pubDate>${new Date(a.publishedAt).toUTCString()}</pubDate>` : "",
      a.excerpt ? `<description>${escape(a.excerpt)}</description>` : "",
      author ? `<dc:creator>${escape(author.name)}</dc:creator>` : "",
      category ? `<category>${escape(category.name)}</category>` : "",
      "</item>",
    ].join("");
  });

  const xml = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom" xmlns:dc="http://purl.org/dc/elements/1.1/">
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
