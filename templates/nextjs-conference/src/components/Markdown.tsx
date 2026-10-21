import Link from "next/link";
import ReactMarkdown, { type Components } from "react-markdown";
import remarkGfm from "remark-gfm";

import type { Locale } from "@/i18n";
import { mediaUrl } from "@/lib/gohead";
import { isExternal, localHref } from "@/lib/paths";

function components(locale?: Locale): Components {
  return {
    // Images uploaded to GoHead may be written as /uploads/… paths.
    img: ({ src, alt, title }) => {
      const url = typeof src === "string" ? mediaUrl(src) : undefined;
      if (!url) return null;
      // Markdown images have no known size, so next/image cannot lay them out.
      // eslint-disable-next-line @next/next/no-img-element
      return <img src={url} alt={alt ?? ""} title={title} loading="lazy" decoding="async" />;
    },
    a: ({ href, children, title }) => {
      if (typeof href !== "string") return <>{children}</>;
      if (isExternal(href)) {
        return (
          <a href={href} title={title} rel="noopener noreferrer">
            {children}
          </a>
        );
      }
      // Site paths (`/schedule`) get the page's locale.
      return (
        <Link href={locale ? localHref(href, locale) : href} title={title}>
          {children}
        </Link>
      );
    },
  };
}

/**
 * Rich text from GoHead (Markdown, with GitHub tables, task lists and
 * strikethrough). Raw HTML in the source is dropped, never rendered.
 */
export function Markdown({ children, locale }: { children: string | null | undefined; locale?: Locale }) {
  if (!children) return null;
  return (
    <div className="prose-conf">
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={components(locale)} skipHtml>
        {children}
      </ReactMarkdown>
    </div>
  );
}
