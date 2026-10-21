import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { Markdown } from "@/components/Markdown";
import { formatDate, formatPrice, getDictionary, isLocale } from "@/i18n";
import { getTickets, setupScreen, stringsOf } from "@/lib/gohead";

export async function generateMetadata({ params }: PageProps<"/[locale]/tickets">): Promise<Metadata> {
  const { locale } = await params;
  return isLocale(locale) ? { title: getDictionary(locale).tickets } : {};
}

/**
 * The `tickets` singleton: a repeatable `ticket_tier` component (float price,
 * enum currency, JSON perks, datetime sale end, bool sold out).
 */
export default async function TicketsPage({ params }: PageProps<"/[locale]/tickets">) {
  const { locale } = await params;
  if (!isLocale(locale)) notFound();
  if (await setupScreen()) return null;
  const t = getDictionary(locale);
  const tickets = await getTickets(locale);
  const tiers = tickets?.tiers ?? [];
  return (
    <main id="main" className="wrap pt-12 sm:pt-16">
      <h1 className="display">{t.tickets}</h1>
      {tickets?.intro && <p className="mt-4 max-w-[56ch] font-serif text-xl leading-relaxed text-muted">{tickets.intro}</p>}
      <ul className="mt-12 grid gap-4 md:grid-cols-3">
        {tiers.map((tier) => {
          const perks = stringsOf(tier.perks);
          return (
            <li key={tier.name} className={`flex flex-col rounded-lg p-6 ${tier.sold_out ? "bg-rule/40 text-muted" : "bg-paper-raised shadow-[inset_0_0_0_1px_var(--rule)]"}`}>
              <h2 className="text-xl font-bold">{tier.name}</h2>
              <p className="tabular mt-3 text-5xl font-extrabold tracking-[-0.05em]">
                <span className={tier.sold_out ? "line-through decoration-2" : undefined}>{formatPrice(tier.price, tier.currency, locale)}</span>
              </p>
              {tier.sale_ends && !tier.sold_out && <p className="mt-1 text-sm">{t.saleEnds(formatDate(tier.sale_ends, locale) ?? "")}</p>}
              <ul className="mt-6 grid gap-2">
                {perks.map((p) => (
                  <li key={p} className="flex gap-2">
                    <span aria-hidden className="text-signal">
                      ✓
                    </span>
                    {p}
                  </li>
                ))}
              </ul>
              <div className="mt-auto pt-8">
                {tier.sold_out ? (
                  <p className="eyebrow">{t.soldOut}</p>
                ) : tier.url ? (
                  <a href={tier.url} className="button w-full" rel="noopener noreferrer">
                    {t.buy}
                  </a>
                ) : null}
              </div>
            </li>
          );
        })}
      </ul>
      {tickets?.refund_policy && (
        <section aria-labelledby="refunds" className="mt-16 max-w-[68ch]">
          <h2 id="refunds" className="text-2xl font-extrabold tracking-[-0.03em]">
            {t.refunds}
          </h2>
          <div className="mt-3">
            <Markdown locale={locale}>{tickets.refund_policy}</Markdown>
          </div>
        </section>
      )}
    </main>
  );
}
