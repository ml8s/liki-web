# 系统化修复计划

> 状态：已实施，等待生产验证  
> 原则：不生产任何提交/推送；先定数据契约，再改领域服务，最后改前端与运维。  
> 目标：修复评审中的 P0，并完成影响资金正确性、权限边界、运营闭环的 P1。

## 0. 修复边界

本轮优先处理：

1. 退款 intake 的 `failed` 状态不可能落库。
2. KOL / Scout 多币种金额不能再跨币种相加。
3. Scout console 必须能申请提现。
4. KOL 提升为 Scout 后，历史 KOL 收益仍可结算。
5. payout 金额从“可变单值”改为“原始申请 + 调整后金额 + 调整流水”。
6. 退款分摊落到显式 ledger，不再只靠隐式累计字段。
7. payout 收款账号不再明文暴露/明文落库。
8. freeze / unfreeze 权限拆分。
9. 管理端资金操作必须可确认、可填原因、可审计。
10. 后台错误必须映射到精确 HTTP 语义。
11. 已支付佣金退款必须生成显式财务追回任务。
12. payout 明文账号必须显式 reveal，并写审计。
13. liveness 与 database readiness 分离，Docker 使用 /api/ready。
14. metrics 支持 Prometheus text exposition。
15. console 逻辑外置为共享核心 + 页面模块，消除大段内联脚本。
16. console 路径使用更严格 CSP，静态 JS 不再 immutable。
17. account erasure 覆盖 auth / profile / chat / commerce / payment / product domains。
18. Referral 删除采用财务保留策略：active payout 阻断；其他场景冻结合作关系并生成财务任务。

暂不在本轮实现自动打款、跨币种结算、复杂组织树和完整 CRM。

## 1. 数据契约

### partner_roles

`referrers.role` 只保留 primary role，用于运营展示；可结算能力由 `partner_roles` 表决定。

```text
partner_roles(
  user_id,
  role,
  status,
  started_at,
  ended_at,
  created_at,
  updated_at,
  PRIMARY KEY(user_id, role)
)
```

一个用户可以有历史 KOL 能力和当前 Scout 能力。

### referral_payouts

新增：

```text
beneficiary_role
requested_amount_minor
```

`amount_minor` 是当前可支付金额；`requested_amount_minor` 是用户申请时的金额，永不覆盖。

### referral_payout_adjustments

```text
id, payout_id, before_minor, after_minor, reason, actor, created_at
```

退款导致 payout 金额变化时必须写流水。

### commission_entry_refunds

```text
id, entry_id, payment_refund_id, amount_minor, created_at
```

每笔 provider refund 在每个 commission entry 上的分摊都可追溯。

### commission_refund_intents

状态恢复为完整状态机：

```text
queued / running / succeeded / failed / skipped
```

`failed` 表示不可自动重试的账务错误，必须保留 last_error。

## 2. API 契约

金额汇总不得返回一个跨币种标量。

KOL / Scout overview 使用：

```json
{
  "balances": [
    {"currency":"CNY","amount_minor":5000},
    {"currency":"USD","amount_minor":1000}
  ]
}
```

佣金额外返回 `balances`。payout API 返回受益人角色、原始申请金额、当前金额。

## 3. 前端契约

- 所有金额渲染必须带后端返回 currency。
- overview 显示每币种余额。
- Scout 与 KOL 都有 payout form。
- admin 破坏性 / 资金操作使用 dialog，不再使用 `prompt`/`confirm`。
- admin HTML escaping 必须覆盖 `"。
- admin 提现审批必须有 reason；标记打款必须有 payment reference。

## 4. 权限契约

`partner.freeze` 只能冻结；`partner.unfreeze` 只能解冻。super admin 继续拥有全部能力。

## 5. 实施顺序

1. 数据库迁移与 migration fixture tests。
2. Store / service 语义改造。
3. API handler 错误映射。
4. KOL / Scout / Admin 前端改造。
5. capability 分权与 admin console。
6. refund ledger / payout adjustment 回归测试。
7. 多币种、角色迁移、退款幂等、payout 状态机 race tests。
8. `make check`、`make test-go`、`make build-web`、`make build-go`。
9. 全部通过后再更新数据库与工业加固文档。

## 6. 验收标准

- fresh DB 与 legacy fixture DB 迁移均通过。
- CNY + USD 混合余额不再相加。
- KOL promoted Scout 后仍能结算 pending KOL entry。
- Scout 可完成 payout request。
- 重复 provider refund ID 不重复扣账。
- payout 原始金额不被退款覆盖。
- 每次金额调整有 audit/adjustment。
- freeze 与 unfreeze 由不同 capability 控制。
- `make check`、`make test-go`、`make build-web`、`make build-go` 全绿。
