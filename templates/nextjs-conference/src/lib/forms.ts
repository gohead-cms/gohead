/**
 * Writes made by the site's forms, on the server only.
 *
 * The public role of GoHead never writes. Visitors' questions, proposals and
 * sign-ups go through Server Actions that call GoHead with GOHEAD_FORMS_TOKEN,
 * an API token of the custom role `site-forms` (npm run gohead:access): it may
 * create questions, proposals, subscribers and files, and reach nothing else.
 * The token never leaves the server.
 *
 * Feedback goes elsewhere: to the `feedback-desk` agent's inbound webhook,
 * which sorts the message and stores it (npm run gohead:agents).
 */
import "server-only";

import { isGoHeadError } from "@gohead-cms/client";

import { gohead, GOHEAD_URL } from "@/lib/gohead";

const formsToken = process.env.GOHEAD_FORMS_TOKEN;
// Writes are POST and PUT requests, which Next.js never caches.
const forms = formsToken ? gohead.with({ token: formsToken }) : null;

export const formsConfigured = forms !== null;

export class FormError extends Error {}

function client() {
  if (!forms) throw new FormError("GOHEAD_FORMS_TOKEN is not set: run `npm run gohead:access` and restart the site.");
  return forms;
}

/** A question under a talk: a draft until a moderator publishes it. */
export async function createQuestion(input: { talkId: number; talkSlug: string; name: string; body: string }) {
  // `questions` has draft and publish: entries are created as drafts, which
  // visitors (the public role) never see. `question-moderator` flags spam;
  // an editor or the program committee publishes the rest.
  await client().collection("questions").create(
    { talk: input.talkId, talk_slug: input.talkSlug, name: input.name, body: input.body },
    { level: 0 },
  );
}

/** A talk proposal, with an optional PDF outline uploaded to the media library first. */
export async function createProposal(input: {
  title: string;
  abstract: string;
  format: "talk" | "workshop" | "lightning";
  level?: "beginner" | "intermediate" | "advanced";
  language: "en" | "fr";
  speakerName: string;
  speakerEmail: string;
  speakerBio?: string;
  attachment?: File | null;
}) {
  const gohead = client();
  let attachment: number | undefined;
  if (input.attachment && input.attachment.size > 0) {
    // GoHead checks the file's real type (from its content), its size, and the
    // attribute's allowed_types (application/pdf) when the proposal is saved.
    const [file] = await gohead.media.upload(input.attachment, { alt: `Outline of “${input.title}”` });
    attachment = file?.id;
  }
  await gohead.collection("proposals").create(
    {
      title: input.title,
      abstract: input.abstract,
      format: input.format,
      level: input.level,
      language: input.language,
      speaker_name: input.speakerName,
      speaker_email: input.speakerEmail,
      speaker_bio: input.speakerBio || null,
      attachment,
      decision: "pending",
    },
    { level: 0 },
  );
}

/** A newsletter sign-up. Signing up twice is not an error for the visitor. */
export async function createSubscriber(email: string, language: "en" | "fr") {
  try {
    await client().collection("subscribers").create({ email, language, source: "website" }, { level: 0 });
  } catch (err) {
    // `email` is unique: a second sign-up fails validation. Don't tell
    // visitors whether an address is already on the list.
    if (isGoHeadError(err) && err.isValidation && /unique|already/i.test(err.message)) return;
    throw err;
  }
}

export const feedbackConfigured = Boolean(process.env.GOHEAD_FEEDBACK_AGENT_ID && process.env.GOHEAD_FEEDBACK_AGENT_TOKEN);

/**
 * Starts the `feedback-desk` agent: POST /agents/webhook/<id> with its token.
 * GoHead answers 202 at once; the agent runs in the worker, classifies the
 * message with LLM helper functions and stores it in `feedback`.
 */
export async function sendFeedback(message: string, email: string | null) {
  if (!feedbackConfigured) throw new FormError("The feedback agent is not set up: run `npm run gohead:agents` and restart the site.");
  const res = await fetch(`${GOHEAD_URL}/agents/webhook/${process.env.GOHEAD_FEEDBACK_AGENT_ID}`, {
    method: "POST",
    headers: { "Content-Type": "application/json", "Webhook-Token": process.env.GOHEAD_FEEDBACK_AGENT_TOKEN! },
    body: JSON.stringify({ message, email }),
    cache: "no-store",
    signal: AbortSignal.timeout(10_000),
  });
  if (res.status !== 202) throw new Error(`the feedback agent answered ${res.status}`);
}
