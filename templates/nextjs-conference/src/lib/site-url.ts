/**
 * The public URL of this site, for canonical links, Open Graph, the sitemap
 * and the RSS feed. Set NEXT_PUBLIC_SITE_URL in production; on Vercel the
 * production domain is used when it is not set.
 */
export function siteUrl(): URL {
  const configured = process.env.NEXT_PUBLIC_SITE_URL;
  if (configured) return new URL(configured);
  const vercel = process.env.VERCEL_PROJECT_PRODUCTION_URL;
  if (vercel) return new URL(`https://${vercel}`);
  return new URL(`http://localhost:${process.env.PORT || 3000}`);
}

/** An absolute URL on this site. */
export function absoluteUrl(path: string): string {
  return new URL(path, siteUrl()).toString();
}
