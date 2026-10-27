# Redesign plan §7 — decision draft

Status: proposal 2026-10-27. Each open item from
`app-flow-redesign-plan.md` §7 lists the implemented reality, options, and a
recommendation. Approval turns an item into code work or closes it.

## 1. Reader threshold (lines/chars) and Markdown-heading section detection

Reality: the consultation experience is a scrolling document
(`ConsultationDocument`); there is no paginated "reader mode" to threshold.
Recommendation: **close as resolved by design** — the document model made the
reader question obsolete. Revisit only if a print/long-read mode ships.

## 2. Prompt-side length budget per domain

Reality: the frontend sends free-form composer text and focused-form context;
no per-domain prompt budget exists client-side, and prompt assembly belongs to
the experts (backend). Recommendation: **assign to backend** expert design.
Frontend keeps a soft composer cap only if abuse becomes measurable.

## 3. `LegacyIntentBridge` product thresholds

Reality: shipped and contract-tested — explicit seeds/slot-forms map through
`scenarios.json`; outcomes are `ok` / `cross-domain` / `pending-backend` /
`invalid`. Recommendation: **keep the current contract**. Revisit only when
the `huangli` → `natal` migration changes the scenario table.

## 4. Restore rule when the last session ended more than N days ago

Reality: `restoreLatest` restores the newest visible session regardless of
age. Recommendation: **restore within 7 days of `updated_at`; otherwise start
fresh** (7 days matches weekly-consultation habits and avoids reviving stale
context). Small change in `useConsultationHistory` + locale copy; telemetry
gets a `stale` restore status.

## 5. Session "active window" for chips/restore

Reality: no window — the latest session is always the restore candidate.
Recommendation: **adopt the same 7-day window as item 4** so "active" and
"restore" share one definition.

## 6. Server-side history contract (content search, per-profile pagination)

Reality: title filtering is client-side over loaded sessions; volumes are
small. Recommendation: **defer** until a profile exceeds ~100 sessions in
practice; revisit with the backend sessions API.

## 7. Person home composition without an active session

Reality: shipped minimal — person/domain context plus the empty-state
guidance added in the O5/O6 batch. Recommendation: **close as shipped**;
suggested questions are a future O5+ enhancement once backend prompt support
exists (depends on item 2).

## 8. Date-selection artifact copy and ranking model

Reality: blocked by the backend — date selection and Bazi/Ziwei flow-day
ownership have not migrated to `natal`. Recommendation: **keep pending**;
resolved inside the deferred backend phase.

## 9. Web contract exposure for `bazi_liuri` / `ziwei_liuri`

Reality: not exposed in `web/shared/api/engine` (no liuri tools); depends on
the same backend capability move as item 8. Recommendation: **keep pending**
with the backend phase; DTO shape should mirror the existing `bazi` engine
adapter pattern.

## 10. Catalog inventory sign-off

Reality: `web/shared/components/engine/catalog.ts` covers the shipped result
families; every panel is contract-tested and lazy-loaded (O4). Recommendation:
**sign off the current catalog**; new families must add an
`engine-contract` entry and a lazy `ArtifactPanel` branch in the same PR.

## 11. `bazi_yongshen` placement: engine catalog vs separate counsel catalog

Reality: yongshen ships inside the `bazi` engine family and renders within
`BaziFullChartPanel` context; no second counsel result family exists.
Recommendation: **keep it in the engine catalog**. Split into a counsel
catalog only if a second `/counsel/mcp` result family materializes.

## Approval shortcuts

- Items 1, 7, 10, 11: close/sign-off with no code change.
- Items 2, 6: defer to backend/scale triggers, no code change now.
- Items 3, 8, 9: tied to the deferred backend phase.
- Items 4, 5: small frontend batch (`useConsultationHistory` + copy +
  telemetry) — ready to implement on approval.
