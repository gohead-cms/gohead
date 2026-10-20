import { NextResponse, type NextRequest } from "next/server";

import { defaultLocale, isLocale, locales, type Locale } from "@/i18n";

/**
 * Every page lives under a locale (/en/…, /fr/…). A path without one, such as
 * `/` or `/articles/hello`, is redirected to the visitor's preferred locale
 * from the Accept-Language header, or the default locale.
 */
export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const first = pathname.split("/")[1] ?? "";
  if (isLocale(first)) return NextResponse.next();

  const url = request.nextUrl.clone();
  url.pathname = `/${preferredLocale(request.headers.get("accept-language"))}${pathname === "/" ? "" : pathname}`;
  const response = NextResponse.redirect(url);
  response.headers.set("Vary", "Accept-Language");
  return response;
}

/** The best supported locale of an Accept-Language header, e.g. "fr-CH, fr;q=0.9, en;q=0.8". */
function preferredLocale(header: string | null): Locale {
  if (!header) return defaultLocale;
  const ranked = header
    .split(",")
    .map((part, index) => {
      const [tag = "", ...params] = part.trim().split(";");
      const q = params.map((p) => p.trim()).find((p) => p.startsWith("q="));
      return { tag: tag.toLowerCase(), q: q ? Number(q.slice(2)) : 1, index };
    })
    .filter((l) => l.tag && l.q > 0)
    .sort((a, b) => b.q - a.q || a.index - b.index);
  for (const { tag } of ranked) {
    const match = locales.find((l) => tag === l || tag.startsWith(`${l}-`));
    if (match) return match;
  }
  return defaultLocale;
}

export const config = {
  // Everything except API routes, Next.js internals and files (paths with a dot,
  // such as /favicon.ico, /sitemap.xml or /robots.txt).
  matcher: ["/((?!api/|_next/|.*\\.).*)"],
};
