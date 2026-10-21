import type { NextConfig } from "next";

type RemotePattern = NonNullable<NonNullable<NextConfig["images"]>["remotePatterns"]>[number];

/**
 * Images come from GoHead: files of the local media store are served by the
 * API under /uploads/…, files of an S3 store from its public URL (set
 * GOHEAD_MEDIA_URL to it, e.g. https://cdn.example.com/media).
 */
const goheadUrl = process.env.GOHEAD_URL || "http://localhost:8080";
const mediaUrl = process.env.GOHEAD_MEDIA_URL;
const publicUrl = process.env.GOHEAD_PUBLIC_URL;

function pattern(base: string, path: string): RemotePattern {
  const url = new URL(base);
  const prefix = url.pathname.replace(/\/+$/, "");
  return {
    protocol: url.protocol === "https:" ? "https" : "http",
    hostname: url.hostname,
    port: url.port,
    pathname: `${prefix}${path}/**`,
  };
}

const remotePatterns: RemotePattern[] = [pattern(goheadUrl, "/uploads")];
if (mediaUrl) remotePatterns.push(pattern(mediaUrl, ""));
if (publicUrl) remotePatterns.push(pattern(publicUrl, "/uploads"));

/**
 * Next.js refuses to optimize images whose host resolves to a private or
 * loopback address (localhost, a Docker service name…). When GoHead itself
 * runs on such a host, allow it: only the hosts above can be fetched anyway.
 */
function isPrivateHost(hostname: string): boolean {
  const h = hostname.replace(/^\[|\]$/g, "");
  return (
    h === "localhost" ||
    h.endsWith(".localhost") ||
    h === "::1" ||
    !h.includes(".") || // single-label names: Docker services, hosts on a LAN
    /^127\./.test(h) ||
    /^10\./.test(h) ||
    /^192\.168\./.test(h) ||
    /^172\.(1[6-9]|2\d|3[01])\./.test(h)
  );
}

const nextConfig: NextConfig = {
  experimental: {
    // The call for papers uploads a PDF of up to 5 MB through a Server Action
    // (1 MB by default).
    serverActions: { bodySizeLimit: "6mb" },
  },
  images: {
    remotePatterns,
    dangerouslyAllowLocalIP: remotePatterns.some((p) => isPrivateHost(p.hostname)),
  },
};

export default nextConfig;
