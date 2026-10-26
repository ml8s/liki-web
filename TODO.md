# TODO

Updated: 2026-10-27

## Current checkpoint

- Branch: `feat/app-flow-redesign`
- PR: [#8](https://github.com/ml8s/liki-web/pull/8)
- Latest commit: `69b22f3 feat(app): harden consultation accessibility and resilience`
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

Production/staging validation and cleanup remain deferred.

## Next: O4 — performance and bundle discipline

O4 is the next planned track in
[`docs/frontend/app-flow-optimization-plan.md`](./docs/frontend/app-flow-optimization-plan.md).
Do an audit first; do not start optimizing before the baseline is recorded.

### O4 TODO

- [ ] Confirm PR #8 review/staging status before baselining a new branch.
- [ ] Build the Consultation preview with
      `VITE_FEATURE_APP_FLOW_REDESIGN=true`.
- [ ] Record route-level performance baselines for `/app`, `/app/me`, and a
      consultation session:
  - [ ] First load and hydration.
  - [ ] LCP and INP on a mid-tier mobile profile.
  - [ ] Consultation send and streaming interaction.
  - [ ] Initial JS/CSS assets per route.
- [ ] Audit current bundle composition:
  - [ ] `chat` chunk.
  - [ ] `AppMarkdown` and markdown dependencies.
  - [ ] Element Plus imports.
  - [ ] Locale chunk loading.
  - [ ] Artifact/engine modules.
  - [ ] Duplicate runtime dependencies.
- [ ] Verify these stay out of the initial route graph:
  - [ ] Admin views.
  - [ ] Partner views.
  - [ ] Chart/engine implementations not required by the visible route.
  - [ ] Heavy locale/content modules.
- [ ] Audit lazy boundaries:
  - [ ] Route-level code splitting.
  - [ ] Artifact expansion.
  - [ ] Chat transport.
  - [ ] Markdown rendering.
- [ ] Define explicit budgets for:
  - [ ] Initial JS/CSS.
  - [ ] Critical Consultation route chunks.
  - [ ] Chat/Markdown chunks.
- [ ] Record intentional trade-offs and reject unbounded regressions.
- [ ] Implement only the highest-impact, low-risk improvements.
- [ ] Run the full verification matrix after the batch.

### O4 acceptance criteria

- No duplicated chart, markdown, runtime, or locale dependency enters the
  initial bundle.
- Artifact expansion does not regress initial page load.
- Performance and frontend budget checks remain green.
- New budgets are explicit and checked in CI.
- Classic and Consultation both continue to work.
- No ECharts, generic JSON renderer, or second design system is introduced.

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
