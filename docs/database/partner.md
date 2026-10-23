# Partner DB

Owner: partner/partnership domain
Engine: SQLite (`referral.db`)  
Migration owner: `internal/partner`

## Tables

### partners

A partner able to share `/r/{code}`.

| Field | Meaning |
| --- | --- |
| user_id | Unified `platform.db.users.id`; soft reference |
| code | Public link token |
| role | `kol` or `scout` |
| status | `active` / `frozen` |
| recruiter_id | Scout that recruited this KOL; soft reference |

### partner_roles

Role capabilities are separate from the primary display role. KOL and Scout are
independent roles, not levels; one role is never upgraded into the other. A
historical role can be retired and then keeps only settlement capability for
earnings that already exist.

| Field | Meaning |
| --- | --- |
| user_id | Unified `platform.db.users.id`; soft reference |
| role | `kol` / `scout` |
| status | `active` / `frozen` / `retired` |
| started_at / ended_at | Role capability lifetime |

Payout creation and payout history accept `active` or `retired`. Partner
dashboard access accepts only `active`, and a frozen partner cannot withdraw.
Role applications require review and do not require product purchase.
Settlement is a money-only view: balances and role-scoped payouts, without
dashboard or recruiting data.

### partner_attributions

Attribution from the latest valid `/r/{code}` visit before a newly created
account is authenticated.

One row per attributed user.

### partner_applications

KOL and Scout applications.
Status: `pending`, `approved`, `rejected`.

One pending application per user.

### partner_commission_intake

Durable intake for fulfilled Dodo/Xunhu payments.  
Status: `queued`, `running`, `succeeded`, `failed`, `skipped`.

Eligible product is `liki-chat`. Unsupported providers/products remain as auditable `skipped` rows.

### partner_commissions

The original sale-level commission fact: payment identity, referrer/recruiter,
gross commission rates and cumulative refunded sale amount. It is not the
payable owner record.

### partner_commission_entries

One payable entry per beneficiary. A sale can have an independent KOL entry and
Scout entry; either can be refunded, locked into a payout, rejected, or marked
paid without changing the other.

| Status | Meaning |
| --- | --- |
| pending | fulfillment succeeded, not withdrawn |
| paid | included in a marked-paid payout |
| voided | payment refunded or manually voided |

`amount_minor - refunded_minor` is the currently payable net. `payout_id` locks
only that beneficiary entry.

### partner_commission_refund_intake

Durable hand-off for verified payment refunds. Unique by provider refund ID.
Status: `queued`, `running`, `succeeded`, `skipped`, `failed`.

### partner_payouts

Partner withdrawal request.

| Status | Meaning |
| --- | --- |
| requested | created by partner; pending finance review |
| approved | approved for manual payment |
| rejected | rejected; commissions released |
| paid | manually paid; linked commissions marked paid |
| failed | payment attempt failed |

`requested_amount_minor` is immutable and records what the partner requested.
`amount_minor` is the current payable amount and can be reduced by refund. If
the net drops to zero or below the payout minimum, finance review is rejected
and non-voided entries are released.

`beneficiary_id` and `beneficiary_role` make the payout owner explicit. `account` stores an AES-GCM
sealed envelope, never plaintext payment credentials.

### partner_payout_adjustments

Append-only money movement explanation. Every refund-driven payout change
records before amount, after amount, reason and actor.

### partner_commission_entry_refunds

Per-entry refund allocation ledger. `(entry_id, provider_refund_id)` is unique.

### partner_finance_tasks

Human finance queue for refunds against paid commission entries.

| Field | Meaning |
| --- | --- |
| provider_refund_id | Provider refund that caused the task |
| entry_id | Commission entry requiring clawback |
| amount_minor | Amount assigned for finance review |
| status | `pending`, `resolved`, `rejected` |
| assignee / resolution_note | Auditable finance decision |

## Cross-database references

`user_id`, `referrer_id`, `recruiter_id`, and `buyer_user_id` are soft references  
to `platform.db.users.id`; no cross-database foreign keys are allowed.

## Naming rule

Do not reuse the retired `channel_partner` value in new code. `013_canonical_partner_tables` renamed the historical referral tables to the current partner-owned names; 001 still creates those historical names for fresh databases before 013 runs.
