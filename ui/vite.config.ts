/// <reference types="vitest/config" />
import tailwindcss from "@tailwindcss/vite";
import { devtools } from "@tanstack/devtools-vite";

import { tanstackRouter } from "@tanstack/router-plugin/vite";

import viteReact from "@vitejs/plugin-react";
import { defineConfig } from "vite";

const config = defineConfig({
  resolve: { tsconfigPaths: true },
  test: {
    environment: "node",
  },
  optimizeDeps: {
    include: [
      "class-variance-authority",
      "cn",
      "@base-ui/react/button",
      "@base-ui/react/input",
      "@base-ui/react/separator",
      "@tanstack/react-devtools",
      "@tanstack/react-router-devtools",
    ],
  },
  server: {
    port: 5173,
    strictPort: true,
    proxy: {
      "/health": { target: "http://localhost:8080", changeOrigin: true },
      "/api": { target: "http://localhost:8080", changeOrigin: true },
    },
  },
  build: {
    outDir: "../pkg/web/dist",
    emptyOutDir: true,
  },
  plugins: [
    devtools(),
    tailwindcss(),
    tanstackRouter({ target: "react", autoCodeSplitting: true }),
    viteReact(),
  ],
});

export default config;
