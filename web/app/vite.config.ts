import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'

export default defineConfig({
  base: '/app/',
  plugins: [
    vue(),
    // Element Plus 按需引入（官方标准做法）：模板里的 el-* 组件与 ElMessageBox
    // 等 API 自动导入并带样式，去掉 app.use(ElementPlus) 全量打包。
    AutoImport({
      resolvers: [ElementPlusResolver()],
      dts: 'src/auto-imports.d.ts',
    }),
    Components({
      resolvers: [ElementPlusResolver()],
      dts: 'src/components.d.ts',
    }),
  ],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  server: {
    proxy: {
      '/api': { target: 'http://localhost:8081', changeOrigin: false },
    },
  },
  build: {
    outDir: '../dist/app',
    emptyOutDir: true,
    sourcemap: true,
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
    // element-plus 按需引入后组件带 CSS side-effect import；测试里需内联
    // 让 Vite 处理 CSS，否则 Node ESM 会报 Unknown file extension ".css"。
    server: {
      deps: {
        inline: ['element-plus'],
      },
    },
  },
})