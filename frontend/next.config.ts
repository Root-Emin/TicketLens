import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  /*
    Standalone output traces the server's actual module graph into
    .next/standalone, so the runtime image ships that instead of the full
    node_modules tree. Required by frontend/Dockerfile — without it there is
    nothing to copy into the runtime stage.
  */
  output: "standalone",
};

export default nextConfig;
