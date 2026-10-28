# TODO

Updated: 2026-10-28

## Current checkpoint

- Branch: `feat/app-flow-redesign`
- PR: [#8](https://github.com/ml8s/liki-web/pull/8)
- State: BFF-only transport hardening implemented locally. `make check`,
  `make test`, `make e2e-a11y`, and the 22 targeted chat/domain-flow E2E
  journeys are green (2026-10-28).
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

Production validation is covered by the local full-matrix test run; there is
no separate staging environment. The two-layout coexistence (Consultation at
`/app`, Classic at `/classic`) is the deliberate product architecture — see
the decision note below.

## Decision: two-layout coexistence is permanent, not a migration phase

The original redesign plan framed Classic as a temporary rollback surface
with eventual cleanup (removing `AppSidebar`/`AppTabbar`, old tool shells,
duplicate projections, dead locale keys). That framing is **retired**: the
product ships two parallel layouts by design.

Why keeping both is correct:

1. **No staging environment exists.** Local full-matrix E2E is the release
   gate, so there is no rollout-evidence pipeline that could justify retiring
   one layout. "Wait for evidence" would mean waiting forever.
2. **`huangli` compatibility requires Classic regardless.** Until the backend
   migrates date-selection ownership to `natal`, Classic is the only surface
   that serves it.
3. **Coexistence is cheap.** The O4 audit proved the layouts are fully lazy:
   Consultation (`/app`, 646 KiB JS) and Classic dashboard (`/classic`,
   323 KiB) never enter each other's route graphs, and the Wave 4 engine
   catalog already shares result components across both.
4. **They serve different jobs.** Consultation is the guided four-domain
   product; Classic is the direct tool surface. Removing either removes
   product capability, not dead code.

Consequences:

- Do **not** schedule the Phase 6 "cleanup" deletions (`AppSidebar`,
  `AppTabbar`, classic tool shells, locale keys used by Classic). They are
  live surface, not dead code.
- Behavior changes must keep both layouts working; shared logic continues to
  live in composables and the engine-result catalog.
- The only remaining long-term backend dependency is the `huangli` → `natal`
  ownership migration, which changes Classic's date-selection entry point —
  it does not remove Classic.

## Next: manual validation and remaining polish

The O5/O6 implementation slice is done; the remaining items in
[`docs/frontend/app-flow-optimization-plan.md`](./docs/frontend/app-flow-optimization-plan.md).

### Pending (requires manual passes or backend work)

- [x] O1 smoke coverage (marketing handoff, login, payment return, restore,
      four-domain send, history, Me, entitlement lock, offline) — every item
      maps to an existing local Playwright journey (`tool-handoff`,
      `login-journey`, `payment-return`, `purchase-unlock`,
      `domain-flow*`, `chat-list-context`, `domain-flow-resilience`); the
      full local suite run is the release gate.
- [ ] O1 production telemetry review for the four-domain funnel and redirect
      counts (optional signal once real traffic exists; not a gate).
- [ ] O2 manual assistive-technology review: one screen reader per major
      platform, keyboard journeys on real devices.
- [ ] O5 five-breakpoint visual review (320/390/768/1024/1440) on real
      devices; transition feel sign-off.
- [x] O6 engine-result catalog: audited 2026-10-27 — both layouts already
      consume the same canonical panels and typed domain functions; the
      remaining per-view orchestration differs by design (see the O6
      appendix in `docs/frontend/o4-performance-audit.md`). Closed.
- [ ] Deferred backend phase: domain-owned consultation contexts and
      date-selection ownership. Settled direction (2026-10-27): when this
      phase is resourced, model 择日 as an independent job-shaped agent
      (option B, `docs/frontend/section7-decisions.md` §8); until then the
      shipped explicit pending-backend behavior stays.

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
3. Keep `huangli` / personalized date selection explicitly pending backend.
   Decision (2026-10-27): no `natal` mapping, no fifth agent for now; the
   independent 择日 agent (option B) is the default assumption for the eventual
   backend phase (see `docs/frontend/section7-decisions.md` §8).
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
