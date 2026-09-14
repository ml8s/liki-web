# Platform Chat 架构与契约

Liki Chat 是统一 Chat 商业产品。它不依赖 legacy naming / mingshu / hepan 订单，也不把四个内部 skill 当作四个商业产品。

## 概念模型

| 层 | 概念 | 说明 |
|---|---|---|
| Product / SKU | `liki-chat` | 用户购买对象，购买后解锁全部 Chat 能力 |
| Skill | `liki-naming`、`liki-bazi`、`liki-divination`、`liki-fengshui` | 内部领域能力，不是商业 SKU |
| Runtime | unified chat | 会话、消息、SSE、Agent 编排 |
| Platform | auth / entitlement / commerce / payment | 用户、权益、订单和支付 |

legacy `naming`、`mingshu`、`hepan` 保持独立。它们不会被隐式转换成 `liki-chat` 权益。

## Runtime contract

1. 一个 session 永久绑定一个 skill。
2. 所有 API 都要求平台 session cookie。
3. 创建、读取 session，读取/写入 message 都要求有效 skill entitlement。
4. SQL 查询始终带 `user_id`，保证 session owner 校验。
5. 每个 skill 有独立 Runner、system prompt 和工具范围。
6. 工具访问是 deny-by-default；LLM 可见 schema 会先过滤，执行时再次校验。
7. 同一 session 的生成串行执行，避免并发写和 transcript 分叉。
8. session 创建和 message generation 有 rate limit。
9. Agent 只产出 transport-neutral 事件；HTTP handler 使用公共 `internal/sse` 输出。
10. Python 工具固定入口、固定函数名、JSON 输入输出，30 秒超时。
11. report generator 与 Chat 隔离，不会继承 Chat 工具范围。

## API

JSON API 使用统一 envelope：

```json
{"data": {}}
```

错误：

```json
{"error": {"code": "...", "message": "..."}}
```

message generation 使用 SSE，不用 JSON envelope。

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/api/me/entitlements` | 读取有效权益和 allowed skills |
| `GET` | `/api/chat/products` | 读取唯一的 Liki Chat 产品与当前购买状态 |
| `POST` | `/api/chat/products/{id}/checkout` | 创建 checkout |
| `GET` | `/api/chat/orders` | 读取当前用户 Chat 订单 |
| `GET` | `/api/chat/sessions` | 读取当前用户 session 列表 |
| `POST` | `/api/chat/sessions` | 创建 session |
| `GET` | `/api/chat/sessions/{id}` | 读取单个 session |
| `GET` | `/api/chat/sessions/{id}/messages` | 读取消息 |
| `POST` | `/api/chat/sessions/{id}/messages` | SSE 流式生成 |

## SSE protocol

返回头：

```text
Content-Type: text/event-stream
Cache-Control: no-cache
```

每个事件是一行 JSON data：

```text
data: {"type":"thinking"}
data: {"type":"phase","content":"正在计算命理数据…"}
data: {"type":"thinking-delta","content":"..."}
data: {"type":"text-delta","content":"..."}
data: {"type":"done","data":{...}}
data: {"type":"error","content":"服务暂时不可用，请稍后重试"}
```

`done.data` 包含：

```json
{
  "session": {},
  "active_skill": "liki-naming",
  "user_message": {},
  "assistant_message": {}
}
```

规则：

- user message 先持久化；
- assistant message 只在 Agent 成功产出完整回复后持久化；
- 失败不会写入半截 assistant 回复；
- `done` 事件是前端替换临时 UI 消息的信号；
- 客户端断开会取消派生 context，Agent 停止执行。

## Storage

| Table | Purpose |
|---|---|
| `products` | Chat 商业目录，当前只有 `liki-chat` |
| `entitlements` | 用户权益、skill 集合、有效期、来源 |
| `chat_orders` | Liki Chat 订单与 payment intent 绑定 |
| `unified_chat_sessions` | 统一 Chat 会话 |
| `unified_chat_messages` | 统一 Chat 消息 |

时间戳使用 RFC 3339 固定纳秒精度，保证快速写入时仍可稳定排序。

## Entitlement mapping

| Product / SKU | Skills |
|---|---|
| `liki-chat` | `liki-naming`, `liki-bazi`, `liki-divination`, `liki-fengshui` |

授权来源必须显式，例如：

```text
commerce_order
payment_intent
admin_grant
legacy_migration
```

`GrantForProduct` 按 `(source, source_id, product_id)` 幂等。  
重复购买同一产品会从当前到期时间向后顺延。

## Payment lifecycle

```text
登录用户
-> POST /api/chat/products/liki-chat/checkout
-> 创建 chat_orders(pending)
-> 创建 payment_intents(user_id)
-> provider checkout
-> webhook / return reconciliation
-> payment_intents(paid)
-> durable fulfillment task
-> chat_orders(paid)
-> GrantForProduct("liki-chat")
-> entitlements 生效
```

payment return 会主动向 provider 查询一次，避免浏览器返回早于 webhook 时被误判为 unpaid。  
legacy payment capture 不会写入 unified Chat entitlement。

## Tool isolation

每个 skill 只能使用 `SkillToolScopes[skill]` 内定义的工具。

规则：

- schema 阶段先过滤；
- execution 阶段再次检查；
- `read_file` 只能读取当前 skill 目录；
- 跨 skill 工具既不可见，也不可调用；
- report prompt 与 `webapp/**` 不进入 Chat。
