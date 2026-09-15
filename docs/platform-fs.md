# Liki Chat 新平台 · 功能规格说明书（FS）

- 版本：v1.4（模型修正：档案=用户信息，不存 pan；排盘由 agent/前端承担）
- 状态：已确认
- 范围：新平台（认证 / 档案 / 商品支付 / 权益 / Chat / 订单 / 账户）
- 配套：`docs/backend-profiles-plan.md`（后端方案）、`docs/frontend/README.md`（前端架构）

## 1. 范围与边界

| | 内容 |
|---|---|
| **在范围内** | 新平台：认证、档案、商品/支付、权益、Chat、订单、账户 |
| **不在范围内** | legacy naming / mingshu / hepan 及其 admin、报告、独立 chat、渲染器——**冻结不动** |
| **领域边界** | `liki-web` 后端=平台域（用户/产品/缴费/权益/档案存储/会话）；**命理计算=引擎/skill 域**，liki-web 不实现任何排盘编排 |

## 2. 核心概念模型

```
User（平台用户）1 ── N Profile（档案 = 一个人的用户信息，聚合根）
                        └── 出生信息 + 称呼等（纯输入，不存排盘结果）
                              └── 1 ── N Session（会话，绑定档案 + skill）
Product：liki-chat（唯一 SKU）→ 解锁 Chat 自由对话（Entitlement）
```

**一句话业务逻辑**：免费 = 建档案（保存用户信息）；付费 = 围绕档案的自由对话——agent 拿到档案的用户信息后**用 skill 工具自行排盘并解读**。

**排盘在哪算（三处，都不落库）**：
| 场景 | 计算方 | 说明 |
|---|---|---|
| 免费看盘（档案页预览） | 前端 TS 直连引擎 RPC | 临时展示，不进库 |
| 付费对话（解读） | agent（skill python 工具） | 用户信息注入 agent，agent 自行排盘解读 |
| 会话内追问新盘 | 前端 TS 直连引擎 RPC | 展示统一走 RPC；**agent 内部数据不与前端交互** |

## 3. 功能模块（6 个模块）

### M1 认证与账户
| 功能 | 规则 |
|---|---|
| 邮箱验证码登录 / 登出 | 已有，沿用 |
| 会话管理（当前会话、其他设备、revoke、revoke-all） | 后端已具备，前端补账户页 |
| 账户页 | 邮箱、登录态、会话列表、登出 |

### M2 档案管理（新域，免费）
| 功能 | 规则 |
|---|---|
| 档案列表 | 支持多档案（本人/家人/朋友），默认选中最近使用（`last_used_at`，服务端存） |
| 新建档案 | 登录即可；**免费**；受配额限制（默认 20） |
| 档案字段 | 称呼、姓名、性别、出生信息（公历/农历、出生时间、出生地）、备注、关系标签 |
| 档案详情 | 展示用户信息 + **排盘预览入口（前端直连引擎，不进库）** + 解读入口 |
| 编辑档案 | 改出生信息 → 更新；乐观锁（见 6.3） |
| 删除档案 | UI 软删（30 天恢复窗口） |
| 归属 | 严格 owner 校验（`user_id`） |

### M3 商品与支付
| 功能 | 规则 |
|---|---|
| 商品目录 | 唯一 SKU `liki-chat`，已有 |
| checkout / 支付返回 | 已有；支付返回后刷新权益 + 引导对话 |
| 币种 | 服务端按 geo 推导，前端只展示不决策 |

### M4 权益
| 功能 | 规则 |
|---|---|
| 购买解锁 Chat | `liki-chat` → 4 skill 全解锁，已有 |
| 权益校验 | 每次新建会话、发消息时后端校验；读取档案永不查权益 |
| 过期边界 | 权益过期 → 禁止新建会话与发消息；历史会话只读；档案仍可看 |

### M5 Chat（改造）
| 功能 | 规则 |
|---|---|
| 选档案 + 选 skill → 建会话 | `POST /api/chat/sessions` 带 `profile_id`；会话永久绑定一个档案 + 一个 skill |
| **agent 注入用户信息** | 会话上下文 = 档案身份 + 出生信息（**不含 pan**）；agent 用 skill 工具自行排盘 |
| 会话首条系统消息 | 系统写入"已载入档案信息"消息（含用户信息摘要） |
| 自由对话 | SSE 流式（thinking/phase/text-delta/done/error） |
| 盘面展示 | 前端 TS 直连引擎 RPC（与免费预览同路径）；**agent 只输出解读文字** |
| 定盘写回 | agent 用 `calibrate` 反推时辰后，经 `update_profile_birth_time` 工具走**网络调用**与前端相同的 `PATCH /api/profiles/{id}`（携带用户会话 cookie）——API 层防线（owner/乐观锁/校验/限流）全部生效；agent 是平台 API 客户端（同前端） |
| 会话管理 | 列表、恢复（URL 直达）、停止、刷新恢复 |
| 串行生成 | 同一会话同时只允许一个 generation（已有） |

### M6 订单
| 功能 | 规则 |
|---|---|
| 订单列表 | 已有；展示订单号/产品/金额/状态/时间 |

## 4. 档案字段（M2 展开）

| 字段 | 说明 | 必填 |
|---|---|---|
| `id` | 档案 ID | 系统 |
| `user_id` | 归属用户 | 系统 |
| `version` | 行版本（乐观锁，任何编辑 +1） | 系统 |
| `nickname` | 称呼（如"妈妈"） | ✅ |
| `name` | 姓名（姓 + 名） | 建议 |
| `gender` | 性别（male/female） | ✅ |
| `calendar` | `solar` / `lunar` | ✅ |
| `birth_date` | 出生日期（公历 `YYYY-MM-DD` / 农历 `YYYY-闰M-D`） | ✅ |
| `birth_time` | 出生时间（精确 `HH:mm` 或时辰） | 视 `birth_time_known` |
| `birth_time_known` | `known / approximate / unknown` | ✅ |
| `birth_place` | 出生地（城市，真太阳时修正用） | 可选 |
| `note` / `relation` | 备注 / 关系标签 | 可选 |
| `third_party_consent` | 为他人建档的同意标记 | 他人时必填 |
| `last_used_at` | 最近被会话使用时间 | 系统 |
| `deleted_at` | 软删除标记 | 系统 |
| `created_at / updated_at` | 时间戳 | 系统 |

**无 pan / pan_status / pan_version**：排盘结果不落库（见 §2）。

## 5. 真太阳时两路（前端直连引擎时遵守，agent 内同样语义）

| `birth_time_known` | 处理 |
|---|---|
| `known` + 出生地 | 路 A：`city.coords` → `tianwen.time` 校正后排盘 |
| `known` 无出生地 | 标注"未做真太阳时修正"，不静默按北京时间 |
| `approximate` | 路 B：按给定时辰直接排，不校正（防二次偏移） |
| `unknown` | 提示"不确定也能算，可对话定盘"；agent 用 `calibrate` 反推后经 `update_profile_birth_time` 工具写回档案（与 PATCH 同一更新路径） |

## 6. 数据实体与规则

```
users (id, email, status, locale, last_login_at, created_at)
profiles (id, user_id, version, nickname, name, gender, relation, calendar,
          birth_date, birth_time, birth_time_known, birth_place, note,
          third_party_consent, last_used_at, deleted_at, created_at, updated_at)
chat_sessions (+profile_id, skill, status, timestamps)
chat_messages (+kind: text|system_profile, content)
entitlements / chat_orders / products（已有）
```

### 6.1 会话绑定
- 会话创建时写入 `profile_id`，之后**不可更换档案**。
- 会话标题默认 `档案称呼 + skill 领域`（如"妈妈 · 八字紫微"）；标题是创建时快照。
- 创建会话时更新该档案 `last_used_at`；**系统写入首条 `system_profile` 消息**（用户信息摘要）。

### 6.2 快照语义
- 会话注入的是创建时的档案出生信息快照；修改档案不影响旧会话。
- 解读所用的盘由 agent 按该快照现算（skill 工具），不依赖档案存储。

### 6.3 并发控制
- `PATCH /api/profiles/{id}` 必须带 `expected_version`；不匹配 → `409 profile_conflict`。

### 6.4 删除档案
- UI 软删（30 天恢复窗口），关联会话保留但标注"档案已删除"。

### 6.5 隐私与合规（PII）
- 出生信息、姓名属敏感个人数据。
- 第三人同意：为他人建档时 `third_party_consent` 必填。
- 删除权：账号注销/合规请求 → 匿名化档案（PII 清空、置 `deleted_at`）。
- 日志不落出生信息；接口仅 owner 访问。

## 7. 端到端主流程

```
登录 → 工作台
 ├─ 无档案 → "新建档案"（免费）→ 填用户信息 → 保存
 │        → 可"免费预览排盘"（前端直连引擎，临时，不进库）
 ├─ 有档案 → 选档案 → 看用户信息 + 解读入口
 │        → 未购：解锁 Chat → checkout → 支付返回 → 权益刷新
 │        → 已购：选 skill → 建会话(绑定档案+系统消息) → agent 注入用户信息 → 自行排盘解读
 └─ 多档案 → 再建一个（算别人 = 多一个档案）
```

## 8. 后端契约建议（新增，不动已有契约）

```
POST   /api/profiles           建档（免费）→ 201 {profile}
GET    /api/profiles           → {profiles, next_cursor}（(updated_at,id) 游标，limit 默认 50）
GET    /api/profiles/{id}      详情
PATCH  /api/profiles/{id}      编辑（必带 expected_version → 409）
DELETE /api/profiles/{id}      UI 软删
POST   /api/chat/sessions      body 增加 profile_id → 校验 + 系统写入 system_profile 首条
GET    /api/chat/sessions       响应带 profile 摘要
SSE    仅 thinking/phase/text-delta/done/error；**agent 与前端不交换盘面数据**
```

错误码：`400 invalid_request` / `401 unauthorized` / `404 not_found` / `409 profile_conflict` / `422 validation_error` / `429 rate_limited`。

## 9. 非功能要求

- 安全：owner 校验全覆盖、HttpOnly cookie、CSRF、限流（建档/建会话/发消息）
- 防滥用：免费建档限流 + 配额（默认 20）；引擎 `/jsonrpc` 是既有公开端点，前端直连仅用于展示
- 并发一致性：行级乐观锁（`version`）
- 隐私合规：第三人同意、删除权、匿名化（见 6.5）
- 配置/灰度：配额、注入策略可配置（env / feature flag）
- i18n：暂中文，字段留 locale 位
- 可测试：档案/chat 各层单测 + E2E 全链路

## 10. 本期明确不做

- legacy 任何改动、admin 后台、报告产品、多语言
- pan 落库、pan_status、排盘 job（架构上永不做）
- agent 与前端不交换盘面数据（架构决定：agent 为解读黑盒，盘面展示走前端 RPC 直连）
- 定盘写回：agent 工具（`update_profile_birth_time`）仅允许更新 `birth_time` + `birth_time_known`，且只能作用于会话绑定档案

## 11. 验收标准

- 免费用户：建档保存用户信息，可"免费预览排盘"（前端直连引擎）；全程无购买阻断
- 已购用户：选档案 + skill → 会话首条系统消息 + agent 自行排盘解读，无需重复交代生辰
- 修改档案 → 乐观锁；旧会话快照不变
- 会话：SSE 流式、可停止、刷新恢复、不可换档案
- 4 个 skill 永不呈现为 4 个商业商品
- `liki-web` 后端无任何排盘计算代码