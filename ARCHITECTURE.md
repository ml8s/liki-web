# Architecture

`liki-web` is the product-facing web/BFF service for Liki. It owns product state and commercial workflows. It does not own deterministic domain computation, generic agent execution, or production deployment.

## Runtime modules

| Module | Responsibility |
| --- | --- |
| `cmd/liki-web` | Service entrypoint |
| `internal/http` | HTTP route composition, middleware, security headers, rate limiting |
| `internal/auth` | email-code identity, sessions, verification codes |
| `internal/profile` | birth profile aggregate |
| `internal/payment` | provider-neutral payment runtime |
| `internal/liki` | Liki Chat product, order, entitlement, and chat domain |
| `internal/account` | account erasure orchestration |
| `internal/referral` | KOL/Scout applications, attribution, commission, and payout domain |
| `internal/admin` | admin operations and audit |
| `internal/agent` | AG-UI event contract shared with the liki-agents runtime |
| `internal/adminauth` | admin console session gate |
| `internal/aipay` | Alipay A2M gateway, sandbox proof-of-payment records |
| `internal/dodo` | Dodo payment gateway client |
| `internal/xunhu` | XunHuPay gateway client |
| `internal/email` | Resend email client |
| `internal/feedback` | feedback lifecycle store and API |
| `internal/httputil` | shared HTTP primitives: client IP, forwarded headers, JWT validation, response envelope, decode, SSE |
| `internal/i18n` | supported language tag constants |
| `internal/llm` | chat message model shared by protocol surfaces |
| `internal/metrics` | in-memory lifecycle counters and the internal metrics endpoint |
| `internal/migrate` | forward-only SQL migration runner for embedded SQLite |
| `internal/sqliteutil` | standard SQLite connection policy for every database file |

## Frontend modules

| Path | Role |
| --- | --- |
| `web/src` | Astro marketing and static locale pages |
| `web/app` | Vue 3 authenticated product SPA with zh-Hans/zh-Hant/en i18n |
| `web/js` | static assets for marketing pages (report demo renderers) |

New product functionality belongs in `web/app` and its backing Go APIs.

## External boundaries

| Repository / service | Responsibility |
| --- | --- |
| `liki` | deterministic engine, skills, Engine MCP, Counsel MCP source |
| `liki-agents` | generic agent runtime, AG-UI/A2A adapters, execution audit |
| payment providers | XunHuPay and Dodo |
| `liki-deploy` | images, Compose, Caddy, release and rollback |

`liki-web` never accepts browser-supplied identity, entitlement, price, or provider state as authoritative.

## Key flows

### Free chart

```text
User enters birth data
→ Vue App creates or selects a profile
→ liki-web requests deterministic engine facts
→ Vue App renders the chart
```

Charting is free. AI interpretation is a paid boundary.

### Liki Chat purchase

```text
User starts checkout
→ liki-web creates Order and PaymentIntent
→ server route policy selects XunHuPay or Dodo
→ provider checkout is returned
→ provider webhook or reconciliation captures payment
→ durable FulfillmentTask is queued
→ Entitlement is granted
→ customer and admin notifications are queued
```

Provider callbacks are verified and idempotent.

### Liki Chat run

```text
Vue App sends AG-UI to /api/chat/sessions/{id}/ag-ui
→ liki-web checks identity, entitlement, session ownership, and rate limits
→ liki-web loads trusted persisted history
→ liki-web routes every session (profile-bound or not) to the liki-agents runtime
→ profile birth facts travel in trusted AG-UI forwardedProps, not system prompts
→ liki-web persists the authoritative final answer
```

Product browser traffic only talks to `liki-web`. The public deterministic
`/jsonrpc` surface is the deliberate exception: the edge exposes the engine's
read/compute protocol to the Vue charting workspace and external Skill users.
Identity, entitlement, payment, profile writes, and AI generation never cross
that boundary.

## Storage

Storage uses bounded SQLite databases under `LIKI_DATA_DIR`. SQLite is intentional at the current scale.

See [`docs/database/overview.md`](./docs/database/overview.md).

Cross-context writes are orchestrated in application code. There is no ad hoc foreign-key graph across all databases.

## Operational boundaries

| Concern | Owner |
| --- | --- |
| application behavior | `liki-web` |
| domain rules and deterministic tools | `liki` |
| generic agent runtime and protocol adapters | `liki-agents` |
| images, Compose, Caddy, rollback | `liki-deploy` |
| rate limiting state | in-memory, per-instance; scale-out requires a shared limiter |

## Security header ownership

| Layer | Headers |
| --- | --- |
| Edge Caddy (`liki-deploy/caddy/Caddyfile`, `dev/Caddyfile.local`) | CSP, HSTS, Referrer-Policy, Permissions-Policy, COOP, CORP, `-Server` |
| App (`internal/http` `SecurityHeaders`) | `X-Content-Type-Options`, `X-Frame-Options` — fallback if the edge is bypassed |

CSP is owned by the edge. Never duplicate it in the Go service — two
sources will drift.
