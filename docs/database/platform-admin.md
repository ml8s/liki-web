# Platform DB — Admin Auth & Audit

Owner: platform/security  
Engine: SQLite (`platform.db`)  
Migration owner: `internal/adminauth`

## admin_users

| Field | Meaning |
| --- | --- |
| id | Internal admin record ID |
| email | Unified app login email; immutable business key |
| user_id | Bound `platform.db.users.id`; bound on first authorized session |
| status | `active` / `disabled` |

## admin_capabilities

Composite key: `(admin_email, capability)`.  
A capability is effective only when `status='active'` and `admin_users.status='active'`.

Internal capabilities:

```text
admin.super
partner.read
partner.approve
partner.freeze
partner.unfreeze
commission.read
commission.void
payout.read
payout.approve
payout.reject
payout.mark_paid
audit.read
admin.user.manage
```

## admin_audit_logs

Append-only record of privileged access and state changes.

Required fields: actor email, action, object, reason.  
Optional: actor user id, capability, before/after, request id, hashed IP, user agent.
