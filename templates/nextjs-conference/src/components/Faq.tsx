import type { ReactNode } from "react";

import type { FaqItem } from "@/gohead";

/** A repeatable `faq_item` component, as disclosure widgets. */
export function Faq({ items }: { items: FaqItem[] | null | undefined }) {
  if (!items?.length) return null;
  return (
    <div className="border-b border-rule">
      {items.map((item, i) => (
        <details key={i} className="group border-t border-rule py-4 [&_summary::-webkit-details-marker]:hidden">
          <summary className="flex cursor-pointer list-none items-start justify-between gap-6 text-lg font-bold tracking-[-0.01em]">
            {item.question}
            <span aria-hidden className="mt-1 text-signal transition-transform group-open:rotate-45">
              +
            </span>
          </summary>
          <p className="mt-3 max-w-[62ch] font-serif text-lg leading-relaxed text-muted">{item.answer}</p>
        </details>
      ))}
    </div>
  );
}

/** A section title with an optional link on the right. */
export function SectionTitle({ id, children, action }: { id: string; children: ReactNode; action?: ReactNode }) {
  return (
    <div className="mb-6 flex flex-wrap items-baseline justify-between gap-4">
      <h2 id={id} className="text-3xl font-extrabold tracking-[-0.04em] sm:text-4xl">
        {children}
      </h2>
      {action}
    </div>
  );
}
