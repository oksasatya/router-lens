/// <reference types="vitest/config" />
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "node:path";

// The TanStack Router plugin is added in the routing step (it needs src/routes/
// to generate routeTree.gen.ts). Verify dev + proxy work first.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: { alias: { "@": path.resolve(import.meta.dirname, "./src") } },
  server: {
    port: 5173,
    proxy: { "/api": { target: "http://localhost:8080", changeOrigin: true } },
  },
  test: { environment: "jsdom", globals: true, setupFiles: "./src/test/setup.ts" },
});
