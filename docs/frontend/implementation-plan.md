# Liki App 前端技术实施方案（Phase 2）

- 版本：v1.0
- 关联：`docs/platform-fs.md`（功能规格 v1.3）、`docs/frontend/ux-design.md`（交互设计 v1.0）、`docs/frontend/mocks/`（v0 草稿）
- 范围：登录后 Vue 3 SPA（`/app`），只覆盖新平台；legacy 页面不动

## 1. 目标与约束

- **目标**：落地"档案 + 排盘 + 自由对话"的完整前端，按 FS v1.3 与 UX v1.0。
- **后端依赖**：profiles API、chat `profile_id` 为**后端新契约**——本方案只消费，不改后端；缺失项作为前置依赖提出。
- **约束**：不引入新框架/状态库；API 调用一律走 `src/api`；SSE 解析集中在 `src/api/sse.ts`；不落 token 到 localStorage。

## 2. 现状盘点

| 层 | 现状 |
|---|---|
| 路由 | login / `/(dashboard)` / chat / chat/:id / reports / orders / account |
| stores | `auth` / `entitlements` / `chat`（Pinia） |
| api | `http` / `me` / `chat` / `commerce` / `orders` / `sse` |
| 组件 | `app/*`(5)、`chat/*`(6)、`commerce/ProductPurchaseCard` |
| 配置 | `config/chat.ts`（4 skill） |
| 测试 | `src/tests/*`（8 文件 17 用例）+ Playwright E2E |

## 3. 目标架构

### 3.1 路由（新增 profiles 域）
```
/login                       公开
/                            工作台（dashboard，改造：主 CTA"继续上次咨询" + 档案概览）
/profiles                    档案列表（新）
/profiles/new                建档表单（新）
/profiles/:id                档案详情 + 盘面（新）
/chat                        会话中心（改造：档案选择 + skill + 会话）
/chat/:sessionId             会话（改造：system_profile 首条消息）
/orders /account /reports    已有（orders/account 收编到 app 组件族）
```
- 路由 guard 沿用 `router/index.ts` 现有逻辑（登录态）。

### 3.2 组件树
```
components/profiles/         新增
├── ProfileList.vue          档案列表（空/加载/错误 + 删除确认）
├── ProfileCard.vue          列表项（称呼 + 最近使用排序）
├── BirthForm.vue            渐进式表单（known/approximate/unknown + 两路提示）
├── ProfileInfo.vue          基本信息展示 + 编辑（乐观锁）
└── PaipanPreview.vue        免费排盘预览（前端直连引擎，临时展示）

components/chat/             改造
├── ProfilePicker.vue        档案选择器（默认最近使用）
├── SystemProfileCard.vue    system_profile 首条消息渲染（档案卡，非对话轮）
├── ChatMessageItem.vue      + kind 分支（text / system_profile）
├── ChatThread.vue / SessionList / ChatComposer / ChatStreamStatus   复用
└── SessionEmpty.vue         空会话引导

components/commerce/         ProductPurchaseCard.vue 复用（档案详情 CTA 展开）

components/app/*             统一组件族，OrdersView/AccountView 收编
```

### 3.3 stores
```
stores/profiles.ts           新增
  state: list, detail, loading, error, 表单草稿
  actions: loadList / loadDetail / create / update / remove
stores/chat.ts                改造
  state: + activeProfileId
  actions: openProfile / createSession(profileId, skill)
           send()（agent 只回文字，前端不处理盘面事件）
stores/auth / entitlements    已有，不动
```

### 3.4 api clients
```
api/profiles.ts              新增
  fetchProfiles / fetchProfile(id) / createProfile(body)
  updateProfile(id, body, expectedVersion) / deleteProfile(id)
  → 封装 409 profile_conflict 与 cursor 分页（无 pan）
api/engine.ts                新增（免费排盘预览：前端直连引擎 RPC，只读消费）
  engineRpc(method, params) → 取 result.data；两路真太阳时编排
api/chat.ts                  改造
  createChatSession(profileId, skill, title)
  fetchChatMessages()：消息含 kind（text|system_profile）
api/me.ts                    复用（账户页 sessions）
```

## 4. 关键实现细节

### 4.1 建档表单（用户价值优先，V1/V3）
- `BirthForm.vue`：**第一屏仅称呼 + 出生日期**（公历/农历切换）；性别/时间/出生地/关系/备注在第二步展开（30 秒点出第一屏）。
- 时间三分：精确 `HH:mm` → `known`；"只知道时辰" → `approximate`；"不确定" → `unknown`（文案无压力"可对话定盘"）。
- 出生地只在 `known` 时展示"将做真太阳时修正"提示；`approximate` 明确"按已定时辰排，不校正"。
- 提交 → `POST /api/profiles` → 跳详情，**渐进呈现**：先出八字，紫微/八宅逐个补齐（不做整体干等骨架）。

### 4.2 免费排盘预览（前端直连引擎）
- `PaipanPreview.vue`：档案详情页"免费预览排盘"入口，临时展示、**不进库**。
- 编排（仅八字，作为引流钩子）：`city.coords`(可选) → `tianwen.time` → `bazi.chart`，取 `result.data`。
- 两路语义：`known`+出生地 → 真太阳时校正；`approximate` → 直接按给定时辰排；`unknown` → 提示定盘。
- 展示：四柱宫格 + 一行确定性摘要；加载/错误/重试态。

### 4.3 会话创建（档案归属）
- `createSession(profileId, skill)`：后端校验档案归属，越权/不存在 → `404`。
- **skill 默认推荐（V6）**：默认选八字（档案核心是出生信息），用户可改。

### 4.4 SSE chat + 消息类型
- 发送：占位 user/assistant → `text-delta` 追加 → `done` 用权威消息替换。
- 会话首条 `system_profile`（系统写入，JSON 摘要）→ 前端渲染为档案卡，**不是 LLM 对话轮**。
- **盘面展示与 agent 解耦**：agent 只输出解读文字；盘面一律走前端 RPC 直连（同免费预览）。
- `error` → 重拉权威消息；停止 → abort + 重拉。
- 滚动 sticky-to-bottom，用户上滚暂停。

### 4.5 支付返回 + 权益轮询
- 复用现有 `handlePaymentReturn`：`status=paid` → 轮询 entitlements → 成功后**直接建会话并跳转**（或聚焦档案详情"开始解读"），不回流再找（V4）。

### 4.6 账户会话管理
- AccountView 用 `GET /api/me/sessions`、`DELETE /api/me/sessions/{id}` 实现当前会话/其他设备/revoke。

### 4.7 设计系统落地
- 全局 CSS 变量覆盖 Element Plus：`--el-color-primary: #6d5acf` + 派生 shades。
- `styles/app.css` 扩展：纸白底、墨文字、宋体品牌字类（`.font-serif`）、盘面金线类。
- 收编 OrdersView/AccountView 到 `AppPageHeader/AppAlert/AppEmpty/AppSkeleton/AppErrorState`。

## 5. 里程碑

| 阶段 | 内容 | 依赖 |
|---|---|---|
| **M0 现状稳固** | orders/account 收编、chat 状态矩阵与测试补全、设计系统 token 注入 | 无 |
| **M1 档案域** | profiles API → store + api → ProfileList/ProfileCard/BirthForm（第一屏 2 项）→ 档案详情 + 盘面（schema-aware + 兜底 + 人话摘要 + 渐进呈现）+ **移动端适配** | 后端 profiles API |
| **M2 Chat 改造** | ProfilePicker + skill 默认推荐 + system_profile 首条 + 支付返回直达解读 | 后端 chat profile_id |
| **M3 打磨** | dashboard"继续上次咨询"、E2E 全链路 | — |

## 6. 测试计划

| 层 | 覆盖 |
|---|---|
| api | `profiles` CRUD/乐观锁/分页；`engine` RPC 直连（排盘预览） |
| store | `profiles` 状态流转；`chat` system_profile、SSE done/error/stop |
| 组件 | `BirthForm` 两路校验、`SystemProfileCard` 渲染、`ProfileCard` 状态徽标 |
| E2E | 登录→建档（known/unknown）→盘面→(未购)解锁→(已购)建会话→对话→停止→刷新恢复 |

## 7. 风险与开放项

- **后端契约未就绪**（profiles API / chat profile_id）→ M1/M2 前置，需后端排期。
- **pan schema 演进** → schema-agnostic 兜底 + `pan_schema_version` 适配。
- **乐观锁冲突**（多设备编辑）→ 409 处理 + 刷新重试。
- **排盘异步 UX** → pending 骨架 + 轮询，超时降级文案（固定）。
- **定盘写回（已实现，后端）**：agent 经 `update_profile_birth_time` 工具走网络 `PATCH /api/profiles/{id}` 写回出生时间（与前端同一接口）；前端档案详情页刷新即显示新时间，无需额外 UI。

## 8. 后端前置依赖清单（提需求，不改后端）

```
1. POST /api/profiles（免费建档，201 + pan_status）
2. GET /api/profiles（cursor 分页）+ GET /api/profiles/{id}
3. PATCH /api/profiles/{id}（expected_version 乐观锁 → 409）
4. DELETE /api/profiles/{id}（软删）
5. POST /api/chat/sessions（body+profile_id；创建写 system_profile 首条）
6. messages 含 kind（text|system_profile）
7. GET /api/me/sessions / DELETE /api/me/sessions/{id}（账户页，已有）
```