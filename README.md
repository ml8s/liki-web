# Liki Web

<p align="center">
  <img alt="Liki Web" src="https://img.shields.io/badge/Liki_Web-命理_Skill-6d5acf?style=for-the-badge&logo=openai&logoColor=white&labelColor=30305c">
</p>

<p align="center">
  <strong>Liki Web — liki.hk 官方网站</strong><br>
  按命理师的专业标准构建：起名 · 灵机命书 · 灵机合盘
</p>

<p align="center">
  <a href="https://github.com/ml8s/liki-web"><img src="https://img.shields.io/badge/GitHub-ml8s/liki_web-4a9e6b?style=flat&logo=github&logoColor=white&labelColor=30305c"></a>
  <a href="https://liki.hk"><img src="https://img.shields.io/badge/liki.hk-官方网站-6d5acf?style=flat&logo=safari&logoColor=white&labelColor=30305c"></a>
  <a href="https://github.com/ml8s/liki-web/actions/workflows/ci.yml"><img src="https://github.com/ml8s/liki-web/actions/workflows/ci.yml/badge.svg"></a>
  <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-1.26-00ADD8?style=flat&logo=go&logoColor=white&labelColor=30305c"></a>
  <a href="https://nodejs.org"><img src="https://img.shields.io/badge/Node-24.14-339933?style=flat&logo=nodedotjs&logoColor=white&labelColor=30305c"></a>
</p>

按命理师的专业标准构建的命理 Skill 官网——起名、灵机命书、灵机合盘三个产品。

## 为什么可信

**① 确定性计算下沉引擎** — 所有排盘/派生由 liki-engine（Go）完成：31 个 RPC 全部有数据驱动锚点锁定（八字 115 golden 命例、紫微 150+100、调候穷通宝鉴 120 条原文对照、纳甲 64、定局 72、世应 64、命卦 12…），LLM 不参与计算。

**② 断语走真值表而非自由发挥** — 八字/紫微断语由规则引擎执行：597 条断语 / 46 张真值表（bazi 26 + ziwei 20，含 7 张 yearly 流年表）精确匹配，LLM 只做翻译与综合——防 agent 不稳定（精度不靠 LLM 发挥）；四门（六爻/奇门/八宅/玄空）断语用统一断语表（吉凶=符号固有属性）。

**③ 三层测试保障** — Go 393 单测（`-race` + 覆盖率 ≥50% 门槛）+ JS 99 + E2E 55 + skill 规则引擎 pytest（集成测试在服务起后全跑）+ govulncheck/Trivy 安全扫描；三仓库 CI 全绿。

**④ 独立部署** — Docker Compose（engine + web + caddy）；python3 仅服务 skill 规则引擎（纯标准库、白名单 5 工具、无任意代码执行）。

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

提示词位于 `../liki/skills/`（ml8s/liki 仓库），通过 `sync-skills.sh` 同步到 `web/skills/`（skill 根形态 + webapp 部署提示词）。

**chatagent 主提示词** = `web/skills/liki/SKILL.md`（`LIKI_PROMPT_FILE` 可覆盖）——与 skill 单一事实源一致：真太阳时校正（路A/B）、关卡制 Gate 强制清单、按需 discover、路由分发全生效。`app/fatechart.md` 等作为子卡被 SKILL.md 路由引用。

**agent 工具链**（`internal/agent/tools.go` + `skill_tools.go`）：
- 引擎 RPC（`rpc.discover` 自动发现，31 方法）+ `read_file`（读 skills 目录）
- **skill 规则引擎 5 工具**（`full_paipan`/`liunian`/`make_factors`/`make_liunian_factors`/`query`）——schema 从 `web/skills/liki/tools/skill-tools.json` 加载（OpenAI function calling 单一来源），执行 `exec python3 agent_cli.py`（stdin {fn,args} → stdout {ok,data}；白名单分派、30s 超时、无任意代码执行）；**部署需 python3**（Dockerfile 已含），容器内经 `LIKI_RPC_URL` 内网直连引擎

**chat 开场白（greeting）**：前端 i18n（`chat.greeting`）显示 + 首条用户消息前经 `context` 参数传给后端，后端在 history 为空时持久化为 assistant 首条消息——LLM 上下文始终含 greeting，用户输「1/2/3」能对应前端选项。

**报告产品**各一个提示词目录：
```
webapp/hepan/
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
| Go 单元 | 393 | `go test ./...` |
| JS 单元 | 99 | `npx vitest run` |
| API 集成 | 冒烟集 | `scripts/test-api.sh $URL`（数量随脚本，需服务在线） |
| E2E Playwright | 55 | `npx playwright test e2e/journeys/`（chat 21/hepan 6/landing 9/pages 7/uncovered 11/a11y 1） |
| 可访问性 | 8 页审计 | `npx playwright test e2e/journeys/a11y.spec.js` |

## 部署

```bash
make deploy-cn     # 部署国内（xunhu 支付）
make deploy-us     # 部署海外（dodo 支付）
```

部署脚本将二进制 + 静态文件 + Docker Compose 打包到服务器。
