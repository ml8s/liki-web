# Referral DB

Owner: referral/partnership domain  
Engine: SQLite (`referral.db`)  
Migration owner: `internal/referral`

## Tables

### referrers

A partner able to share `/r/{code}`.

| Field | Meaning |
|---|---|
| user_id | Unified `platform.db.users.id`; soft reference |
| code | Public link token |
| role | `kol` or `scout` |
| status | `active` / `frozen` |
| recruiter_id | Scout that recruited this KOL; soft reference |

### partner_roles

Role capabilities are separate from the primary display role. A promoted KOL
keeps a retired KOL capability for pending historical earnings, but cannot open
the KOL console.

| Field | Meaning |
|---|---|
| user_id | Unified `platform.db.users.id`; soft reference |
| role | `kol` / `scout` |
| status | `active` / `frozen` / `retired` |
| started_at / ended_at | Role capability lifetime |

Payout creation accepts `active` or `retired`; console access accepts only
`active`.
### referral_attributions

First-touch attribution from a valid `/r/{code}` visit to a newly created account.

One row per attributed user.

### referral_applications

Scout applications.  
Status: `pending`, `approved`, `rejected`.

One pending application per user.

### commission_intents

Durable intake for fulfilled Dodo/Xunhu payments.  
Status: `queued`, `running`, `succeeded`, `failed`, `skipped`.

Eligible product is `liki-chat`. Unsupported providers/products remain as auditable `skipped` rows.

### referral_commissions

The original sale-level commission fact: payment identity, referrer/recruiter,
gross commission rates and cumulative refunded sale amount. It is not the
payable owner record.

### referral_commission_entries

One payable entry per beneficiary. A sale can have an independent KOL entry and
Scout entry; either can be refunded, locked into a payout, rejected, or marked
paid without changing the other.

| Status | Meaning |
|---|---|
| pending | fulfillment succeeded, not withdrawn |
| paid | included in a marked-paid payout |
| voided | payment refunded or manually voided |

`amount_minor - refunded_minor` is the currently payable net. `payout_id` locks
only that beneficiary entry.

### commission_refund_intents

Durable hand-off for verified payment refunds. Unique by provider refund ID.
Status: `queued`, `running`, `succeeded`, `skipped`, `failed`.

### referral_payouts

Partner withdrawal request.

| Status | Meaning |
|---|---|
| requested | created by partner; pending finance review |
| approved | approved for manual payment |
| rejected | rejected; commissions released |
| paid | manually paid; linked commissions marked paid |
| failed | payment attempt failed |

`requested_amount_minor` is immutable and records what the partner requested.
`amount_minor` is the current payable amount and can be reduced by refund. If
the net drops to zero or below the payout minimum, finance review is rejected
and non-voided entries are released.

`beneficiary_role` makes the payout owner explicit. `account` stores an AES-GCM
sealed envelope, never plaintext payment credentials.

### referral_payout_adjustments

Append-only money movement explanation. Every refund-driven payout change
records before amount, after amount, reason and actor.

### commission_entry_refunds

Per-entry refund allocation ledger. `(entry_id, provider_refund_id)` is unique.

### finance_tasks

Human finance queue for refunds against paid commission entries.

| Field | Meaning |
|---|---|
| provider_refund_id | Provider refund that caused the task |
| entry_id | Commission entry requiring clawback |
| amount_minor | Amount assigned for finance review |
| status | `pending`, `resolved`, `rejected` |
| assignee / resolution_note | Auditable finance decision |

## Cross-database references

`user_id`, `referrer_id`, `recruiter_id`, and `buyer_user_id` are soft references  
to `platform.db.users.id`; no cross-database foreign keys are allowed.


## Migration notes

- `005_scout_role` renames historical `channel_partner` partners to `scout`.
- The same migration rebuilds referral applications with `type='scout'`.
- Do not reuse the legacy `channel_partner` value in new code.
- `006_commission_entries` splits sale commissions into beneficiary-owned
  entries. Historical payouts are attached only to the entry matching the
  payout owner's role.
- `007_commission_refund_intake` makes refund handling retryable and idempotent.
- `008_partner_roles` separates role capability from primary display role.
- `009_financial_ledger` adds failed refund intake, payout role/original amount,
  payout adjustments, refund allocations and encrypted payout accounts.
- `010_finance_tasks` creates an explicit queue for paid-entry refund clawback.
- `011_finance_task_erasure` permits `user_erasure_review`; account erasure freezes partners, redacts payout accounts after settlement, and queues unpaid obligations for finance.
