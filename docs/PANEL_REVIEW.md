# 专家组评审报告（Panel Review）

评审日期：2026-10-09
方式：4 位领域专家并行独立评审（互不可见）→ 组长交叉汇总去重
基线：master @ 778b9f2 后（含 aipay amount_minor、same-origin 单点化、死代码清扫等近期修复）

## 0. 专家组构成与统计

| 专家 | 领域 | P1 | P2 | P3 | 小计 |
| --- | --- | --- | --- | --- | --- |
| Planck | 安全与合规 | 1 | 8 | 5 | 14 |
| Turing | Go 后端与数据架构 | 4 | 8 | 2 | 14 |
| Wegener | 测试与交付（资金关键路径） | 5 | 8 | 4 | 17 |
| Kant | 前端（Vue/Astro/legacy） | 2 | 4 | 8 | 14 |
| **合计（去重前）** | | **12** | **28** | **19** | **59** |

**交叉验证亮点**：并发退款超退被 Planck/Turing/Wegener **三方独立发现**（置信度最高）；
erasure 编排非持久化被 Turing/Wegener 双方独立发现；checkout 幂等/取消冲突被
Turing/Wegener 双方发现。多位专家一致认为资金状态机的**并发与乱序保护**是当前
第一优先主题。

## 1. 已知基线（本次未重复上报的已修项）

same-origin 中间件单点化、全量错误 envelope、CSP 边缘归属、CI 65% 覆盖率门禁 +
coverage artifact、products 死表停止建表播种、aipay amount_minor、胶水收敛
httputil、一轮死代码清扫。

## 2. P1 汇总（去重后 7 项）

### P1-1 并发退款可超退（三方独立确认 🔒⚙️🧪）

`payment/store.go RecordRefund`：剩余额度检查在事务外读旧快照，事务内
`refunded_amount_minor = refunded_amount_minor + ?` 无上限断言；两个不同
provider refund id 并发均可通过旧快照检查 → 累计退款超过支付金额。
referral `applyCommissionRefund` 的 stale balance 盲写同类。
**修复**：校验移入事务，条件更新 `WHERE refunded_amount_minor + ? <= amount_minor`，
RowsAffected=0 回滚；补并发 overdraft 测试。（0.5–1 天）

### P1-2 checkout 无幂等键 + 跨库取消冲突（⚙️🧪）

每次 checkout 新建 order+intent，无 Idempotency-Key/活跃订单唯一约束；
StartCheckout 失败后无条件 CancelOrder——若 provider checkout 已建但本地
回写失败，webhook 迟到捕获成功后订单已是 canceled，权益不发放、履约进 failed。
前端 `buy()` 双击/重试可生成多笔可支付意图。
**修复**：幂等键或"同用户同产品活跃订单唯一"；provider 外呼成功后禁止本地
cancel；canceled/paid 冲突生成补偿任务。（2–4 天）

### P1-3 admin.user.manage 等价于超级管理员（🔒）

持有 `admin.user.manage` 者可创建任意邮箱管理员再授予 `admin.super`（禁止
改自己但可先建可控账号）。**修复**：授予/撤销 `admin.super` 拆独立 capability，
要求 actor 显式持有；可选双人复核。（0.5–1 天）

### P1-4 erasure 编排非持久化、无补偿（⚙️🧪）

跨 6 库串行删除，中途失败留部分状态；与 webhook/通知 worker 并发存在隐私窗口。
**修复**：durable erasure request/step 状态表 + per-domain 幂等补偿扫描，
handler 只 enqueue；补失败注入与并发 webhook 测试。（3–5 天）

### P1-5 前端登出状态清理不一致（🖥️）

AppLayout 登出 reset 不全（漏 profiles/commerce），AccountView 登出/擦除完全
不 reset；`likiChatDraft` 留 sessionStorage。换号登录可见前用户数据。
**修复**：统一 workspace 清理函数（abort stream + reset 全部 store + 清用户级
sessionStorage），两处登出都调用。（S）

### P1-6 聊天流式发送跨 session 写入（🖥️）

发送中切换 session/离开页面，旧闭包继续写共享 messages/activeSession；unmount
只 resetActive 不 stopGeneration。旧结果可能污染新 session。
**修复**：切换/卸载时 abort；completion 回调校验 generation token/session id。（M）

### P1-7 发布门槛未强制 E2E（⚙️）

`docker-publish` 不依赖 `e2e` job；`make gate` 不含 e2e/build/check-packaging，
与文档"pre-release gate"不符。**修复**：publish 增加 needs e2e；提供
`make release-gate` 统一入口。（小）

## 3. P2 汇总（去重后 20 项，按域分组）

### 资金/后台正确性

- refund intake `running` 状态无重启恢复（对比 fulfillment/intake 均有 recover）（🧪 0.5d）
- webhook 处理失败统一 400——provider 对 4xx 不重试，内部错误应 5xx（🧪 0.5–1d）
- MarkPaid/EnsureQueued/通知入队非事务；EnqueueOnce SELECT-then-INSERT TOCTOU（🧪 1–2d）
- 部分退款按 beneficiary share 舍入漂移，未断言 sum=AmountMinor（🧪 1d）
- 迁移 isApplied 事务外检查——多实例重复执行风险；无 checksum；referral 008/009
   legacy CHECK 缺 failed 状态的时序约束（🧪 2–3d，多实例风险待部署确认）
- 补偿扫描 `updated_at DESC LIMIT 200` 老积压饥饿 → 改 ASC/watermark（🧪 0.5d）
- 退款金额语义不一致：payment 允许 0=全额，referral intake 强制 >0（⚙️ 小，待验证 provider 行为）

### 可观测/运维

- 优雅关停不等待 payment/referral worker，DB 先于 goroutine 关闭 → errgroup（🧪 1–2d）
- partner 决策邮件无 durable outbox，重启/失败即丢通知（🧪 2d）
- CI job 无 timeout-minutes、顶层无 permissions: contents: read（⚙️ 小）
- CI Node cache 不含 web/app lock；Playwright 无缓存；artifact 无 retention-days（⚙️ 小）
- E2E 固定端口 + ready 误判边界 + dev-start `fuser -k` 盲杀（⚙️ 小–中）
- Docker 无 HEALTHCHECK；compose depends_on 仅 service_started；Caddy local 未 digest pin（⚙️ 小）
- Go/前端构建在镜像外，provenance 不完整（⚙️ 中）

### 安全（Planck 专项）

- aipay 非浏览器 402 流会被生产 same-origin 拒（bypass 仅 webhook）→ 显式 bypass
    或 agent 专用 token（🔒 0.25–0.5d，待验证边缘注入）
- payment return 的 status_token 整段 URL 进 slog.Info（🔒 0.25d）
- admin 审计表：未认证请求可无限插入 denied 记录，无采样/清理（🔒 0.25–0.5d）
- erasure 覆盖缺口三处：chat session title/profile_id 残留（🔒 0.25d）、
    feedback raw JSON 不在删除编排（🔒 1d）、admin_audit_logs 邮箱明文残留（🔒 0.5–1d）

### 前端

- loadMe 把 401/网络/5xx 全吞成未登录跳转，无区分提示（🖥️ M）
- BirthForm dirty 判断恒真（snapshot 结构不匹配）；Partner 面板直显 err.message
    绕过统一错误层；profiles loadList 失败清缓存 + cursor 未编码（🖥️ 各 S）

## 4. P3 汇总（22 项，摘要）

后端：schema 演进双轨（手写 DDL vs migrate runner，3–5d）；错误包装一致性；
chat message insert 与 session touch 无事务；aipay pending bill 无限增长无清理；
Xunhu webhook 无重放窗口（MD5 签名，幂等缓解）；same-origin 不校验 scheme/大小写；
MaxSessions 配置存在但未生效、过期 session 无限保留；pay_before 本地时区 vs UTC
字典序比较；metrics/feedback token 无失败限速；SECURITY.md 与 docs/SECURITY.md 关系未声明。

前端：免费排盘 RPC 未传 locale（引擎文案中文直出，待验证）；lang-switcher cookie
无 Secure + aria 硬编码；Axe E2E 缺 Partner×3/NotFound 页；404 数字对比度 1.4:1；
chat chunk（292KB raw）被 dashboard 依赖链拖入首屏；OrdersView row 类型弱化；
profiles 分页 UI 无 load more（>50 档案不可达）；admin.html 冻结面行为与 SPA 不一致
未声明；checkout_url 无前端 allowlist。

## 5. 做得好（各专家一致）

- **SQLite 基线与资金幂等基建**：WAL/FK/单连接/权限收紧统一出口；payment events
  四元组去重、refund 唯一键、durable queue、claim 事务 + 启动 recover（🔒⚙️🧪）。
- **安全基础扎实**：256-bit token 只存 SHA-256、bcrypt 验证码、keyed HMAC ip_hash、
  payout 加密 + reveal 审计、Dodo Standard Webhooks 时间戳校验、Xunhu 常数时间比较（🔒）。
- **前端/XSS/i18n 基础好**：唯一 v-html 集中且 DOMPurify allowlist；三语 605 key
  完全对齐；路由懒加载 + 按需引入 + Axe 基线（🖥️）。
- **供应链卫生**：actions/Caddy/skill bundle hash 校验、SBOM/provenance/cosign、
  无生产 secret 入库（⚙️）。

## 6. 汇总行动矩阵（建议迭代排序）

| 批次 | 内容 | 估算 |
| --- | --- | --- |
| 第一批（资金正确性） | P1-1 退款条件更新；P1-2 checkout 幂等+取消补偿；P2-1/3/4 refund recover、webhook 5xx、EnqueueOnce、分摊断言 | 5–8 天 |
| 第二批（隐私与提权） | P1-3 super capability 拆分；P1-4 erasure durable 化；P2-18 三处 erasure 覆盖缺口 | 6–9 天 |
| 第三批（前端正确性） | P1-5 登出清理统一；P1-6 流式 abort；P2-19/20 | 3–4 天 |
| 第四批（交付与卫生） | P1-7 release-gate；P2-8/10/11/12/13；P3 批量小项 | 3–5 天 |

## 7. 组长横切观察

- **发布前必查**：catalog 仍是测试价 ¥1/$1（`internal/liki/catalog.go` 注释
   Temporary test price）——不在任何待办系统里，上线前必须换。
- 本地 master 领先远端多个提交（含本轮全部评审修复），评审结论以推送后为准。
- P1 高度集中于一个主题：**资金状态机的并发与乱序保护**——现有串行测试全部
   通过但并发保护缺失，建议第一批修复时同步补 goroutine 级 race/overdraft 测试。
- 三位后端侧专家均独立给出"durable queue/outbox 方向正确、继续沿用"的结论，
   erasure 与 partner 邮件应复用该模式而非新造机制。

## 8. 修复进度（2026-10-09，同日启动）

实施前逐项验伪（读代码为证），7 个 P1 全部确认为真；两处子细节修正：

- `MarkPaid` 对 refunded 状态有保护（WHERE 限定 created/checkout_created），Turing 该子细节不成立；真实危害路径是 `capture` 忽略 MarkPaid 生效与否、`EnsureQueued` 无状态守卫——迟到支付在退款后仍触发履约流，核心成立。
- 额外发现报告未载细节：部分退款后收到 `Amount==0`（全额语义）会绕过剩余校验，`refunded > amount_minor`。已随 P1-1 一并修复。

| 批次 | 内容 | 状态 |
| --- | --- | --- |
| 1 | RecordRefund 事务内重读余额 + 条件 UPDATE + 0=补剩余；附超退回归测试 | ✅ |
| 2 | MarkPaidByIntent 放宽含 canceled（迟到 webhook 复活，不吞钱），附注释 | ✅ |
| 3 | admin.super 授予/撤销要求 actor 持有 super（新增 ErrSuperRequiresSuper + 403 映射）；SafetyRules 测试按新语义重写（含提权拒绝用例） | ✅ |
| 4 | erasure 覆盖：chat session title/profile_id 清除；admin_audit_logs 邮箱化名（account 编排新增 AdminAuditEraser，nil 容错兼容旧调用）；feedback 无用户关联 → 声明不适用（非 DSAR 工程） | ✅ |
| 5 | 前端：auth.logout 统一清理五 store + likiChatDraft；chat 切换会话即 abort + 完成回调 session 校验；commerce 补 reset | ✅ |
| 6 | CI：docker-publish needs +e2e；顶层 permissions + 各 job timeout；Node cache 双 lock | ✅ |
| 明确不做 | Idempotency-Key 全套、erasure 全新 outbox 表、DSAR 工具、kol/scout 路由重构 | 记录为已知取舍 |

验证：go test ./... 全过、golangci 0、vue-tsc + vitest 47/47、ESLint、markdownlint、actions SHA-pinned、YAML 解析通过。

## 9. P2 收尾批次（2026-10-09 第二轮）

- refund intake `running` 启动恢复（`recoverCommissionRefunds`，与 intake 对称）
- webhook 错误分级：验签失败 → `ErrWebhookVerification`（HTTP 400，provider 不重试）；DB/内部错误 → 500（可重试）；未知订单 400
- `EnqueueOnce` 单语句 `ON CONFLICT DO NOTHING`（依赖既有唯一索引，消除 TOCTOU）
- `RefundRecord` 携带实际入账金额：0-amount 观察者契约归一化（referral intake 不再因 0 失败）
- aipay `pay_before` 统一 UTC（消除本地时区字典序比较隐患）
- referral 补偿扫描 `DESC` → `ASC`（老积压不再饥饿）
- admin 未认证 deny 不再写审计表（存储耗尽放大消除；已认证 capability deny 仍审计）
- 前端：`loadMe` 区分 401/网络·5xx（后者登录页提示重试）；BirthForm dirty 快照同构修复；Partner 四视图 9 处错误统一 `errorMessage()`；profiles 列表失败保留 stale + cursor 归零 + URLSearchParams 编码

同步遗留说明：优雅关停已确认**根因更深**——`handleSignals` 从未 cancel root ctx，本次以 `workers.Wait()` + 显式 `cancel()` 修复。
