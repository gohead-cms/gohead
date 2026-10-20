import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { ArticleList } from "@/components/Articles";
import { Avatar } from "@/components/Cover";
import { getDictionary, isLocale, locales } from "@/i18n";
import { getArticlesByAuthor, getAuthor, getSlugs, mediaUrl, setupScreen } from "@/lib/gohead";

export async function generateStaticParams() {
  const slugs = await getSlugs("authors");
  return locales.flatMap((locale) => slugs.map((slug) => ({ locale, slug })));
}

export async function generateMetadata({ params }: PageProps<"/[locale]/authors/[slug]">): Promise<Metadata> {
  const { locale, slug } = await params;
  if (!isLocale(locale) || (await setupScreen())) return {};
  const author = await getAuthor(slug, locale);
  if (!author) return {};
  const avatar = mediaUrl(author.avatar);
  return {
    title: author.name,
    description: author.bio || author.role || undefined,
    alternates: {
      canonical: `/${locale}/authors/${slug}`,
      languages: Object.fromEntries(locales.map((l) => [l, `/${l}/authors/${slug}`])),
    },
    openGraph: { type: "profile", title: author.name, images: avatar ? [{ url: avatar, alt: author.name }] : undefined },
  };
}

export default async function AuthorPage({ params }: PageProps<"/[locale]/authors/[slug]">) {
  const { locale, slug } = await params;
  if (!isLocale(locale)) notFound();
  if (await setupScreen()) return null;

  const author = await getAuthor(slug, locale);
  if (!author) notFound();
  const articles = await getArticlesByAuthor(author.id, locale);
  const t = getDictionary(locale);

  return (
    <main id="main" className="wrap">
      <header className="flex flex-col gap-8 pt-10 pb-12 sm:flex-row sm:items-center sm:pt-16">
        <Avatar media={author.avatar} name={author.name} size={128} />
        <div className="min-w-0">
          <h1 className="text-[clamp(2.75rem,7vw,5.5rem)] leading-[0.92] font-extrabold tracking-[-0.045em]">{author.name}</h1>
          {author.role && <p className="mt-3 text-lg font-medium text-muted">{author.role}</p>}
          {author.bio && <p className="mt-4 max-w-[52ch] font-serif text-xl leading-relaxed">{author.bio}</p>}
        </div>
      </header>
      <h2 className="mb-4 text-2xl font-extrabold tracking-[-0.03em]">
        {t.articlesBy(author.name)} <span className="font-medium text-muted">({articles.length})</span>
      </h2>
      {articles.length > 0 ? (
        <ArticleList articles={articles} locale={locale} />
      ) : (
        <p className="border-t border-rule pt-8 font-serif text-lg text-muted">{t.noArticles}</p>
      )}
    </main>
  );
}
