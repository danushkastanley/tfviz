/// <reference types="vitest/config" />
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// The production build feeds the Go HTML exporter, which inlines exactly one
// script and one stylesheet and hashes them into the report's CSP. Code
// splitting, module preloading and emitted assets are therefore all disabled.
export default defineConfig({
  plugins: [react()],
  base: "./",
  build: {
    outDir: "dist",
    emptyOutDir: true,
    cssCodeSplit: false,
    modulePreload: false,
    assetsInlineLimit: 0,
    sourcemap: false,
    rolldownOptions: {
      output: {
        codeSplitting: false,
        entryFileNames: "tfviz.js",
        assetFileNames: "tfviz[extname]",
      },
    },
  },
  server: {
    fs: { allow: [".."] },
  },
  test: {
    include: ["src/**/*.test.ts"],
    environment: "node",
  },
});
