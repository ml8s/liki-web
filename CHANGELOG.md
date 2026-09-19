# Changelog

## 2026.09.19.2 —— Alipay A2M Golden 校准

- 新增测试资产 `internal/donation/testdata/aipay`：纯标准库 Python Golden oracle 与已签名协议 vectors，用于独立校准 Go A2M 实现；Go CI 直接消费 vectors，不依赖 Python。
- 新增 A2M conformance tests：覆盖商家账单签名、Payment-Needed/Payment-Proof 编码、verify/confirm 同步响应验签、空值跳过与篡改拒绝。
- 修复 Go OpenAPI 签名串：同步响应 JSON 空字符串/null 按规则跳过；跳过后不再生成前导 `&`。
- 新增官方沙箱 `.alipay-sandbox.json` 显式加载器：仅本地显式 `ALIPAY_CONFIG_FILE` 启用，强制 0600、固定沙箱网关和 service ID，不引入 mock 旁路；生产仍走环境变量。

## 2026.09.19.1 —— Alipay AI Pay 捐赠凭证

- 新增独立的 `POST /api/donations/redeem`：无凭证返回 402 + Payment-Needed；凭证经支付宝 A2M 验付、本地幂等落库并履约确认后返回捐赠凭证。
- 捐赠与商业 payment / entitlement 完全隔离：不创建权益、不解锁 RPC、不改 engine；所有功能继续免费。
- 新增 `internal/donation` 协议客户端、SQLite 状态机、handler 和契约测试；未配置 Alipay 时端点返回 503 disabled。
- Skill 新增 `donation.py`：只在完整结果获用户满意确认后提示一次；凭证整块原子写入 `~/.liki/donation.json`。
- Make 新增 `test-python`，CI test job 同步执行；环境样例补充 Alipay 捐赠配置。

## 2026.09.19 —— Makefile 领域解耦 + 全量质量修复

- **Makefile 重构**：34 target → 20 个，按领域解耦为 `check`（静态检查+安全扫描）→ `test`（全量测试+E2E）→ `gate`（check+test 并行组合=部署门槛）→ `deploy` → `smoke`；零内联编排 shell，复杂逻辑提取到 6 个新脚本（check-caddy/check-skills/check-packaging/e2e/deploy/smoke）
- **CI 对齐**：5 job → 3 job（check/test/e2e），直接调 make target；补 vue-tsc（CI 原缺失）、check-caddy（CI 原假绿灯）；govulncheck/Trivy 从独立 security job 并入 check job
- **测试修复**：profile cursor 分页 INSERT 改用纳秒时间戳（datetime('now') 秒级精度导致同秒排序不确定）；List cursor 统一用 `created_at`（原 ORDER BY 和 cursor 列不匹配）；新增 migration 003 补 `idx_profiles_user_created` 索引
- **错误架构**：`unhandledrejection` 不再调 `reportFatal`（网络异常不崩 app）；AppLayout/ChatListView/ProfileDetailView/ChatSessionView onMounted 加 try/catch；`auth.loadMe` 移除 re-throw 统一为 return null
- **安全**：`payment/store.go` SQL 拼接 → 白名单 + Sprintf；`agent/tools.go` panic → slog.Error + nil guard；`ALTER TABLE` 拼接加列名白名单
- **类型安全**：`usePaipanData.ts` 10 处 `any` → 引用已有类型；`engineSolarTime` `lunar: unknown` → `string`；`engineZiweiChart` 参数 `unknown` → `string`
- **死代码清理**：删除 `usePaipanData.ts`（零消费者）、`canSend` getter（零引用）、`stream-renderer.test.js`（测试代码副本非生产码）、`brand.test.js`/`i18n.test.js`（E2E 已覆盖）、`check-no-legacy-skills.sh`（迁移已完成）
- **冗余收敛**：`PaipanPreview` 30 行排盘逻辑 → 复用 `resolvePaipanContext()`；错误消息 4 处字面量 → `ERR_CITY_NOT_FOUND`/`ERR_CITY_INVALID` 常量
- **RPC 方法名**：`RPC_METHODS` 常量从 `engine.ts` 导出，API 层和 E2E mock 统一引用，消除方法名漂移
- **types.ts 删除**：`Me`→`api/me.ts`、`EmailCodeResponse`→`api/auth.ts`，类型 co-location 到产生方
- **Caddyfile**：`caddy fmt --overwrite` 修复格式
- **pre-push hook**：从 30 行旧 target 引用简化为 `make check`

# Changelog

## 2026.09.16 —— brand renormalized to Liki

- 主品牌统一为 `Liki`，不再设置中文核心品牌名；中文展示口径为 `Liki 命理`。
- 唯一对外 slogan 更新为「懂命理，用 Liki。」，同步 README、i18n、静态页、SPA、E2E 和品牌契约测试。
- 旧报告提示词口径改为 `Liki 命书` / `Liki 合盘`；内嵌 unified skill 同步到 runtime `2026.09.16.1`。
- 历史报告与归档营销材料不重写。

## 2026.09.14 —— 统一 Chat 平台：档案域 + Chat 绑定 + 工具切面 + feedback-v1

- **档案域（新平台）**：profiles CRUD + 乐观锁 + 软删 + cursor 分页 + 配额 + owner 校验（纯用户信息，无 pan）；前端建档/编辑/免费排盘预览（TS 直连引擎 RPC，不进库）
- **Chat 档案绑定**：会话绑定 `profile_id`、首条 `system_profile` 系统消息、档案用户信息注入 agent；定盘写回工具 `update_profile_birth_time`（走网络 PATCH，与前端同一接口，会话绑定 guard）
- **工具切面一致性**：命理向结果契约（去 `_product`/`content` 传输壳）、统一 30s 工具超时、429 语义化、`requireBoundProfile` guard；引擎/skill 侧审计零改动（提示词无信封引用）
- **feedback 存原文**：`POST /api/feedback` 改收 `feedback-v1` 原文 JSON，不解析字段（schema 演进零后端改动），内容 hash 去重，UTF-8 强制
- **前端**：档案列表/新建/详情三页、dashboard"继续上次咨询"、支付返回直达解读、设计系统（墨/纸/紫/金 + 宋体）、账户会话管理
- **文档**：`platform-fs.md` v1.4、`backend-profiles-plan.md` v2.1、`backend-tool-consistency-plan.md`、UX 交互设计、前端实施计划、双 v0 mock

## 2026.08.27 —— 版本制切换 semver → CalVer（日期版本）

- VERSION 写入日期戳（2026.08.27），make version-* 三命令合并为 make version；里程碑按需 `git tag -a <日期>`
- 动因：无依赖解析场景（网站持续部署），semver 判断是仪式负担；历史版本条目（0.x）保留
- 同步：CI skills 锁 ref v5.0.1 → 2026.08.27（liki-skills 同日切换）；删除全部历史 tag（v0.2.0…v0.8.0）

## 0.22.0 —— skill 规则引擎工具注册（web agent 执行 python 工具链）

- agent 新增 5 个 skill 规则引擎工具：full_paipan / liunian / make_factors / make_liunian_factors / query
  - schema 从 web/skills/liki-bazi/tools/skill-tools.json 加载（OpenAI function calling，单一来源）
  - 执行：exec python3 web/skills/liki-bazi/tools/agent_cli.py（stdin {fn,args} → stdout {ok,data}；白名单分派、30s 超时、参数化无代码注入）
  - python3 缺失/失败 → 工具报错（不降级，保真值表精度）
- 部署：Dockerfile 加 python3；compose/env 加 LIKI_RPC_URL=http://engine:8082/jsonrpc（python 子进程内网直连引擎）
- 测试：skill_tools_test（schema 注册 5 工具 / 错误透传 / query 空结果）

- 0.21.2: chatagent 提示词统一 + greeting 持久化
  - PromptFile: liki/app/fatechart.md → liki/SKILL.md（与 skill 单一事实源：真太阳时路A/B/Gate强制清单/路由分发生效）
  - chat.greeting 三语言：先问服务类型（①宝宝起名 ②成人改名 ③评估），不再假定「宝宝起名」、不含公司名（qiming 不支持）
  - greeting 持久化：前端 messages 无 user 时带 context，后端 history 空持久化为 assistant 首条——LLM 上下文始终含 greeting
  - E2E 断言同步新 greeting 文案
- 0.21.1: 单 skill 部署链路修复
  - sync-skills.sh 重写：适配 liki 单库（源 ../skills/liki/，同步 web/skills/liki/）
  - Makefile: SKILLS=liki + check-skills/check-packaging 引用 liki.tar.gz（清 liki-master 残留）
  - 清理 liki-master 残留（测试/e2e/i18n/llms.txt/ci.yml/index.astro/README）
  - 删过时拆分规划文档（prompt-switch-plan）
- 0.21.0: 同步 2.0.0 skill 架构——naming/SKILL.md→app/naming.md；打包路径 dist/→根目录；check-skills 适配新结构
- 0.19.0: feedback agent SKILL.md + FEEDBACK_API_KEY 配置
- 0.18.0: feedback API + 隐私政策更新 + 首页间距调整
- 0.17.1: 首页关键词云+SEO词库+CSS清理+hero布局调整
- 0.17.0: 品牌定位对齐 — 命理师的 Skill / Professional Skill for Chinese Metaphysics
- 0.16.2: 结构化数据修复 — availability + image + 空标签删除 + brand
- 0.16.1: 英文品牌统一 Liki — Chinese Astrology Skill
- 0.16.0: 英文品牌 Chinese Astrology（去掉 & Metaphysics Skill）
- 0.14.0: 英文 lifebook title 去重 + CTA 统一 Life Reading + zh-Hans 空格统一
- 0.13.7: 修复报告页 catch () 语法错误导致 E2E 报 Unexpected token ')'
- 0.13.6: E2E a11y 修复（scrollable-region-focusable + color-contrast 4 轮），hepan/lifebook 产品页可访问性全部通过
- 0.13.5: checkout HTTP handler 测试 + httputil 59% 覆盖；报告页骨架屏 + 30s 超时 + 错误/重试
- 0.13.4: 报告页独立路由 /hepan/report + /lifebook/report；pipeline 测试补齐
- 0.13.3: 代码清理（死代码、404 CSS/JS 引用、route.go Go doc、Makefile help）
- 0.13.2: 统一报告渲染器 render-report.js；hepan + lifebook JSON 格式统一
- 0.13.1: naming 独立域 + admin 独立域 + payment 回调化
- 0.13.0: chat 去 Vue 改 vanilla JS，三产品页组件化
- 0.12.8: Astro 迁移 — 旧 build-html 体系替换为 Astro 静态生成
- 0.11.0: 架构重构 — 邮件事件回调 + Pipeline→Generator
