# chat 归入命名域 — `/chat` → `/naming/chat`

## 改动清单

### 1. Astro 页面移动
```bash
git mv web/src/pages/[locale]/chat.astro web/src/pages/[locale]/naming/chat.astro
# import 路径不变（同级目录下）
```

### 2. Go 后端 — RedirectPath
**文件**: `internal/naming/route.go:19`
```diff
- RedirectPath: "/chat",
+ RedirectPath: "/naming/chat",
```
支付成功后跳转 `/naming/chat?order_id=xxx`。

### 3. 前端链接更新

#### naming.astro（起名产品页）
```diff
- <a href={`/${locale}/chat`}>已有订单</a>
+ <a href={`/${locale}/naming/chat`}>已有订单</a>
```

#### naming/activate.astro（激活码页）
两处：
```diff
- <a href={`/${locale}/chat`}>已有订单</a>
+ <a href={`/${locale}/naming/chat`}>已有订单</a>
```
```diff
- location.href = '/' + loc + '/chat?order_id=' + orderID;
+ location.href = '/' + loc + '/naming/chat?order_id=' + orderID;
```

### 4. Caddyfile 改动

#### 4a 正则路径加入 naming/chat（2 文件）
**文件**: `deploy/liki/Caddyfile` 和 `deploy/liki/Caddyfile.local`
```diff
- path_regexp pages ^/(chat|naming|mingshu|...)?$
+ path_regexp pages ^/(chat|naming/chat|naming|mingshu|...)?$
```
（4 个 locale 各一处）

#### 4b 旧链接 301 跳转
```diff
+ # Old /chat → /naming/chat (compat)
+ @oldChat path /chat /zh-Hans/chat /zh-Hant/chat /en/chat
+ redir @oldChat {path}/naming/chat 301
```

#### 4c noindex 更新
```diff
- @noindex path_regexp noindex .*/(chat|report)(/.*)?$
+ @noindex path_regexp noindex .*/(chat|naming/chat|report)(/.*)?$
```

### 5. E2E 测试更新

| 文件 | 改动 |
|------|------|
| `web/e2e/journeys/a11y.spec.js` | `path: '/zh-Hans/chat'` → `'/zh-Hans/naming/chat'` |
| `web/e2e/journeys/landing.spec.js` | `page.goto('/zh-Hans/chat?order_id=...')` → `'/zh-Hans/naming/chat?order_id=...'` |

### 6. sitemap
chat 页面当前不在 sitemap 中（硬编码列表），无需改动。

### 7. 单元测试
`internal/naming/route.go` 的 `RedirectPath` 在 handler_test.go 中通过 `checkout.Config` 验证，会自动生效。

---

## 验证清单

- [ ] `go build ./...` 编译通过
- [ ] `go test ./...` 全部通过
- [ ] Astro 39 pages 构建成功
- [ ] `node web/scripts/check-html.cjs` 通过
- [ ] `node web/scripts/check-classes.cjs` 通过
- [ ] `Caddyfile` 解析正常
- [ ] 旧链接 `/zh-Hans/chat` 301 → `/zh-Hans/naming/chat`
- [ ] 支付跳转正确到 `/naming/chat?order_id=xxx`
