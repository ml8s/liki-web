import js from '@eslint/js'
import pluginVue from 'eslint-plugin-vue'
import tseslint from 'typescript-eslint'

// Baseline: JS recommended + Vue recommended. The TS parser is attached so
// lang="ts" script blocks parse; type-aware rules are deliberately not
// enabled yet (vue-tsc already covers type checking).
export default [
  js.configs.recommended,
  ...pluginVue.configs['flat/recommended'],
  {
    // Standalone .ts files need the TS parser directly; inside .vue files the
    // vue parser delegates via parserOptions.parser below.
    files: ['src/**/*.ts'],
    languageOptions: {
      parser: tseslint.parser,
    },
  },
  {
    // Element Plus styles enter through app/src/styles/element-plus.css; vendor
    // imports elsewhere bypass the cascade-layer theme boundary.
    files: ['src/**/*.{ts,vue}'],
    rules: {
      'no-restricted-imports': ['error', {
        patterns: [{
          group: ['element-plus/theme-chalk/*'],
          message: 'Use src/styles/element-plus.css; vendor styles are cascade-layered there.',
        }],
      }],
    },
  },
  {
    files: ['src/**/*.{vue,ts}'],
    plugins: {
      '@typescript-eslint': tseslint.plugin,
    },
    languageOptions: {
      parserOptions: {
        parser: tseslint.parser,
        sourceType: 'module',
      },
    },
    rules: {
      // TypeScript (vue-tsc) owns undefined-variable detection; no-undef
      // false-positives on browser globals and unplugin auto-imports.
      'no-undef': 'off',
      // The base rule flags TS type-signature parameters it does not
      // understand; the TS variant parses them correctly.
      'no-unused-vars': 'off',
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
      // Props are declared with TS optional types and handled with ?? / ?.
      // — withDefaults would duplicate those fallbacks.
      'vue/require-default-prop': 'off',
    },
  },
  {
    // The project's single, deliberately sanitized v-html injection point
    // (DOMPurify allowlist in utils/markdown.ts).
    files: ['src/components/common/AppMarkdown.vue'],
    rules: {
      'vue/no-v-html': 'off',
    },
  },
  {
    ignores: [
      'dist/',
      'node_modules/',
      'src/auto-imports.d.ts',
      'src/components.d.ts',
    ],
  },
]
