# App flow unified optimization plan

Status: planning follow-up to
[App flow redesign plan](./app-flow-redesign-plan.md). The consultation
implementation and Phase 6 rollout redirects are complete; this plan sequences
hardening and quality work without reopening the settled product IA.

## 1. Baseline and constraints

The current branch keeps two intentional experiences:

- Consultation is the flagged four-domain product surface:
  `natal`, `divination`, `fengshui`, and `naming`.
- The backend already has four matching experts. It also retains seven
  consultation context scenarios for structured tool input; these are context
  types, not seven product skills.
- Classic is the rollback and compatibility surface. It remains intact until
  rollout is stable and `huangli` context ownership is migrated from
  `divination` to `natal`.

The current gate baseline is:

| Gate | Result |
| --- | --- |
| App unit/component tests | 100 files and 441 tests passed |
| App accessibility E2E | 38 tests passed, including Consultation, Classic, Admin, and mobile smoke |
| Static web checks | Route contract, token, i18n, responsive, CSS, performance, and budget checks passed |
| Rollout route contract | 36 App routes checked |
| Pre-push gate | `make check` and `make test` passed before the up-to-date remote check |

Constraints for every follow-up:

1. Do not add a fifth consultation domain.
2. Do not delete Classic, the consultation context resolver currently named
   `LegacyIntentBridge`, or compatibility routes yet.
3. Do not pretend `huangli` / personalized date selection has migrated; its
   natal-expert migration is a backend contract.
4. Put reusable behavior in shared components, composables, tokens, or
   contracts before adding another view-local implementation.
5. Keep behavior-changing PRs separate from behavior-neutral extraction PRs.

## 2. Optimization tracks

### O1 — Stabilize the rollout surface

Goal: prove that the new default is safe before any cleanup.

The first implementation adds the bounded `/api/analytics/product-event`
beacon and Consultation entry, domain/person selection, restore, submit,
bridge, failure, and recovery counters. Remaining work is production review and
staging validation.

Work:

1. Extend production telemetry review for consultation entry, domain selection,
   person restore, seed bridge result, pending-backend `huangli`, failure
   class, and recovery action.
2. Track redirect use from `/app/hepan`, `/app/divination`,
   `/app/fengshui`, `/app/liunian`, and `/app/naming`.
3. Review client errors and funnel counters by feature flag, route class,
   browser, viewport, and auth state.
4. Run a staging smoke checklist for marketing handoff, login, payment return,
   restore, four-domain send, history, Me, entitlement lock, and offline state.

Exit criteria:

- The four-domain funnel and bridge outcomes are observable.
- Every consultation failure can be classified without guesswork.
- Rollout redirects have stable production counts and no unexpected Classic
  fallback spike.
- Staging sign-off is recorded before cleanup discussion.

### O2 — Complete accessibility

Goal: move from automated AA coverage to practical assistive-technology
quality.

The first hardening slice adds a screen-reader consultation-context
announcement, programmatic focus for consultation errors, and an E2E assertion
that closing the person sheet restores focus to its trigger. The remaining
manual assistive-technology review is still required.

Work:

1. Audit live regions across streaming, artifact computation, restore, save,
   entitlement, offline, and pending-backend states.
2. Audit focus restoration for sheets, artifact expansion, route redirects, and
   failed submissions.
3. Complete keyboard journeys for domain chooser, person chooser, focused
   forms, composer, artifact controls, and history filters.
4. Test one screen reader per major platform and record known limitations.
5. Keep accessibility primitives in shared components so Classic benefits too.

Exit criteria:

- Keyboard-only users can complete all four domains without hidden traps.
- Focus is never lost or sent to a decorative container.
- All async state changes have an appropriate accessible announcement.
- Existing axe, reflow, 320px, and keyboard E2E remain green.

### O3 — Resilience and draft safety

Goal: make transient network and lifecycle failures harmless.

The first hardening slice adds shared request timeouts, retry only for
idempotent reads, a single-flight consultation submit guard, an unload guard
for unsent drafts, accessible draft-preservation announcements, and E2E
coverage for creation failure and unreachable streaming.

Work:

1. Define one shared API timeout and retry policy for idempotent reads; keep
   non-idempotent consultation sends explicit and recoverable.
2. Preserve domain drafts across offline, refresh, login expiry, and failed
   session creation.
3. Add E2E coverage for permission failure, service unreachable, offline
   send, payment return interruption, and restore after refresh.
4. Ensure retry never duplicates a paid or consultation submission.
5. Preserve raw errors in stores while showing only safe localized categories.

Exit criteria:

- A failed send never silently loses user text or selected person/domain.
- Refresh after an interrupted payment or session creation has a deterministic
  recovery path.
- Failure-state tests cover all `resourceStatus` categories.

### O4 — Performance and bundle discipline

Goal: keep the consultation fast while Classic and Consultation coexist.

Work:

1. Capture route-level baseline metrics for first load, hydration, LCP, INP,
   and consultation send on a mid-tier mobile profile.
2. Audit the current `chat` and `AppMarkdown` chunks for duplicate
   dependencies, eager imports, locale loading, and markdown extensions.
3. Keep artifact engines, charts, admin views, partner views, and heavy locale
   content out of the initial route graph.
4. Prefer lazy module boundaries and typed imports over a new rendering
   library.
5. Add performance budgets for initial assets and critical Consultation route
   chunks before adding new capabilities.

Exit criteria:

- No duplicated chart/markdown/runtime dependency enters the main bundle.
- Artifact expansion does not regress initial page load.
- Performance and frontend budget checks remain green with explicit new
  baselines for intentional trade-offs.

### O5 — Consultation UX polish

Goal: improve comprehension and motion without changing the IA.

Work:

1. Review person home, active reading, follow-up questions, artifact capsules,
   and action hierarchy at 320, 390, 768, 1024, and 1440px.
2. Harmonize empty, loading, locked, error, pending-backend, and recovery
   visuals through shared state components.
3. Polish focused-form progression: context, required slots, validation,
   submission, artifact handoff, and return to consultation.
4. Reduce sheet and capsule transition cost; respect reduced-motion and avoid
   animating layout properties.
5. Validate copy for uncertainty, domain limits, and next actions.

Exit criteria:

- The primary consultation document remains the only main scroll owner.
- Artifacts remain reference layers and never compete with the reading.
- A first-time user can understand person, domain, question, result, and next
  step without product training.

### O6 — Maintainability and result reuse

Goal: reduce accidental complexity only after behavior is locked.

Work:

1. Split `DomainFlowView` further only where it still composes stores,
   routing, and session orchestration in one place.
2. Extract repeated follow-up matching, context projection, and status
   mapping into tested composables or domain helpers.
3. Continue the engine-result catalog work; page-local projections should
   become typed adapters, not copied renderers.
4. Keep component APIs small and contract-tested.
5. Remove dead code only when redirect telemetry and backend migration plans
   prove the compatibility surface is no longer needed.

Exit criteria:

- Every reusable mechanism has one canonical home.
- Result rendering has one typed source of truth per engine family.
- New features compose shared primitives instead of expanding view-local
  logic.

## 3. Delivery order

| Order | Track | Reason |
| --- | --- | --- |
| 1 | O1 rollout telemetry and staging validation | safety evidence gates cleanup |
| 2 | O2 accessibility completion | correctness is part of release quality |
| 3 | O3 resilience and draft safety | prevents irreversible user-data loss |
| 4 | O4 performance baseline and bundle audit | coexistence cost must be measured |
| 5 | O5 UX polish | best done on stable contracts |
| 6 | O6 maintainability | behavior-neutral cleanup comes after stabilization |

O1 and O2 may run in parallel when owners and fixtures are separate. O3 may
begin with tests while O2 owns interactive focus work. O4 should establish its
baseline before O5 changes visuals. O6 is intentionally last.

## 4. Unified verification matrix

Run all verification after a batch, not per cosmetic edit:

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

Every PR must state:

1. Whether behavior is unchanged.
2. Which track and acceptance criterion it advances.
3. Which shared mechanism absorbs the change.
4. Which compatibility or rollback surface remains.
5. Which tests and telemetry prove the outcome.

## 5. Explicit non-goals

1. Do not introduce ECharts for existing chart components.
2. Do not build a generic JSON-to-UI renderer.
3. Do not create a second design system.
4. Do not replace free-form consultation with automatic text classification.
5. Do not remove the seven marketing islands; they remain acquisition language
  and hand off to the four domains.
6. Do not move `huangli` into another visible domain before the backend owns
  date selection and Bazi/Ziwei day charts.
