# Liki Web 本地项目评审

> 评审日期：2026-09-26。  
> 评审对象：本仓当前工作区，基线提交 `47e4fda`，包含尚未提交的架构收敛改动。  
> 结论只覆盖本仓；外部仓只评审本仓对其的契约和使用方式。

## 1. 结论摘要

本仓当前整体工程质量较好：模块边界清楚，Liki Chat 已收敛为唯一商业产品，
naming / mingshu / hepan 的独立购买与激活入口已改为 fail closed；认证、支付、
退款、履约、权益、审计和账号删除的主链路有较完整的测试覆盖。

当前工作区还有一批重要收敛改动：

- Liki Chat 全量走 `liki-agents` 的 AG-UI 路径；
- 删除进程内 Liki Chat runner；
- profile 出生事实通过受控 `forwardedProps` 交给远端运行时；
- skill bundle 改为消费 `liki` 的 immutable release asset；
- CI 不再 checkout 或重建 `liki` 源码仓。

方向正确，且本地验证通过。但在合并/发布前还有一个 CI 工作流阻断项：

```yaml
e2e:
  needs: [check, test, e2e]
```

`e2e` job 不能依赖自己。应改为：

```yaml
  needs: [check, test]
```

## 2. 评审范围与基线

| 项目 | 结果 |
| --- | --- |
| Git 基线 | `47e4fda`，`origin/master` |
| 工作区 | 包含未提交的 AG-UI runtime、skill bundle、CI 和本地开发链路收敛改动 |
| Go 包 | 29 |
| Go 文件 | 204 |
| Web 文件 | 187 |
| Python 文件 | 9 |
| 外部产品 | Liki Chat / `liki-chat` 是唯一商业产品 |
| retired identifiers | `naming`、`mingshu`、`hepan` |

## 3. 架构现状

### 3.1 运行形态

```text
Browser / Astro / Vue App
  → Caddy
  → liki-web BFF
    → SQLite bounded contexts
    → XunHuPay / Dodo / Alipay AI Pay
    → Engine JSON-RPC
    → liki-agents AG-UI
```

`liki-web` 负责产品体验、平台身份、档案、订单、支付、权益、会话持久化、
账号删除、referral 和管理后台。它不负责确定性命理计算，也不实现通用
Agent runtime。

### 3.2 Liki Chat 执行链路

当前工作区已经把 Liki Chat 的执行路径收敛为：

```text
Browser AG-UI
  → /api/chat/sessions/{id}/ag-ui
  → liki-web 校验身份、权益、会话 ownership、限流
  → 读取可信持久化历史
  → AG-UI 调用 liki-agents
  → liki-web 持久化 RUN_FINISHED.result.answer
```

`internal/liki.Service` 现在只有一个 runner：

```go
func (s *Service) SetAgentRunner(runner Runner)
```

进程内 runner 已删除。profile-bound session 与普通 session 都走远端
`liki-agents`。

### 3.3 Profile 数据边界

`AgentProfileContext` 只包含远端排盘所需的最小出生事实：

- calendar；
- gender；
- birth date；
- birth time；
- birth time known；
- birth place。

它不传 user id、email、note、third-party consent、profile ID 或数据库 ID。
这个边界设计是正确的。

### 3.4 Skill 产物供应链

当前工作区引入了 `prepare-skills` composite action：

```text
GitHub release asset
  → liki-web-skill-bundle.tar.gz
  → 校验外部 SHA256
  → 解包并校验内嵌 liki.tar.gz digest
  → 校验 discovery index / optional manifest
  → 放入 web/skills 与 web/.well-known/agent-skills
```

必需输入：

```text
LIKI_SKILLS_RELEASE_TAG
LIKI_SKILLS_BUNDLE_SHA256
```

这个设计消除了“CI checkout 并重建 sibling `liki` 仓”的可重现性风险。
内容更新必须先发 `liki` release，再构建本仓镜像。

## 4. 产品收敛状态

| 场景 / 标识 | 新商业销售 | 公开页面 | 兼容行为 |
| --- | --- | --- | --- |
| `liki-chat` | 是 | Vue App | 唯一商业产品 |
| `naming` | 否，`410 product_retired` | CTA 到 App | 历史报告 / chat 只读兼容 |
| `mingshu` | 否，`410 product_retired` | CTA 到 App | 历史报告只读 |
| `hepan` | 否，`410 product_retired` | CTA 到 App | 历史报告只读 |

已确认：

- `/api/naming/orders` 返回 `410 product_retired`；
- `/api/naming/verification-codes/redeem` 返回 `410 product_retired`；
- `/api/naming/auth/login` 不再用 email 建立订单会话，返回 `410`；
- `/api/naming/orders/select` 返回 `410`；
- `/api/mingshu/orders` 返回 `410`；
- `/api/hepan/orders` 返回 `410`；
- 管理端创建独立订单、生成激活码也返回 `410`；
- 历史 webhook / reconciliation / fulfillment flow 保留，只用于存量处理。

## 5. 验证证据

本次评审在本地工作区执行并全部通过：

```text
make check
make test-go
make test-js
make test-python
make build
make check-packaging
```

其中：

```text
Go:      ./... race tests passed
Legacy:  5 files / 55 tests passed
Vue App: 17 files / 47 tests passed
Python:  18 helper tests passed
```

新增的 `web/e2e/journeys/live-system.spec.js` 是 gated full-system journey。
它默认 skip，必须显式提供：

```text
LIVE_SYSTEM_E2E=1
LIKI_E2E_SESSION_VALUE
```

用于验证真实登录态、profile 创建、会话创建、AG-UI run、前端流式输出和
assistant message 持久化。

还执行：

```text
docker compose config --quiet
```

需要提供本地测试变量：

```text
LIKI_ENGINE_IMAGE
LIKI_AGENTS_INTERNAL_TOKEN
```

Compose 配置校验通过。

## 6. 风险与问题

### 6.1 Blocker：e2e job 自依赖

位置：

```text
.github/workflows/ci.yml
```

当前：

```yaml
e2e:
  needs: [check, test, e2e]
```

问题：

GitHub Actions job 不能依赖自身。这会导致工作流无法按预期运行，或在后续
平台校验中直接失败。

修复：

```yaml
e2e:
  needs: [check, test]
```

### 6.2 P1：账号删除未清理普通聊天正文

位置：

```text
internal/liki/chat.go
```

当前删除逻辑只清空：

```sql
kind = system_profile
```

但普通 `user` 与 `assistant` message 也可能包含姓名、出生信息、家庭关系、
健康、感情、事业等个人上下文。文档口径是“chat-embedded personal context”
应匿名化，当前实现只覆盖嵌入 profile 的结构化消息。

建议：

1. 删除该用户全部 chat message content；
2. 或至少同时清空 `text` kind 的 user / assistant content；
3. 保留计数、时间、session 结构供审计；
4. 增加普通聊天正文删除测试。

### 6.3 P1：SQLite 文件与目录权限未强制收敛

`sqliteutil.Open` 只在目录不存在时创建 `0700` 目录。目录已存在时不会纠正权限。
SQLite 文件仍受进程 umask 控制。

本地实际状态显示存在：

```text
0755 data/
0644 data/*.db
```

这些库包含用户、订单、聊天、报告等敏感数据。

建议：

1. 启动时强制已有 data dir 为 `0700`；
2. DB / WAL / SHM 文件强制 `0600`；
3. 增加权限测试；
4. 生产部署明确拒绝过宽权限。

### 6.4 P2：chat session lock 存在删除竞态

位置：

```text
internal/liki/service.go
```

当前模式：

```go
m.Lock()
return func() {
    m.Unlock()
    s.sessionLocks.Delete(sessionID)
}
```

可能发生：

1. A 持有 mutex M；
2. B 拿到 M 并等待；
3. A unlock；
4. A 删除 map entry；
5. C 创建 M2；
6. B 持有 M，C 持有 M2；
7. 同一 session 出现并发写入窗口。

建议：

1. 简单方案：不删除 session mutex；
2. 或改用 DB 级 session claim；
3. 或实现引用计数，只在无等待者时清理。

### 6.5 P2：旧产品 drain 路径仍运行 LLM / Generator

新销售已经 fail closed，但当前启动仍会构造：

- naming LLM agent；
- report LLM agent；
- mingshu generator；
- hepan generator；
- retired payment flows。

这对存量 webhook / reconciliation 是必要的，但边界应更明确：

1. 代码注释和配置命名应表达“drain-only compatibility”；
2. 只处理已有 pending / paid historical intent；
3. 不允许重新创建 checkout；
4. 未来可增加 drain cutoff；
5. 逐步删除 naming legacy chat runner。

### 6.6 P2：支付 webhook 未单独限流

```text
POST /api/payments/webhook
```

有签名验证，但没有独立限流。公网攻击者可以反复触发 body 读取、验签和
provider 查询。

建议：

1. 增加按 IP / provider 的宽松限流；
2. 保留 provider retry 余量；
3. 对验签失败单独计数；
4. 对连续失败告警。

### 6.7 P2：多副本仍不可安全启用

当前仍有多处进程内状态：

- in-memory rate limiter；
- in-memory session mutex；
- payment / referral workers；
- SQLite bounded contexts；
- startup recovery；
- process-local fulfillment claim。

当前部署必须保持单副本。未来需要水平扩容前，必须先设计：

1. shared task ownership；
2. transactional worker claim；
3. shared / external rate limiter；
4. SQLite 迁移策略；
5. 显式 `LIKI_WEB_TOPOLOGY=single|multi` guard。

不要仅通过增加 replica 数量扩容。

### 6.8 P2：文档口径仍有少量不一致

新增 `docs/SYSTEM.md` 大方向正确，但存在几处应继续收敛：

1. “确定性产品直接调 engine JSON-RPC” 与 retired 产品口径不完全一致；
   这些现在是历史兼容 / drain 行为；
2. `LIKI_LLM_*` 注释仍称 in-process report/chat runner；
   chat 已走 `liki-agents`，只有 retired report / naming compatibility 仍使用；
3. `.env.example` 中 `DEV_MODE=1` 作为默认值对生产复制不友好；
4. docs 中应明确 naming / mingshu / hepan 不是当前商业产品。

建议把 `docs/SYSTEM.md` 与 `LEGACY_PRODUCTS.md`、`ARCHITECTURE.md` 的措辞统一。

### 6.9 P3：JSON decode 可继续统一

多数 handler 使用 `json.Decoder.Decode`，部分路径没有显式拒绝 trailing JSON，
也没有统一 `DisallowUnknownFields`。

建议增加统一 helper：

```text
strict JSON decode
single value only
optional unknown-field rejection
stable 400/422 mapping
```

至少覆盖所有写操作和 admin API。

### 6.10 P2：HTTP 安全头责任与 CSP 强度需要文档化

应用层 `SecurityHeaders` 只设置：

```text
X-Content-Type-Options
X-Frame-Options
```

CSP、Referrer Policy、Permissions Policy、COOP / CORP 主要依赖 Caddy。这个分层
可以接受，但必须在架构文档中明确责任归属，避免后续维护者误以为应用层缺失、
或边缘与应用重复配置导致规则漂移。

本地 `dev/Caddyfile.local` 的 CSP 当前包含：

```text
script-src 'unsafe-inline' 'unsafe-eval'
style-src 'unsafe-inline'
```

Astro 静态页面和 Vue 生产构建通常不需要 `unsafe-eval`。建议：

1. 在架构文档中增加 security header ownership 表；
2. 生产 CSP 移除 `unsafe-eval`；
3. 评估内联脚本 hash / nonce 后再移除 `unsafe-inline`；
4. E2E 中加入 CSP violation 观察。

### 6.11 P3：前端 lint 与测试工具仍有漂移

`web/js` 有 ESLint，但 Vue App 没有 ESLint flat config；`vue-tsc` 只能覆盖类型
问题，不能覆盖 Vue 生态 lint 规则。

另外，root web 工程与 `web/app` 的 Vitest 主版本不一致。建议：

1. 给 `web/app` 增加 `eslint.config.js` 与 `eslint-plugin-vue`；
2. 将 Vue App lint 纳入 `make lint-js`；
3. 统一 Vitest 主版本；
4. 评估 npm workspaces 收敛双 package 工程。

## 7. 做得好的部分

### 7.1 产品收敛方向正确

退役产品的新建入口已经 fail closed，管理端也不能再创建独立订单或激活码。
历史记录仍可审计。

### 7.2 Liki Chat 边界清晰

`liki-web` 保留身份、权益、档案、持久化和支付；`liki-agents` 只负责执行。
AG-UI 是唯一 chat 通道，没有额外私有执行 API。

### 7.3 认证主线扎实

Email code 登录具备：

- random code；
- bcrypt hash；
- token hash 存储；
- session 管理；
- HttpOnly / SameSite cookie；
- IP hash；
- auth audit。

legacy naming 的 email-only 登录已移除。

### 7.4 支付设计较完整

服务端拥有价格、currency、provider routing、webhook 验签、金额 / 币种 /
provider 一致性检查、事件去重、durable fulfillment、notification 和 refund
clawback。

### 7.5 Admin 有能力模型和审计

管理后台不是简单 email allowlist，而是 capability-based authorization，并记录
privileged action audit。

### 7.6 前端 Markdown 处理安全

Chat Markdown 使用：

```text
markdown-it html:false
DOMPurify
受限 tag / attr
noopener noreferrer
```

### 7.7 Skill 供应链改进明显

当前工作区使用 immutable release bundle，外层 SHA256、内层 archive digest、
discovery index 和 optional manifest 都会校验。相比 checkout sibling source 是
显著进步。

## 8. 建议执行顺序

| 优先级 | 动作 | 目标 |
| --- | --- | --- |
| 0 | 修复 CI e2e self-dependency | 工作流可运行 |
| 1 | 补齐普通 chat content 的删除/匿名化 | 隐私合规闭环 |
| 1 | 强制 SQLite 目录和文件权限 | 数据最小暴露 |
| 2 | 修复 session lock 删除竞态 | 防止并发写 |
| 2 | 给 webhook 增加独立限流 | 降低滥用成本 |
| 2 | 明确 retired flow 的 drain-only 边界 | 减少长期维护面 |
| 2 | 文档化 security header ownership 并收敛 CSP | 保持纵深防御不漂移 |
| 3 | 统一 strict JSON decode | API 一致性 |
| 3 | 增加 explicit topology guard | 防止误水平扩容 |
| 3 | 统一 `docs/SYSTEM.md` / env / architecture 文案 | 减少理解偏差 |
| 3 | 为 Vue App 补 ESLint 并统一 Vitest | 前端质量门完整 |

## 9. 总体判断

本仓不是简单 CRUD 站点，支付、权益、履约、审计、认证和 Agent 边界都有清晰
设计。当前工作区的 Liki Chat 单产品收敛和 skill bundle 供应链改造方向正确。

建议在解决 CI self-dependency 后合并当前收敛改动；随后优先处理隐私删除闭环、
SQLite 权限和 session lock 竞态。当前阶段继续保持单副本部署。
