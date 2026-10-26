# TODO

Updated: 2026-10-27

## Current checkpoint

- Branch: `feat/app-flow-redesign`
- PR: [#8](https://github.com/ml8s/liki-web/pull/8)
- Latest commit: `5f95b35 fix(app): surface record save outcomes and artifact readiness`
- State: pushed and clean; CI `check`, `race`, and `test` are green.
- Backend contract: four experts (`natal`, `divination`, `fengshui`,
  `naming`) plus seven durable consultation context scenarios. The scenario
  list is not a seven-product API. `huangli` is still mapped to
  `divination` and remains explicitly pending backend ownership migration.
- Completed tracks:
  - Phase 0-6 consultation rollout.
  - O1 bounded rollout telemetry implementation.
  - O2 automated accessibility hardening slice.
  - O3 resilience and draft-safety hardening slice.
  - O4 performance and bundle discipline (see
    [`docs/frontend/o4-performance-audit.md`](./docs/frontend/o4-performance-audit.md)).
  - O5/O6 consultation batch: consultation-document empty state, context
    equality helpers moved to `@shared/domain/consultation.ts`, session
    submit routing extracted to `useConsultationSubmit`, history actions
    extracted to `useConsultationHistory` (view reduced from 979 lines).
  - O2 live-region audit fix: record saves and artifact readiness are now
    announced in the consultation flow (previously completely silent).

Production/staging validation and cleanup remain deferred.

## Next: manual validation and remaining polish

The O5/O6 implementation slice is done; the remaining items in
[`docs/frontend/app-flow-optimization-plan.md`](./docs/frontend/app-flow-optimization-plan.md).

### Pending (requires external environments or manual passes)

- [ ] O1 staging smoke checklist (marketing handoff, login, payment return,
      restore, four-domain send, history, Me, entitlement lock, offline).
- [ ] O1 production telemetry review for the four-domain funnel and redirect
      counts.
- [ ] O2 manual assistive-technology review: one screen reader per major
      platform, keyboard journeys on real devices.
- [ ] O5 five-breakpoint visual review (320/390/768/1024/1440) on real
      devices; transition feel sign-off.
- [ ] O6 engine-result catalog completion: page-local projections → typed
      adapters (behavior-neutral, schedule after staging sign-off).
- [ ] Phase 6 cleanup (remove AppSidebar/AppTabbar, dead locale keys,
      compatibility facades) — blocked on O1 staging sign-off and the
      `huangli` backend ownership migration.
- [ ] Deferred backend phase: domain-owned consultation contexts and the
      `huangli` → `natal` expert migration.

### Redesign plan section 7 (product/design decisions, not code)

The eleven open items in
[`app-flow-redesign-plan.md` §7](./app-flow-redesign-plan.md#7-open-items)
remain product decisions (restore windows, prompt budgets, catalog
sign-off, `bazi_liuri` contract exposure, etc.). They gate later code work,
not the current frontend batch.

## Hard constraints

1. Keep four consultation domains only: `natal`, `divination`, `fengshui`,
   and `naming`.
2. Keep Classic and the consultation context resolver until rollout evidence
   and the `huangli` ownership migration support cleanup.
3. Keep `huangli` / personalized date selection explicitly pending backend;
   do not map it to `natal` until the backend moves its scenario/expert
   ownership.
4. Reuse existing components, tokens, composables, and libraries.
5. Implement a whole batch first, run unified tests, then make one commit.
6. Do not delete compatibility surfaces during O4.

## Verification floor

Run after implementation, not per cosmetic edit:

```bash
make check
make test
cd web/app
VITE_FEATURE_APP_FLOW_REDESIGN=true npm run build
npx vite preview --host 127.0.0.1 --port 5185 --strictPort
cd ../..
BASE_URL=http://127.0.0.1:5185 npm run test:e2e --prefix web -- \
  e2e/journeys/domain-flow.spec.ts \
  e2e/journeys/domain-flow-casting.spec.ts \
  e2e/journeys/domain-flow-hepan.spec.ts \
  e2e/journeys/domain-flow-fengshui.spec.ts \
  e2e/journeys/domain-flow-naming.spec.ts \
  e2e/journeys/domain-flow-me.spec.ts \
  e2e/journeys/domain-flow-a11y.spec.ts \
  e2e/journeys/domain-flow-resilience.spec.ts \
  e2e/journeys/domain-flow-reflow.spec.ts \
  e2e/journeys/domain-flow-visual.spec.ts \
  e2e/journeys/domain-flow-rollout.spec.ts
```

## New-session prompt

```text
Continue Liki Web frontend optimization.

Read TODO.md and docs/frontend/app-flow-optimization-plan.md first.
The completed checkpoint is PR #8 at commit 69b22f3; CI is green.
Start O4 performance and bundle discipline.

First produce an audit and implementation plan. Do not optimize before
recording the baseline. Reuse existing assets. No ECharts, no generic JSON
renderer, no second design system. Keep Classic and the compatibility
resolver. Remember: the backend already has four experts; seven scenario IDs
are consultation context types, not seven skills.
Implement the approved batch, run the full verification matrix, then make one
commit.
```
