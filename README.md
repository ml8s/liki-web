# Liki Web

命理Skills（灵机） — 起名、灵机命书、灵机合盘产品的独立域名系统。

## 架构

### 产品域（独立闭环）

```
internal/
├── naming/       # 起名 — 八字起名 + AI 聊天
├── mingshu/     # 灵机命书 — 个人命理报告
├── hepan/        # 灵机合盘 — 双人合盘报告
└── admin/        # 管理后台 — 订单/验证码管理
```

每个产品域自包含：

| 组件 | 说明 |
|------|------|
| `store.go` | 独立 SQLite（订单 + 报告各自一张 DB） |
| `handler.go` | HTTP handler（create/select/login/chat） |
| `checkout.go` | 支付流程（createCheckout/paymentReturn/orderStatus） |
| `webhook.go` | 回调处理（VerifyWebhook + onPaid 回调） |
| `generate.go` | 报告流水线（generate → review → revise） |
| `route.go` | 路由注册（`RegisterRoutes(mux, store, svc, agent)`） |

### 支付架构

`internal/payment/service.go` 提供通用能力：
- `Provider()` — 支付网关（dodo 海外 / xunhu 国内）
- `VerifyWebhook()` — 回调验签
- `Store` 接口已移除（各域用自有 Store）
- `OnPaid` 回调由各域注入（邮件内容在产品域编排）

### 前端

```
web/src/pages/[locale]/  （Astro SSG，3 locales × 13 页面 = 39 pages）
├── index.astro           # 首页
├── naming.astro          # 起名产品页
├── mingshu.astro        # 灵机命书产品页
├── mingshu/report.astro # 报告页（独立纯净页）
├── hepan.astro           # 合盘产品页
├── hepan/report.astro    # 报告页
├── chat.astro            # 聊天（Vue CSR）
└── about/activate/contact/disclaimer/privacy/terms
```

### 报告渲染

LLM 输出的 JSON 通过 `render-report.js` 统一渲染：

```json
{
  "title": "报告标题",
  "persons": [{"name":"甲方","label":"乾造"}],
  "sections": {
    "bazi": {"title":"八字合盘","analysis":"...","rating":"上等","advice":"..."}
  }
}
```

`render-report.js`（24 行）遍历 `Object.keys(sections)` → HTML 卡片，无外部依赖。

### 提示词管理

提示词位于 `../liki/reports/`（ml8s/liki 仓库），通过 `sync-skills.sh` 同步到 `web/skills/`。

每个产品一个报告目录：
```
reports/hepan/
├── SKILL.md          # AI agent 入口
├── generate.md       # 生成 prompt
├── review.md         # 审查 prompt
└── revise.md         # 修订 prompt
```

## 开发

```bash
make dev          # 启动本地开发环境（engine + web + Caddy）
make build        # 构建 Go + Astro
make check        # lint + vet + HTML + i18n + classes + skills
make test-all     # 全量测试（Go + JS + Docker E2E）
```

## 测试

| 层 | 数量 | 命令 |
|----|------|------|
| Go 单元 | 84 | `go test ./...` |
| JS 单元 | 81 | `npx vitest run js/__tests__/` |
| API 集成 | 48 | `scripts/test-api.sh` |
| E2E Playwright | 58 | `scripts/test-ui.sh` |
| 可访问性 | 8 | `npx playwright test e2e/journeys/a11y.spec.js` |

## 部署

```bash
make deploy-cn     # 部署国内（xunhu 支付）
make deploy-us     # 部署海外（dodo 支付）
```

部署脚本将二进制 + 静态文件 + Docker Compose 打包到服务器。
