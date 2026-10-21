import Link from "next/link";

import type { Dictionary } from "@/i18n";

/** Newer / Older links for `?page=` lists. */
export function Pagination({ basePath, page, pageCount, t }: { basePath: string; page: number; pageCount: number; t: Dictionary }) {
  if (pageCount <= 1) return null;
  const href = (p: number) => (p <= 1 ? basePath : `${basePath}?page=${p}`);
  const button = "inline-block rounded-full border border-ink px-4 py-2 font-semibold whitespace-nowrap hover:bg-ink hover:text-paper sm:px-5 sm:py-2.5";
  return (
    <nav aria-label={t.pageOf(page, pageCount)} className="mt-10 flex items-center justify-between gap-4">
      <div className="min-w-0 flex-1">
        {page > 1 && (
          <Link href={href(page - 1)} rel="prev" className={button}>
            {t.newer}
          </Link>
        )}
      </div>
      <p className="text-center text-sm text-muted">{t.pageOf(page, pageCount)}</p>
      <div className="flex min-w-0 flex-1 justify-end">
        {page < pageCount && (
          <Link href={href(page + 1)} rel="next" className={button}>
            {t.older}
          </Link>
        )}
      </div>
    </nav>
  );
}
