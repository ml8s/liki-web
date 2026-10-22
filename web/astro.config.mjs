import { fileURLToPath, URL } from 'node:url';
import { defineConfig } from 'astro/config';
import vue from '@astrojs/vue';

export default defineConfig({
  // 站点绝对 URL：生产默认 liki.hk；本地 SEO 审计可用 SITE_URL=http://localhost:8080 构建。
  site: process.env.SITE_URL ?? 'https://liki.hk',
  integrations: [vue()],
  build: {
    format: 'directory',
    assets: 'assets',
  },
  vite: {
    resolve: {
      // 共享层（web/shared）跨边界引用统一走 @shared（层内保持相对路径）。
      alias: { '@shared': fileURLToPath(new URL('./shared', import.meta.url)) },
      dedupe: ['vue', 'vue-i18n', 'element-plus'],
    },
    build: {
      // Keep Astro-processed scripts as external files so the production CSP
      // can continue to forbid executable inline scripts.
      assetsInlineLimit: 0,
    },
  },
});
