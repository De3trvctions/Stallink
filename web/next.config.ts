import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  /* config options here */
  async headers() {
    return [
      {
        source: "/:path*",
        headers: [
          { key: "X-Content-Type-Options", value: "nosniff" },
          { key: "X-Frame-Options", value: "DENY" },
          { key: "Referrer-Policy", value: "no-referrer" },
        ],
      },
    ];
  },
  cacheComponents: true,
  partialPrefetching: true,
  // Next 16 blocks dev runtime resources from origins it does not trust.
  // Without this, opening the app via 127.0.0.1 or the LAN address leaves the
  // client un-hydrated: every useEffect-driven control (login link, theme
  // toggle) silently never renders while the SSR shell still looks fine.
  allowedDevOrigins: ["127.0.0.1", "localhost", "192.168.132.138"],
  turbopack: {
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
};

export default nextConfig;
