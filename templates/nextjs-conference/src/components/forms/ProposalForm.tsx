"use client";

import { useActionState } from "react";

import { submitProposal } from "@/app/actions";
import { Field, FormStatus, Honeypot, initialState } from "@/components/forms/Fields";
import { getDictionary, type Locale } from "@/i18n";

/** The call for papers: a proposal, and an optional PDF uploaded to GoHead's media library. */
export function ProposalForm({ locale }: { locale: Locale }) {
  const t = getDictionary(locale);
  const [state, action, pending] = useActionState(submitProposal, initialState);
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
      <Field label={t.proposalTitle} name="title" state={state}>
        {(p) => <input {...p} className="input" required minLength={5} maxLength={120} />}
      </Field>
      <Field label={t.abstract} name="abstract" state={state} hint={t.abstractHint}>
        {(p) => <textarea {...p} className="input min-h-40" required minLength={50} maxLength={1500} />}
      </Field>
      <div className="grid gap-6 sm:grid-cols-3">
        <Field label={t.formatLabel} name="format" state={state}>
          {(p) => (
            <select {...p} className="input" required defaultValue={p.defaultValue || "talk"}>
              <option value="talk">{t.format.talk}</option>
              <option value="workshop">{t.format.workshop}</option>
              <option value="lightning">{t.format.lightning}</option>
            </select>
          )}
        </Field>
        <Field label={t.levelLabel} name="level" state={state}>
          {(p) => (
            <select {...p} className="input" defaultValue={p.defaultValue || "intermediate"}>
              <option value="beginner">{t.level.beginner}</option>
              <option value="intermediate">{t.level.intermediate}</option>
              <option value="advanced">{t.level.advanced}</option>
            </select>
          )}
        </Field>
        <Field label={t.languageLabel} name="language" state={state}>
          {(p) => (
            <select {...p} className="input" defaultValue={p.defaultValue || locale}>
              <option value="en">English</option>
              <option value="fr">Français</option>
            </select>
          )}
        </Field>
      </div>
      <div className="grid gap-6 sm:grid-cols-2">
        <Field label={t.speakerName} name="speakerName" state={state}>
          {(p) => <input {...p} className="input" required maxLength={80} autoComplete="name" />}
        </Field>
        <Field label={t.email} name="speakerEmail" state={state} hint={t.emailHint}>
          {(p) => <input {...p} type="email" className="input" required autoComplete="email" />}
        </Field>
      </div>
      <Field label={t.bio} name="speakerBio" state={state}>
        {(p) => <textarea {...p} className="input min-h-24" maxLength={600} />}
      </Field>
      <Field label={t.outline} name="attachment" state={state}>
        {(p) => (
          // A file input can't be refilled after an error: no defaultValue.
          <input id={p.id} name={p.name} aria-invalid={p["aria-invalid"]} aria-describedby={p["aria-describedby"]} type="file" accept="application/pdf" className="input file:mr-4 file:rounded-full file:border-0 file:bg-ink file:px-4 file:py-1.5 file:font-semibold file:text-paper" />
        )}
      </Field>
      <div>
        <button type="submit" className="button" disabled={pending}>
          {pending ? t.sending : t.submit}
        </button>
      </div>
      {state.message && <FormStatus state={state} />}
    </form>
  );
}
