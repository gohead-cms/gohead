import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { FeedbackForm } from "@/components/forms/FeedbackForm";
import { getDictionary, isLocale } from "@/i18n";
import { feedbackConfigured } from "@/lib/forms";
import { setupScreen } from "@/lib/gohead";

export async function generateMetadata({ params }: PageProps<"/[locale]/feedback">): Promise<Metadata> {
  const { locale } = await params;
  return isLocale(locale) ? { title: getDictionary(locale).feedback } : {};
}

/** Feedback goes to an agent's inbound webhook (see sendFeedback in src/lib/forms.ts). */
export default async function FeedbackPage({ params }: PageProps<"/[locale]/feedback">) {
  const { locale } = await params;
  if (!isLocale(locale)) notFound();
  if (await setupScreen()) return null;
  const t = getDictionary(locale);
  return (
    <main id="main" className="wrap pt-12 sm:pt-16">
      <div className="grid gap-12 lg:grid-cols-12">
        <div className="lg:col-span-4">
          <h1 className="display">{t.feedback}</h1>
          <p className="mt-6 font-serif text-xl leading-relaxed text-muted">{t.feedbackIntro}</p>
        </div>
        <div className="lg:col-span-7 lg:col-start-6">
          {feedbackConfigured ? <FeedbackForm locale={locale} /> : <p className="text-lg text-muted">{t.formUnavailable}</p>}
        </div>
      </div>
    </main>
  );
}
