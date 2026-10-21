"use client";

import { useId, type ReactNode } from "react";

import type { FormState } from "@/app/actions";

/** A labelled field with its hint and error, wired for screen readers. */
export function Field({
  label,
  name,
  state,
  hint,
  optional,
  children,
}: {
  label: string;
  name: string;
  state: FormState;
  hint?: string;
  optional?: string;
  children: (props: { id: string; name: string; "aria-invalid"?: boolean; "aria-describedby"?: string; defaultValue?: string }) => ReactNode;
}) {
  const id = useId();
  const error = state.errors?.[name];
  const described = [hint && `${id}-hint`, error && `${id}-error`].filter(Boolean).join(" ") || undefined;
  return (
    <div className="field">
      <label htmlFor={id}>
        {label}
        {optional && <span className="font-normal text-muted"> ({optional})</span>}
      </label>
      {children({ id, name, "aria-invalid": error ? true : undefined, "aria-describedby": described, defaultValue: state.values?.[name] })}
      {hint && (
        <p id={`${id}-hint`} className="hint">
          {hint}
        </p>
      )}
      {error && (
        <p id={`${id}-error`} className="error-text">
          {error}
        </p>
      )}
    </div>
  );
}

/** The form's outcome, announced to screen readers. */
export function FormStatus({ state }: { state: FormState }) {
  return (
    <p role="status" aria-live="polite" className={state.ok ? "font-semibold text-ok" : "font-semibold text-error"}>
      {state.message}
    </p>
  );
}

/** A field people don't see and bots fill in (see read() in actions.ts). */
export function Honeypot() {
  return (
    <div aria-hidden className="absolute -left-[9999px] h-px w-px overflow-hidden">
      <label>
        Website
        <input type="text" name="website" tabIndex={-1} autoComplete="off" />
      </label>
    </div>
  );
}

export const initialState: FormState = { ok: false };
