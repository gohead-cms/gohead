import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { ProposalForm } from "@/components/forms/ProposalForm";
import { getDictionary, isLocale } from "@/i18n";
import { formsConfigured } from "@/lib/forms";
import { setupScreen } from "@/lib/gohead";

export async function generateMetadata({ params }: PageProps<"/[locale]/cfp">): Promise<Metadata> {
  const { locale } = await params;
  return isLocale(locale) ? { title: getDictionary(locale).cfp } : {};
}

/**
 * The call for papers. A proposal lands in the private `proposals`
 * collection; the `proposal-triage` agent scores it for the committee.
 */
export default async function CfpPage({ params }: PageProps<"/[locale]/cfp">) {
  const { locale } = await params;
  if (!isLocale(locale)) notFound();
  if (await setupScreen()) return null;
  const t = getDictionary(locale);
  return (
    <main id="main" className="wrap pt-12 sm:pt-16">
      <div className="grid gap-12 lg:grid-cols-12">
        <div className="lg:col-span-4">
          <h1 className="display">{t.cfp}</h1>
          <p className="mt-6 font-serif text-xl leading-relaxed text-muted">{t.cfpIntro}</p>
        </div>
        <div className="lg:col-span-7 lg:col-start-6">
          {formsConfigured ? <ProposalForm locale={locale} /> : <p className="text-lg text-muted">{t.formUnavailable}</p>}
        </div>
      </div>
    </main>
  );
}
