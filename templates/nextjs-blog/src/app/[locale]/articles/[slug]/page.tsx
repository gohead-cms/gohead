import { populated } from "@gohead-cms/client";
import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";

import { ArticleRow } from "@/components/Articles";
import { Avatar, Cover } from "@/components/Cover";
import { Markdown } from "@/components/Markdown";
import { defaultLocale, formatDate, getDictionary, isLocale, locales } from "@/i18n";
import {
  authorOf,
  categoryOf,
  getArticle,
  getArticlesInCategory,
  getSite,
  getSlugs,
  mediaUrl,
  readingMinutes,
  setupScreen,
  tagsOf,
} from "@/lib/gohead";

export async function generateStaticParams() {
  const slugs = await getSlugs("articles");
  return locales.flatMap((locale) => slugs.map((slug) => ({ locale, slug })));
}

export async function generateMetadata({ params }: PageProps<"/[locale]/articles/[slug]">): Promise<Metadata> {
  const { locale, slug } = await params;
  if (!isLocale(locale) || (await setupScreen())) return {};
  const article = await getArticle(slug, locale);
  if (!article) return {};

  // The starter model shares the seo component between locales, so its texts
  // are used in the default locale only. Make `seo` localized in
  // gohead/model.json to translate them too, and drop this condition.
  const seo = locale === defaultLocale ? article.seo : null;
  const title = seo?.meta_title || article.title;
  const description = seo?.meta_description || article.excerpt || undefined;
  const site = await getSite(locale);
  const image = populated(article.seo?.share_image) ?? populated(article.cover) ?? populated(site.seo?.share_image);
  const imageUrl = mediaUrl(image);

  return {
    title,
    description,
    alternates: {
      canonical: `/${locale}/articles/${slug}`,
      languages: Object.fromEntries(locales.map((l) => [l, `/${l}/articles/${slug}`])),
    },
    openGraph: {
      type: "article",
      title,
      description,
      locale,
      publishedTime: article.publishedAt ?? undefined,
      modifiedTime: article.updatedAt,
      authors: authorOf(article)?.name ? [authorOf(article)!.name] : undefined,
      tags: tagsOf(article),
      images: imageUrl
        ? [{ url: imageUrl, width: image?.width ?? undefined, height: image?.height ?? undefined, alt: image?.alt || title }]
        : undefined,
    },
  };
}

export default async function ArticlePage({ params }: PageProps<"/[locale]/articles/[slug]">) {
  const { locale, slug } = await params;
  if (!isLocale(locale)) notFound();
  if (await setupScreen()) return null;

  const article = await getArticle(slug, locale);
  if (!article) notFound();

  const t = getDictionary(locale);
  const author = authorOf(article);
  const category = categoryOf(article);
  const tags = tagsOf(article);
  const date = formatDate(article.publishedAt, locale);
  const related = category
    ? (await getArticlesInCategory(category.id, locale)).filter((a) => a.id !== article.id).slice(0, 3)
    : [];

  return (
    <main id="main">
      <article>
        <header className="wrap pt-10 sm:pt-16">
          <div className="max-w-[58rem]">
            {category && (
              <Link href={`/${locale}/categories/${category.slug}`} className="marker font-semibold">
                {category.name}
              </Link>
            )}
            <h1 className="mt-4 text-[clamp(2.5rem,6.5vw,5.25rem)] leading-[0.95] font-extrabold tracking-[-0.045em] text-balance">
              {article.title}
            </h1>
            {article.excerpt && (
              <p className="mt-6 max-w-[48ch] font-serif text-xl leading-relaxed text-muted sm:text-2xl">{article.excerpt}</p>
            )}
          </div>

          <div className="mt-8 flex flex-wrap items-center gap-x-8 gap-y-4 border-t border-rule pt-5">
            {author && (
              <Link href={`/${locale}/authors/${author.slug}`} className="group flex items-center gap-3">
                <Avatar media={author.avatar} name={author.name} size={44} />
                <span className="flex flex-col leading-tight">
                  <span className="font-semibold group-hover:underline">{author.name}</span>
                  {author.role && <span className="text-sm text-muted">{author.role}</span>}
                </span>
              </Link>
            )}
            <dl className="flex flex-wrap gap-x-8 gap-y-2 text-sm">
              {date && (
                <div className="flex flex-col leading-tight">
                  <dt className="text-muted">{t.publishedOn}</dt>
                  <dd className="font-medium">
                    <time dateTime={article.publishedAt ?? undefined}>{date}</time>
                  </dd>
                </div>
              )}
              <div className="flex flex-col leading-tight">
                <dt className="text-muted">{t.readingTimeLabel}</dt>
                <dd className="font-medium">{t.readingTime(readingMinutes(article))}</dd>
              </div>
            </dl>
          </div>
        </header>

        {article.cover && (
          <div className="wrap mt-10">
            <Cover media={article.cover} ratio="16 / 8" sizes="(min-width: 1216px) 1136px, 100vw" preload />
          </div>
        )}

        <div className="wrap mt-12 grid gap-12 lg:grid-cols-12">
          <div className="min-w-0 lg:col-span-8 lg:col-start-3 xl:col-span-7 xl:col-start-3">
            <Markdown>{article.body}</Markdown>

            {tags.length > 0 && (
              <div className="mt-12 flex flex-wrap items-center gap-2">
                <h2 className="sr-only">{t.tags}</h2>
                {tags.map((tag) => (
                  <span key={tag} className="rounded-full border border-rule px-3 py-1 text-sm text-muted">
                    #{tag}
                  </span>
                ))}
              </div>
            )}

            {author && (author.bio || author.role) && (
              <aside className="mt-12 flex gap-5 rounded-md bg-paper-raised p-6 shadow-[inset_0_0_0_1px_var(--rule)]">
                <Avatar media={author.avatar} name={author.name} size={64} />
                <div>
                  <p className="text-lg font-bold">
                    <Link href={`/${locale}/authors/${author.slug}`} className="hover:underline">
                      {author.name}
                    </Link>
                  </p>
                  {author.role && <p className="text-sm text-muted">{author.role}</p>}
                  {author.bio && <p className="mt-3 font-serif leading-relaxed">{author.bio}</p>}
                </div>
              </aside>
            )}
          </div>
        </div>
      </article>

      {category && related.length > 0 && (
        <section aria-labelledby="related" className="wrap mt-24">
          <div className="mb-4 flex flex-wrap items-baseline justify-between gap-4">
            <h2 id="related" className="text-3xl font-extrabold tracking-[-0.03em]">
              {t.moreIn(category.name)}
            </h2>
            <Link href={`/${locale}/categories/${category.slug}`} className="font-semibold text-link hover:underline">
              {t.seeAll}
            </Link>
          </div>
          <div className="border-b border-rule">
            {related.map((a) => (
              <ArticleRow key={a.id} article={a} locale={locale} />
            ))}
          </div>
        </section>
      )}
    </main>
  );
}
