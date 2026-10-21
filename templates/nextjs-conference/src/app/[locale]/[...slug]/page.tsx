import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { Faq } from "@/components/Faq";
import { Markdown } from "@/components/Markdown";
import { defaultLocale, getDictionary, isLocale, locales } from "@/i18n";
import { getPage, getSlugs, setupScreen } from "@/lib/gohead";

/**
 * Pages of the `pages` collection, at /<locale>/<slug>: code of conduct,
 * practical information… Editors add one in GoHead and it exists. Any other
 * path under a locale gets the localized 404 page.
 */
export async function generateStaticParams() {
  const slugs = await getSlugs("pages");
  return locales.flatMap((locale) => slugs.map((slug) => ({ locale, slug: [slug] })));
}

async function load(params: PageProps<"/[locale]/[...slug]">["params"]) {
  const { locale, slug } = await params;
  if (!isLocale(locale) || slug.length !== 1 || (await setupScreen())) return null;
  const page = await getPage(slug[0]!, locale);
  return page ? { locale, page } : null;
}

export async function generateMetadata({ params }: PageProps<"/[locale]/[...slug]">): Promise<Metadata> {
  const found = await load(params);
  if (!found) return {};
  const { locale, page } = found;
  const seo = locale === defaultLocale ? page.seo : null;
  return {
    title: seo?.meta_title || page.title,
    description: seo?.meta_description || undefined,
    alternates: { canonical: `/${locale}/${page.slug}`, languages: Object.fromEntries(locales.map((l) => [l, `/${l}/${page.slug}`])) },
  };
}

export default async function CmsPage({ params }: PageProps<"/[locale]/[...slug]">) {
  const found = await load(params);
  if (!found) notFound();
  const { locale, page } = found;
  const t = getDictionary(locale);
  return (
    <main id="main" className="wrap pt-12 sm:pt-16">
      <h1 className="display max-w-[18ch]">{page.title}</h1>
      <div className="mt-12 grid gap-14 lg:grid-cols-12">
        <div className="min-w-0 lg:col-span-7">
          <Markdown>{page.body}</Markdown>
        </div>
        {page.faq && page.faq.length > 0 && (
          <section aria-labelledby="faq" className="lg:col-span-5">
            <h2 id="faq" className="mb-4 text-2xl font-extrabold tracking-[-0.03em]">
              {t.faq}
            </h2>
            <Faq items={page.faq} />
          </section>
        )}
      </div>
    </main>
  );
}
