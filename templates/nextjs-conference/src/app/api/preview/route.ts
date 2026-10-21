import { draftMode } from "next/headers";
import { redirect } from "next/navigation";

import { gohead, previewConfigured } from "@/lib/gohead";

/**
 * Draft mode: /api/preview?secret=<GOHEAD_PREVIEW_SECRET>&type=talks&slug=surprise-act
 *
 * Checks the shared secret, checks that the entry exists (drafts included,
 * with the preview token), turns draft mode on and redirects to its page.
 * Pages then read with GOHEAD_PREVIEW_TOKEN and `status: "all"`, and skip
 * Next.js caches, until "Exit draft mode" in the banner.
 *
 * GoHead has no preview button of its own: bookmark this URL, or put it in
 * your editors' notes for each type.
 */
const PAGES = { talks: "talks", posts: "news" } as const;

export async function GET(request: Request) {
  const params = new URL(request.url).searchParams;
  const secret = process.env.GOHEAD_PREVIEW_SECRET;
  if (!previewConfigured || !secret) {
    return new Response("Draft mode is not set up: run `npm run gohead:access`, then restart the site.", { status: 501 });
  }
  if (params.get("secret") !== secret) return new Response("Invalid secret", { status: 401 });

  const type = params.get("type") === "posts" ? "posts" : "talks";
  const slug = params.get("slug") ?? "";
  const locale = params.get("locale") === "fr" ? "fr" : "en";
  const entry = await gohead
    .with({ token: process.env.GOHEAD_PREVIEW_TOKEN, init: { cache: "no-store" } })
    .collection(type)
    .findFirst({ filters: { slug }, status: "all", level: 0 });
  if (!entry) return new Response(`No ${type} entry with slug "${slug}"`, { status: 404 });

  (await draftMode()).enable();
  // Redirect to the path built from the entry, not from the query: no open redirect.
  redirect(`/${locale}/${PAGES[type]}/${entry.slug}`);
}
