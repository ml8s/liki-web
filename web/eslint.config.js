import js from '@eslint/js'
import globals from 'globals'
import tseslint from 'typescript-eslint'

// web/ 的分层基线（与 web/app 同一条口径：类型检查归 astro check，lint 不做 type-aware）：
// - .astro 模板由 astro check 负责（eslint-plugin-astro 2.x/3.x 要求 Node >= 24.16，
//   1.x 带一个无修复版本的 high advisory，会让 CI 的 npm audit 变红），所以 eslint
//   只覆盖 .ts/.js。
// - .ts 交给 typescript-eslint；未定义标识符由 astro check 报告，no-undef 关掉
//   （base 规则会在浏览器/Node 全局上误报）。
export default [
  {
    ignores: ['app/', 'dist/', 'node_modules/', 'public/', 'test-results/', '.astro/'],
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    // 浏览器运行时代码：Astro 静态页的交互脚本。
    files: ['src/scripts/**/*.ts'],
    languageOptions: { globals: globals.browser },
  },
  {
    // Node 侧：构建脚本、检查脚本、测试与工具配置。
    files: ['scripts/**/*.{js,mjs}', 'test/**/*.ts', '**/*.config.{js,mjs,ts}'],
    languageOptions: { globals: globals.node },
  },
  {
    // Playwright：spec 跑在 Node，page.evaluate 回调跑在浏览器，两套全局都要有。
    files: ['e2e/**/*.{js,ts}'],
    languageOptions: { globals: { ...globals.node, ...globals.browser } },
  },
  {
    // journey 的 test/expect 只能来自 fixtures：那里的 API 契约是强制的。
    // 直接从 @playwright/test 取会静默绕过契约；打真实后端的用例用
    // `test.use({ liveBackend: true })` 显式退出。类型导入不受限制。
    files: ['e2e/journeys/**/*.ts'],
    rules: {
      'no-restricted-imports': ['error', {
        paths: [{
          name: '@playwright/test',
          importNames: ['test', 'expect'],
          message: '从 ../fixtures 取 test/expect（API 契约在那里生效）；真实后端用例用 test.use({ liveBackend: true })。',
        }],
      }],
    },
  },
  {
    files: ['**/*.ts'],
    rules: {
      'no-undef': 'off',
      'no-unused-vars': 'off',
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
    },
  },
]
