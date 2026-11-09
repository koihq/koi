import { resolve } from "node:path";
import tailwindcss from "@tailwindcss/vite";
import viteReact from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// The public site at getkoi.co. It is a separate static build, so self-hosted binaries never ship
// marketing pages and the page reaches answer engines as plain HTML.
export default defineConfig({
  root: resolve(import.meta.dirname, "site"),
  resolve: {
    alias: [
      // The app's blocks render on the landing, with logos saved beside the page.
      {
        find: "@/components/blocks/brand-icon",
        replacement: resolve(import.meta.dirname, "site/brand-icon.tsx"),
      },
      { find: "@", replacement: resolve(import.meta.dirname, "src") },
    ],
  },
  server: {
    port: 5174,
    strictPort: true,
    fs: { allow: [resolve(import.meta.dirname, "..")] },
  },
  build: {
    outDir: "dist",
    assetsInlineLimit: 0,
    emptyOutDir: true,
    rollupOptions: {
      input: {
        main: resolve(import.meta.dirname, "site/index.html"),
      },
    },
  },
  plugins: [tailwindcss(), viteReact()],
});
