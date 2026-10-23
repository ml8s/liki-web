# Architecture

`liki-web` is the product-facing web/BFF service for Liki. It owns product state and commercial workflows. It does not own deterministic domain computation, generic agent execution, or production deployment.

## Runtime modules

| Module | Responsibility |
| --- | --- |
| `cmd/liki-web` | Service entrypoint |
| `internal/http` | HTTP route composition, middleware (request-id, RED metrics), security headers, rate limiting |
| `internal/auth` | email-code identity, sessions, verification codes |
| `internal/birthprofile` | birth profile aggregate |
| `internal/payment` | provider-neutral payment runtime |
| `internal/product` | product definitions, skills, offers, and pricing |
| `internal/sales` | checkout, orders, and subscription lifecycle |
| `internal/entitlement` | user product entitlements |
| `internal/chat` | Liki Chat sessions, messages, and AG-UI runtime |
| `internal/partner` | KOL/Scout applications, attribution, commission, and payout domain |
| `internal/account` | account erasure orchestration |
| `internal/admin` | admin operations and audit |
| `internal/agent` | AG-UI event adapter shared with the liki-agent-runtime |
| `internal/adminauth` | admin console session gate |
| `internal/aipay` | Alipay A2M gateway, sandbox proof-of-payment records |
| `internal/dodo` | Dodo payment gateway client |
| `internal/xunhu` | XunHuPay gateway client |
| `internal/email` | Resend email client |
| `internal/feedback` | feedback lifecycle store and API |
| `internal/httputil` | shared HTTP primitives: client IP, forwarded headers, JWT validation, request-id context and slog handler, response envelope, decode, SSE |
| `internal/i18n` | supported language tag constants |
| `internal/llm` | chat message model shared by protocol surfaces |
| `internal/metrics` | in-memory lifecycle counters, HTTP RED metrics, and the internal metrics endpoint |
| `internal/migrate` | forward-only SQL migration runner for embedded SQLite |
| `internal/sqliteutil` | standard SQLite connection policy for every database file |

## Frontend modules

Every browser surface is framework-rendered and written in TypeScript:

| Path | Role | Checked by |
| --- | --- | --- |
| `web/src` | Astro marketing and static locale pages; page behaviour in `web/src/scripts` | `astro check` |
| `web/app` | Vue 3 authenticated product SPA with zh-Hans/zh-Hant/en i18n | `vue-tsc` |
| `web/e2e` | Playwright journeys against a running stack | `tsc -p e2e/tsconfig.json` |
| `web/scripts` | static build and consistency checks used by `make check` | eslint (Node ESM) |

New product functionality belongs in `web/app` and its backing Go APIs; new
static-page behaviour belongs in `web/src/scripts`. There is no vanilla-JS
browser runtime: the retired `web/js` layer is replaced by those Astro scripts
and the Vue app.

## External boundaries

| Repository / service | Responsibility |
| --- | --- |
| `liki` | deterministic engine, skills, Engine MCP, Counsel MCP source |
| `liki-agent-runtime` | generic agent runtime, AG-UI/A2A adapters, execution audit |
| payment providers | XunHuPay and Dodo |
| `liki-deploy` | local images, Compose, Caddy, deploy and rollback |

`liki-web` never accepts browser-supplied identity, entitlement, price, or provider state as authoritative.

## Key flows

### Free chart

```text
User enters birth data
→ Vue App creates or selects a profile
→ the App calls deterministic engine facts over the same-origin MCP gateway
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

Payment self-service reads (`/api/payments/return/{id}` and the two intent
status endpoints) are anonymous by design — the browser return can land without
a session, on another device, or after session expiry. The credential is a
**capability URL**: the unguessable intent id mints a 1-hour, `payment-status`
scoped token that is accepted only by these read-only status endpoints. The
status responses carry no PII (status, order id, redirect URLs only). Because
the return endpoint may trigger one provider reconciliation call, all three
endpoints carry a rate-limit tier.

### Liki Chat run

```text
Vue App sends AG-UI to /api/chat/sessions/{id}/ag-ui
→ liki-web checks identity, entitlement, session ownership, and rate limits
→ liki-web loads trusted persisted history
→ liki-web routes the session to its profile's experts runtime
→ profile birth facts travel in trusted AG-UI forwardedProps, not system prompts
→ liki-web persists the authoritative final answer
```

The relay above is the default transport and the rollback lever. A build-time
`VITE_CHAT_TRANSPORT=direct` web image instead posts to `/rt/<profile>/ag-ui`:
Caddy performs ForwardAuth introspection against liki-web
(`/api/internal/introspect`) and opens a mutual-TLS connection to the runtime;
the client carries the conversation history and profile context, and history is
persisted by the signed run-completed callback
(`/api/internal/runs/completed`). See liki-deploy's `docs/DIRECT_CONNECT.md`.

Product browser traffic talks either to `liki-web` (identity, entitlement,
payment, profiles, chat) or directly to the MCP services over the same-origin
gateway prefixes `/engine/*` and `/counsel/*` — the same paths and protocol
external skills use. Charting calls only the deterministic MCP tools;
identity, entitlement, payment, profile writes, and AI generation stay inside
`liki-web`.

## Storage

Storage uses bounded SQLite databases under `LIKI_DATA_DIR`. SQLite is intentional at the current scale.

See [`docs/database/overview.md`](./docs/database/overview.md).

Cross-context writes are orchestrated in application code. There is no ad hoc foreign-key graph across all databases.

## Operational boundaries

| Concern | Owner |
| --- | --- |
| application behavior | `liki-web` |
| domain rules and deterministic tools | `liki` |
| generic agent runtime and protocol adapters | `liki-agent-runtime` |
| images, Compose, Caddy, rollback | `liki-deploy` |
| rate limiting state | in-memory, per-instance; scale-out requires a shared limiter |

## Security header ownership

| Layer | Headers |
| --- | --- |
| Edge Caddy (`liki-deploy/caddy/Caddyfile`, `dev/Caddyfile.local`) | CSP, HSTS, Referrer-Policy, Permissions-Policy, COOP, CORP, `-Server` |
| App (`internal/http` `SecurityHeaders`) | `X-Content-Type-Options`, `X-Frame-Options` — fallback if the edge is bypassed |

CSP is owned by the edge. Never duplicate it in the Go service — two
sources will drift.

## Domain dependency rules

Bounded contexts are enforced by `internal/arch`:

```text
product      may not import sales, payment, entitlement, chat, partner, or birthprofile
sales        may import product and payment; must not import entitlement, chat, or partner
payment      is provider-neutral and must not import product/sales/entitlement/chat/partner
entitlement  may import product; must not import sales/payment/chat/partner
chat         may import product and birthprofile; must not import sales/payment/partner
partner      may import payment; must not import product/sales/chat/entitlement
```

Cross-context workflows use injected ports in the composition root. Sales hands
paid facts to an `EntitlementProvisioner`; chat queries an `AccessAuthorizer`.
Neither imports the concrete entitlement store.
