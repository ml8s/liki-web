┌─────────────────────────────────────────────────────────────────────┐
│                    liki.hk — 部署技术栈总览                          │
│                          (系统架构图)                               │
└─────────────────────────────────────────────────────────────────────┘

                                 ▲ HTTPS :443
                                 │
                    ┌────────────┴────────────┐
                    │      Caddy 2             │
                    │  · TLS 终止 (HTTPS→HTTP) │
                    │  · 反向代理              │
                    │  · Accept-Language 重定向 │
                    │  · 静态资源缓存头         │
                    │  · CSP 安全头             │
                    │  · 路径路由 (见下文)       │
                    └────────────┬────────────┘
                                │ HTTP :8081
                ┌───────────────┼────────────────┐
                │               │                │
         ┌──────┴──────┐ ┌─────┴──────┐  ┌──────┴──────┐
         │  静态文件    │ │  Go API    │  │  JSON-RPC   │
         │  (Astro构建) │ │  (业务逻辑) │  │  (引擎)     │
         │  /zh-Hans/*  │ │  /api/*    │  │  /jsonrpc   │
         │  /js/*       │ │            │  │             │
         │  /css/*      │ └──────┬─────┘  │  liki-engine│
         │  /img/*      │        │        │  :8082      │
         └──────────────┘        │        └─────────────┘
                                 │
                    ┌────────────┴────────────┐
                    │  Go (liki-web)          │
                    │  http.ServeMux 路由     │
                    │  port :8081             │
                    └────────────┬────────────┘
                                 │
          ┌──────────────────────┼──────────────────────┐
          │                      │                      │
   ┌──────┴──────┐       ┌──────┴──────┐       ┌──────┴──────┐
   │ 命名域       │       │ 命书域       │       │ 合盘域       │
   │ naming/     │       │ mingshu/    │       │ hepan/      │
   │             │       │             │       │             │
   │ · store.go  │       │ · store.go  │       │ · store.go  │
   │ · handler   │       │ · handler   │       │ · handler   │
   │ · orders    │       │ · orders    │       │ · orders    │
   │ · chat (SSE)│       │ · generate  │       │ · generate  │
   │ · auth/JWT  │       │ · render    │       │ · render    │
   └──────┬──────┘       └──────┬──────┘       └──────┬──────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │
                    ┌────────────┴────────────┐
                    │   共享基础设施            │
                    ├─────────────────────────┤
                    │ · payment.Runtime 支付   │
                    │   (dodo 海外 + xunhu国内)│
                    │ · agent/ LLM Agent      │
                    │ · llm/ LLM 客户端       │
                    │ · email/ 邮件发送       │
                    │ · httputil/ 工具函数     │
                    │ · payment/runtime.go    │
                    └─────────────────────────┘

═══════════════════════════════════════════════════════════════════════
                      组件调用链详细图谱
═══════════════════════════════════════════════════════════════════════

┌─ 用户请求配置 ─────────────────────────────────────────────────────┐
│                                                                     │
│ 证书: Caddy 自动 HTTPS (Let's Encrypt)                              │
│ 域名: liki.hk (指向 Caddy 所在服务器 IP)                             │
│ 端口: Caddy :443 (HTTPS) → Go :8081 (HTTP)                          │
│ 引擎: Caddy /jsonrpc → engine :8082                                  │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘

┌─ Caddy 路由配置 ───────────────────────────────────────────────────┐
│                                                                     │
│ 路径模式         → 后端        → 说明                               │
│────────────────────────────────────────────────────────────────────│
│ /jsonrpc         → engine:8082  → LLM 引擎 (外部AI可直接调用)      │
│ /api/*           → web:8081    → Go API (CRUD+支付+Agent)          │
│ /zh-Hans/*       → 静态文件    → Astro 构建的静态页面               │
│ /zh-Hant/*       → 静态文件    → 同上                               │
│ /en/*            → 静态文件    → 同上                               │
│ /chat → 302 /zh-Hant/chat      → 旧 URL 兼容 (现已改成 naming/chat)│
│ /skills          → 静态文件    → AI Agent 发现协议                   │
│                                                                     │
│ 安全头:                                                             │
│ - CSP: script-src 'self' 'unsafe-inline'                            │
│ - HSTS: max-age=31536000                                            │
│ - X-Frame-Options: DENY                                             │
│ - 缓存: /js/* /css/* → immutable 1年                                │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘

┌─ Go 服务 (liki-web) ──────────────────────────────────────────────┐
│                                                                     │
│ 启动 (main.go)                                                      │
│  ├─ 读取 .env 配置                                                  │
│  ├─ 初始化 DB (SQLite × 3)                                          │
│  │   ├─ liki.db        (命名域 orders + chat)                       │
│  │   ├─ mingshu.db     (命书域 orders + reports)                    │
│  │   └─ hepan.db       (合盘域 orders + reports)                    │
│  ├─ 初始化基础设施                                                   │
│  │   ├─ payment.Runtime (canonical intents + dodo/xunhu adapters + notifications)                     │
│  │   ├─ email.Sender (Resend API)                                   │
│  │   ├─ llm.Client (DeepSeek API)                                   │
│  │   └─ agent.Agent (LLM Prompt + Tools)                             │
│  ├─ 初始化 AI Agent                                                  │
│  │   ├─ prompt = 读取 LIKI_PROMPT_FILE (naming/SKILL.md)             │
│  │   ├─ tools.register("read_file", ...)                            │
│  │   └─ tools.discoverEngineTools() → engine rpc.discover           │
│  ├─ 注册 HTTP 路由                                                   │
│  │   ├─ naming.RegisterRoutes(...)                                   │
│  │   │   ├─ POST /api/naming/auth/login                             │
│  │   │   ├─ POST /api/naming/orders                                 │
│  │   │   ├─ POST /api/naming/orders/select                          │
│  │   │   ├─ GET  /api/naming/orders/{id}/status                     │
│  │   │   ├─ GET  /api/naming/orders/{id}/messages                   │
│  │   │   ├─ DELETE /api/naming/orders/{id}/messages                 │
│  │   │   ├─ POST /api/naming/agent/chat  (SSE stream)               │
│  │   │   ├─ POST /api/payments/intents (rate limited)                │
│  │   │   ├─ POST /api/payments/intents/{id}/checkout (rate limited)  │
│  │   │   ├─ GET  /api/payments/intents/{id}/status                  │
│  │   │   ├─ GET  /api/payments/return/{id}                          │
│  │   │   ├─ POST /api/payments/webhook                              │
│  │   │   └─ POST /api/naming/verification-codes/redeem              │
│  │   ├─ hepan.RegisterRoutes(...)                                   │
│  │   │   └─ 同命名模式（hepan 替换 naming）                          │
│  │   ├─ mingshu.RegisterRoutes(...)                                 │
│  │   │   └─ 同命名模式（mingshu 替换 naming）                       │
│  │   ├─ admin.RegisterRoutes(...)  (naming-specific)                │
│  │   └─ fileServer(webDir, handler)  (静态文件中间件)                │
│  ├─ 中间件链                                                         │
│  │   ├─ Recover (panic 恢复)                                        │
│  │   ├─ SecurityHeaders (安全头 8 项)                                │
│  │   ├─ CORSMiddleware (跨域)                                        │
│  │   └─ BodyLimit (1MB 请求体限制)                                   │
│  └─ 启动 HTTP 监听 :8081                                             │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘

┌─ 请求处理流程图 ────────────────────────────────────────────────────┐
│                                                                     │
│ 浏览器 → https://liki.hk/zh-Hans/naming                             │
│                                                                     │
│ ① Caddy 接收 TLS 握手 (443)                                          │
│ ② Caddy 检查路径 → /zh-Hans/naming → 静态文件目录                    │
│ ③ Caddy 返回 web/zh-Hans/naming/index.html (Astro 构建产物)          │
│                                                                     │
│ 浏览器 → https://liki.hk/api/naming/orders (POST)                   │
│                                                                     │
│ ① Caddy 接收 TLS 握手 (443)                                          │
│ ② Caddy → 反向代理 → Go :8081                                       │
│ ③ Go 中间件链: Recover → Security → CORS → BodyLimit                │
│ ④ Go mux 匹配 POST /api/naming/orders                               │
│ ⑤ naming.HandleCreateOrder → store.CreateOrder                      │
│ ⑥ 返回 JSON {data:{order_id:"..."}}                                 │
│                                                                     │
│ 浏览器 SSE → https://liki.hk/api/naming/agent/chat                  │
│                                                                     │
│ ① JWT 验证 (cookie → email+orderID)                                 │
│ ② 查询订单 (paid + 未过期)                                           │
│ ③ 加载聊天历史                                                       │
│ ④ ensureNamingPrompt → 拼入 system prompt                           │
│ ⑤ agent.Execute → tool loop                                         │
│    ├─ LLM 决定调用 bazi.chart                                        │
│    │   └─ tools.Execute → POST engine:8082/jsonrpc                   │
│    ├─ LLM 决定调用 naming.chat                                       │
│    │   └─ engine 返回命理计算数据                                     │
│    └─ LLM 输出回复文本                                               │
│ ⑥ SSE 流式返回给浏览器                                               │
│                                                                     │
│ 支付完成回调 → https://liki.hk/api/payments/webhook                   │
│                                                                     │
│ ① Caddy 443 → Go :8081                                              │
│ ② paymentRuntime.HandleWebhook → provider VerifyWebhook             │
│    ├─ xunhu: MD5 验签、OD 成功态、精确 success ACK                  │
│    └─ dodo: Standard Webhooks HMAC 验签                             │
│ ③ MarkPaid → 幂等 enqueue notifications + fulfillment task           │
│    ├─ notification worker: 客户回执 / 管理员通知                    │
│    └─ fulfillment worker: 重试型 OnPaid（开通 / 报告生成）           │
│ ④ ReconcilePendingIntents 使用官方状态查询兜底                      │
│    ├─ xunhu: POST /payment/query.html + 金额校验                    │
│    └─ dodo: GET /checkouts/{id} → GET /payments/{payment_id}        │
│ ⑤ 用户浏览器 → GET /zh-Hans/naming/chat?order_id=xxx                │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘

┌─ 支付与履约领域契约 ─────────────────────────────────────────────────┐
│                                                                     │
│ · product catalog 是产品 ID 和定价唯一来源：naming / mingshu / hepan。│
│   历史 lifebook ID 会在迁移时归一为 mingshu。                         │
│ · billing currency 由服务端从 edge/geo header 推导，不由请求体决定。 │
│ · ProviderRoutePolicy 是服务端权威路由：CNY → xunhu，非 CNY → dodo。 │
│   请求里的 provider 只是 hint，不参与最终路由。                      │
│ · webhook / reconciliation 成功后写入幂等 fulfillment_tasks。        │
│   worker 负责产品履约，支持重试、backoff、running 恢复和终态失败。   │
│ · notifications 与 fulfillment 是独立 durable work stream。          │
│   重复回调只补齐缺失记录，不会重复执行已完成工作。                   │
│ · mingshu / hepan 报告读取要求订单 paid，并通过 JWT 归属或           │
│   有过期时间的 order access token 授权。                             │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘

┌─ 支付运营与状态授权 ────────────────────────────────────────────────┐
│                                                                     │
│ · payment_events 记录 webhook / reconciliation 的稳定事件指纹，       │
│   不保存原始回调和客户 PII。                                         │
│ · checkout 返回短时 status_token；状态与履约状态接口不再只依赖 UUID。 │
│ · fulfillment status 暴露 queued / running / succeeded / failed。     │
│ · provider 配置不完整时启动即禁用；两个渠道都缺失则启动失败。         │
│ · metrics token 为空时 /api/internal/metrics 返回 404。               │
│ · retention worker 关闭 7 天废弃 intent，并清理 90 天已发送通知。     │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘

┌─ LLM 工具调用链 ───────────────────────────────────────────────────┐
│                                                                     │
│ naming chat 场景:                                                    │
│                                                                     │
│ 用户: "我是1990年5月20日上海出生，想取个好名字"                       │
│                                                                     │
│ LLM Agent 内部循环:                                                  │
│                                                                     │
│ Round 1: LLM → 调用 tianwen_time(birth, location)                   │
│          ↓                                                          │
│          Engine → 返回真太阳时修正数据                                │
│                                                                     │
│ Round 2: LLM → 调用 bazi_chart(timeset)                              │
│          ↓                                                          │
│          Engine → 返回四柱干支/十神/藏干                              │
│                                                                     │
│ Round 3: LLM → 调用 bazi_yongshen(bazi)                              │
│          ↓                                                          │
│          Engine → 返回用神/喜神/忌神                                  │
│                                                                     │
│ Round 4: LLM → 调用 ziwei_chart(timeset)                             │
│          ↓                                                          │
│          Engine → 返回紫微命盘 + 十二宫                               │
│                                                                     │
│ Round 5: LLM 综合以上数据 → 生成起名建议输出                          │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
