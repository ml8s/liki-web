import { defineConfig } from 'astro/config';

export default defineConfig({
  build: {
    format: 'directory',
    assets: 'assets',
  },
  vite: {
    build: {
      // Keep Astro-processed scripts as external files so the production CSP
      // can continue to forbid executable inline scripts.
      assetsInlineLimit: 0,
    },
  },
});
