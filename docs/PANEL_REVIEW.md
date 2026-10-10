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

### 8.1 第二、三轮（同日）：P2 尾项与 B 批全部完成

| 项 | 状态 |
| --- | --- |
| durable erasure（B1 最小化） | ✅ `erasure_requests` 表 + Handler.runErasureSteps 单一实现 + worker 续跑；优雅关停根因（handleSignals 不 cancel root ctx）一并修复（workers.Wait + 显式 cancel） |
| 错误包装约定（B2 窄版） | ✅ CONTRIBUTING 落约定；新 erasure worker 全 wrapped；全仓 blanket 维持不做 |
| chat chunk 动态 import（B3） | ✅ 附 stopGeneration 信号竞态修复（await 前捕获 signal） |
| aipay pending bill 清理（B4） | ✅ 挂 6h retention |
| Axe 扩 4 页 + profiles load more | ✅ |
| P2 尾项（refund recover、webhook 5xx、EnqueueOnce、ASC 扫描、aipay UTC、admin deny、status_token、aipay bypass、前端 4 项） | ✅ 前批已完成 |

**CI 验证迭代记录（2026-10-10）**：新 Axe 网在 CI 首跑即抓到 9 处真实对比度
违规（EP 次级灰 2.8/3.07:1）。修复经三轮收敛：组件选择器 → 变量级提升
（`--el-text-color-secondary`）→ **ID 特异性（`#app` 前缀，终版）**——EP 按需
注入样式后置于 app.css，类特异性与顺序博弈不可靠，ID 特异性与加载顺序无关。
同批把 playwright 工件上传改为 always()（失败即有 DOM 快照可终审）。

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

## 10. 尾批清零（2026-10-09 第三轮）——P2 归零

- 迁移并发：applied 检查移入迁移事务内——并发 migrator 输家在 (scope,id)
  主键上 fail-closed，不再双跑；checksum 记为已知取舍（forward-only、单实例部署）
- feedback：legacy 回填错误不再吞掉（fresh 表按列存在性跳过）；hash 内容去重
  升级为 schema 级（去重 DELETE + `WHERE hash != ''` 部分唯一索引）；store 方法
  ctx 化记为已知取舍（本地单文件 SQLite）
- aipay 402 流：same-origin 显式 bypass（Payment-Proof 即凭证，CSRF 不适用；
  限流保留）——SKILL.md 契约与实现对齐
- status_token 不再进结构化日志（只记 path；1 小时 capability 不泄漏）
- partner 决策邮件：`context.WithoutCancel` 硬化（不随请求取消）；完整 outbox
  维持已知取舍（低频 admin 动作）
- **验伪否决一项**："分摊舍入漂移"的前提不成立——entries 只覆盖 beneficiary
  份额，部分退款的其余部分归买家（sale 级计数跟踪），`sum(deltas) !=
  refund 总额` 是正确语义；首轮实施过 exact-sum 断言后被既有测试抓到回归，
  已撤销并在代码内注明理由

至本轮结束：**P2 20/20 闭环**（含 3 项验伪否决/取舍）；P3 22 项为择期
结构性项，未在本轮范围。

## 11. 未完成项逐项裁决（2026-10-09 终版，全部经代码核验）

裁决口径：**修问题/优化的做；新功能审慎，仅"补齐既有契约的缺口"才做**。

### 11.1 做清单（21 项，3 个提交批次）

### A 批 — 快赢（9 项）

| # | 项 | 裁决理由 |
| --- | --- | --- |
| 1 | chat AppendMessage 双语句包事务 | 排序一致性缺陷，tx 包裹即可 |
| 2 | same-origin 补 scheme 校验 + host 小写 | 现接受 `http://liki.hk`，真实松弛 |
| 3 | `MaxSessions` 生效 + 过期/撤销 session 30 天清除 | 死配置 + PII 无限保留，双重缺陷 |
| 4 | metrics / feedback GET·PATCH 套限流 | 限流基建已有，接线即可 |
| 5 | SECURITY.md ↔ docs 关系声明 | 文档治理 |
| 6 | 404 数字装饰化（aria-hidden） | 实测对比度 1.4:1 |
| 7 | checkout_url 前端 allowlist | 开放重定向纵深，~10 行 |
| 8 | lang-switcher cookie `Secure` | 与 SPA setAppLocale 对齐 |
| 9 | OrdersView 统一 typed row helper | 唯一类型边界弱化 |

### B 批 — 本质修复（6 项）

| # | 项 | 裁决理由 |
| --- | --- | --- |
| 10 | durable erasure 编排（最小化：enqueue-first + 剩余步骤续跑，无多步状态机） | P1-4 本体，PIPL 合规闭环 |
| 11 | 错误包装窄版：worker/编排边界 `domain: operation: %w` | 排障定位；不做全仓 132 处 blanket |
| 12 | chat store 对 `@/api/chat` 改动态 import | 实测 chat chunk 295KB（最大 chunk）被 dashboard 依赖链拖入首屏 |
| 13 | aipay pending bill 过期清理（挂既有 6h retention） | 表体积无界增长 |
| 14 | Axe 扫 Partner×3 + NotFound（既有 spec 扩路由） | 给既有资金页面补无障碍回归网 |
| 15 | profiles load more（store append + 按钮） | 后端分页契约已存在，>50 不可达是可达性缺陷 |

### P3 小项 — 并入 A/B 批顺带做（6 项，其中 4 项前批已完成）

| # | 项 | 裁决 |
| --- | --- | --- |
| 16 | SECURITY.md 独立条目（同 #5） | 做 |
| 17 | aipay `pay_before` UTC | ✅ 已做（前批） |
| 18 | admin 未认证 deny 不落审计 | ✅ 已做（前批） |
| 19 | refund observer 0-amount 契约归一 | ✅ 已做（前批） |
| 20 | EnqueueOnce ON CONFLICT | ✅ 已落地（2026-10-10 复核发现前批仅改了提交信息未改代码——本轮真正落地并移除孤儿 `exists()`） |
| 21 | referral 补偿扫描 ASC | ✅ 已做（前批） |

### 11.2 不做清单（10 项，含理由）

| 项 | 裁决 | 理由 |
| --- | --- | --- |
| 引擎 `safety_advisory` locale | 本仓不做 | 已核验：字段来自 liki 确定性引擎 RPC，中文文案引擎域所有；加 locale 参数属 liki 仓契约变更，已记录跨仓需求 |
| 分摊 exact-sum 断言 | 否决 | entries 仅覆盖 beneficiary 份额，买家自留部分走 sale 级计数；既有测试已抓到该回归 |
| `MarkPaid` refunded 状态保护 | 非缺陷 | WHERE 已限定 created/checkout_created；真实缺口在 capture/EnsureQueued，已修 |
| kol/scout 路由树收敛 | 不做 | 结构性重构，收益低于回归风险 |
| schema 演进双轨统一 | 不做 | 行为等价、纯风格，3–5 天高风险低收益 |
| 迁移 checksum | 不做 | 事务内检查已 fail-closed；forward-only + 单实例部署 |
| feedback store 全量 ctx 化 | 不做 | 本地单文件 SQLite，无超时传播需求 |
| partner 邮件完整 outbox | 不做 | 低频 admin 动作；WithoutCancel 硬化 + Warn 日志已够 |
| erasure 全新多步状态机 | 不做（被 #10 最小化替代） | enqueue-first + 幂等续跑已满足合规 |
| Idempotency-Key 全套 | 不做 | checkout 幂等以"活跃订单复用 + 迟到捕获复活"最小闭环实现 |
| Axe 全站扩展到 40+ 路由 | 只做 4 页 | 既有基线 13 页已覆盖主路径，避免测试维护面膨胀 |
| lang-switcher 完整键盘菜单管理 | 冻结面不做 | legacy 冻结面，仅补 cookie Secure |

### 11.3 执行批次

- 提交一：A 批 9 项 + P3 顺带（16/17/18/19 已在前批完成的标注）
- 提交二：#10 durable erasure + #12 动态 import + #13 aipay 清理
- 提交三：#14 Axe 扩页 + #15 load more + 文档收口
- 每批全量验证（go test ./...、golangci、vue-tsc、vitest、eslint、markdownlint）

## 13.1 死代码清扫 · 第二轮（2026-10-10，代码演进后复扫）

方法同 13 节（导出零引用 + `golangci --tests=false` 抓 test-only 存活）。

本轮清除：

| 死代码 | 位置 | 死因 |
| --- | --- | --- |
| `langToLocale` + locale.go 整文件 + 其测试 | internal/http | products/胶水收敛后全仓零引用（含测试外无调用方） |
| `NotificationStore.exists` | internal/payment | EnqueueOnce 改单语句后孤儿化（本轮清扫才发现前批该修复未落地，见 8.1 更正） |

核验为存活（保留）：`canonicalHost`、`absoluteNotificationURL`、`formatMinorAmount`、
`actorFromRequest`、`decodeAPI`（测试 helper）、`createKOLPartner`/`commission`
（refund 测试经真实 store 方法构造状态，属正当 fixture，记录不删）。

前端未使用导出复扫：零。

## 14. D 类挂起项终审（2026-10-10）——验伪否决，无需改动

**引擎 `safety_advisory` locale**（原 D 类：en 用户疑见中文警示）：

三重证据否决：

1. liki **Go 引擎全仓从不返回** `safety_advisory`（唯一命中是
   skill_docs_contract_test 的字段清单字符串，非 RPC 响应）；
2. 该字段实为 counsel **chat-agent 工具路径**的
   `divination_safety.assess()` 输出（`liuyao_ask`/`qimen_ask` 等
   `*_ask` 工具），语言由 agent 对话自行处理，与前端免费排盘 RPC 无关；
3. 前端 `safety_advisory?: string` 为防御性可选类型，
   `v-if="result.safety_advisory"` 永不触发。

结论：不存在报告所担心的问题，两个仓库均无需改动。前端类型保留
（面向未来 counsel 契约的防御性声明）。

## 17. 专项评审：库 / 不造轮子 / 公开协议 / 一致性（2026-10-10 终审）

### 17.1 能用库用库——零重复轮子

逐项核对全部自研代码，**每一处手写都属于三类必须手写之一**：

1. **域算法**：八字/六爻/紫微计算、黄历、divination_safety——领域核心，
   无库可用；
2. **供应商协议适配**：XunHu MD5 签名 + 私有表单协议（无 Go SDK）；
   Alipay A2M（RSA2 协议）；Dodo 走官方 `dodopayments-go` SDK ✓；
3. **有界自研且已文档化**：capability token（比 JWT 更小的有界设计）、
   `internal/migrate`（60 行 forward-only runner，替代 golang-migrate
   全家桶）、`internal/money`（严格 2dp 解析，替代 decimal 度依赖）。

其余全部用库/标准设施：bcrypt、HMAC-SHA256、x/time rate、ozzo、
modernc sqlite、Standard Webhooks、resend、Intl.NumberFormat、
DOMPurify、markdown-it、vue-i18n。**零重复轮子确认**。

### 17.2 公开协议 vs 私有协议

对外协议全部公开标准：

| 面 | 协议 |
| --- | --- |
| 引擎公共计算 | JSON-RPC 2.0 |
| Dodo 回调 | Standard Webhooks 规范 |
| 前端↔BFF 流式 | SSE + AG-UI（公开 SDK） |
| 数据格式 | JSON（snake_case 全对齐，61 处抽查）+ RFC 3339 + ISO 4217 minor units |

私有定义仅 2 处且均不对外：capability token 内部格式（HMAC 有界
capability，比 JWT 更小、无 claims 泄漏、常数时间校验；本轮已完成
HKDF 密钥分离加固）；响应信封 `{data}/{error}`（行业惯例非私有协议，
契约写于 API.md）。

### 17.3 一致性终审

| 维度 | 结果 |
| --- | --- |
| JSON tag | snake_case 全对齐（61 处抽查零偏差） |
| slog 键 | `err` ×77 主导，实体键（user_id/intent_id/session_id）成对出现 |
| 前端网络层 | 零裸 fetch——全部经统一 api 层（错误/信封/重定向一处处理） |
| | 前端 `console.error` ×4 均为有意诊断（带组件前缀+全局兜底），非调试残留 |
| 前端 `as unknown as` | ×2（EP slot/引擎响应类型窄化，非安全路径，P3 可选收窄） |
| TODO/FIXME | 前后端均零 |
| `JWTSecret()` 全局读 env | 全局状态模式的小瑕疵：每次调用读 env，且密钥注入非显式。pragmatic 可接受；记录为风格项不 churn |

### 17.4 本轮修复

- capability MAC 密钥分离：stdlib `crypto/hkdf` 派生（15.2），附
  回归测试（原始密钥伪造 MAC 必须被拒）——已落地推送。

## 15. 专项评审：库 / 标准 / 一致性（2026-10-10）

### 15.1 库 vs 手写

依赖面精简且全部正当：uuid / bcrypt（验证码）/ HMAC-SHA256（token）/
golang-jwt（平台会话）/ x/time rate（限流）/ ozzo（校验）/ modernc sqlite
（纯 Go 驱动）/ standard-webhooks（Dodo）/ dodopayments SDK / resend /
ag-ui SDK / x/crypto。手写部分逐项核对均为"必须手写"：域引擎算法、
供应商协议适配（XunHu MD5 / Alipay A2M）、边界胶水。

发现：**金额解析/格式化共 5 套实现**——
`xunhu.parseMinorUnits`（>2 位小数静默截断）、`aipay.parseAmountMinor`
（>2 位拒绝，严格）、`xunhu.formatMinorUnits`（无符号十进制）、
`payment.formatMinorAmount`（币种符号，邮件用）、前端 `formatMinorUnit`
（Intl locale-aware，浏览器用）。

裁决：三种**展示**格式各有正当表面（供应商协议/邮件/浏览器），不收敛；
两个**解析**语义分歧（截断 vs 拒绝）真实存在但输入域均受协议约束
（供应商回调固定 2 位小数）。处置：收敛为 `internal/money` 严格解析器，
列入"下次触碰时"清单（P3，不立即做）。

### 15.2 标准

对齐项：JSON-RPC 2.0（引擎公共面）/ Standard Webhooks 规范（Dodo）/
RFC 3339 时间戳 / BCP 47 locale / ISO 4217 minor-units 金额 / WCAG
（axe 基线）/ `de facto` 信封（{data}/{error}）。

私有定义仅 2 处且均合理：capability token 自有格式（比 JWT 更小、无
claims 泄漏、action+subject 绑定、常数时间校验、TTL 有界，已文档化）；
响应信封（行业惯例非标准）。

**修复一项密钥分离反模式**：capability token 的 MAC 原与平台 JWT 共用
同一 `JWT_SECRET` 原始密钥。已改为 stdlib `crypto/hkdf` 派生独立 MAC
密钥（info 域隔离，零新依赖），附密钥分离回归测试（原始密钥伪造 MAC
必须被拒）。

### 15.3 一致性

- 时间精度：RFC3339（秒）×55 处 vs RFC3339Nano ×5 处。chat 已用固定
  9 位填充自解"剥零破坏字典序"问题（注释在案）；auth erasure 写 Nano
  进秒精度列——列内无字符串比较依赖，影响≈0，记录不churn。
- 错误响应/限流桶模式/构造器模式：全 API 统一（前轮已收敛）。
- 前端 `formatMinorUnit`（Intl）与后端展示格式双轨：UI locale-aware vs
  邮件简式，各自正确。

### 15.4 行动项

| 优先级 | 事项 | 工作量 |
| --- | --- | --- |
| ✅ 已做 | capability MAC 密钥分离（stdlib HKDF）+ 回归测试 | 0.5h |
| P3 | 金额解析收敛 `internal/money`（统一 strict 语义），下次触碰时 | 0.5 天 |
| 记录 | 时间精度策略（Nano 列内不混比较依赖） | — |

## 16. 专项评审：最佳工程实践 / Go 惯用法 / 前端标准实践（2026-10-10）

### 16.1 Go 语言标准实践

用官方 `modernize` 分析器（gopls modernize）全仓扫描 + 自动修复：

| 发现 | 处置 |
| --- | --- |
| goroutine 创建可用 Go 1.25 `WaitGroup.Go` ×8 | ✅ 已应用（本次新增的 worker 均命中） |
| 循环内成员检查可用 `slices.Contains` ×2 | ✅ 已应用 |
| `strings.Split` 可用 `strings.SplitSeq`（Go 1.24 零分配）×3 | ✅ 已应用 |
| 循环内 `string +=` 可用 `strings.Builder` ×1 | ✅ 已应用 |
| `payment/route.go`、`profile/store_test.go` 微现代化 | ✅ 已应用 |

复检 **0 违规**。其余标准实践核验通过：`interface{}` ×0（全 `any`）、
`sort.Slice` ×0（无手写排序）、`time.After` 循环 ×0、错误包装 `%w`
103:1、结构化日志零残留、`-race` 常态、Context 全链传播、密钥分离
（15.2 已修）。

### 16.2 前端标准实践

核验通过：`tsconfig strict: true`；script setup / Composition API；
路由级懒加载 + Element Plus 按需；ESLint flat + `vue/` 规则集；唯一
`v-html` 集中消毒（13 节）；i18n 605 key 三语对齐；Axe 无障碍基线 +
partner/not-found 扩页（本轮新增）。

### 16.3 工程实践

核验通过：Conventional commits 全程；pre-commit（vet+lint+测试）与
pre-push（make check）双层钩子；CI check/test/e2e 三 job + SHA-pinned
actions + 65% 覆盖率门禁 + coverage artifact。

**可选未采纳（记录决策）**：CODEOWNERS / PR template / dependabot——
单人维护 + 主动 SHA-pin + 人工验证的供应链策略下，自动化依赖机器人
与该策略冲突；单维护者仓库 CODEOWNERS/PR template 价值有限。若团队
扩大再启用。

## 18. 专项评审：两处私有定义的标准对标（2026-10-10）

### 18.1 capability token

| 对标 | 契合度 |
| --- | --- |
| JWT（RFC 7519） | 实用标准；仓库已有 golang-jwt；aud/sub/exp 恰好映射三要素 |
| PASETO v4 | 只允许安全算法的 JWT 替代；但为"签名 bearer"，非"验证方注入上下文的 MAC capability"，映射不完美 |
| Macaroons/Biscuit | 概念最贴合（可衰减 capability），生态小众 |

现状（HMAC 密封 + 验证方注入 action/subject + 常数时间比较 + TTL）在安全
模型上优于把 claims 装进 token（更小、不泄漏资源 ID、无法跨路由重放）。
**裁决：保留**。迁移触发：第二个服务需验证同一 capability，或 token 需
携带更多 claims → 迁 PASETO v4 / pinned-JWT。

### 18.2 响应信封 {data}/{error}

错误半边有 IETF 正牌标准——**RFC 9457 Problem Details**
（`application/problem+json`；`type/title/status/detail/instance`）。
映射：`code → type`、`message → detail`、HTTP status → `status`。
成功半边 `{data}` 为 Google 风格 de facto，JSON:API 同构。

**裁决：保留现状 + 记录映射**。错误响应可随外部集成需求平滑迁移到
`application/problem+json`（前端 `ApiError.code ← type` 一行改动）；
映射已写入 API.md。

### 18.3 标准对照补全（含标准组织/规范号）

| 私有定义 | 对标标准 | 规范出处 |
| --- | --- | --- |
| 错误信封 `{code,message}` | RFC 9457 Problem Details（`application/problem+json`） | IETF RFC 9457（2024，取代 RFC 7807） |
| 成功信封 `{data}` | JSON:API v1.1 `{data,links,meta}`（`application/vnd.api+json`）；Google AIP 风格 | 社区规范 |
| capability token | RFC 7519 JWT + RFC 9068 access token profile（`aud/sub/exp` 恰好映射 action/subject/exp）；备选 PASETO v4（非 IETF） | IETF / de facto |
| 可衰减 capability 语义 | Macaroons / Biscuit（IETF 个人草案） | 概念对标，生态未成熟 |
| sender-constrained 模型 | RFC 9449 DPoP | IETF（非本仓需求） |

**迁移触发条件**（记录在案，当前均未触发）：

1. capability 出现第二验证服务或需要更多 claims → RFC 9068 / PASETO v4；
2. 出现第三方 API 消费者 → 错误响应迁 `application/problem+json`。
