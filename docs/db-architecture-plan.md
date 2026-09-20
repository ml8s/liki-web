# 数据库架构重构计划

> 状态：代码重构完成；生产迁移前必须在 staging 备份库上完整演练
> 更新：2026-09-20

---

## 一、目标

将当前 `naming.db` 单文件大杂烩拆分为按领域隔离的数据库，实现：

- **删产品不影响平台**：删掉 liki.db 不会丢用户数据
- **加产品不碰平台**：新产品只需要新建一个 .db 文件
- **正交性**：跨域用 TEXT 软引用，同域用 FK
- **线上数据零丢失**：提供安全迁移脚本

## 二、当前问题

1. naming.db 包含 21 张表，来自 8 个不同领域（严重）
2. entitlements 表每次启动 DROP TABLE 重建，线上会丢授权数据（严重）
3. entitlements.user_id REFERENCES users(id)，跨域外键
4. 文件名叫 naming.db，实际是全平台的库

## 三、目标结构

```
${LIKI_DATA_DIR}/               ← 单一环境变量，默认 /var/lib/liki
│
├── platform.db                 ← 身份 + 支付运行时（删任何产品不影响）
│   ├── users
│   ├── auth_identities         ─FK→ users
│   ├── auth_codes
│   ├── auth_sessions           ─FK→ users
│   ├── auth_audit_logs
│   ├── payment_intents
│   ├── payment_notifications
│   ├── fulfillment_tasks
│   ├── payment_events
│   └── schema_migrations
│
├── profiles.db                 ← 用户档案（属于用户，不属于任何产品）
│   ├── profiles
│   └── schema_migrations
│
├── referral.db                 ← 分销系统（业务功能，可独立开关）
│   ├── referrers
│   ├── referral_attributions
│   ├── referral_commissions
│   ├── referral_payouts
│   └── schema_migrations
│
├── feedback.db                 ← 反馈
│   ├── feedback
│   └── schema_migrations
│
├── liki.db                     ← liki 产品
│   ├── chat_orders
│   ├── products                ← Liki Chat 商品目录
│   ├── entitlements            （user_id: TEXT，无 FK）
│   ├── unified_chat_sessions
│   ├── unified_chat_messages   ─FK→ sessions
│   └── schema_migrations
│
├── naming.db                   ← naming 产品
│   ├── orders
│   ├── chat_messages
│   ├── verification_codes
│   └── schema_migrations
│
├── mingshu.db                  ← mingshu 产品（不变）
│   ├── orders
│   ├── reports                 ─FK→ orders
│   └── schema_migrations
│
├── hepan.db                    ← hepan 产品（不变）
│   ├── orders
│   ├── reports                 ─FK→ orders
│   └── schema_migrations
│
└── aipay.db                    ← aipay 产品（用支付宝收款的产品）
    ├── aipay_orders
    └── schema_migrations
```

## 四、设计规则

### 外键规则

- 同库内：可以用 REFERENCES + ON DELETE CASCADE
- 跨库：必须用 TEXT，无 REFERENCES

### ID 规则

- 所有主键用 UUID TEXT（统一，不用自增）
- 金额用最小单位（分）INTEGER
- 时间用 RFC3339 TEXT

### 状态机规则

- 所有状态字段用 CHECK 约束
- 软删除用 deleted_at TEXT（可空），不物理删用户数据

### SQLite 连接规则

所有库统一通过 `internal/sqliteutil.Open` 打开：

```text
journal_mode=WAL
busy_timeout=5000
foreign_keys=ON
max_open_conns=1
```

不引入 GORM。数据库结构由各模块自己的 forward-only migration 管理；
跨库一致性通过 fulfillment task、幂等 retry 和状态机实现。

## 五、代码包结构调整

```
internal/
├── auth/            ← 不变，读写 platform.db
├── payment/         ← 不变，读写 platform.db
├── profile/         ← 不变，读写 profiles.db
├── feedback/        ← 不变，读写 feedback.db
├── aipay/           ← 不变，读写 aipay.db
│
├── liki/            ← 合并 commerce + entitlement + chat
│   ├── product.go       产品定义（Product, Skill）
│   ├── entitlement.go   EntitlementStore
│   ├── catalog.go       定价 + 货币检测
│   ├── order.go         OrderStore
│   ├── chat.go          ChatStore
│   ├── service.go       聊天编排
│   ├── flow.go          Fulfiller (payment.ProductFlow)
│   ├── handler.go       统一 HTTP Handler
│   └── profile_tool.go  Server-side tool
│
├── naming/          ← 不变，读写 naming.db
├── mingshu/         ← 不变，读写 mingshu.db
├── hepan/           ← 不变，读写 hepan.db
│
├── referral/        ← 新增（分销系统就绪）
│   ├── store.go
│   ├── handler.go
│   └── types.go
│
├── agent/           ← 不变（LLM 基础设施）
├── llm/             ← 不变
├── sse/             ← 不变
├── httputil/        ← 不变
├── migrate/         ← 不变
├── sqliteutil/      ← SQLite 连接策略
└── metrics/         ← 不变

删除：
├── internal/commerce/   → 合并入 internal/liki/
├── internal/entitlement/ → 合并入 internal/liki/
└── internal/chat/       → 合并入 internal/liki/
```

## 六、关键修复

### entitlements DROP TABLE bug（必须在做拆库之前修）

当前代码每次启动都执行：

```sql
DROP TABLE IF EXISTS entitlements;  -- 线上已有的授权全部丢失！
DROP TABLE IF EXISTS products;
CREATE TABLE products (...);
CREATE TABLE entitlements (...);
```

修复：改为 CREATE TABLE IF NOT EXISTS，只在首次创建时 seed。
同时去掉 user_id REFERENCES users(id)（跨域 FK 改为 TEXT）。

## 七、数据迁移方案

### 前提

线上已经有订单数据在 naming.db 中，迁移必须保证零丢失。

### 迁移脚本 scripts/migrate-db.sh

步骤：

1. 停止 `liki-web`，并以 `LIKI_MIGRATION_CONFIRM_STOPPED=1` 显式确认
2. 对源库执行 `PRAGMA integrity_check`
3. 用 `VACUUM INTO` 生成一致性冷备份
4. 写入一个全新的空 staging 目录，不允许覆盖已有目标库
5. 从旧 naming.db 按表搬到 staging 下的新库
6. 校验每个表的行数和内容 digest
7. 校验每库 `integrity_check` / `foreign_key_check`
8. 校验 `entitlements.user_id`、`chat_orders.user_id` 都能命中 platform users
9. 用 staging 目录启动新版本，人工 smoke test
10. 确认无误后由人工执行生产目录切换；源库和冷备份继续保留

### 表迁移映射

| 旧表（naming.db） | 新库 | 新表 |
|---|---|---|
| users | platform.db | users |
| auth_identities | platform.db | auth_identities |
| auth_codes | platform.db | auth_codes |
| auth_sessions | platform.db | auth_sessions |
| auth_audit_logs | platform.db | auth_audit_logs |
| profiles | profiles.db | profiles |
| payment_intents | platform.db | payment_intents |
| payment_notifications | platform.db | payment_notifications |
| fulfillment_tasks | platform.db | fulfillment_tasks |
| payment_events | platform.db | payment_events |
| feedback | feedback.db | feedback |
| products | liki.db | products |
| chat_orders | liki.db | chat_orders |
| entitlements | liki.db | entitlements（去 FK） |
| unified_chat_sessions | liki.db | unified_chat_sessions |
| unified_chat_messages | liki.db | unified_chat_messages |
| aipay_orders | aipay.db | aipay_orders |
| orders（naming 的） | naming.db | orders |
| chat_messages（naming 的） | naming.db | chat_messages |
| verification_codes | naming.db | verification_codes |

mingshu.db 和 hepan.db 不需要迁移（已经是独立的）。

### 校验清单

- 每个表行数 = 旧表行数
- users 表的 email 全部匹配
- entitlements 的 user_id 全部能在 users 表找到
- chat_orders 的 user_id 全部能在 users 表找到
- 登录一个已有用户，能正常获取 session
- 打开一个已有的 chat session，消息完整
- 查看一个已支付的 naming 订单

## 八、实施顺序

### Phase 1：修复 entitlements DROP TABLE bug（紧急）

- 修改 entitlement Migrate，改为 CREATE TABLE IF NOT EXISTS + 条件 seed（已完成）
- 去掉跨域 FK（已完成）
- 部署，验证线上数据不受影响（待生产验证）

### Phase 2：合并 commerce + entitlement + chat → liki 包

- 创建 internal/liki/ 包（已完成）
- 更新 main.go、server.go、account.go 的引用（已完成）
- 删除 internal/commerce/、internal/entitlement/、internal/chat/（已完成）
- 跑通所有测试（已完成）

### Phase 3：拆库 + 迁移脚本

- main.go 改用 LIKI_DATA_DIR 统一管理数据目录（已完成）
- 所有库统一走 internal/sqliteutil（已完成）
- 编写并加固 scripts/migrate-db.sh（已完成）
- 在 staging 备份库上验证迁移（待执行）
- 线上执行迁移（待执行）

### Phase 3.5：生产切换

- 用生产备份副本执行 `scripts/migrate-db.sh`
- 用 `LIKI_DATA_DIR=<staging>` 启动应用
- 验证老用户登录、老订单、老 session、老 entitlement
- 记录迁移前后行数和 digest
- 由人工完成生产目录切换
- 保留旧库和冷备份直到上线观察期结束

### Phase 4：实现分销系统

- 创建 internal/referral/ 包
- 实现 referrers / attributions / commissions / payouts
- 对接 fulfillment hook（付款成功 → 计算佣金）
- Partner Portal 前端
- 详见 docs/referral-system-review.md（评审报告 + 必须修复项）

## 九、环境变量变更

### 之前

```
DB_PATH=/var/lib/liki/naming.db
MINGSHU_DB_PATH=/var/lib/liki/mingshu.db
HEPAN_DB_PATH=/var/lib/liki/hepan.db
FEEDBACK_DB_PATH=/var/lib/liki/feedback.db
```

### 之后

```
LIKI_DATA_DIR=/var/lib/liki
```

容器挂载点的常见配置是：

```
LIKI_DATA_DIR=/data
```

宿主机目录或 named volume 挂到该路径即可。所有 `.db` 文件都在这个目录下，一个变量管全部。

## 十、风险与缓解

| 风险 | 缓解 |
|------|------|
| 迁移丢数据 | 迁移前备份 + 行数校验 + 回滚方案 |
| 跨库查询变复杂 | 仅 admin 后台需要跨产品查询，非热路径 |
| SQLite 无法跨库事务 | 现有代码已是最终一致，不引入新问题 |
| 多文件 WAL 管理 | SQLite WAL 按文件独立，无互相影响 |

## 十一、不在本次范围

- PostgreSQL 迁移（当前用户量下 SQLite 足够）
- 双写过渡（直接停机迁移，维护窗口 < 5 分钟）
- 优惠券系统
- 发票系统
