"use client";

import { useActionState } from "react";

import { askQuestion } from "@/app/actions";
import { Field, FormStatus, Honeypot, initialState } from "@/components/forms/Fields";
import { getDictionary, type Locale } from "@/i18n";

/** Creates a draft question under a talk (published after moderation). */
export function QuestionForm({ talkSlug, locale }: { talkSlug: string; locale: Locale }) {
  const t = getDictionary(locale);
  const [state, action, pending] = useActionState(askQuestion, initialState);
  if (state.ok) return <FormStatus state={state} />;
  return (
    <form action={action} className="relative grid gap-4">
      <input type="hidden" name="locale" value={locale} />
      <input type="hidden" name="talk" value={talkSlug} />
      <Honeypot />
      <Field label={t.yourName} name="name" state={state}>
        {(p) => <input {...p} className="input" required maxLength={60} autoComplete="given-name" />}
      </Field>
      <Field label={t.yourQuestion} name="body" state={state}>
        {(p) => <textarea {...p} className="input min-h-28" required minLength={10} maxLength={500} />}
      </Field>
      <div className="flex flex-wrap items-center gap-4">
        <button type="submit" className="button" disabled={pending}>
          {pending ? t.sending : t.send}
        </button>
        <p className="hint">{t.askNote}</p>
      </div>
      {state.message && <FormStatus state={state} />}
    </form>
  );
}
