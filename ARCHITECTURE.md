# Architecture

`liki-web` is the product-facing web/BFF service for Liki. It owns product state and commercial workflows. It does not own deterministic domain computation, generic agent execution, or production deployment.

## Runtime modules

| Module | Responsibility |
| --- | --- |
| `cmd/liki-web` | Service entrypoint |
| `internal/http` | HTTP route composition, middleware, security headers, rate limiting |
| `internal/auth` | email-code identity, sessions, verification codes |
| `internal/profile` | birth profile aggregate |
| `internal/product` | canonical product IDs and prices |
| `internal/payment` | provider-neutral payment runtime |
| `internal/paymentflow` | legacy product fulfillment flows |
| `internal/liki` | unified consultation domain: product, order, entitlement, chat |
| `internal/account` | account erasure orchestration |
| `internal/referral` | KOL/Scout applications, attribution, commission, and payout domain |
| `internal/admin` | admin operations and audit |
| `internal/agent` | in-process local runner and tool scoping |

## Frontend modules

| Path | Role |
| --- | --- |
| `web/src` | Astro marketing and static locale pages |
| `web/app` | Vue 3 authenticated product SPA with zh-Hans/zh-Hant/en i18n |
| `web/js` | legacy vanilla JS compatibility pages |

New product functionality belongs in `web/app` and its backing Go APIs. Legacy vanilla JS pages are compatibility surfaces only.

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
→ unprofiled sessions may use liki-agents; profile-bound sessions use the local runner
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
