import { defineConfig } from "vitest/config";
import { svelteTesting } from "@testing-library/svelte/vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

export default defineConfig({
  plugins: [
    svelte(),
    svelteTesting(),
    {
      name: "keep-dist-gitkeep",
      generateBundle() {
        this.emitFile({ type: "asset", fileName: ".gitkeep", source: "" });
      },
    },
  ],
  build: { outDir: "dist", emptyOutDir: true },
  test: { environment: "jsdom", include: ["src/**/*.test.ts"], setupFiles: ["src/test/setup.ts"] },
  server: { proxy: { "/api": "http://localhost:8080" } },
});
