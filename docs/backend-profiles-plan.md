# 后端实施方案：档案域 + Chat 集成（平台域）

- 版本：v2.1（按最终模型：档案=用户信息，不存 pan；免费排盘预览=前端直连引擎）
- 关联：`docs/platform-fs.md`（功能规格 v1.4）、`docs/frontend/implementation-plan.md`（前端）
- 范围：`liki-web` 后端；**不含 `liki/engine`、不含 skill 契约、不含排盘计算**

## 0. 架构边界（原则）

| 域 | 归属 | 职责 |
|---|---|---|
| 平台域 | `liki-web` 后端 | 用户、产品、缴费、权益、订单、**档案存储（用户信息）**、会话编排 |
| 命理计算 | `liki/engine` + skill | 排盘 RPC（`bazi.chart` 等）、真太阳时、断语 |
| 免费排盘预览 | **前端 TS 直连引擎 RPC** | 临时展示，不进库（引流钩子，成本极低） |
| 付费解读排盘 | **agent（skill python 工具）** | 注入用户信息后自行排盘 |

**原则**：
- `liki-web` **不实现/不编排任何命理计算**（无 engine 客户端、无 paipan worker、无 pan 存储）。
- 引擎 RPC 契约只读消费；`skill-tools.json` / python `agent_cli.py` 不动。
- 档案只存用户信息（输入），**无 pan / pan_status / pan_version**。
- 免费预览与 agent 解读共用同一引擎，一致性靠"前端调用契约钉死"控制（见 §4）。

## 1. 目标（平台域交付）

1. **profiles 域**：CRUD + 乐观锁 + 软删 + cursor 分页 + 配额 + owner 校验（纯用户信息）
2. **Chat 集成**：会话绑定 `profile_id`、首条 `system_profile` 系统消息、档案用户信息注入 system prompt
3. 无排盘编排、无 pan 存储

## 2. 新增/修改清单

```
internal/profile/               新增（纯平台域）
├── types.go                    Profile（纯用户信息，无 pan）/ 入参校验
├── store.go                    SQLite CRUD + 乐观锁 + 软删 + cursor + 配额
└── http.go                     POST/GET/PATCH/DELETE 路由 + 限流
internal/chat/                  修改
├── store.go                    migration 002：sessions+profile_id、messages+kind
├── types.go                    Session.ProfileID / Message.Kind
├── service.go                  建档会话写 system_profile、档案用户信息注入 system prompt
└── http.go                     create session 接收 profile_id
cmd/liki/main.go                wiring：profile store + 路由
internal/http/server.go         register profile routes
```

**明确不做**：`internal/engine` 客户端、paipan job worker、pan 存储、任何排盘 RPC 编排。

## 3. 数据模型（platform DB，与 auth/chat 同库）

```
profiles (
  id TEXT PK, user_id TEXT NOT NULL,
  version INTEGER NOT NULL DEFAULT 1,          -- 行乐观锁，任何编辑 +1
  nickname TEXT NOT NULL, name TEXT, gender TEXT NOT NULL,
  relation TEXT, calendar TEXT NOT NULL,       -- solar | lunar
  birth_date TEXT NOT NULL, birth_time TEXT,
  birth_time_known TEXT NOT NULL,              -- known | approximate | unknown
  birth_place TEXT, note TEXT, third_party_consent INTEGER NOT NULL DEFAULT 0,
  last_used_at TEXT, deleted_at TEXT,
  created_at TEXT NOT NULL, updated_at TEXT NOT NULL
)
```
- **无 pan / pan_status / pan_version 列**（排盘不落库）。
- 索引：`(user_id, deleted_at, updated_at)`。

## 4. Chat 集成

- 会话创建：`POST /api/chat/sessions` body + `profile_id`；校验档案归属（owner）+ 未删除；不存在/越权 → `404`。
- 创建时**系统写入首条 `system_profile`** 消息（内容=档案用户信息摘要；无 pan）。
- 档案信息注入：`SendMessage` 拼 system prompt 时附加"档案身份 + 出生信息"，**不含 pan**；agent 用 skill 工具自行排盘。
- SSE：仅 thinking/phase/text-delta/done/error；**agent 与前端不交换盘面数据**（agent 为解读黑盒）。
- **agent = 平台 API 客户端（同前端）**：`update_profile_birth_time` 工具（liki-bazi scope 白名单）走网络调用与前端相同的 `GET/PATCH /api/profiles/{id}`，携带用户会话 cookie——API 层全部防线（owner/乐观锁/入参校验/限流）对 agent 同样生效，无 store 后门。约束：只改 `birth_time`+`birth_time_known`、只作用于会话绑定档案。

## 5. API 契约（新增）

```
POST   /api/profiles          建档（免费）→ 201 {profile}
GET    /api/profiles          → {profiles, next_cursor}（(updated_at,id) 游标，limit 默认 50）
GET    /api/profiles/{id}     详情
PATCH  /api/profiles/{id}     body 必带 expected_version → 409 profile_conflict
DELETE /api/profiles/{id}     软删（deleted_at，30 天窗口）
```

错误码：`400 invalid_request` / `401 unauthorized` / `404 not_found` / `409 profile_conflict` / `422 validation_error` / `429 rate_limited`。

## 6. 非功能

- owner 校验 SQL 层强制（`user_id`）；建档/编辑限流 + 单用户档案上限 20。
- 软删 30 天窗口（本期只置 `deleted_at`，物理清理下期）。
- 出生信息为敏感 PII：接口仅 owner 可访问；日志不落出生信息/pan。
- **配置**：`LIKI_SELF_URL`（agent 自调平台 API 的 base URL，须与监听地址一致，默认 `http://127.0.0.1:8080`）。

## 7. 测试

- `profile/store`：CRUD / 乐观锁 409 / 软删 / cursor / 配额
- `profile/http`：路由 + 校验 + 限流
- `chat`：profile_id 绑定、system_profile 首条、档案用户信息注入 system prompt、越权 404

## 8. 本期不做（明确）

- Go 端任何排盘编排（engine 客户端 / paipan worker）——**按架构原则永不进 liki-web**
- agent 与前端交换盘面数据（架构决定：盘面展示走前端 RPC 直连，agent 只输出文字）
- 定盘写回更宽的权限（工具仅 `birth_time`+`birth_time_known`，仅绑定档案）
- 合规强删 `/api/account/erasure`（本期只存字段）
- 软删 30 天物理清理定时器