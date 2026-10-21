import { populated } from "@gohead-cms/client";
import type { Metadata, Viewport } from "next";
import { notFound } from "next/navigation";

import "../globals.css";

import { grotesk, literata } from "@/fonts";
import { PreviewBanner } from "@/components/PreviewBanner";
import { SetupScreen } from "@/components/SetupScreen";
import { SiteFooter } from "@/components/SiteFooter";
import { SiteHeader } from "@/components/SiteHeader";
import { getDictionary, isLocale, locales } from "@/i18n";
import { getSite, mediaUrl, setupScreen } from "@/lib/gohead";
import { siteUrl } from "@/lib/site-url";

export function generateStaticParams() {
  return locales.map((locale) => ({ locale }));
}

export const viewport: Viewport = {
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#f6f1e8" },
    { media: "(prefers-color-scheme: dark)", color: "#151311" },
  ],
};

export async function generateMetadata({ params }: LayoutProps<"/[locale]">): Promise<Metadata> {
  const { locale } = await params;
  if (!isLocale(locale) || (await setupScreen())) return { title: "GoHead conference starter" };
  const site = await getSite(locale);
  const shareImage = mediaUrl(site.seo?.share_image);
  return {
    metadataBase: siteUrl(),
    title: { default: site.seo?.meta_title || site.name, template: `%s · ${site.name}` },
    description: site.seo?.meta_description || site.description || undefined,
    openGraph: {
      siteName: site.name,
      locale,
      type: "website",
      images: shareImage ? [{ url: shareImage, alt: populated(site.seo?.share_image)?.alt || site.name }] : undefined,
    },
    twitter: { card: shareImage ? "summary_large_image" : "summary" },
    alternates: {
      types: { "application/rss+xml": [{ url: `/${locale}/feed.xml`, title: site.name }] },
    },
  };
}

export default async function LocaleLayout({ children, params }: LayoutProps<"/[locale]">) {
  const { locale } = await params;
  if (!isLocale(locale)) notFound();

  const fonts = `${grotesk.variable} ${literata.variable}`;
  const issue = await setupScreen();
  if (issue) {
    return (
      <html lang="en" className={fonts}>
        <body>
          <SetupScreen issue={issue} />
        </body>
      </html>
    );
  }

  const t = getDictionary(locale);
  const site = await getSite(locale);
  return (
    <html lang={locale} className={fonts}>
      <body className="flex min-h-dvh flex-col">
        <a
          href="#main"
          className="sr-only z-50 rounded-md bg-signal px-4 py-2 font-semibold text-on-signal focus:not-sr-only focus:fixed focus:top-3 focus:left-3"
        >
          {t.skipToContent}
        </a>
        <PreviewBanner locale={locale} />
        <SiteHeader site={site} locale={locale} />
        <div className="flex-1">{children}</div>
        <SiteFooter site={site} locale={locale} />
      </body>
    </html>
  );
}
