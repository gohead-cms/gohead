import ReactMarkdown, { type Components } from "react-markdown";
import remarkGfm from "remark-gfm";

import { mediaUrl } from "@/lib/gohead";

const components: Components = {
  // Images uploaded to GoHead may be written as /uploads/… paths.
  img: ({ src, alt, title }) => {
    const url = typeof src === "string" ? mediaUrl(src) : undefined;
    if (!url) return null;
    // Markdown images have no known size, so next/image cannot lay them out.
    // eslint-disable-next-line @next/next/no-img-element
    return <img src={url} alt={alt ?? ""} title={title} loading="lazy" decoding="async" />;
  },
  a: ({ href, children, title }) => {
    const external = typeof href === "string" && /^https?:\/\//.test(href);
    return (
      <a href={href} title={title} {...(external ? { rel: "noopener noreferrer" } : {})}>
        {children}
      </a>
    );
  },
};

/**
 * Rich text from GoHead (Markdown, with GitHub tables, task lists and
 * strikethrough). Raw HTML in the source is dropped, never rendered.
 */
export function Markdown({ children }: { children: string | null | undefined }) {
  if (!children) return null;
  return (
    <div className="article-prose">
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={components} skipHtml>
        {children}
      </ReactMarkdown>
    </div>
  );
}
