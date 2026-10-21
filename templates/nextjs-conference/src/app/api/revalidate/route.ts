import { parseWebhook } from "@gohead-cms/client";
import { revalidateTag } from "next/cache";

import { CACHE_TAG, cacheTag } from "@/lib/gohead";

/**
 * GoHead webhook → fresh pages. `npm run gohead:access` creates the webhook
 * ("website revalidation") and writes its signing secret to
 * GOHEAD_WEBHOOK_SECRET.
 *
 * An event about a collection expires the reads tagged `gohead:<collection>`
 * (singletons: `gohead:singleton:<name>`), so pages showing those entries
 * render fresh on their next visit; a test delivery expires every GoHead read.
 */
export async function POST(request: Request) {
  const secret = process.env.GOHEAD_WEBHOOK_SECRET;
  if (!secret) {
    return Response.json({ error: "GOHEAD_WEBHOOK_SECRET is not set" }, { status: 500 });
  }

  // Checks the HMAC signature and refuses deliveries older than five minutes.
  const event = await parseWebhook(request, secret);
  if (!event) {
    return Response.json({ error: "invalid or expired signature" }, { status: 401 });
  }

  const tags = event.event === "webhook:test" || !event.collection ? [CACHE_TAG] : [cacheTag(event.collection)];
  // A webhook is not a Server Action, so updateTag is not available: expire
  // the data now so the next visit renders fresh content instead of stale.
  for (const tag of tags) revalidateTag(tag, { expire: 0 });

  return Response.json({ revalidated: tags, event: event.event, id: event.id });
}
