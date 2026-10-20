import Link from "next/link";

import { Avatar, Cover } from "@/components/Cover";
import { formatDate, getDictionary, type Locale } from "@/i18n";
import { authorOf, categoryOf, readingMinutes, type ArticleEntry } from "@/lib/gohead";

const articleHref = (locale: Locale, a: ArticleEntry) => `/${locale}/articles/${a.slug}`;

function CategoryLink({ article, locale }: { article: ArticleEntry; locale: Locale }) {
  const category = categoryOf(article);
  if (!category) return null;
  return (
    <Link
      href={`/${locale}/categories/${category.slug}`}
      className="marker relative z-10 self-start text-sm font-semibold"
    >
      {category.name}
    </Link>
  );
}

function Byline({ article, locale, avatar = false }: { article: ArticleEntry; locale: Locale; avatar?: boolean }) {
  const t = getDictionary(locale);
  const author = authorOf(article);
  const date = formatDate(article.publishedAt, locale);
  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-muted">
      {author && (
        <Link
          href={`/${locale}/authors/${author.slug}`}
          className="relative z-10 flex items-center gap-2 font-medium text-ink hover:underline"
        >
          {avatar && <Avatar media={author.avatar} name={author.name} size={28} />}
          {author.name}
        </Link>
      )}
      {date && <time dateTime={article.publishedAt ?? undefined}>{date}</time>}
      <span>{t.readingTime(readingMinutes(article))}</span>
    </div>
  );
}

/** The first featured article: a large cover beside a large headline. */
export function LeadStory({ article, locale }: { article: ArticleEntry; locale: Locale }) {
  return (
    <article className="group relative grid gap-6 lg:grid-cols-12 lg:gap-10">
      <Cover
        media={article.cover}
        sizes="(min-width: 1216px) 680px, (min-width: 1024px) 58vw, 100vw"
        preload
        className="lg:col-span-7"
      />
      <div className="flex flex-col gap-4 lg:col-span-5 lg:justify-end lg:pb-2">
        <CategoryLink article={article} locale={locale} />
        <h3 className="text-[2rem] leading-[1.05] font-extrabold tracking-[-0.03em] text-balance sm:text-5xl">
          <Link href={articleHref(locale, article)} className="headline-link after:absolute after:inset-0">
            {article.title}
          </Link>
        </h3>
        {article.excerpt && <p className="font-serif text-lg leading-relaxed text-muted">{article.excerpt}</p>}
        <Byline article={article} locale={locale} avatar />
      </div>
    </article>
  );
}

/** A featured article after the lead. */
export function FeatureCard({ article, locale }: { article: ArticleEntry; locale: Locale }) {
  return (
    <article className="group relative grid grid-cols-[7rem_1fr] gap-5 sm:grid-cols-[11rem_1fr]">
      <Cover media={article.cover} ratio="1 / 1" sizes="(min-width: 640px) 176px, 112px" />
      <div className="flex flex-col gap-2">
        <CategoryLink article={article} locale={locale} />
        <h3 className="text-xl leading-tight font-bold tracking-[-0.02em] text-balance sm:text-2xl">
          <Link href={articleHref(locale, article)} className="headline-link after:absolute after:inset-0">
            {article.title}
          </Link>
        </h3>
        <Byline article={article} locale={locale} />
      </div>
    </article>
  );
}

/** A row of an article list: date, headline and excerpt, thumbnail. */
export function ArticleRow({ article, locale }: { article: ArticleEntry; locale: Locale }) {
  const date = formatDate(article.publishedAt, locale);
  const author = authorOf(article);
  const t = getDictionary(locale);
  return (
    <article className="group relative grid grid-cols-[1fr_5.5rem] gap-x-5 gap-y-2 border-t border-rule py-7 sm:grid-cols-[1fr_9rem] md:grid-cols-[10rem_1fr_11rem] md:gap-x-8">
      <div className="col-span-2 flex flex-wrap gap-x-4 text-sm text-muted md:col-span-1 md:flex-col md:gap-1 md:pt-1.5">
        {date && <time dateTime={article.publishedAt ?? undefined}>{date}</time>}
        <span>{t.readingTime(readingMinutes(article))}</span>
      </div>
      <div className="flex min-w-0 flex-col gap-2">
        <h3 className="text-xl leading-tight font-bold tracking-[-0.02em] text-balance sm:text-[1.65rem]">
          <Link href={articleHref(locale, article)} className="headline-link after:absolute after:inset-0">
            {article.title}
          </Link>
        </h3>
        {article.excerpt && (
          <p className="line-clamp-3 max-w-[60ch] font-serif leading-relaxed text-muted">{article.excerpt}</p>
        )}
        <div className="flex flex-wrap items-center gap-x-4 gap-y-1 pt-1 text-sm">
          <CategoryLink article={article} locale={locale} />
          {author && (
            <Link href={`/${locale}/authors/${author.slug}`} className="relative z-10 text-muted hover:text-ink hover:underline">
              {author.name}
            </Link>
          )}
        </div>
      </div>
      <Cover media={article.cover} ratio="4 / 3" sizes="(min-width: 768px) 176px, (min-width: 640px) 144px, 88px" className="self-start" />
    </article>
  );
}

export function ArticleList({ articles, locale }: { articles: ArticleEntry[]; locale: Locale }) {
  return (
    <div className="border-b border-rule">
      {articles.map((a) => (
        <ArticleRow key={a.id} article={a} locale={locale} />
      ))}
    </div>
  );
}
