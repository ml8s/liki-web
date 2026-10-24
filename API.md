# HTTP API

This document describes the stable HTTP API exposed by `liki-web`.
The legacy public `/jsonrpc` engine runtime was retired together with the
engine RPC server; `liki-web` does not proxy it, and it no longer proxies
engine/counsel MCP either. Browser charting calls the MCP services directly
over the same-origin gateway paths `/engine/*` and `/counsel/*`; the gateway
strips the prefix and forwards to engine-mcp / counsel-mcp. The browser speaks
the same MCP Streamable HTTP protocol as external skills.

`liki-web`'s own API (profiles, records, orders, chat, …) uses the shared
envelope (`application/json`):

```json
{"data": {}}
```

## Engine & counsel MCP (same-origin gateway)

`POST /engine/mcp` and `POST /counsel/mcp` accept MCP JSON-RPC 2.0. Both
services are stateless with JSON responses — no `initialize` handshake is
required. Clients must send `Accept: application/json, text/event-stream`
(MCP Streamable HTTP spec).

Request:

```json
{"jsonrpc": "2.0", "id": 1, "method": "tools/call",
 "params": {"name": "bazi_chart", "arguments": {
   "instant": "1958-05-11T22:00:00Z",
   "timezone": "Asia/Shanghai",
   "longitude": 121.47,
   "gender": "female"
 }}}
```

The tool payload is a JSON string inside `result.content[0].text`. Tool
errors return HTTP 200 with `result.isError: true` and the diagnostic message
in the same field. Engine tools (charting/calendar) live under `/engine/mcp`;
counsel tools (`bazi_yongshen`, `naming_*`, `liuyao_query`, …) under
`/counsel/mcp`.

Chart tools accept the canonical `{instant, timezone, longitude}` context. `instant` is
an RFC 3339 UTC instant, `timezone` is an IANA zone, and `longitude` is WGS-84.
The browser converts exact Gregorian wall-clock input locally with Temporal and
never sends raw calendar or profile wall-time fields to chart tools.
Lunar calendar arithmetic remains engine-internal output, not browser input.

Browser callers must materialize Engine MCP arguments rather than relying on
server defaults. In particular, LiuYao sends an explicit `liuyao_qigua` mode,
captures its ritual UTC instant once at submit, and passes that instant plus the
visible casting timezone and numeric longitude to `liuyao_chart`. It also
materializes `yong_shen` explicitly. The display place string is never
substituted for longitude or timezone.

Error responses follow [RFC 9457 Problem Details](https://www.rfc-editor.org/rfc/rfc9457)
(`application/problem+json`):

```json
{
  "type": "not_found",
  "title": "not_found",
  "status": 404,
  "detail": "..."
}
```

`type` is the stable machine identifier (a URI reference; short codes like
`not_found` are valid relative references). The full code inventory (every code
must carry App copy in all three locales) lives in
[`docs/error-codes.md`](./docs/error-codes.md), enforced by `make check-web-static`.
`detail` is diagnostic and is not
guaranteed to be in the browser's UI language; browser clients map types to
their own locale resources.

The problem shape applies to every handler-owned error response, including
operator endpoints and middleware (rate limiting, recovery). The only
exception is the `net/http` ServeMux's own 404/405 replies, which remain the
framework default. If the API ever gains third-party consumers, the error
shape already matches RFC 9457; full JSON:API v1.1 (`application/vnd.api+json`
with links/meta) was evaluated and deliberately not adopted.

Authentication uses HttpOnly platform session cookies. Ownership and entitlement are always checked server-side.

## Identity

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| POST | `/api/auth/email/code` | public | Send email verification code |
| POST | `/api/auth/email/verify` | public | Verify code and create session |
| POST | `/api/auth/logout` | user | Destroy current session |
| POST | `/api/auth/sessions/revoke-all` | user | Revoke every other session |
| GET | `/api/me` | user | Read current user |
| GET | `/api/me/auth/identities` | user | List login identities |

Verification codes are rate limited. Production does not return the verification code in the response.

## Birth profiles

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| GET | `/api/birth-profiles` | user | List birth profiles |
| POST | `/api/birth-profiles` | user | Create birth profile |
| GET | `/api/birth-profiles/{id}` | owner | Read birth profile |
| PATCH | `/api/birth-profiles/{id}` | owner | Update birth profile with expected version |
| DELETE | `/api/birth-profiles/{id}` | owner | Delete birth profile |

Profile updates use `expected_version`. Cross-user access does not reveal another user's data.

New and updated profiles carry birth facts in one nested `birth` object.
Input requires `gender`, `date` (`YYYY-MM-DD`), `time` (`HH:mm`), and
`location.timezone` (IANA zone) plus `location.longitude` (WGS-84,
`-180..180`). `location.place` is display-only. The server computes and stores
`instant`; responses project `date` and `time` for editing and display. There
is no approximate/unknown time state. Incomplete profiles cannot be charted
until they are edited and completed.
`third_party_consent: true` is required; the account owner must confirm consent
before storing another person's birth facts.

```json
{
  "nickname": "妈妈",
  "third_party_consent": true,
  "birth": {
    "gender": "female",
    "date": "1958-05-12",
    "time": "06:00",
    "location": {
      "place": "上海",
      "longitude": 121.4737,
      "timezone": "Asia/Shanghai"
    }
  }
}
```

Responses use the same `birth` object and add canonical `instant`.

## Products and checkout

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| GET | `/api/sales/products` | user | List products and purchase state |
| POST | `/api/sales/checkouts` | user | Create order or subscription checkout (`product_id` and `billing_mode` in body); returns `201 Created` |

A one-time checkout response is:

```json
{
  "order_id": "...",
  "payment_intent_id": "...",
  "provider": "xunhu",
  "status": "checkout_created",
  "checkout_url": "...",
  "qr_code_url": "...",
  "status_token": "..."
}
```

`qr_code_url` is omitted when the selected provider does not return one. A
subscription checkout instead returns `subscription_id`, `provider`, `status`,
and `checkout_url`.

The server selects currency, amount, provider, and provider SKU. A client provider value is only a hint.
The product response carries the server-selected `display_currency` and `display_amount_minor`, and
`purchase_options` lists only the offers that currency can settle — every option's currency equals
`display_currency`. The UI renders those fields as-is: a CNY buyer sees Xunhu one-time pricing while an
overseas buyer sees Dodo one-time and subscription pricing.

## Payments

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| GET | `/api/payments/intents/{id}/status` | capability/session | Check payment status |
| GET | `/api/payments/intents/{id}/payment-completion-status` | capability/session | Check payment completion status |
| GET | `/api/payments/return/{id}` | capability/session | Browser return and reconciliation |
| POST | `/api/payments/webhook` | provider signature | Provider callback |
| POST | `/api/aipay` | public, rate limited | Redeem an Alipay A2M sandbox payment proof (`Payment-Proof` header, 402 flow); 503 when the channel is not enabled |

Webhooks verify signature, amount, currency, provider, and event fingerprint. Browser returns may reconcile a provider state before the webhook arrives.

Payment status is a minimal polling fact:

```json
{
  "payment_intent_id": "...",
  "order_id": "...",
  "status": "paid"
}
```

Completion status adds:

```json
{
  "payment_intent_id": "...",
  "payment_status": "paid",
  "fulfillment_status": "succeeded"
}
```

Redirect URLs are not exposed by polling; `/api/payments/return/{id}` owns the redirect decision.

## Entitlements

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| GET | `/api/me/entitlements` | user | List active entitlements and allowed skills |

Paid chat operations recheck entitlement server-side.

## Chat

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| GET | `/api/chat/sessions` | user | List sessions |
| POST | `/api/chat/sessions` | entitled user | Create session, optionally attaching deterministic tool context; returns `201 Created` |
| GET | `/api/chat/sessions/{id}` | owner + entitled | Read session |
| PATCH | `/api/chat/sessions/{id}` | owner | Rename session (title, 1-60 chars) |
| DELETE | `/api/chat/sessions/{id}` | owner | Soft-delete session |
| GET | `/api/chat/sessions/{id}/messages` | owner + entitled | Read messages (cursor paging: `?before=&limit=`; response `{messages, next_cursor}`) |
| POST | `/api/chat/sessions/{id}/ag-ui` | owner + entitled | AG-UI/SSE generation (relay) |

The direct transport (`/rt/<profile>/ag-ui`) is not a liki-web route: Caddy
reaches the runtime directly after calling `/api/internal/introspect`, and the
runtime posts `/api/internal/runs/completed` back. Both internal endpoints are
404 on the public path; see liki-deploy's `docs/DIRECT_CONNECT.md`.

Current canonical skill is `liki` (four experts: `natal`, `divination`, `fengshui`, `naming`).

`POST /api/chat/sessions` accepts an optional strict `context` object. The
versioned context preserves the exact tool facts required to continue a paid
consultation. It is persisted with the session, returned only by the session
detail endpoint (not the list projection), placed under
`forwardedProps.consultation_context` by both transports, and cleared by
account erasure. Invalid or unknown context shapes return HTTP 400 with
`invalid_consultation_context` or `invalid_request`, respectively.

Scenario-specific requirements include:

| Scenario | Required context |
| --- | --- |
| `hepan` | `second_birth` with date, time, longitude, timezone, and `third_party_consent: true` |
| `divination` | `question`, non-reproducible `casting`, `casting_instant`, and atomic `location` |
| `huangli` | `event`, `start_date`, and `end_date` |
| `fengshui` | `layout` with `zuo_shan` and `period_date` |
| `natal`, `naming` | atomic `birth` facts |
| `liunian` | atomic `birth` facts and `year` |

The object also enforces a per-scenario allowlist. For example, a divination
context cannot smuggle in birth facts, and a He Pan context cannot carry a
casting receipt. Feng Shui public-handoff facts use `birth_year` and `gender`
only as a pair. Beyond unknown members, the object rejects invalid calendar
dates/times, invalid longitude or IANA timezone, incomplete or internally
inconsistent casting receipts, unsupported versions, and contexts that do not
route to the selected expert. A persisted context is revalidated before relay
generation, so corrupt tool state fails closed instead of becoming generic
advice. Natural-language intent is never a substitute for these computation
facts.

### AG-UI forwardedProps

Both the relay runner and direct browser transport use one profile shape:

```json
{
  "product": "liki",
  "locale": "zh-Hant",
  "consultation_context": {
    "version": 1,
    "scenario": "divination",
    "question": "項目能否成交",
    "casting": {
      "mode": "auto",
      "order": "bottom_up",
      "yaos": [7, 8, 8, 7, 8, 9],
      "dong_yao": [6],
      "casting_id": "..."
    },
    "casting_instant": "2026-10-23T04:00:00Z",
    "location": { "longitude": 121.4726, "timezone": "Asia/Shanghai" }
  },
  "profile": {
    "gender": "female",
    "instant": "1958-05-11T22:00:00Z",
    "location": {
      "longitude": 121.4737,
      "timezone": "Asia/Shanghai"
    }
  }
}
```

`profile` is present only for profile-bound sessions. The relay transport
constructs it server-side from the owned profile; direct transport may send the
same trusted shape after the browser loads it from `/api/birth-profiles/{id}`.
`location.place`, identity, contact details, notes, and database IDs are
deliberately excluded. The only consent fact in this object is the required
affirmative `third_party_consent` for He Pan.

## Records

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| GET | `/api/records?kind=` | user | List tool records (newest 50; `kind` one of `liuyao`, `huangli`, `bond`, `bazhai`) |
| POST | `/api/records` | user | Save a tool record; returns `201 Created` |
| PATCH | `/api/records/{id}` | owner | Rename a record (title, 1-60 chars) |
| DELETE | `/api/records/{id}` | owner | Soft-delete a record |

Records are opaque engine I/O snapshots for the service pages; the engine contract stays authoritative for their shape.

A record response is owner-scoped and exposes only:

```json
{
  "id": "...",
  "kind": "liuyao",
  "title": "...",
  "input": {},
  "output": {},
  "created_at": "...",
  "updated_at": "..."
}
```

`output` is omitted until a result exists. Requests reject unknown members; `kind` must be one of the four values above, `input` and `output` must be JSON objects, and `title` is 1-60 characters.

## Orders

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| GET | `/api/sales/orders` | user | List sales orders (annotated with `intent_status`) |
| GET | `/api/sales/orders/{id}` | owner | Order detail (order + payment intent facts) |
| POST | `/api/sales/orders/{id}/cancel` | owner | Cancel a pending order |
| POST | `/api/sales/orders/{id}/checkout` | owner | Re-open checkout for a pending order (fresh provider session; 409 `order_expired` / `order_already_paid` / `order_refunded`) |

Orders expose user-safe state, not provider secrets or internal metadata. Owner
order responses omit `user_id` and `email`; authentication already proves
ownership and account email remains in identity/account APIs.

## Account

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| GET | `/api/me/sessions` | user | List authentication sessions |
| DELETE | `/api/me/sessions/{id}` | user | Revoke one session |
| POST | `/api/account/erasure` | user | Erase or anonymize account |

Erasure is rate limited and spans identity, birth profiles, chat, tool records,
direct-run answers, sales, entitlements, payment, and partner contexts.

## Partners

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| GET | `/api/partner-portal` | user | Read KOL/Scout role and application states |
| GET | `/api/partner-portal/kol/application` | user | Read KOL application state |
| POST | `/api/partner-portal/kol/application` | user | Submit a KOL application |
| GET | `/api/partner-portal/scout/application` | user | Read Scout role/application state |
| POST | `/api/partner-portal/scout/application` | user | Submit a Scout application |
| GET | `/api/partner-portal/kol/session` | active KOL | Read KOL dashboard summary |
| GET | `/api/partner-portal/kol/commissions` | active KOL | List KOL commission entries |
| GET | `/api/partner-portal/kol/payouts` | active or retired KOL | List payouts |
| POST | `/api/partner-portal/kol/payouts` | active or retired KOL | Request a payout |
| GET | `/api/partner-portal/kol/settlement` | active or retired KOL | Read money-only settlement state |
| GET | `/api/partner-portal/scout/session` | active Scout | Read Scout dashboard summary |
| GET | `/api/partner-portal/scout/recruits` | active Scout | List recruited KOL partners |
| GET | `/api/partner-portal/scout/commissions` | active Scout | List Scout commission entries |
| GET | `/api/partner-portal/scout/payouts` | active or retired Scout | List payouts |
| POST | `/api/partner-portal/scout/payouts` | active or retired Scout | Request a payout |
| GET | `/api/partner-portal/scout/settlement` | active or retired Scout | Read money-only settlement state |
| GET | `/r/{code}` | public | Store a signed-in-new-account attribution cookie |

Partner payout responses name their owner explicitly:

```json
{
  "beneficiary_id": "...",
  "beneficiary_code": "...",
  "beneficiary_role": "kol"
}
```

KOL and Scout are independent role capabilities, not levels. The public code
link is the last valid link before a new account is created. A Scout link that
leads to an approved KOL creates that Scout's direct channel relationship.
Applications do not require product purchase; platform review controls access.
A retired role exposes only settlement data and cannot reopen the partner dashboard.

## Subscriptions

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| GET | `/api/sales/subscriptions` | user | List subscriptions |
| GET | `/api/sales/subscription-payments` | user | List subscription payments |
| POST | `/api/sales/subscriptions/{id}/cancellation` | owner | Schedule cancellation at period end |
| DELETE | `/api/sales/subscriptions/{id}/cancellation` | owner | Cancel a scheduled cancellation |

Subscription lifecycle state and money records are separate read models; both
are scoped to the signed-in user.

## Feedback

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| POST | `/api/feedback` | public, rate limited | Submit a UTF-8 JSON `feedback-v1` payload |
| GET | `/api/feedback` | feedback API key | List feedback lifecycle records (`status`, `limit` ≤ 50, `offset`) |
| PATCH | `/api/feedback/{id}` | feedback API key | Update processing state and issue URL |

The server stores and returns the self-describing payload verbatim. It does not
reinterpret skill-specific fields as a fixed backend DTO.

Operators can decode records for triage with:

```bash
FEEDBACK_API_KEY=... python3 scripts/feedback-read.py --status pending
FEEDBACK_API_KEY=... python3 scripts/feedback-read.py --all
```

`GET /api/feedback` pages oldest-first with `limit` (default 20, cap 50) and
`offset`. Traversing `offset` until a short page covers records beyond the
single-page cap.

## Retired product compatibility

The naming, Ming Shu, and He Pan public pages are Liki Chat CTA surfaces, not
separate products. Their historical commerce backends (orders, checkout,
naming-chat sessions, email-only login) have been fully removed. Requests to the
retired `/api/naming`, `/api/mingshu`, and `/api/hepan` routes return `404`;
they never establish a session or grant an entitlement.

## Internal

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| GET | `/api/health` | public | Liveness |
| GET | `/api/ready` | public, rate limited | Readiness |
| POST | `/api/analytics/pageview` | public, rate limited | Increment aggregate page-view counter |
| GET | `/api/version` | public | Build time and version |
| GET | `/api/internal/metrics` | token | JSON or Prometheus counters |

Metrics are disabled when `LIKI_METRICS_TOKEN` is empty.

## Admin

The back-office API lives under `/api/admin/*` (administrator and
capability management, user entitlement support and compensation, finance
aggregates, partner application decisions, payout approval, and partner freeze
controls). They are gated by admin sessions
and capabilities
and are intentionally outside this stable product API; their shape may
change with the back-office without notice. See `internal/admin/` and
`internal/partner/admin_handler.go` for the authoritative list.

Compensation accepts `idempotency_key`; an unchanged retry maps to the same
beneficiary and returns the same entitlement instead of stacking access. The
response reports whether the customer notification entered the durable email
queue (`email_queued`). A false value means the grant is committed but the
notification requires operator follow-up.

Finance aggregates use UTC half-open periods `[start,end)`. `net_product_revenue`
means known product cash receipts less subscription refunds in the period;
one-time provider refunds are not represented in the sales ledger. The
pending payout aggregate is a current operational balance, not a period-limited
total.

Admin list endpoints use offset pagination:

```text
page=1
page_size=100
```

Responses include the original collection key plus `page`, `page_size`,
`total`, and `has_more`. `page_size` is capped at 100.
