"use client";

import { useActionState } from "react";

import { giveFeedback } from "@/app/actions";
import { Field, FormStatus, Honeypot, initialState } from "@/components/forms/Fields";
import { getDictionary, type Locale } from "@/i18n";

/** Sends the message to the `feedback-desk` agent's webhook (it stores it, sorted). */
export function FeedbackForm({ locale }: { locale: Locale }) {
  const t = getDictionary(locale);
  const [state, action, pending] = useActionState(giveFeedback, initialState);
  if (state.ok) {
    return (
      <div className="rounded-lg bg-paper-raised p-6 shadow-[inset_0_0_0_1px_var(--rule)]">
        <FormStatus state={state} />
      </div>
    );
  }
  return (
    <form action={action} className="relative grid gap-6">
      <input type="hidden" name="locale" value={locale} />
      <Honeypot />
      <Field label={t.feedbackMessage} name="message" state={state}>
        {(p) => <textarea {...p} className="input min-h-40" required minLength={5} maxLength={2000} />}
      </Field>
      <Field label={t.feedbackEmail} name="email" state={state}>
        {(p) => <input {...p} type="email" className="input" autoComplete="email" />}
      </Field>
      <div>
        <button type="submit" className="button" disabled={pending}>
          {pending ? t.sending : t.send}
        </button>
      </div>
      {state.message && <FormStatus state={state} />}
    </form>
  );
}
