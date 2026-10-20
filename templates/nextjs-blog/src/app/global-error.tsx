"use client";

import "./globals.css";

/**
 * Shown when a whole page fails, e.g. GoHead is down and this page was never
 * rendered before (pages rendered earlier keep being served from the cache).
 * The server log has the reason. It has no locale, so it speaks both.
 */
export default function GlobalError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return (
    <html lang="en">
      <body className="font-sans">
        <main className="wrap py-20 sm:py-28">
          <h1 className="max-w-[18ch] text-5xl leading-[0.95] font-extrabold tracking-[-0.045em] sm:text-6xl">
            This site will be back in a moment
          </h1>
          <p className="mt-6 max-w-[52ch] text-xl leading-relaxed text-muted">
            Its content could not be loaded. Try again in a minute.
          </p>
          <p lang="fr" className="mt-2 max-w-[52ch] text-xl leading-relaxed text-muted">
            Le contenu n’a pas pu être chargé. Réessayez dans une minute.
          </p>
          {error.digest && <p className="mt-4 text-sm text-muted">Digest: {error.digest}</p>}
          <button
            type="button"
            onClick={reset}
            className="mt-8 rounded-full bg-ink px-6 py-3 font-semibold text-paper hover:opacity-90"
          >
            Try again
          </button>
        </main>
      </body>
    </html>
  );
}
