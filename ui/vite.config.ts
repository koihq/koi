/// <reference types="vitest/config" />
import tailwindcss from "@tailwindcss/vite";

import { tanstackRouter } from "@tanstack/router-plugin/vite";

import viteReact from "@vitejs/plugin-react";
import { playwright } from "@vitest/browser-playwright";
import { msw } from "msw/vite";
import { defineConfig } from "vite";

const config = defineConfig({
  envDir: "..",
  resolve: { tsconfigPaths: true },
  test: {
    projects: [
      {
        extends: true,
        test: {
          name: "unit",
          environment: "node",
          include: ["src/**/*.test.ts"],
        },
      },
      {
        extends: true,
        plugins: [msw({ mode: "worker-only" })],
        test: {
          name: "browser",
          include: ["src/**/*.test.tsx"],
          // Route tests wait on real polling intervals (2-5 s) and the full onboarding flow.
          testTimeout: 30_000,
          setupFiles: ["src/test/setup-browser.ts"],
          browser: {
            enabled: true,
            headless: true,
            provider: playwright({
              contextOptions: { permissions: ["clipboard-read", "clipboard-write"] },
            }),
            instances: [{ browser: "chromium" }],
            commands: {
              emulateMedia: async (
                context,
                options: { reducedMotion: "reduce" | "no-preference" },
              ) => {
                await context.page.emulateMedia(options);
              },
            },
          },
        },
      },
    ],
  },
  optimizeDeps: {
    include: [
      "class-variance-authority",
      "cn",
      "@base-ui/react/button",
      "@base-ui/react/input",
      "@base-ui/react/separator",
    ],
  },
  server: {
    port: 5173,
    strictPort: true,
    // Tests answer every request with MSW; one that escapes must fail, not reach a dev API.
    proxy: process.env.VITEST
      ? undefined
      : {
          "/health": { target: "http://localhost:8080", changeOrigin: true },
          "/api": { target: "http://localhost:8080", changeOrigin: true },
          "/ingest": { target: "http://localhost:8080", changeOrigin: true },
        },
  },
  build: {
    outDir: "../pkg/web/dist",
    emptyOutDir: true,
  },
  plugins: [
    tailwindcss(),
    tanstackRouter({ target: "react", autoCodeSplitting: true }),
    viteReact(),
  ],
});

export default config;
