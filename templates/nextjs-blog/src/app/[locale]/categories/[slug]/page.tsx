import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { ArticleList } from "@/components/Articles";
import { getDictionary, isLocale, locales } from "@/i18n";
import { getArticlesInCategory, getCategory, getSlugs, setupScreen } from "@/lib/gohead";

export async function generateStaticParams() {
  const slugs = await getSlugs("categories");
  return locales.flatMap((locale) => slugs.map((slug) => ({ locale, slug })));
}

export async function generateMetadata({ params }: PageProps<"/[locale]/categories/[slug]">): Promise<Metadata> {
  const { locale, slug } = await params;
  if (!isLocale(locale) || (await setupScreen())) return {};
  const category = await getCategory(slug, locale);
  if (!category) return {};
  return {
    title: category.name,
    description: category.description || undefined,
    alternates: {
      canonical: `/${locale}/categories/${slug}`,
      languages: Object.fromEntries(locales.map((l) => [l, `/${l}/categories/${slug}`])),
    },
  };
}

export default async function CategoryPage({ params }: PageProps<"/[locale]/categories/[slug]">) {
  const { locale, slug } = await params;
  if (!isLocale(locale)) notFound();
  if (await setupScreen()) return null;

  const category = await getCategory(slug, locale);
  if (!category) notFound();
  const articles = await getArticlesInCategory(category.id, locale);
  const t = getDictionary(locale);

  return (
    <main id="main" className="wrap">
      <header className="grid gap-6 pt-10 pb-12 sm:pt-16 lg:grid-cols-12 lg:items-end lg:gap-10">
        <h1 className="text-[clamp(3rem,9vw,7.25rem)] leading-[0.9] font-extrabold tracking-[-0.05em] lg:col-span-8">
          {category.name}
        </h1>
        <div className="lg:col-span-4 lg:pb-3">
          {category.description && <p className="font-serif text-xl leading-relaxed text-muted">{category.description}</p>}
          <p className="mt-2 text-sm font-medium">{t.articleCount(articles.length)}</p>
        </div>
      </header>
      {articles.length > 0 ? (
        <ArticleList articles={articles} locale={locale} />
      ) : (
        <p className="border-t border-rule pt-8 font-serif text-lg text-muted">{t.noArticles}</p>
      )}
    </main>
  );
}
