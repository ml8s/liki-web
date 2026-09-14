# Liki App 前端架构

`/app` 是登录后的 Vue 3 SPA，负责任务流、状态呈现和交互；命理规则、权益、订单和权限判定以后端为事实源。

## 技术栈

- Vue 3 + TypeScript
- Vite
- Vue Router
- Pinia
- Element Plus
- Vitest / Vue Test Utils
- Playwright

## 分层

```text
src/views/       页面编排
src/components/  通用 UI 与业务组件
src/stores/      server state 与跨页面状态
src/api/         typed API client 与 SSE 解析
src/config/      页面配置，例如内部 skill 展示项
src/router/      路由和 auth guard
```

### 页面层

页面只做组合和生命周期管理，不直接实现复杂 UI 细节。

```text
LoginView           邮箱验证码登录
AppLayout           登录后 App Shell
DashboardView       工作台入口
ChatListView        Liki Chat 产品、内部 skill、session 列表
ChatSessionView     SSE 会话
OrdersView          Chat 订单
AccountView         账户与登录态
ReportsView         预留报告入口
```

### 组件层

```text
components/app/
├── AppPageHeader.vue
├── AppAlert.vue
├── AppEmpty.vue
├── AppSkeleton.vue
└── AppErrorState.vue

components/commerce/
└── ProductPurchaseCard.vue

components/chat/
├── SkillCard.vue
├── SessionList.vue
├── ChatThread.vue
├── ChatMessageItem.vue
├── ChatComposer.vue
└── ChatStreamStatus.vue
```

页面状态矩阵必须覆盖 loading、empty、success、error、unauthorized、forbidden、expired entitlement。

## Server state

当前使用 Pinia 管理：

```text
auth            当前用户、登录态
entitlements    allowed skills 和权益列表
chat            sessions、active session、messages、streaming 状态
```

规则：

1. 后端是身份、权益、订单、消息和权限的事实源。
2. 页面不要直接散落 `fetch`，统一走 `src/api`。
3. API client 不做权限决策，只转换契约和错误。
4. HttpOnly cookie 之外，前端不得持久化 session token。
5. SSE 只是传输；最终状态必须来自 `done` 或后端查询。

## Chat streaming

发送消息使用 `POST /api/chat/sessions/:id/messages`，响应是 SSE。

事件：

```text
thinking
phase
thinking-delta
text-delta
done
error
```

前端行为：

1. 发送时创建临时 user/assistant placeholder。
2. `text-delta` 实时追加。
3. `thinking` / `phase` 更新状态行。
4. `done` 用后端持久化的 user/assistant/session 替换 placeholder。
5. `error` 显示错误并重新拉取权威消息。
6. 用户停止后重新拉取权威消息。
7. 只允许同一 session 同时存在一个 generation。

SSE 解析集中在 `src/api/sse.ts`。

## 页面设计流程

新增页面或重构页面时，先阅读 [`page-design-workflow.md`](page-design-workflow.md)：先定义用户任务、路由、状态矩阵和 API contract，再拆组件、接 store/API、补测试和文档。

## 测试

| 层 | 工具 | 覆盖 |
|---|---|---|
| API client | Vitest | envelope、错误、SSE parser |
| Store | Pinia + Vitest | 权益、session、SSE done/error/stop |
| Component | Vue Test Utils | product card、composer、message |
| E2E | Playwright | 登录、受保护路由、session 恢复、支付返回 |

运行：

```bash
cd web/app
npm run test
npm run build
```
