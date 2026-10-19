import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

export default defineConfig({
  plugins: [
    svelte(),
    {
      name: "keep-dist-gitkeep",
      generateBundle() {
        this.emitFile({ type: "asset", fileName: ".gitkeep", source: "" });
      },
    },
  ],
  build: { outDir: "dist", emptyOutDir: true },
  server: { proxy: { "/api": "http://localhost:8080" } },
});
