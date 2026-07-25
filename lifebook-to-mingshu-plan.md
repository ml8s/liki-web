# lifebook → mingshu 重命名计划

## 范围总览

| 类别 | 位置 | 变更数 | 风险 |
|------|------|--------|------|
| Go 包 | `internal/lifebook/` → `internal/mingshu/` | mv 1 目录 | 🔴 |
| Go 引用 | `liki-web/internal/lifebook` 导入 | 26 处 | 🔴 |
| Go 产品常量 | `product.ProductLifeBook` | 1 处 | 🟡 |
| API 路径 | `/api/lifebook/*` → `/api/mingshu/*` | 12 处 | 🟡 |
| URL 路径 | `/lifebook` → `/mingshu` | 8 处 | 🟡 |
| Caddyfile | 重定向规则 + noindex | 19 行 | 🟢 |
| Astro 页面 | `lifebook.astro` + `lifebook/report.astro` | 2 文件 mv | 🟢 |
| JS 文件 | `lifebook.js` + `lifebook-report.js` | 2 文件 mv | 🟢 |
| JS 变量 | `renderLifeBook` → `renderMingShu` | 3 处 | 🟢 |
| i18n | `lifebook.*` (62 key) → `mingshu.*` | 186 行（3 locale） | 🟢 |
| CSS | `.lb-*` → `.ms-*` | 2 处 | 🟢 |
| DB | `lifebook.db` + `LIFEBOOK_DB_PATH` | 4 处 | 🟢 |
| E2E | pages.spec.js + test-api.sh | 6 行 | 🟢 |
| Caddy 旧链接跳转 | `/lifebook` → `/mingshu` 301 | 新增规则 | 🟢 |

## 实施顺序

### Phase 1: Go 包重命名（🔴 关键路径）

```bash
# 1a. 移动目录
git mv internal/lifebook internal/mingshu

# 1b. 修改包内所有文件的 package 声明
# internal/mingshu/*.go — 全部改为 package mingshu

# 1c. 更新所有外部引用（26 处）
# internal/http/server.go:   lifebook. → mingshu.
# internal/http/*_test.go:   lifebook. → mingshu.
# cmd/liki/main.go:          lifebook. → mingshu.
# internal/naming/route.go:  ProductLifeBook → ProductMingShu
# internal/admin/handler.go: ProductLifeBook → ProductMingShu
```

**具体文件清单：**

| 文件 | 改动 |
|------|------|
| `internal/http/server.go:8` | import `liki-web/internal/lifebook` → `mingshu` |
| `internal/http/server.go:47` | `lifebook.RegisterRoutes` → `mingshu.RegisterRoutes` |
| `internal/http/server.go:48` | `deps.LifeBookGenerator` → `deps.MingShuGenerator` |
| `internal/http/server_test.go:7` | import + ServerDeps 字段 |
| `internal/http/handler_integration_test.go` | 引用更新 |
| `internal/naming/route.go:23` | `namingProductFromOrder` 中 `"lifebook"` → `"mingshu"` |
| `internal/product/product.go` | `ProductLifeBook` → `ProductMingShu` |
| `internal/admin/handler.go` | `ProductLifeBook` → `ProductMingShu` |
| `internal/payment/service.go` | comment |
| `internal/checkout/checkout_test.go` | `ProductLifeBook` → `ProductMingShu` |
| `internal/hepan/handler_test.go` | ProductLifeBook → ProductMingShu |
| `internal/lifebook/*.go` | 全部改为 package mingshu |
| `cmd/liki/main.go:5` | import |
| `cmd/liki/main.go:67-70` | `lifebookDB` → `mingshuDB` |
| `cmd/liki/main.go:71-74` | `lifebookStore` → `mingshuStore` |
| `cmd/liki/main.go:75` | `lifebook.NewStore` → `mingshu.NewStore` |
| `cmd/liki/main.go:76-79` | `lifebookGenerator` → `mingshuGenerator` |
| `cmd/liki/main.go:80-83` | `hepan.OnReportReady` 中的 `"lifebook"` |
| `cmd/liki/main.go:86` | `lifebookDB.Close` → `mingshuDB.Close` |
| `cmd/liki/main.go:114` | `cleanupStaleOrders` 中的 `"lifebook"` |
| `cmd/liki/main.go` | `envOr("LIFEBOOK_DB_PATH"` → `"MINGSHU_DB_PATH"` |

### Phase 2: API + URL 路径更改

**route.go（2 文件）：**
```
ReturnPath:   "/api/mingshu/payments/return/"
RedirectPath: "/mingshu/report"
LogPrefix:    "mingshu:"
所有 HandleFunc 路径 /api/lifebook/ → /api/mingshu/
webhook 第一个参数 "lifebook" → "mingshu"
```

**handler_test.go（lifebook → mingshu）：**
```
所有 /api/lifebook/ → /api/mingshu/
所有 /lifebook/report → /mingshu/report
所有 LogPrefix "lifebook:" → "mingshu:"
```

### Phase 3: i18n 重命名

```python
# 3 个 locale 文件，每个 62 个 lifebook.* key → mingshu.*
# 脚本替换 key 前缀，value 内容不变
```

### Phase 4: Astro 文件重命名

```bash
git mv web/src/pages/\[locale\]/lifebook.astro web/src/pages/\[locale\]/mingshu.astro
git mv web/src/pages/\[locale\]/lifebook/report.astro web/src/pages/\[locale\]/mingshu/report.astro
# 更新 import 路径（不需要，同一目录层级不变）
```

### Phase 5: JS 文件重命名

```bash
git mv web/js/lifebook.js web/js/mingshu.js
git mv web/js/lifebook-report.js web/js/mingshu-report.js
# JS 内部变量名：renderLifeBook → renderMingShu
```

### Phase 6: CSS

```
.lb-card → .ms-card（2 处，如果存在）
```

### Phase 7: DB + 环境变量

```
LIFEBOOK_DB_PATH → MINGSHU_DB_PATH
lifebook.db → mingshu.db（DB 文件名）
```

### Phase 8: Caddyfile 改规则 + 加 301

**改路径：**
```
/lifebook → /mingshu（所有重定向规则）
```

**加旧链接跳转（deploy + deploy.local）：**
```
@oldLifebook path /lifebook /zh-Hans/lifebook /zh-Hant/lifebook /en/lifebook
redir @oldLifebook {path} 301  # 或按 locale 重定向
```

### Phase 9: 测试更新

```
scripts/test-api.sh: /lifebook → /mingshu（4 处）
web/e2e/journeys/pages.spec.js: lifebook → mingshu（2 处）
web/js/__tests__/lifebook.test.js → mingshu.test.js（重命名）
```

### Phase 10: 构建验证

```bash
go test ./... -count=1
cd web && npx astro build && npx vitest run js/__tests__/
node web/scripts/check-classes.cjs
node web/scripts/check-i18n.cjs
make check-lint
```

## 风险点

| 风险 | 缓解 |
|------|------|
| Go import 漏改导致编译失败 | 逐步构建，每改一个包就 `go build` |
| DB 表内 `product` 字段含 `"lifebook"` | 线上数据需迁移脚本 `UPDATE orders SET product='mingshu' WHERE product='lifebook'` |
| 旧 URL 没有 301 | Caddyfile 加跳转规则 |
| 外部链接/邮件中的旧链接 | 301 跳转兜底 |

## 预计耗时

| Phase | 估计 |
|-------|------|
| P1 Go 包名 | 30min |
| P2 API 路径 | 10min |
| P3 i18n | 10min |
| P4 Astro | 5min |
| P5 JS | 5min |
| P6 CSS | 2min |
| P7 DB | 2min |
| P8 Caddy | 5min |
| P9 测试 | 10min |
| P10 验证 | 10min |
| **总计** | **~1.5h** |
