import Image from "next/image";
import type { Media } from "@gohead-cms/client";

import { mediaUrl } from "@/lib/gohead";

/**
 * A GoHead image cropped to a frame. `ratio` is the frame's CSS aspect ratio;
 * `sizes` tells the browser how wide it is drawn so it picks a small file.
 */
export function Cover({
  media,
  alt,
  ratio = "16 / 9",
  sizes,
  preload = false,
  className = "",
}: {
  media: Media | null | undefined;
  alt?: string;
  ratio?: string;
  sizes: string;
  preload?: boolean;
  className?: string;
}) {
  const src = mediaUrl(media);
  if (!src) return null;
  return (
    <div className={`relative overflow-hidden rounded-[3px] bg-rule ${className}`} style={{ aspectRatio: ratio }}>
      <Image
        src={src}
        alt={alt ?? media?.alt ?? ""}
        fill
        sizes={sizes}
        preload={preload}
        className="object-cover"
      />
    </div>
  );
}

/** A round avatar. */
export function Avatar({ media, name, size = 40 }: { media: Media | null | undefined; name: string; size?: number }) {
  const src = mediaUrl(media);
  if (!src) {
    return (
      <span
        aria-hidden
        className="inline-grid shrink-0 place-items-center rounded-full bg-ink font-bold text-paper"
        style={{ width: size, height: size, fontSize: size * 0.42 }}
      >
        {name.slice(0, 1)}
      </span>
    );
  }
  return (
    <Image
      src={src}
      alt=""
      width={size}
      height={size}
      className="shrink-0 rounded-full object-cover"
      style={{ width: size, height: size }}
    />
  );
}
