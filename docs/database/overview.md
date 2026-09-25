# Database Inventory

## Runtime database layout

All SQLite files live in `${LIKI_DATA_DIR}`.

| File | Owner | Purpose |
| --- | --- | --- |
| platform.db | auth / payment / admin | users, auth sessions, payment intents/refunds, fulfillment tasks, payment events, admin users/capabilities/audit |
| profiles.db | profile | customer profiles |
| referral.db | referral/partnership | partners, attribution, applications, commission ledger, payouts, commission/refund intake, admin actions |
| liki.db | liki product | products, entitlements, orders, chat sessions/messages |
| naming.db | naming product | naming orders/messages/codes |
| mingshu.db | mingshu product | orders/reports |
| hepan.db | hepan product | orders/reports |
| aipay.db | aipay product | voluntary AI-pay orders |
| feedback.db | feedback | customer feedback |

## Cross-database rules

1. Foreign keys are valid only within one database file.
2. Cross-domain IDs are plain TEXT and validated by the owning service.
3. Financial state changes must be idempotent and audited.
