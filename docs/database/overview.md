# Database Inventory

## Runtime database layout

All SQLite files live in `${LIKI_DATA_DIR}`.

| File | Owner | Purpose |
| --- | --- | --- |
| platform.db | auth / payment / admin | users, auth sessions, payment intents/refunds, fulfillment tasks, payment events, admin users/capabilities/audit |
| profiles.db | profile | customer profiles |
| referral.db | referral/partnership | partners, attribution, applications, commission ledger, payouts, commission/refund intake, admin actions |
| liki.db | liki product | entitlements, orders, chat sessions/messages |
| aipay.db | aipay product | voluntary AI-pay orders |
| feedback.db | feedback | customer feedback |

## Historical archive

The retired naming / mingshu / hepan product databases (`naming.db`,
`mingshu.db`, `hepan.db`) are no longer opened by the runtime. Local copies may
remain in `${LIKI_DATA_DIR}` from before the product removal; they are inert.

## Cross-database rules

1. Foreign keys are valid only within one database file.
2. Cross-domain IDs are plain TEXT and validated by the owning service.
3. Financial state changes must be idempotent and audited.

## Schema design notes

Normalisation deviations are deliberate; none are accidental:

| Deviation | Location | Rationale |
| --- | --- | --- |
| Contact email snapshotted per context (users, auth_identities, auth_codes, payment_intents, entitlements, chat_orders, payment_notifications) | platform / liki | Each bounded context keeps its own contact copy; account erasure tombstones every copy while retaining terms for audit |
| Verbatim JSON payload | feedback.payload | Self-describing `feedback-v1` schema; the backend never interprets skill fields |
| JSON array column | entitlements.skills | Skill snapshot is frozen at grant time and never filtered in SQL |
| JSON extension object | payment_intents.metadata | Extension field for provider round-tripping |
| Redundant user_id on message rows | unified_chat_messages | Serves the per-user PII erasure sweep |

Naming and legacy notes:

- `unified_chat_*` keeps its prefix from the session-merge migration; it is
  historical, not a second chat model. Renaming is cosmetic and deferred.
- The legacy `products` table in old liki.db files is no longer created,
  seeded, or read; product truth is the in-code catalog. Existing copies are
  inert like the retired product databases.
- `aipay_orders.amount` mirrors the provider wire string (fixed 2-decimal);
  `amount_minor` is the canonical integer column derived at write time.
