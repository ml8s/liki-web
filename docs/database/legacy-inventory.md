# SQLite legacy inventory

Inventory date: 2026-10-24  
Scope: production `/opt/liki/data`.

## Removed

| Object | State before cleanup | Action |
| --- | ---: | --- |
| `profiles.db: birth_profiles_legacy` | 65 rows | Dropped by `010_drop_birth_profile_legacy_archive`; canonical `birth_profiles` is authoritative. |
| `hepan.db`, `mingshu.db`, `naming.db` | all product tables had 0 rows | Retired product databases; not opened by runtime. Deployment removes them only after a successful pre-deploy backup. |

## Removed from runtime

| Runtime fallback | Replacement |
| --- | --- |
| feedback legacy `category` / `message` migration | Canonical `feedback.payload` + content hash only. |
| payment `fulfillment_tasks` compatibility rename | Fresh databases create canonical `payment_completion_tasks` directly. |

## Kept

| Object | Reason |
| --- | --- |
| `platform.db: fulfillment_tasks` history renamed into `payment_completion_tasks` | Existing payment completion rows are operational audit/state. |
| financial partner tables prefixed `partner_*` | These are current canonical names, not legacy compatibility. |

No runtime code may read or write the removed objects. If a retired object is
rediscovered, treat it as a defect and add it to this inventory.
