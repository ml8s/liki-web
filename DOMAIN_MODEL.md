# Domain Model

This document defines the stable domain model and invariants behind Liki Web.

## Ubiquitous language

| Term | Meaning |
| --- | --- |
| User | Platform identity created through email verification |
| Session | Authentication session for one user |
| BirthProfile | Birth data used to calculate charts |
| Product / SKU | Purchasable product |
| Order | Commercial order owned by a user |
| PaymentIntent | Provider-neutral payment state |
| PaymentEvent | Deduplicated provider or reconciliation event |
| Entitlement | Right to use a skill during a period |
| ChatSession | Durable consultation conversation |
| ChatMessage | Durable message inside a session |
| FulfillmentTask | Durable post-payment work |
| Notification | Durable email work |

`liki-chat` is the only commercial product. `naming`, `mingshu`, and `hepan` are retired product identifiers retained for historical records and for Liki Chat scenario landing pages.

## Identity

### User

A user owns profiles, orders, entitlements, chat sessions, and account operations.

Invariants:

- Internal records are scoped by stable `user_id`.
- Email is not sufficient to prove ownership.
- Erasure anonymizes identity and personal data while preserving required financial audit records.

### Session

A session authenticates a caller.

Invariants:

- Session cookies are HttpOnly.
- Session presence proves identity, not authorization for another user's data.
- Product reads and writes recheck ownership and entitlement.

## Birth profile

A birth profile is sensitive personal data used for deterministic charting.

Key fields:

```text
id
user_id
nickname
gender
calendar
birth_date
birth_time
birth_time_known
birth_place
version
```

Invariants:

- A profile belongs to exactly one user.
- Updates use optimistic locking through `version`.
- Cross-user reads are forbidden.
- Erasure anonymizes birth data and embedded profile messages.

## Product and price

Products are defined by the service, not by the browser.

Only purchasable product:

```text
liki-chat
```

Retired identifiers:

```text
naming
mingshu
hepan
```

Invariants:

- Amount is selected from the product catalog using currency.
- Provider SKU configuration is separate from the product price.
- Unsupported product/currency combinations fail closed.
- Retired identifiers cannot create a new order, checkout, activation code, or entitlement.
- Client-supplied provider is only a hint.

## Order and payment

Order lifecycle:

```text
pending
→ provider checkout
→ paid
→ fulfilled
```

Payment intent lifecycle includes:

```text
created
checkout_created
paid
refunded
failed
expired
```

Invariants:

- Order creation precedes payment intent creation.
- Payment webhooks verify signature, amount, currency, and expected intent.
- Payment events use stable fingerprints for deduplication.
- Duplicate success callbacks do not create duplicate business effects.
- Missing provider configuration fails closed.
- Canceled orders never grant entitlements.

## Entitlement

An entitlement grants one or more skills for a period.

Invariants:

- Grants require explicit `source` and `source_id`.
- `(source, source_id, product_id)` is idempotent.
- Expired entitlements cannot read or write paid resources.
- Legacy entitlements do not implicitly become `liki-chat`.
- Entitlement is rechecked on session read, creation, message read, and generation.

## Fulfillment

Fulfillment is durable work performed after payment capture.

Statuses:

```text
queued
running
succeeded
failed
```

Invariants:

- Webhook acknowledgement and product fulfillment are independent.
- `OnPaid` is idempotent.
- Failed fulfillment retries with backoff.
- A terminal fulfillment result is not reset by replaying the same event.
- Notification is a separate durable stream.

## Chat

A chat session belongs to one user and one skill.

Invariants:

- Chat session creation requires an active entitlement.
- Chat session and message reads require ownership and active entitlement.
- Generation requires an active entitlement.
- User messages persist before generation.
- `RUN_FINISHED.result.answer` is the authoritative assistant answer.
- `RUN_ERROR` does not persist a formal assistant message.
- Browser AG-UI events are transport, not a substitute for persisted state.

### Scenario

A scenario is an entry intent for Liki Chat, not a product. Its public page is a CTA surface into the app, comparable to the home page.
Current public scenarios are `naming`, `mingshu`, and `hepan`.

## Notification

Notification kinds:

```text
customer_receipt
admin_new_order
report_ready
```

Invariants:

- `(payment_intent_id, kind)` is deduplicated.
- Customer receipts state that payment completed and fulfillment is in progress.
- Customer receipts do not claim that service has already been enabled.
- Admin notices contain order, intent, email, amount, provider, and payment time.
- Admin notices are fixed to Simplified Chinese.
- Customer notices follow `zh-Hans`, `zh-Hant`, or `en`.

## Account erasure

Erasure balances privacy with financial auditability.

Invariants:

- Birth profile data and chat-embedded personal context are anonymized.
- Authentication sessions and verification codes are removed or anonymized.
- Commerce and payment records may remain for accounting, but user identity is anonymized.
- Referral payouts with financial holds block erasure until resolved.

## Referral

Referral state tracks attribution, commission eligibility, payout, and adjustment.

Invariants:

- KOL and Scout are independent role capabilities, not levels.
- Recruiter binding comes only from partner-link attribution before a new account is created.
- Product purchase is normal product usage, not a paid ticket to apply for either role.
- Financial records remain auditable.
- A retired role can settle existing earnings; a frozen partner cannot withdraw.
- Payouts cannot be erased while a financial hold exists.
- Referral rules never implicitly mutate Liki Chat entitlements.
