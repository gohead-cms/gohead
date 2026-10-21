import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";

import { Cover } from "@/components/Cover";
import { Pagination } from "@/components/Pagination";
import { formatDate, getDictionary, isLocale } from "@/i18n";
import { getPosts, setupScreen } from "@/lib/gohead";

export async function generateMetadata({ params }: PageProps<"/[locale]/news">): Promise<Metadata> {
  const { locale } = await params;
  return isLocale(locale) ? { title: getDictionary(locale).news } : {};
}

/** Posts, newest first, 6 per page: GoHead's `page` / `pageSize` and `meta.pagination`. */
export default async function NewsPage({ params, searchParams }: PageProps<"/[locale]/news">) {
  const { locale } = await params;
  if (!isLocale(locale)) notFound();
  if (await setupScreen()) return null;
  const t = getDictionary(locale);
  const raw = (await searchParams).page;
  const page = Math.max(1, Math.min(1000, Number.parseInt(Array.isArray(raw) ? raw[0]! : (raw ?? "1"), 10) || 1));
  const { data, meta } = await getPosts(locale, page);
  if (page > 1 && data.length === 0) notFound();

  return (
    <main id="main" className="wrap pt-12 sm:pt-16">
      <h1 className="display">{t.news}</h1>
      <p className="mt-4 text-lg text-muted">{t.newsIntro}</p>
      {data.length === 0 && <p className="mt-10 text-muted">{t.noPosts}</p>}
      <div className="mt-12 grid gap-x-6 gap-y-12 md:grid-cols-2 lg:grid-cols-3">
        {data.map((post) => (
          <article key={post.id} className="group relative">
            <Cover media={post.cover} ratio="16 / 9" sizes="(min-width: 1024px) 380px, (min-width: 768px) 50vw, 100vw" />
            <p className="eyebrow mt-4 flex gap-3 text-muted">
              {post.kind && <span>{t.kind[post.kind]}</span>}
              {post.status === "draft" ? (
                <span className="rounded bg-signal px-1.5 text-on-signal">{t.draftBadge}</span>
              ) : (
                <time dateTime={post.publishedAt ?? undefined}>{formatDate(post.publishedAt, locale)}</time>
              )}
            </p>
            <h2 className="mt-1 text-2xl leading-tight font-extrabold tracking-[-0.03em] text-balance">
              <Link href={`/${locale}/news/${post.slug}`} className="after:absolute after:inset-0 group-hover:underline group-hover:decoration-signal group-hover:decoration-2 group-hover:underline-offset-4">
                {post.title}
              </Link>
            </h2>
            {post.excerpt && <p className="mt-2 text-muted">{post.excerpt}</p>}
          </article>
        ))}
      </div>
      <Pagination basePath={`/${locale}/news`} page={meta.pagination.page} pageCount={meta.pagination.pageCount} t={t} />
    </main>
  );
}
