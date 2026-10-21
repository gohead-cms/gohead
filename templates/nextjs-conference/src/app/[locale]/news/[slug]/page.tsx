import { populated } from "@gohead-cms/client";
import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { Cover } from "@/components/Cover";
import { Markdown } from "@/components/Markdown";
import { defaultLocale, formatDate, getDictionary, isLocale, locales } from "@/i18n";
import { getPost, getSlugs, mediaUrl, setupScreen } from "@/lib/gohead";

export async function generateStaticParams() {
  const slugs = await getSlugs("posts");
  return locales.flatMap((locale) => slugs.map((slug) => ({ locale, slug })));
}

export async function generateMetadata({ params }: PageProps<"/[locale]/news/[slug]">): Promise<Metadata> {
  const { locale, slug } = await params;
  if (!isLocale(locale) || (await setupScreen())) return {};
  const post = await getPost(slug, locale);
  if (!post) return {};
  const seo = locale === defaultLocale ? post.seo : null;
  const image = populated(post.seo?.share_image) ?? populated(post.cover);
  return {
    title: seo?.meta_title || post.title,
    description: seo?.meta_description || post.excerpt || undefined,
    alternates: { canonical: `/${locale}/news/${slug}`, languages: Object.fromEntries(locales.map((l) => [l, `/${l}/news/${slug}`])) },
    openGraph: { type: "article", publishedTime: post.publishedAt ?? undefined, images: image ? [{ url: mediaUrl(image)!, alt: image.alt ?? "" }] : undefined },
    robots: post.status === "draft" ? { index: false } : undefined,
  };
}

export default async function PostPage({ params }: PageProps<"/[locale]/news/[slug]">) {
  const { locale, slug } = await params;
  if (!isLocale(locale)) notFound();
  if (await setupScreen()) return null;
  const post = await getPost(slug, locale);
  if (!post) notFound();
  const t = getDictionary(locale);
  return (
    <main id="main">
      <article>
        <header className="wrap pt-12 sm:pt-16">
          <p className="eyebrow flex gap-3 text-muted">
            {post.kind && <span className="text-signal">{t.kind[post.kind]}</span>}
            {post.status === "draft" ? (
              <span className="rounded bg-signal px-1.5 text-on-signal">{t.draftBadge}</span>
            ) : (
              <time dateTime={post.publishedAt ?? undefined}>
                {t.publishedOn} {formatDate(post.publishedAt, locale)}
              </time>
            )}
          </p>
          <h1 className="mt-4 max-w-[20ch] text-[clamp(2.3rem,5.5vw,4.5rem)] leading-[0.95] font-extrabold tracking-[-0.045em] text-balance">
            {post.title}
          </h1>
          {post.excerpt && <p className="mt-6 max-w-[52ch] font-serif text-xl leading-relaxed text-muted">{post.excerpt}</p>}
        </header>
        {post.cover && (
          <div className="wrap mt-10">
            <Cover media={post.cover} ratio="16 / 7" sizes="(min-width: 1248px) 1168px, 100vw" preload />
          </div>
        )}
        <div className="wrap mt-12">
          <div className="max-w-[68ch] lg:ml-[16.66%]">
            <Markdown locale={locale}>{post.body}</Markdown>
          </div>
        </div>
      </article>
    </main>
  );
}
