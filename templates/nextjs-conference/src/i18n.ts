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
  language: "Language",
  menu: "Menu",
  search: "Search",
  searchPlaceholder: "Talks, speakers, news…",
  searchResults: (q: string) => `Results for “${q}”`,
  searchEmpty: "Nothing matches. Try another word.",
  searchHint: "Search titles of talks and posts, and names of speakers.",

  // Home
  stats: { talks: "Sessions", speakers: "Speakers", tracks: "Tracks" },
  featuredTalks: "Don’t miss",
  featuredSpeakers: "Speakers",
  allSpeakers: "All speakers",
  fullSchedule: "Full schedule",
  faq: "Questions",
  latestNews: "News",
  allNews: "All news",
  sponsors: "Sponsors",

  // Schedule
  schedule: "Schedule",
  scheduleIntro: "Times are in the venue’s time zone.",
  allDays: "Both days",
  day: (n: number) => `Day ${n}`,
  filterBy: "Filter the schedule",
  allLevels: "All levels",
  allFormats: "All formats",
  allTracks: "All tracks",
  apply: "Apply",
  reset: "Reset",
  noTalks: "No session matches these filters.",
  addToCalendar: "Add the schedule to your calendar (.ics)",
  minutes: (n: number) => `${n} min`,
  format: { keynote: "Keynote", talk: "Talk", workshop: "Workshop", lightning: "Lightning talk", panel: "Panel" },
  level: { beginner: "Beginner", intermediate: "Intermediate", advanced: "Advanced" },
  room: { main_hall: "Main hall", room_a: "Room A", room_b: "Room B", lab: "Lab" },

  // Talk
  when: "When",
  where: "Where",
  track: "Track",
  levelLabel: "Level",
  slides: "Slides (PDF)",
  recording: "Watch the recording",
  inShort: "In short",
  inShortNote: "Written by the talk-summarizer agent.",
  photos: "Photos",
  questions: "Questions from attendees",
  noQuestions: "No question yet. Be the first to ask one.",
  answer: "Answer",
  askTitle: "Ask a question",
  askNote: "Questions appear here once a moderator has published them.",
  yourName: "Your name",
  yourQuestion: "Your question",
  send: "Send",
  sending: "Sending…",
  questionSent: "Thanks! Your question will appear once a moderator has published it.",
  draftBadge: "Draft",

  // Speakers
  speakers: "Speakers",
  speakersIntro: "The people on stage.",
  talksBy: (name: string) => `Sessions with ${name}`,

  // News
  news: "News",
  newsIntro: "Announcements and stories from the team.",
  newer: "Newer",
  older: "Older",
  pageOf: (page: number, total: number) => `Page ${page} of ${total}`,
  noPosts: "No news yet.",
  publishedOn: "Published",
  kind: { announcement: "Announcement", programme: "Programme", community: "Community" },
  rss: "RSS feed",

  // Sponsors
  sponsorsIntro: "The companies and communities that make the conference possible. This page is read with GraphQL.",
  tier: { platinum: "Platinum", gold: "Gold", silver: "Silver", community: "Community" },
  visit: "Visit website",

  // Tickets
  tickets: "Tickets",
  soldOut: "Sold out",
  buy: "Buy",
  saleEnds: (date: string) => `Until ${date}`,
  refunds: "Refunds and transfers",

  // Call for papers
  cfp: "Call for papers",
  cfpIntro: "Propose a talk, a workshop or a lightning talk. Proposals go straight to the programme committee in GoHead.",
  proposalTitle: "Title",
  abstract: "Abstract",
  abstractHint: "What will people learn? 50 to 1,500 characters.",
  formatLabel: "Format",
  languageLabel: "Language of the talk",
  speakerName: "Your name",
  email: "Email",
  emailHint: "Only the programme committee sees it.",
  bio: "Short bio (optional)",
  outline: "Outline (optional PDF, 5 MB at most)",
  submit: "Send the proposal",
  proposalSent: "Thank you! The programme committee will get back to you by email.",
  required: "Required",
  optional: "optional",

  // Newsletter and feedback
  newsletter: "Get the news",
  newsletterIntro: "One email a month about the programme. No spam.",
  subscribe: "Subscribe",
  subscribed: "You’re on the list. See you in Lyon!",
  feedback: "Feedback",
  feedbackIntro: "Tell us what works and what doesn’t. A GoHead agent reads every message, sorts it and passes it on to the team.",
  feedbackMessage: "Your message",
  feedbackEmail: "Email (optional, if you want an answer)",
  feedbackSent: "Thank you! Your message is with the team.",

  // Errors and states
  invalidEmail: "Enter a valid email address.",
  tooShort: (n: number) => `At least ${n} characters.`,
  tooLong: (n: number) => `At most ${n} characters.`,
  formError: "Something went wrong. Please try again in a minute.",
  formUnavailable: "This form isn’t set up on this site yet.",
  pdfOnly: "The outline must be a PDF of 5 MB or less.",
  previewOn: "Draft mode: you see unpublished talks and posts.",
  previewExit: "Exit draft mode",
  notFoundTitle: "This page isn’t here",
  notFoundText: "The link may be old, or the page was unpublished.",
  backHome: "Back to the home page",
  errorTitle: "Something went wrong",
  errorText: "The page could not be rendered. Try again, or come back in a minute.",
  retry: "Try again",
  poweredBy: "Built with GoHead",
};

export type Dictionary = typeof en;

const fr: Dictionary = {
  languageName: "Français",
  skipToContent: "Aller au contenu",
  language: "Langue",
  menu: "Menu",
  search: "Rechercher",
  searchPlaceholder: "Sessions, intervenants, actualités…",
  searchResults: (q: string) => `Résultats pour « ${q} »`,
  searchEmpty: "Aucun résultat. Essayez un autre mot.",
  searchHint: "Cherche dans les titres des sessions et des actualités, et les noms des intervenants.",

  stats: { talks: "Sessions", speakers: "Intervenants", tracks: "Thématiques" },
  featuredTalks: "À ne pas manquer",
  featuredSpeakers: "Intervenants",
  allSpeakers: "Tous les intervenants",
  fullSchedule: "Tout le programme",
  faq: "Questions",
  latestNews: "Actualités",
  allNews: "Toutes les actualités",
  sponsors: "Partenaires",

  schedule: "Programme",
  scheduleIntro: "Les horaires sont à l’heure du lieu.",
  allDays: "Les deux jours",
  day: (n: number) => `Jour ${n}`,
  filterBy: "Filtrer le programme",
  allLevels: "Tous niveaux",
  allFormats: "Tous formats",
  allTracks: "Toutes thématiques",
  apply: "Appliquer",
  reset: "Réinitialiser",
  noTalks: "Aucune session ne correspond à ces filtres.",
  addToCalendar: "Ajouter le programme à votre agenda (.ics)",
  minutes: (n: number) => `${n} min`,
  format: { keynote: "Keynote", talk: "Conférence", workshop: "Atelier", lightning: "Lightning talk", panel: "Table ronde" },
  level: { beginner: "Débutant", intermediate: "Intermédiaire", advanced: "Avancé" },
  room: { main_hall: "Grande salle", room_a: "Salle A", room_b: "Salle B", lab: "Labo" },

  when: "Quand",
  where: "Où",
  track: "Thématique",
  levelLabel: "Niveau",
  slides: "Supports (PDF)",
  recording: "Voir la vidéo",
  inShort: "En bref",
  inShortNote: "Écrit par l’agent talk-summarizer.",
  photos: "Photos",
  questions: "Questions du public",
  noQuestions: "Pas encore de question. Posez la première.",
  answer: "Réponse",
  askTitle: "Poser une question",
  askNote: "Les questions apparaissent ici une fois publiées par un modérateur.",
  yourName: "Votre nom",
  yourQuestion: "Votre question",
  send: "Envoyer",
  sending: "Envoi…",
  questionSent: "Merci ! Votre question apparaîtra une fois publiée par un modérateur.",
  draftBadge: "Brouillon",

  speakers: "Intervenants",
  speakersIntro: "Les personnes sur scène.",
  talksBy: (name: string) => `Sessions avec ${name}`,

  news: "Actualités",
  newsIntro: "Annonces et nouvelles de l’équipe.",
  newer: "Plus récentes",
  older: "Plus anciennes",
  pageOf: (page: number, total: number) => `Page ${page} sur ${total}`,
  noPosts: "Pas encore d’actualité.",
  publishedOn: "Publié le",
  kind: { announcement: "Annonce", programme: "Programme", community: "Communauté" },
  rss: "Flux RSS",

  sponsorsIntro: "Les entreprises et communautés qui rendent la conférence possible. Cette page est lue avec GraphQL.",
  tier: { platinum: "Platine", gold: "Or", silver: "Argent", community: "Communauté" },
  visit: "Voir le site",

  tickets: "Billets",
  soldOut: "Épuisé",
  buy: "Acheter",
  saleEnds: (date: string) => `Jusqu’au ${date}`,
  refunds: "Remboursement et cession",

  cfp: "Appel à propositions",
  cfpIntro: "Proposez une conférence, un atelier ou un lightning talk. Les propositions arrivent directement au comité de programme, dans GoHead.",
  proposalTitle: "Titre",
  abstract: "Résumé",
  abstractHint: "Qu’apprendra le public ? De 50 à 1 500 caractères.",
  formatLabel: "Format",
  languageLabel: "Langue de la session",
  speakerName: "Votre nom",
  email: "E-mail",
  emailHint: "Seul le comité de programme le voit.",
  bio: "Courte bio (facultatif)",
  outline: "Plan (PDF facultatif, 5 Mo au plus)",
  submit: "Envoyer la proposition",
  proposalSent: "Merci ! Le comité de programme vous répondra par e-mail.",
  required: "Obligatoire",
  optional: "facultatif",

  newsletter: "Recevoir les nouvelles",
  newsletterIntro: "Un e-mail par mois sur le programme. Pas de spam.",
  subscribe: "S’inscrire",
  subscribed: "C’est noté. À bientôt à Lyon !",
  feedback: "Votre avis",
  feedbackIntro: "Dites-nous ce qui marche et ce qui ne marche pas. Un agent GoHead lit chaque message, le classe et le transmet à l’équipe.",
  feedbackMessage: "Votre message",
  feedbackEmail: "E-mail (facultatif, pour recevoir une réponse)",
  feedbackSent: "Merci ! Votre message est transmis à l’équipe.",

  invalidEmail: "Saisissez une adresse e-mail valide.",
  tooShort: (n: number) => `Au moins ${n} caractères.`,
  tooLong: (n: number) => `Au plus ${n} caractères.`,
  formError: "Une erreur est survenue. Réessayez dans une minute.",
  formUnavailable: "Ce formulaire n’est pas encore configuré sur ce site.",
  pdfOnly: "Le plan doit être un PDF de 5 Mo au plus.",
  previewOn: "Mode brouillon : vous voyez les sessions et actualités non publiées.",
  previewExit: "Quitter le mode brouillon",
  notFoundTitle: "Cette page n’existe pas",
  notFoundText: "Le lien est peut-être ancien, ou la page a été dépubliée.",
  backHome: "Retour à l’accueil",
  errorTitle: "Une erreur est survenue",
  errorText: "La page n’a pas pu être affichée. Réessayez, ou revenez dans une minute.",
  retry: "Réessayer",
  poweredBy: "Construit avec GoHead",
};

const dictionaries: Record<Locale, Dictionary> = { en, fr };

export function getDictionary(locale: Locale): Dictionary {
  return dictionaries[locale];
}

/** "20 October 2026" / "20 octobre 2026" (dates without a time are read in UTC). */
export function formatDate(iso: string | null | undefined, locale: Locale, timeZone = "UTC"): string | null {
  if (!iso) return null;
  const date = new Date(iso.length === 10 ? `${iso}T12:00:00Z` : iso);
  if (Number.isNaN(date.getTime())) return null;
  return new Intl.DateTimeFormat(locale === "en" ? "en-GB" : locale, { dateStyle: "long", timeZone }).format(date);
}

/** "11–12 March 2027" / "11–12 mars 2027". */
export function formatDateRange(start: string | null | undefined, end: string | null | undefined, locale: Locale): string | null {
  if (!start) return null;
  const fmt = new Intl.DateTimeFormat(locale === "en" ? "en-GB" : locale, { day: "numeric", month: "long", year: "numeric", timeZone: "UTC" });
  const a = new Date(`${start}T12:00:00Z`);
  return end && end !== start ? fmt.formatRange(a, new Date(`${end}T12:00:00Z`)) : fmt.format(a);
}

/** "Thursday 11 March" / "jeudi 11 mars". */
export function formatDay(iso: string, locale: Locale, timeZone = "UTC"): string {
  const date = new Date(iso.length === 10 ? `${iso}T12:00:00Z` : iso);
  return new Intl.DateTimeFormat(locale === "en" ? "en-GB" : locale, { weekday: "long", day: "numeric", month: "long", timeZone }).format(date);
}

/** "09:30" in the venue's time zone. */
export function formatTime(iso: string, locale: Locale, timeZone = "UTC"): string {
  return new Intl.DateTimeFormat(locale === "en" ? "en-GB" : locale, { hour: "2-digit", minute: "2-digit", timeZone }).format(new Date(iso));
}

/** The calendar day (YYYY-MM-DD) of an instant in a time zone. */
export function dayOf(iso: string, timeZone = "UTC"): string {
  return new Intl.DateTimeFormat("en-CA", { year: "numeric", month: "2-digit", day: "2-digit", timeZone }).format(new Date(iso));
}

/** €249 / 249 € */
export function formatPrice(amount: number, currency: string | null | undefined, locale: Locale): string {
  return new Intl.NumberFormat(locale === "en" ? "en-GB" : locale, {
    style: "currency",
    currency: currency || "EUR",
    maximumFractionDigits: Number.isInteger(amount) ? 0 : 2,
  }).format(amount);
}
