import { draftMode } from "next/headers";

import { exitPreview } from "@/app/actions";
import { getDictionary, type Locale } from "@/i18n";

/** Shown on every page while draft mode is on (see /api/preview). */
export async function PreviewBanner({ locale }: { locale: Locale }) {
  const { isEnabled } = await draftMode();
  if (!isEnabled) return null;
  const t = getDictionary(locale);
  return (
    <aside role="status" className="sticky top-0 z-40 bg-signal text-on-signal">
      <div className="wrap flex flex-wrap items-center justify-between gap-3 py-2 text-sm font-semibold">
        <p>{t.previewOn}</p>
        <form action={exitPreview}>
          <input type="hidden" name="path" value={`/${locale}`} />
          <button type="submit" className="rounded-full bg-on-signal px-3 py-1 text-signal hover:opacity-90">
            {t.previewExit}
          </button>
        </form>
      </div>
    </aside>
  );
}
