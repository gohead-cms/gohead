"use server";

/**
 * Server Actions of the site's forms. They validate what visitors send,
 * then write to GoHead with the forms token (src/lib/forms.ts). Forms work
 * without JavaScript too: they post to these actions like plain HTML forms.
 */
import { isGoHeadError } from "@gohead-cms/client";
import { draftMode } from "next/headers";
import { redirect } from "next/navigation";

import { defaultLocale, getDictionary, isLocale, type Dictionary, type Locale } from "@/i18n";
import { createProposal, createQuestion, createSubscriber, FormError, sendFeedback } from "@/lib/forms";
import { getTalk } from "@/lib/gohead";

export interface FormState {
  ok: boolean;
  message?: string;
  /** Errors by field name. */
  errors?: Record<string, string>;
  /** What the visitor typed, to refill the form after an error. */
  values?: Record<string, string>;
}

const EMAIL = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

function read(form: FormData) {
  const raw = form.get("locale");
  const locale: Locale = typeof raw === "string" && isLocale(raw) ? raw : defaultLocale;
  const t = getDictionary(locale);
  const text = (name: string) => {
    const v = form.get(name);
    return typeof v === "string" ? v.trim() : "";
  };
  // A field hidden from people: bots fill it in, and get a fake success.
  const isBot = text("website") !== "";
  return { locale, t, text, isBot };
}

function check(errors: Record<string, string>, t: Dictionary, name: string, value: string, min: number, max: number) {
  if (value.length < min) errors[name] = min <= 1 ? t.required : t.tooShort(min);
  else if (value.length > max) errors[name] = t.tooLong(max);
}

/** The message to show when GoHead refused a write; details go to the server log. */
function failure(err: unknown, t: Dictionary, values: Record<string, string>): FormState {
  if (err instanceof FormError) {
    console.error(`[forms] ${err.message}`);
    return { ok: false, message: t.formUnavailable, values };
  }
  console.error("[forms]", isGoHeadError(err) ? `${err.status} ${err.code}: ${err.message}` : err);
  return { ok: false, message: t.formError, values };
}

export async function askQuestion(_prev: FormState, form: FormData): Promise<FormState> {
  const { locale, t, text, isBot } = read(form);
  const values = { name: text("name"), body: text("body") };
  if (isBot) return { ok: true, message: t.questionSent };
  const errors: Record<string, string> = {};
  check(errors, t, "name", values.name, 1, 60);
  check(errors, t, "body", values.body, 10, 500);
  if (Object.keys(errors).length) return { ok: false, errors, values };

  // The talk is looked up again: only a published talk takes questions.
  const talk = await getTalk(text("talk"), locale);
  if (!talk || talk.status === "draft") return { ok: false, message: t.formError, values };
  try {
    await createQuestion({ talkId: talk.id, talkSlug: talk.slug, name: values.name, body: values.body });
    return { ok: true, message: t.questionSent };
  } catch (err) {
    return failure(err, t, values);
  }
}

/**
 * Whether a file really is a PDF. GoHead detects an upload's type from its
 * content too, and `proposals.attachment` only accepts application/pdf: a
 * text file renamed .pdf would be uploaded as text/plain, then refused with
 * the proposal. Checking first keeps it out of the media library.
 */
async function isPdf(file: File): Promise<boolean> {
  const head = new Uint8Array(await file.slice(0, 5).arrayBuffer());
  return String.fromCharCode(...head) === "%PDF-";
}

const FORMATS = ["talk", "workshop", "lightning"] as const;
const LEVELS = ["beginner", "intermediate", "advanced"] as const;

export async function submitProposal(_prev: FormState, form: FormData): Promise<FormState> {
  const { locale, t, text, isBot } = read(form);
  const values = {
    title: text("title"),
    abstract: text("abstract"),
    format: text("format"),
    level: text("level"),
    language: text("language"),
    speakerName: text("speakerName"),
    speakerEmail: text("speakerEmail"),
    speakerBio: text("speakerBio"),
  };
  if (isBot) return { ok: true, message: t.proposalSent };
  const errors: Record<string, string> = {};
  check(errors, t, "title", values.title, 5, 120);
  check(errors, t, "abstract", values.abstract, 50, 1500);
  check(errors, t, "speakerName", values.speakerName, 1, 80);
  check(errors, t, "speakerBio", values.speakerBio, 0, 600);
  if (!EMAIL.test(values.speakerEmail)) errors.speakerEmail = t.invalidEmail;
  const format = FORMATS.find((f) => f === values.format);
  if (!format) errors.format = t.required;
  const level = LEVELS.find((l) => l === values.level);
  const file = form.get("attachment");
  const attachment = file instanceof File && file.size > 0 ? file : null;
  if (attachment && (attachment.size > 5 * 1024 * 1024 || !(await isPdf(attachment)))) errors.attachment = t.pdfOnly;
  if (Object.keys(errors).length || !format) return { ok: false, errors, values };

  try {
    await createProposal({
      ...values,
      format,
      level,
      language: values.language === "fr" ? "fr" : locale === "fr" ? "fr" : "en",
      attachment,
    });
    return { ok: true, message: t.proposalSent };
  } catch (err) {
    // GoHead checks the upload's real type: a renamed file is refused (415).
    if (isGoHeadError(err) && (err.status === 413 || err.status === 415)) {
      return { ok: false, errors: { attachment: t.pdfOnly }, values };
    }
    return failure(err, t, values);
  }
}

export async function subscribe(_prev: FormState, form: FormData): Promise<FormState> {
  const { locale, t, text, isBot } = read(form);
  const values = { email: text("email") };
  if (isBot) return { ok: true, message: t.subscribed };
  if (!EMAIL.test(values.email) || values.email.length > 200) return { ok: false, errors: { email: t.invalidEmail }, values };
  try {
    await createSubscriber(values.email.toLowerCase(), locale);
    return { ok: true, message: t.subscribed };
  } catch (err) {
    return failure(err, t, values);
  }
}

export async function giveFeedback(_prev: FormState, form: FormData): Promise<FormState> {
  const { t, text, isBot } = read(form);
  const values = { message: text("message"), email: text("email") };
  if (isBot) return { ok: true, message: t.feedbackSent };
  const errors: Record<string, string> = {};
  check(errors, t, "message", values.message, 5, 2000);
  if (values.email && !EMAIL.test(values.email)) errors.email = t.invalidEmail;
  if (Object.keys(errors).length) return { ok: false, errors, values };
  try {
    await sendFeedback(values.message, values.email || null);
    return { ok: true, message: t.feedbackSent };
  } catch (err) {
    return failure(err, t, values);
  }
}

/** Leaves draft mode (a POST, from the preview banner). */
export async function exitPreview(form: FormData) {
  (await draftMode()).disable();
  const to = form.get("path");
  redirect(typeof to === "string" && to.startsWith("/") && !to.startsWith("//") ? to : "/");
}
