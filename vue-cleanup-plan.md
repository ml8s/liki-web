# 移除 Vue 清理计划

## 改动清单

### 1. astro.config.mjs
```diff
- import vue from '@astrojs/vue';
- integrations: [vue()],
+ integrations: [],  // 或移除整行
```
风险：无，无任何 .astro 文件使用 Vue 组件。

### 2. web/package.json
```diff
- "vue": "^3.5.38"           (dependencies)
- "@astrojs/vue": "^7.0.1"   (devDependencies)
- "@vue/compiler-dom": "^3.5.38"  (devDependencies)
```
运行 `cd web && npm install` 自动更新 `package-lock.json`。

### 3. web/eslint.config.js
```diff
- Vue: "readonly",  (line 54)
```

### 4. web/e2e/journeys/pages.spec.js
```diff
- // Catches: Vue compiler errors, source map 404s, ...
+ // Catches: source map 404s, i18n load failures, ...
- // Render markers: .chat-shell (Vue), null (static resources).
+ // Render markers: .chat-shell (vanilla JS), null (static resources).
- { path: '/zh-Hans/chat',  marker: '.chat-shell', name: 'chat (Vue)', ... }
+ { path: '/zh-Hans/chat',  marker: '.chat-shell', name: 'chat (vanilla)', ... }
- { path: '/en/chat',       marker: '.chat-shell', name: 'chat EN (Vue)', ... }
+ { path: '/en/chat',       marker: '.chat-shell', name: 'chat (vanilla)', ... }
```
### 5. web/js/__tests__/stream-renderer.test.js
```diff
- // The real function lives inside Vue setup() and is not directly importable.
+ // The real function lives inside chat.js and is duplicated here for test isolation.
```

## 验证清单
- [ ] `cd web && npm install` 通过
- [ ] `npx astro build` 39 pages 构建成功
- [ ] `npx vitest run js/__tests__/` 81 测试通过
- [ ] `node web/scripts/check-html.cjs` 通过
- [ ] `node web/scripts/check-classes.cjs` 通过
- [ ] `node web/scripts/check-i18n.cjs` 通过
