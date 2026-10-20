import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { ArticleList, FeatureCard, LeadStory } from "@/components/Articles";
import { Pagination } from "@/components/Pagination";
import { getDictionary, isLocale, locales } from "@/i18n";
import { getHomepage, getLatestArticles, setupScreen } from "@/lib/gohead";

const PAGE_SIZE = 5;

function pageNumber(value: string | string[] | undefined): number | null {
  if (value === undefined) return 1;
  const n = Number(Array.isArray(value) ? value[0] : value);
  return Number.isInteger(n) && n >= 1 ? n : null;
}

export async function generateMetadata({ params, searchParams }: PageProps<"/[locale]">): Promise<Metadata> {
  const [{ locale }, { page }] = await Promise.all([params, searchParams]);
  if (!isLocale(locale)) return {};
  const n = pageNumber(page) ?? 1;
  const path = n > 1 ? `/${locale}?page=${n}` : `/${locale}`;
  return {
    ...(n > 1 ? { title: `${getDictionary(locale).latest}, ${getDictionary(locale).page(n).toLowerCase()}` } : {}),
    alternates: {
      canonical: path,
      languages: Object.fromEntries(locales.map((l) => [l, `/${l}`])),
    },
  };
}

export default async function HomePage({ params, searchParams }: PageProps<"/[locale]">) {
  const [{ locale }, { page: pageParam }] = await Promise.all([params, searchParams]);
  if (!isLocale(locale)) notFound();
  if (await setupScreen()) return null; // the layout shows the setup screen

  const page = pageNumber(pageParam);
  if (page === null) notFound();

  const t = getDictionary(locale);
  const [home, latest] = await Promise.all([getHomepage(locale), getLatestArticles(locale, page, PAGE_SIZE)]);
  const { pageCount } = latest.meta.pagination;
  if (page > 1 && page > pageCount) notFound();

  const featured = (home?.featured ?? []).filter((a) => typeof a === "object" && a !== null);
  const [lead, ...others] = featured;

  return (
    <main id="main">
      {page === 1 && (home?.hero_title || home?.hero_text) && (
        <section className="wrap grid gap-6 pt-10 pb-14 sm:pt-16 lg:grid-cols-12 lg:items-end lg:gap-10 lg:pb-20">
          {home.hero_title && (
            <h1 className="text-[clamp(3rem,9vw,7.25rem)] leading-[0.9] font-extrabold tracking-[-0.05em] text-balance lg:col-span-8">
              {home.hero_title}
            </h1>
          )}
          {home.hero_text && (
            <p className="max-w-[42ch] font-serif text-xl leading-relaxed text-muted lg:col-span-4 lg:pb-3">
              {home.hero_text}
            </p>
          )}
        </section>
      )}

      {page === 1 && lead && (
        <section aria-labelledby="featured" className="wrap">
          <h2 id="featured" className="sr-only">
            {t.featured}
          </h2>
          <LeadStory article={lead} locale={locale} />
          {others.length > 0 && (
            <div className="mt-12 grid gap-10 border-t border-rule pt-10 md:grid-cols-2 md:gap-12">
              {others.map((a) => (
                <FeatureCard key={a.id} article={a} locale={locale} />
              ))}
            </div>
          )}
        </section>
      )}

      <section aria-labelledby="latest" className="wrap mt-20 sm:mt-24">
        <h2 id="latest" className="mb-6 text-3xl font-extrabold tracking-[-0.03em] sm:text-4xl">
          {page > 1 ? `${t.latest}, ${t.page(page).toLowerCase()}` : t.latest}
        </h2>
        {latest.data.length > 0 ? (
          <ArticleList articles={latest.data} locale={locale} />
        ) : (
          <p className="max-w-[56ch] border-t border-rule pt-8 font-serif text-lg text-muted">{t.noArticles}</p>
        )}
        <Pagination basePath={`/${locale}`} page={page} pageCount={pageCount} t={t} />
      </section>
    </main>
  );
}
