# 工业化加固计划（P0）

> 状态：P0 已实现；系统化修复批次完成，待生产验证  
> 范围：管理能力与审计、退款佣金闭环、提现最小闭环、可观测性、数据库字典  
> 原则：先文档、先测试；所有资金/权限状态变更必须幂等、可审计、可重放

---

## 一、目标

把分销和后台从“功能可用”推进到“可运营”：

1. 管理能力不再只依赖一个 `ADMIN_EMAIL`。
2. 管理员敏感操作有统一 capability 与 audit。
3. 全额退款作废未支付受益人佣金；部分退款按原始佣金比例扣减；已支付佣金进入财务处理流程。
4. 合作伙伴可以申请提现；管理员可审核、拒绝、标记打款。
5. payment / referral / payout 关键事件有 metrics。
6. SQLite 多库结构有数据字典。

---

## 二、边界

本阶段不做：

- 自动打款
- 银行/支付宝/ PayPal API 直连
- 复杂组织架构
- 多级分销
- 发票
- 跨币种结算

只做最小可信闭环。

---

## 三、Admin Capability / Audit

### 角色

内部角色：

```text
admin.super        超级管理员
admin.referral.ops 分销审核与运营
admin.finance      佣金 / 提现
admin.auditor      只读审计
```

对外仍不可见。

### Capability

```text
partner.read
partner.approve
partner.freeze
partner.unfreeze

commission.read
commission.void

payout.read
payout.request.review
payout.approve
payout.reject
payout.mark_paid

audit.read
```

### 存储位置

Admin capability 属于平台管理域，不放入 referral 业务域。

```text
platform.db
  admin_users
  admin_capabilities
  admin_audit_logs
```

统一 session 解析出 active user 后，由 admin authorizer 判断 capability。

### 状态

```sql
admin_users.status:
  active
  disabled

admin_capabilities.status:
  active
  revoked
```

### 初始化

部署配置 `ADMIN_EMAIL` 对应的第一个账号初始化为：

```text
admin.super
```

如果该邮箱还没有登录创建 user，则先保存 email；统一登录成功后再绑定 `user_id`。

### Audit

统一表：

```sql
admin_audit_logs (
  id,
  actor_user_id,
  actor_email,
  actor_capability,
  action,
  object_type,
  object_id,
  before,
  after,
  reason,
  request_id,
  ip_hash,
  user_agent,
  created_at
)
```

所有管理写操作必须写 audit。

---

## 四、退款 → 佣金作废

### 事件

新增统一支付事件：

```text
payment.refunded
```

Dodo 的 `refund.succeeded` 映射为该事件。  
Xunhu 如果后台/回调提供退款事件，也使用同一契约；当前未提供时，可由人工管理端触发。

### Runtime 规则

```text
收到 payment.refunded
→ 记录 payment_events
→ 在 payment_refunds 记录 provider refund ID 和金额
→ 累计退款未达原价时 intent 仍为 paid，已退金额可审计
→ 累计退款达到原价时 intent 变为 refunded
→ 只在 durable observer hand-off 成功后标记 refund notified
```

### Referral 规则

```text
sale refund → beneficiary entry net = gross - allocated refund
entry fully refunded / pending → voided
entry partially refunded / pending → 保持 pending，可提现净额减少
entry paid → 不自动改账，创建 finance clawback task
payout locked entry → 重算 payout amount；低于门槛则 reject 并释放
```

作废必须记录：

```text
reason = payment.refunded
provider
payment_intent_id
actor = system
```

---

## 五、提现最小闭环

### 用户状态机

```text
无请求
→ requested
→ approved
→ paid

任意审批前状态可 rejected
支付失败进入 failed
```

### 佣金锁定

申请提现时：

```text
1. 按 beneficiary role 查询 pending commission entries
2. 按 amount_minor - refunded_minor 计算可提现净额
3. 原子创建 payout
4. 将这些 beneficiary entry 的 payout_id 绑定 payout
5. KOL 和 Scout 的 entry 互相独立，pending 佣金不再重复计入可提现
```

拒绝提现时释放佣金。  
混合币种的 pending commission 不能合成一个 payout，必须明确返回错误。  
只有 approved payout 可以标记 paid；标记 paid 时同时将关联佣金置为 paid。

### API

用户侧：

```http
POST /api/referral/me/payouts
GET  /api/referral/me/payouts
```

管理侧：

```http
GET  /api/admin/referral/payouts
POST /api/admin/referral/payouts/{id}/approve
POST /api/admin/referral/payouts/{id}/reject
POST /api/admin/referral/payouts/{id}/mark-paid
```

---

## 六、可观测性

### Payment

```text
payment_intent_created_total
payment_paid_total
payment_refunded_total
payment_webhook_total{result}
payment_reconcile_total{result}
payment_fulfillment_total{status}
payment_notification_total{result}
```

Export:

```text
GET /api/internal/metrics?format=prometheus
```

### Runtime

```text
GET /api/health  # liveness
GET /api/ready   # database readiness; Docker healthcheck uses this
```

### Referral

```text
referral_commission_intake_total{status}
referral_commission_settled_total
referral_commission_retry_total
referral_commission_void_total
referral_commission_refund_intake_total{status}
referral_commission_refund_applied_total
referral_commission_refund_worker_total{result}
referral_payout_created_total
referral_payout_state_total{result}
```

### Admin

```text
admin_action_total{action,result}
admin_forbidden_total
```

---

## 七、测试清单

### Admin capability

- bootstrap `ADMIN_EMAIL` 后拥有 `admin.super`
- 非 admin session 被拒绝
- disabled admin 被拒绝
- 缺少 capability 被拒绝
- 敏感操作写入 audit
- audit 保存 actor email / user id / action / reason

### Refund

- pending commission full refund 后 voided
- partial refund 按 beneficiary entry 扣减 net amount
- repeated provider refund ID 幂等
- paid commission entry 不被自动 void，必须生成 pending finance task
- finance task resolve / reject 必须有 note 并写入 audit
- refund observer 失败时不标记 notified，webhook retry 可重放 hand-off
- 无 commission 时幂等成功
- `payment.refunded` 写入 payment_events

### Payout

- 无 pending commission 时不能创建
- 创建 payout 后关联 pending commissions
- 已绑定佣金不会进入第二次 payout
- reject 后释放佣金
- approve 只允许 requested / under_review
- mark paid 后 payout 和 commission 同步 paid
- 重复 mark paid 幂等
- payout account listing 默认 masked；明文 reveal 必须有 reason 且写入 audit
- 用户只能读取自己的 payout

### Observability

- refunded / payout / admin action 均产生 counter
- counter label 状态稳定，不含 PII

---

## 八、实施顺序

1. 文档与状态机定义
2. Admin capability / audit 测试与实现
3. Refund 事件测试与 referral void 实现
4. Payout store / service 测试
5. Payout user & admin API
6. Metrics 接入
7. 数据字典
8. 全量 `make gate`

## 九、独立 KOL / Scout 前端

- `/kol`：KOL Console，只允许 `role=kol`
- `/scout`：Scout Console，只允许 `role=scout`
- 两个页面互不链接、互不提及对方
- 后端 API 按 `/api/kol/*` 与 `/api/scout/*` 强隔离
- 前端隔离有 regression test

## 十、当前实施结果

- Admin capability / audit：已实现；freeze/unfreeze 已拆分
- Refund -> commission void：已实现
- Payout requested / approved / rejected / paid：已实现；金额调整可审计
- Paid commission refund clawback queue：已实现
- 多币种余额：已实现，不再跨币种相加
- Scout payout form：已实现
- KOL promoted Scout：历史收益可结算，但不再暴露 KOL console
- payout account encryption：已实现
- Payment refund metric：已实现
- Referral payout / settlement metrics：已实现
- Platform admin 与 referral 数据字典：已添加
