"use client";

import { useActionState } from "react";

import { subscribe } from "@/app/actions";
import { FormStatus, Honeypot, initialState } from "@/components/forms/Fields";
import { getDictionary, type Locale } from "@/i18n";

/** Adds an email address to the `subscribers` collection (unique emails). */
export function NewsletterForm({ locale }: { locale: Locale }) {
  const t = getDictionary(locale);
  const [state, action, pending] = useActionState(subscribe, initialState);
  if (state.ok) return <FormStatus state={state} />;
  const error = state.errors?.email;
  return (
    <form action={action} className="relative grid gap-2">
      <input type="hidden" name="locale" value={locale} />
      <Honeypot />
      <label htmlFor="newsletter-email" className="sr-only">
        {t.email}
      </label>
      <div className="grid grid-cols-[minmax(0,1fr)_auto] gap-2">
        <input
          id="newsletter-email"
          name="email"
          type="email"
          required
          autoComplete="email"
          placeholder={t.email}
          defaultValue={state.values?.email}
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? "newsletter-error" : undefined}
          className="w-full rounded-full bg-white/10 px-4 py-2.5 text-on-footer placeholder:text-on-footer/60 focus:bg-white/15 focus:outline-none"
        />
        <button type="submit" disabled={pending} className="shrink-0 rounded-full bg-signal px-5 py-2.5 font-semibold text-on-signal hover:opacity-90 disabled:opacity-60">
          {t.subscribe}
        </button>
      </div>
      {error && (
        <p id="newsletter-error" className="text-sm font-medium text-[#ffb4a2]">
          {error}
        </p>
      )}
      {state.message && <p role="status" className="text-sm font-medium text-[#ffb4a2]">{state.message}</p>}
    </form>
  );
}
