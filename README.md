# Liki Web

<p align="center">
  <img alt="Liki Web" src="https://img.shields.io/badge/Liki_Web-命理_Skill-6d5acf?style=for-the-badge&logo=openai&logoColor=white&labelColor=30305c">
</p>

<p align="center">
  <strong>Liki Web — liki.hk 官方网站</strong><br>
  按命理师的专业标准构建：Liki Chat · 起名 · 八字 · 问卦 · 风水
</p>

<p align="center">
  <a href="https://github.com/ml8s/liki-web"><img src="https://img.shields.io/badge/GitHub-ml8s/liki_web-4a9e6b?style=flat&logo=github&logoColor=white&labelColor=30305c"></a>
  <a href="https://liki.hk"><img src="https://img.shields.io/badge/liki.hk-官方网站-6d5acf?style=flat&logo=safari&logoColor=white&labelColor=30305c"></a>
  <a href="https://github.com/ml8s/liki-web/actions/workflows/ci.yml"><img src="https://github.com/ml8s/liki-web/actions/workflows/ci.yml/badge.svg"></a>
  <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-1.26-00ADD8?style=flat&logo=go&logoColor=white&labelColor=30305c"></a>
  <a href="https://nodejs.org"><img src="https://img.shields.io/badge/Node-24.14-339933?style=flat&logo=nodedotjs&logoColor=white&labelColor=30305c"></a>
</p>

按命理师的专业标准构建的命理咨询平台。新体系以 **Liki Chat** 为统一入口，内部包含起名、八字、问卦、风水四个专业 skill；既有 naming、mingshu、hepan 作为 legacy 产品域继续隔离维护。

## 为什么可信

**① 确定性计算下沉引擎** — 所有排盘/派生由 liki-engine（Go）完成：31 个 RPC 全部有数据驱动锚点锁定（八字 115 golden 命例、紫微 150+100、调候穷通宝鉴 120 条原文对照、纳甲 64、定局 72、世应 64、命卦 12…），LLM 不参与计算。

**② 断语走真值表而非自由发挥** — 八字/紫微断语由规则引擎执行：597 条断语 / 46 张真值表（bazi 26 + ziwei 20，含 7 张 yearly 流年表）精确匹配，LLM 只做翻译与综合——防 agent 不稳定（精度不靠 LLM 发挥）；四门（六爻/奇门/八宅/玄空）断语用统一断语表（吉凶=符号固有属性）。

**③ 分层测试保障** — Go 单测/集成（`-race`）、JS 单测、组件测试、API 冒烟、Playwright E2E、skill 规则引擎 pytest，并接入 govulncheck/Trivy 安全扫描。

**④ 独立部署** — Docker Compose（engine + web + caddy）；python3 仅服务 skill 规则引擎（固定入口、白名单工具、超时控制，无任意代码执行）。

## 架构

### 架构分层

```
internal/
├── auth/          # 平台用户、邮箱验证码登录、登录态
├── entitlement/   # Liki Chat -> 唯一 liki skill 权益
├── commerce/      # Liki Chat 订单、checkout、支付履约适配
├── chat/          # 统一 Chat session / message / runtime 编排
├── agent/         # LLM/tool-call 内核与 skill 工具 scope
├── sse/           # 公共 Server-Sent Events 传输组件
├── payment/       # legacy 与 commerce 共用的支付运行时
├── naming/        # legacy：起名产品
├── mingshu/       # legacy：灵机命书报告
├── hepan/         # legacy：灵机合盘报告
└── admin/         # 管理后台
```

三层边界：

| 层 | 职责 |
|---|---|
| Platform | 用户、登录、权益、订单、支付 |
| Chat Runtime | 会话、消息、SSE、Agent 编排 |
| Skill / Engine | prompt、领域工具、确定性排盘与规则 |
| Legacy Product | naming / mingshu / hepan，保持隔离，不隐式进入新权益体系 |

### 支付架构

`internal/payment` 提供统一支付运行时：provider gateway、checkout、webhook、通知、durable fulfillment、对账和幂等由组合编排；dodo 海外 / xunhu 国内是可替换 Provider。

legacy 产品流继续注册在 runtime 中。Liki Chat 则通过 `internal/commerce` 创建订单，并在 provider capture 后由 `commerce.Fulfiller` 显式调用 entitlement 授权。payment runtime 不会隐式写入 Chat 权益。

### 平台用户与统一 Chat

`/app` 是登录后的 Vue 3 SPA。邮箱验证码登录建立平台用户和 HttpOnly 会话。

新商业模型只有一个产品/SKU：

```text
liki-chat
```

购买后解锁唯一 skill：

```text
liki
```

统一 Chat 只允许会话所属用户读写，并在每次读取、发送、创建会话时重新校验权益。旧 `mingshu`、`hepan`、`naming` 是 legacy 报告/支付链路，不隐式获得新 Chat 权益。

统一 skill（`liki`）持有单一入口 prompt 与 deny-by-default 工具范围，内部按意图路由到八字/起名/问卦/风水领域；`webapp/**` 报告提示词不进入 Chat。详细契约见 [`docs/platform-chat.md`](docs/platform-chat.md)。

Chat 商业上是一个产品、一个全功能 SKU；`liki` 是唯一 skill，领域能力是内部实现。Dodo 需要配置：

```bash
DODO_PRODUCT_LIKI_CHAT=
```

CNY checkout 由 Xunhu 承接；USD checkout 由 Dodo 承接。生产环境按目标市场配置对应 provider。

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

```
web/app/src/            （登录后 Vue 3 + Pinia + Vue Router SPA）
├── views/              # 页面编排：工作台、统一 Chat、订单、账户
├── components/         # app 通用组件、chat/commerce 业务组件
├── stores/             # auth、entitlements、chat
├── api/                # 平台 API 客户端与 SSE 解析
└── config/             # 内部 skill 展示配置
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

**旧 naming chat 与报告产品**继续使用现有产品提示词；**统一 Chat**加载 `../liki/skills/liki/`（单一 skill：`SKILL.md` 路由 + 四域 `ENTRY.md`），并由 `internal/agent/scope.go` 统一限制工具 schema 与执行。

**Agent 工具链**（`internal/agent/tools.go`、`skill_tools.go`、`scope.go`）：

- 引擎 RPC 由 `rpc.discover` 懒发现，`read_file` 限制在当前 skill 目录。
- 八字/问卦域工具分别加载 `liki/bazi/tools/skill-tools.json`、`liki/divination/tools/skill-tools.json`。
- Python 工具通过 `exec python3 agent_cli.py` 执行：stdin `{fn,args}` → stdout `{ok,data|error}`，固定分派、30 秒超时、无任意代码执行。
- `SkillToolScopes` 先过滤 schema，再在执行时再次校验；跨 skill 工具不可见也不可调用。

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

`make dev` 默认设置 `DEV_MODE=true` 和本地数据库路径。本地登录页会在邮件服务不可用时显示 6 位开发验证码；生产环境不受影响，也不会返回该字段。

平台契约文档见 [`docs/platform-chat.md`](docs/platform-chat.md)，前端页面与组件规则见 [`docs/frontend/README.md`](docs/frontend/README.md)。

## 测试

| 层 | 数量 | 命令 |
|----|------|------|
| Go 单测与集成 | 全量 | `go test ./... -count=1`；`make test-db` |
| JS 单测 | 全量 | `make test-js`（含 Astro 站点与 `/app` SPA） |
| API 集成 | 冒烟集 | `scripts/test-api.sh $URL`（需服务在线） |
| E2E Playwright | 全量 | `make test-ui URL=$URL` |
| 静态检查 | 全量 | `make lint vet` |

## 部署

```bash
make deploy-cn     # 部署国内（xunhu 支付）
make deploy-us     # 部署海外（dodo 支付）
```

部署脚本将二进制 + 静态文件 + Docker Compose 打包到服务器。
