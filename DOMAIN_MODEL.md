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

`liki-chat` is the only commercial product. `naming`, `mingshu`, and `hepan` are retired product names now used only as Liki Chat scenario landing page identifiers.

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
birth.location.place # display label (free text; not used for computation)
birth.location.longitude # WGS-84 longitude used for true solar time
birth.location.timezone # IANA timezone used with birth.instant
birth.instant        # canonical RFC 3339 UTC instant for exact-time profiles
birth.date           # Gregorian date derived from birth.instant + timezone
birth.time           # local wall time derived from birth.instant + timezone
version
```

Invariants:

- A profile belongs to exactly one user.
- Updates use optimistic locking through `version`.
- Cross-user reads are forbidden.
- Erasure anonymizes birth data and embedded profile messages.
- Chartable profiles require `birth.location.longitude`,
  `birth.location.timezone`, and `birth.instant` together.
- There is no approximate/unknown time state. If canonical fields are absent,
  the profile cannot be charted until the user supplies complete information.
- New/updated profile input is Gregorian-only and must be complete.
- Creating or updating another person's profile requires affirmative
  `third_party_consent`. Ephemeral He Pan input that is persisted as a tool
  record or converted into a paid consultation follows the same rule.

## Product and price

Products are defined by the service, not by the browser.

Only purchasable product:

```text
liki-chat
```

Scenario identifiers (landing page → Liki Chat handoff, not products) are the
seven user-facing scenarios; each routes to an internal expert deployment:

```text
natal      → expert natal       (life reading)
liunian    → expert natal       (annual fortune)
hepan      → expert natal       (compatibility)
divination → expert divination
huangli    → expert divination  (date selection)
fengshui   → expert fengshui
naming     → expert naming
```

### Scenario naming convention

Scenario identifiers use **pinyin** when they map to one specific Chinese
metaphysics concept with no English equivalent; they use **English** when they
are an umbrella covering multiple methods. This is a deliberate domain-model
decision, not an inconsistency.

| Scenario | Language | Rationale |
| -------- | -------- | --------- |
| natal | English | Umbrella: covers bazi + ziwei (no single pinyin word) |
| liunian | Pinyin | Specific concept: annual fortune (流年) |
| hepan | Pinyin | Specific concept: Chinese synastry (合盘, not Western astrology) |
| divination | English | Umbrella: covers liuyao + qimen (extensible to more methods) |
| huangli | Pinyin | Specific concept: Chinese almanac (黄历) |
| fengshui | Pinyin | Specific concept: already an English loanword (风水) |
| naming | English | Generic service action, not metaphysics-specific |

The same rule applies to frontend code: shared chart components use pinyin
(`BaziPanel`, `ZiweiPanel`), umbrella UI layers use English
(`DivinationView`, `NatalTool`). Engine MCP tool names (`liuyao_qigua`,
`bazi_chart`) are cross-system contracts and always keep their original
pinyin identifiers regardless of the frontend naming layer.

Invariants:

- Amount is selected from the product catalog using currency.
- Provider SKU configuration is separate from the product price.
- Unsupported product/currency combinations fail closed.
- Unregistered product IDs cannot create an order, checkout, or entitlement.
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
- The runtime's terminal `answer`, consumed server-side by the BFF, is the
  authoritative assistant answer; browser AG-UI events are not persisted state.
- `RUN_ERROR` does not persist a formal assistant message.
- A tool-originated session may carry one versioned consultation context. The
  context stores deterministic inputs and non-reproducible receipts (especially
  LiuYao casting), is attached when the session is created, and is the same for
  every BFF runtime call. Prose intent never replaces longitude, timezone,
  canonical instants, dates, or a casting receipt. The BFF constructs it from
  owned session/profile state; the browser never supplies it as authority.

### Scenario

A scenario is an entry intent for Liki Chat, not a product. Its public page is a CTA surface into the app, comparable to the home page.
The seven public scenarios are `natal`, `liunian`, `hepan`, `divination`, `huangli`, `fengshui`, and `naming` — one vocabulary shared by marketing slugs and App handoff parameters. Each scenario routes to an internal expert deployment; the expert layer is never presented to users. `mingshu` (life reading) is the deliverable of the `natal` scenario, not a scenario of its own.

Eight Mansions uses the Gregorian calendar year for both `birth_year` and
annual flying-star boundaries. The engine explicitly returns
`year_boundary: "gregorian_calendar_year"`; it does not apply a Lichun
boundary. Public form copy and fixtures must preserve that contract.

## Notification

Notification kinds:

```text
customer_receipt
admin_new_order
entitlement_compensation
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

- Birth profile data, chat-embedded personal context, tool-record engine I/O,
  and generated direct-run answers are anonymized or cleared.
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
