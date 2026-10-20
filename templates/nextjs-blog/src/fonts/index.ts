import localFont from "next/font/local";

/**
 * Fonts are in the repository (SIL Open Font License, see OFL.txt) so builds
 * need no network. Schibsted Grotesk sets headlines and interface text,
 * Literata the reading text. Swap them here.
 */
export const grotesk = localFont({
  src: [{ path: "./schibsted-grotesk-latin-wght-normal.woff2", weight: "400 900", style: "normal" }],
  variable: "--font-grotesk",
  display: "swap",
  fallback: ["ui-sans-serif", "system-ui", "sans-serif"],
});

export const literata = localFont({
  src: [
    { path: "./literata-latin-wght-normal.woff2", weight: "200 900", style: "normal" },
    { path: "./literata-latin-wght-italic.woff2", weight: "200 900", style: "italic" },
  ],
  variable: "--font-literata",
  display: "swap",
  fallback: ["ui-serif", "Georgia", "serif"],
});
