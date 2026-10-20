/**
 * Site languages. Keep this list in sync with `locales` in gohead/model.json:
 * the first one is the default, served at `/` for visitors whose browser
 * asks for none of the others.
 */
export const locales = ["en", "fr"] as const;
export type Locale = (typeof locales)[number];
export const defaultLocale: Locale = "en";

export function isLocale(value: string): value is Locale {
  return (locales as readonly string[]).includes(value);
}

/** Interface text. Content comes from GoHead; these are the words around it. */
const en = {
  languageName: "English",
  skipToContent: "Skip to content",
  home: "Home",
  sections: "Sections",
  language: "Language",
  featured: "Featured",
  latest: "Latest articles",
  newer: "Newer",
  older: "Older",
  page: (page: number) => `Page ${page}`,
  pageOf: (page: number, total: number) => `Page ${page} of ${total}`,
  noArticles: "No articles yet. Publish one in the GoHead admin and it shows up here.",
  readingTime: (minutes: number) => `${minutes} min read`,
  readingTimeLabel: "Reading time",
  publishedOn: "Published",
  tags: "Tags",
  moreIn: (category: string) => `More in ${category}`,
  seeAll: "See all",
  articlesBy: (name: string) => `Articles by ${name}`,
  articleCount: (n: number) => (n === 1 ? "1 article" : `${n} articles`),
  rss: "RSS feed",
  links: "Links",
  notFoundTitle: "This page isn’t here",
  notFoundText: "The link may be old, or the article was unpublished.",
  backHome: "Back to the home page",
  errorTitle: "Something went wrong",
  errorText: "The page could not be rendered. Try again, or come back in a minute.",
  retry: "Try again",
};

export type Dictionary = typeof en;

const fr: Dictionary = {
  languageName: "Français",
  skipToContent: "Aller au contenu",
  home: "Accueil",
  sections: "Rubriques",
  language: "Langue",
  featured: "À la une",
  latest: "Derniers articles",
  newer: "Plus récents",
  older: "Plus anciens",
  page: (page: number) => `Page ${page}`,
  pageOf: (page: number, total: number) => `Page ${page} sur ${total}`,
  noArticles: "Aucun article pour l’instant. Publiez-en un dans l’admin GoHead et il apparaîtra ici.",
  readingTime: (minutes: number) => `${minutes} min de lecture`,
  readingTimeLabel: "Temps de lecture",
  publishedOn: "Publié le",
  tags: "Étiquettes",
  moreIn: (category: string) => `Aussi dans ${category}`,
  seeAll: "Tout voir",
  articlesBy: (name: string) => `Articles de ${name}`,
  articleCount: (n: number) => (n <= 1 ? `${n} article` : `${n} articles`),
  rss: "Flux RSS",
  links: "Liens",
  notFoundTitle: "Cette page n’existe pas",
  notFoundText: "Le lien est peut-être ancien, ou l’article a été dépublié.",
  backHome: "Retour à l’accueil",
  errorTitle: "Une erreur est survenue",
  errorText: "La page n’a pas pu être affichée. Réessayez, ou revenez dans une minute.",
  retry: "Réessayer",
};

const dictionaries: Record<Locale, Dictionary> = { en, fr };

export function getDictionary(locale: Locale): Dictionary {
  return dictionaries[locale];
}

/** "October 20, 2026" / "20 octobre 2026". */
export function formatDate(iso: string | null | undefined, locale: Locale): string | null {
  if (!iso) return null;
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return null;
  return new Intl.DateTimeFormat(locale, { dateStyle: "long", timeZone: "UTC" }).format(date);
}
