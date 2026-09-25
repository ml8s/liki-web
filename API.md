# HTTP API

This document describes the stable HTTP API exposed by `liki-web`.
The deterministic public `/jsonrpc` engine protocol is owned by the `liki`
engine and documented there; `liki-web` does not proxy or redefine it.

JSON responses use a shared envelope.

Success:

```json
{"data": {}}
```

Error:

```json
{"error": {"code": "not_found", "message": "..."}}
```

`error.code` is the stable machine contract. `error.message` is diagnostic and
is not guaranteed to be in the browser's UI language; browser clients map codes
to their own locale resources.

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

## Profiles

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| GET | `/api/profiles` | user | List birth profiles |
| POST | `/api/profiles` | user | Create birth profile |
| GET | `/api/profiles/{id}` | owner | Read birth profile |
| PATCH | `/api/profiles/{id}` | owner | Update profile with expected version |
| DELETE | `/api/profiles/{id}` | owner | Delete profile |

Profile updates use `expected_version`. Cross-user access does not reveal another user's data.

## Products and checkout

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| GET | `/api/chat/products` | user | List products and purchase state |
| POST | `/api/chat/products/{id}/checkout` | user | Create order and provider checkout |

Checkout response:

```json
{
  "order_id": "...",
  "payment_intent_id": "...",
  "provider": "xunhu",
  "status": "checkout_created",
  "checkout_url": "...",
  "status_token": "..."
}
```

The server selects currency, amount, provider, and provider SKU. A client provider value is only a hint.
The product response includes all catalog prices plus `display_currency` and
`display_amount_minor`; the UI must render those server-selected display fields.

## Payments

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| GET | `/api/payments/intents/{id}/status` | capability/session | Check payment status |
| GET | `/api/payments/intents/{id}/fulfillment-status` | capability/session | Check fulfillment status |
| GET | `/api/payments/return/{id}` | capability/session | Browser return and reconciliation |
| POST | `/api/payments/webhook` | provider signature | Provider callback |

Webhooks verify signature, amount, currency, provider, and event fingerprint. Browser returns may reconcile a provider state before the webhook arrives.

## Entitlements

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| GET | `/api/me/entitlements` | user | List active entitlements and allowed skills |

Paid chat operations recheck entitlement server-side.

## Chat

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| GET | `/api/chat/sessions` | user | List sessions |
| POST | `/api/chat/sessions` | entitled user | Create session |
| GET | `/api/chat/sessions/{id}` | owner + entitled | Read session |
| GET | `/api/chat/sessions/{id}/messages` | owner + entitled | Read messages |
| POST | `/api/chat/sessions/{id}/ag-ui` | owner + entitled | AG-UI/SSE generation |

Current canonical skill is `liki`.

## Orders

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| GET | `/api/chat/orders` | user | List chat orders |

Orders expose user-safe state, not provider secrets or internal metadata.

## Account

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| GET | `/api/me/sessions` | user | List authentication sessions |
| DELETE | `/api/me/sessions/{id}` | user | Revoke one session |
| POST | `/api/account/erasure` | user | Erase or anonymize account |

Erasure is rate limited and spans identity, profile, chat, commerce, payment, and referral contexts.

## Partners

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| GET | `/api/partner` | user | Read KOL/Scout role and application states |
| GET | `/api/kol/application` | user | Read KOL application state |
| POST | `/api/kol/application` | user | Submit a KOL application |
| GET | `/api/scout/application` | user | Read Scout role/application state |
| POST | `/api/scout/application` | user | Submit a Scout application |
| GET | `/api/kol/session` | active KOL | Read KOL dashboard summary |
| GET | `/api/kol/commissions` | active KOL | List KOL commission entries |
| GET | `/api/kol/payouts` | active or retired KOL | List payouts |
| POST | `/api/kol/payouts` | active or retired KOL | Request a payout |
| GET | `/api/kol/settlement` | active or retired KOL | Read money-only settlement state |
| GET | `/api/scout/session` | active Scout | Read Scout dashboard summary |
| GET | `/api/scout/recruits` | active Scout | List recruited KOL partners |
| GET | `/api/scout/commissions` | active Scout | List Scout commission entries |
| GET | `/api/scout/payouts` | active or retired Scout | List payouts |
| POST | `/api/scout/payouts` | active or retired Scout | Request a payout |
| GET | `/api/scout/settlement` | active or retired Scout | Read money-only settlement state |
| GET | `/r/{code}` | public | Store a signed-in-new-account attribution cookie |

KOL and Scout are independent role capabilities, not levels. The public code
link is the last valid link before a new account is created. A Scout link that
leads to an approved KOL creates that Scout's direct channel relationship.
Applications do not require product purchase; platform review controls access.
A retired role exposes only settlement data and cannot reopen the partner dashboard.

## Feedback

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| POST | `/api/feedback` | public, rate limited | Submit a UTF-8 JSON `feedback-v1` payload |
| GET | `/api/feedback` | feedback API key | List feedback lifecycle records |
| PATCH | `/api/feedback/{id}` | feedback API key | Update processing state and issue URL |

The server stores and returns the self-describing payload verbatim. It does not
reinterpret skill-specific fields as a fixed backend DTO.

Operators can decode records for triage with:

```bash
FEEDBACK_API_KEY=... python3 scripts/feedback-read.py --status pending
```

## Legacy product compatibility

The naming, Ming Shu, and He Pan pages retain their historical order/report and
naming chat routes under `/api/naming`, `/api/mingshu`, and `/api/hepan`.
They are frozen compatibility surfaces; new commerce flows use the unified
Liki Chat product and payment APIs above.

## Internal

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| GET | `/api/health` | public | Liveness |
| GET | `/api/ready` | public, rate limited | Readiness |
| POST | `/api/analytics/pageview` | public, rate limited | Increment aggregate page-view counter |
| GET | `/api/version` | public | Build time and version |
| GET | `/api/internal/metrics` | token | JSON or Prometheus counters |

Metrics are disabled when `METRICS_TOKEN` is empty.
