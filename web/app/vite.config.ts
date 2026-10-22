import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'
import compression from 'vite-plugin-compression'

export default defineConfig({
  base: '/app/',
  plugins: [
    vue(),
    // Element Plus styles are centrally loaded through a cascade layer; resolver
    // style side effects are disabled so vendor defaults cannot escape the layer.
    AutoImport({
      resolvers: [ElementPlusResolver({ importStyle: false })],
      dts: 'src/auto-imports.d.ts',
    }),
    Components({
      resolvers: [ElementPlusResolver({ importStyle: false })],
      dts: 'src/components.d.ts',
    }),
    compression({ algorithm: 'gzip' }),
    compression({ algorithm: 'brotliCompress' }),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      // 共享层（web/shared）跨边界引用统一走 @shared（层内保持相对路径，便于整层搬迁）。
      '@shared': fileURLToPath(new URL('../shared', import.meta.url)),
    },
    // 共享层（web/shared）同时被 App 与营销站编译：强制 vue / vue-i18n 解析到
    // 本包副本，避免双实例导致 provide/inject 与 VNode 标记失效。
    dedupe: ['vue', 'vue-i18n', 'element-plus', 'pinia'],
  },
  server: {
    proxy: {
      '/api': { target: 'http://localhost:8081', changeOrigin: false },
    },
  },
  build: {
    outDir: '../dist/app',
    emptyOutDir: true,
    // 生产不公开 sourcemap（6MB 源文件、暴露源码）；dev 由 vite dev server 自带。
    sourcemap: false,
    rollupOptions: {
      output: {
        // 举一反三：稳定 vendor 独立 chunk——浏览器长缓存。
        // 注意：element-plus 不在此强制整包——那会架空按需树摇（曾打回 938kB），
        // 已用的组件由 unplugin 按需导入，Rollup 自动归入共享 chunk。
        manualChunks: {
          'vue-vendor': ['vue', 'vue-router', 'pinia'],
        },
      },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: './src/tests/setup.ts',
    coverage: {
      provider: 'v8',
      reportsDirectory: './coverage',
      reporter: ['text-summary', 'html', 'json'],
      include: [
        'src/api/**',
        'src/stores/**',
        'src/components/**',
        'src/utils/**',
      ],
      thresholds: {
        statements: 29,
        branches: 25,
        functions: 18,
        lines: 30,
      },
    },
    // element-plus 按需引入后组件带 CSS side-effect import；测试里需内联
    // 让 Vite 处理 CSS，否则 Node ESM 会报 Unknown file extension ".css"。
    server: {
      deps: {
        inline: ['element-plus'],
      },
    },
  },
})
