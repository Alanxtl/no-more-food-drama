import type { NextConfig } from "next";
import { dirname } from "node:path";
import { fileURLToPath } from "node:url";

const projectRoot = dirname(fileURLToPath(import.meta.url));

const nextConfig: NextConfig = {
  reactStrictMode: true,
  turbopack: {
    root: projectRoot
  },
  async rewrites() {
    const apiURL = process.env.E2E_API_URL;
    if (!apiURL) {
      return [];
    }

    return [
      {
        source: "/api/rooms",
        destination: `${apiURL}/api/rooms`
      },
      {
        source: "/api/rooms/:path*",
        destination: `${apiURL}/api/rooms/:path*`
      }
    ];
  }
};

export default nextConfig;
