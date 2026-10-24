import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

// The build goes into the Go package that embeds it.
export default defineConfig({
  plugins: [svelte()],
  build: {
    outDir: '../internal/webui/dist',
    // scripts/clean.mjs empties it first and keeps .gitkeep, which the Go embed needs.
    emptyOutDir: false,
  },
  server: {
    port: 5173,
    // During development the interface talks to a running server.
    proxy: { '/admin/api': 'http://127.0.0.1:8080' },
  },
});
