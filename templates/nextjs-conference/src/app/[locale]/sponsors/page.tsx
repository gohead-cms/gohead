import type { Metadata } from "next";
import Image from "next/image";
import { notFound } from "next/navigation";

import { getDictionary, isLocale } from "@/i18n";
import { getSponsors, mediaUrl, setupScreen, type SponsorNode } from "@/lib/gohead";

export async function generateMetadata({ params }: PageProps<"/[locale]/sponsors">): Promise<Metadata> {
  const { locale } = await params;
  return isLocale(locale) ? { title: getDictionary(locale).sponsors } : {};
}

/** Sponsors by tier, read with one GraphQL query (getSponsors in src/lib/gohead.ts). */
export default async function SponsorsPage({ params }: PageProps<"/[locale]/sponsors">) {
  const { locale } = await params;
  if (!isLocale(locale)) notFound();
  if (await setupScreen()) return null;
  const t = getDictionary(locale);
  const sponsors = await getSponsors(locale);
  const tiers = (["platinum", "gold", "silver", "community"] as const)
    .map((tier) => ({ tier, list: sponsors.filter((s) => s.tier === tier) }))
    .filter((g) => g.list.length > 0);
  const size: Record<SponsorNode["tier"], string> = {
    platinum: "",
    gold: "md:grid-cols-2",
    silver: "sm:grid-cols-2 lg:grid-cols-3",
    community: "sm:grid-cols-2 lg:grid-cols-4",
  };
  return (
    <main id="main" className="wrap pt-12 sm:pt-16">
      <h1 className="display">{t.sponsors}</h1>
      <p className="mt-4 max-w-[60ch] text-lg text-muted">{t.sponsorsIntro}</p>
      {tiers.map(({ tier, list }) => (
        <section key={tier} aria-labelledby={`tier-${tier}`} className="mt-14">
          <h2 id={`tier-${tier}`} className="eyebrow text-signal">
            {t.tier[tier]}
          </h2>
          <ul className={`mt-4 grid gap-4 ${size[tier]}`}>
            {list.map((s) => {
              // Bigger tiers get the logo beside the text; smaller ones, above it.
              const wide = tier === "platinum" || tier === "gold";
              const logo = mediaUrl(s.logo);
              return (
                <li key={s.id} className={`grid gap-5 rounded-lg bg-paper-raised p-6 shadow-[inset_0_0_0_1px_var(--rule)] ${wide ? "sm:grid-cols-[minmax(0,14rem)_1fr] sm:items-center" : ""}`}>
                  <div className="grid aspect-[3/1] place-items-center rounded-md bg-white p-3">
                    {logo ? (
                      <Image src={logo} alt={s.logo?.alt || s.name} width={s.logo?.width ?? 360} height={s.logo?.height ?? 120} className="h-auto max-h-full w-full object-contain" />
                    ) : (
                      <span className="font-bold text-[#181614]">{s.name}</span>
                    )}
                  </div>
                  <div>
                    <h3 className="text-xl font-bold">{s.name}</h3>
                    {s.blurb && <p className="mt-1 text-muted">{s.blurb}</p>}
                    {s.website && (
                      <a href={s.website} className="link mt-2 inline-block" rel="noopener noreferrer">
                        {t.visit}
                      </a>
                    )}
                  </div>
                </li>
              );
            })}
          </ul>
        </section>
      ))}
    </main>
  );
}
