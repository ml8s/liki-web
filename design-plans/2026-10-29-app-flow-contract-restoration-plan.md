# App Flow contract restoration plan

Written against: `9621536`

Status: implementation plan, not yet approved for cleanup. The goal is to make the default Consultation experience satisfy the reviewed interaction contract in `docs/frontend/app-flow-redesign-plan.md` before further UX polish or dead-code cleanup.

Execution status (2026-10-30): P0–P7 are implemented in the working tree.
App unit tests (494), web static tests, focused/full Consultation E2E (39),
production build, lint, typecheck, docs lint, route contracts, and web-static
gates pass. Rendered checks: 320px draft 180px / active 136px; 1024px peek has
a 480px document column and pinned keeps the 400px minimum; sidecar widths are
320/384px and reading measure is capped at 760px.

## Outcome

1. `/app` is canonical Consultation; `/app/consultation` is only a preserving alias.
2. Opening the app restores the latest visible `(person, domain)` consultation without losing cross-person or cross-domain sessions.
3. URL state is deterministic: safe `domain` and transient `seed` context may be in the URL; person and session IDs never enter the URL.
4. Mobile restores the reviewed context/document/action zones and 136/180px fixed-chrome budgets.
5. Desktop restores context rail / consultation document / artifact sidecar, with a minimum 400px document at 1024px.
6. No-profile users get a profile-free success path led by divination.
7. Artifacts use inline turn anchors, mobile detents, and desktop closed/peek/pinned states.
8. Focused forms behave as domain slot-filling, not tool pages inside the flow.
9. Dynamic i18n keys are guarded, and Partner Center no longer renders raw translation keys.

## Evidence chain

- Surface: logged-in Vue Consultation at `/app`, plus Partner Center copy.
- Problem: implementation drifted from the reviewed contract while existing tests and visual snapshots encoded the drift.
- Design evidence: `docs/frontend/app-flow-redesign-plan.md` §2–§3 and §6; rendered audit at commit `9621536`.
- Owner: `web/app/src/views/DomainFlowView.vue`, `web/app/src/components/flow/**`, `web/app/src/router/appRoutes.ts`, `web/app/src/composables/useConsultationHistory.ts`, `web/app/src/stores/domain-flow.ts`, `web/routes/app.contract.json`.
- Confirmed defects:
  - 320px fixed chrome measured 405px versus the documented 180px draft budget.
  - Active-session domain strip remained visible.
  - `/app/consultation` became the final canonical-looking URL.
  - Domain selection did not update URL and reverted after refresh.
  - The latest divination session was not restored without an explicit domain query.
  - A restored divination session rendered a new casting form instead of its consultation document.
  - Desktop sidecar compressed the 1024px consultation document to 377px.
  - Partner Center rendered `partner.ruleKol1`, `partner.ruleKol2`, `partner.ruleScout1`, and `partner.ruleScout2` literally.
- Uncertainty: the reviewed plan leaves the active-session window open. Preserve current recency behavior until product explicitly approves an expiry period.

## Design decision

Treat the reviewed redesign document as the acceptance contract, while keeping the two-template rollback decision as an explicit exception:

- Consultation owns `/app` by default when the build flag is enabled.
- Classic remains the rollback surface at `/app/classic`.
- A user-selected Classic preference may redirect `/app` to `/app/classic`, but Consultation's canonical route remains `/app`; `/app/consultation` redirects back to `/app` while preserving query and hash.

Restore behavior is session-first, not profile-list-first. On boot, select the latest restorable session, project its `profile_id` and visible domain into flow state, then open it. Only when no session is restorable should the UI select the most recently used profile and the appropriate empty-state domain.

Do not introduce a new design system, dependency, fifth domain, chat protocol, or generic JSON-to-UI renderer.

## Reuse

- `domainForSession` for projecting persisted sessions to visible domains.
- `bridgeLegacyIntent` for deterministic legacy skill routing.
- `AppLiveRegion`, `AppStatusAlert`, and existing skip/focus behavior.
- `FlowSheet`, `ArtifactCapsule`, `ArtifactSurface`, and typed artifact descriptors.
- Shared engine panels and form components; do not duplicate projections.
- Existing Vue Router, Pinia, Element Plus, VueUse, markdown-it, DOMPurify, and AG-UI dependencies.
- Existing tokens in `web/css/tokens.css` and semantic aliases in `web/app/src/styles/**`.

If mobile needs an `AppFlowContextMenu`, it must be flow-shell-owned, not a second navigation system. If detents need a `FlowArtifactSheet`, wrap or replace `FlowSheet` behavior rather than adding another modal system.

## Delivery strategy

Implement in PR-sized phases. Every phase must keep Classic rollback and existing marketing handoffs green. Do not combine cleanup with behavior restoration.

### P0 — Partner i18n hotfix

Goal: stop rendering machine keys.

Files:

1. `web/app/src/locales/zh-Hans.json`
   - Restore `partner.ruleKol1`, `partner.ruleKol2`, `partner.ruleScout1`, `partner.ruleScout2`.
2. `web/app/src/locales/zh-Hant.json`
   - Restore equivalent traditional-Chinese strings.
3. `web/app/src/locales/en.json`
   - Restore equivalent English strings.
4. `web/app/src/tests/i18n-contract.test.ts`
   - Add an explicit dynamic-key contract for role-constructed partner keys.
5. `docs/frontend/subtraction-review-2026-10-27.md`
   - Correct the false zero-reference record.

Preserve existing `partner.rule3` and `partner.rule4`, role isolation, and invite-only behavior.

Exit:

- KOL and Scout Centers show four human-readable rules in all locales.
- All three locale files contain the same restored keys.
- A missing dynamic key fails the frontend unit gate.

### P1 — Route and restore contract

Goal: make Consultation state deterministic before visual rework.

Files:

1. `web/app/src/router/appRoutes.ts`
   - Register `/app` as Consultation when enabled.
   - Make `/app/consultation` a preserving alias/redirect to `/app`.
   - Keep `/app/classic` and every existing compatibility redirect.
   - Respect user-selected Classic preference without changing Consultation's canonical route.
2. `web/routes/app.contract.json`
   - Update route and experience metadata.
3. `web/scripts/check-route-contracts.mjs`
   - Enforce the reviewed canonical route.
4. `web/app/src/tests/router/app-routes.test.ts`
   - Replace the “stable canonical path = consultation” assertion.
5. `web/app/src/views/DomainFlowView.vue`
   - Sync selected domain into the URL.
   - Remove `seed` / `service` / `scenario` when they no longer belong to the selected domain.
   - Preserve locale/query/hash where applicable.
   - Never place `person` or `sessionId` in the URL.
6. `web/app/src/composables/useConsultationHistory.ts`
   - Add session-first restore.
   - Project the selected session’s person and visible domain before opening it.
   - Keep person/session binding immutable.
7. `web/app/src/stores/domain-flow.ts`
   - Keep domain-keyed drafts; do not add session ownership here.

Required behavior:

- `/app?domain=divination` opens divination.
- Selecting 问卦 updates to `/app?domain=divination` without reload; refresh restores it.
- `/app` restores the latest session even if it belongs to a non-first profile or another domain.
- A restored divination session with a casting receipt renders its consultation document and artifact, not a fresh casting form.
- Switching person switches the visible view and never rewrites `session.profile_id`.
- `/app/hepan`, `/app/divination`, `/app/fengshui`, `/app/liunian`, `/app/naming`, and Classic compatibility redirects remain correct.

Tests:

- Canonical/alias/query preservation.
- Domain URL sync and stale seed cleanup.
- Session-first restore across person and domain.
- Divination restore versus new casting.
- Person switching does not mutate binding.

Exit: `npm run routes:check` passes, and new route/restore E2E fails against the old implementation and passes after the fix.

### P2 — No-profile success path

Goal: prevent profile creation from becoming the first-run gate.

Files: `ConsultationSpine.vue`, `DomainFlowView.vue`, and locale catalogues.

Behavior:

1. With no profiles, default the visible domain to 问卦.
2. Lead with one primary profile-free CTA: 问一卦.
3. Offer generic 黄历择日 as a secondary compatibility path where the backend contract allows it.
4. Present 建档 as the upgrade path for personal charts, annual readings, compatibility, naming, and fengshui—not the first action.
5. For partially available 命理, show the available capability and unlock reason.
6. Do not attach a person’s name to person-free divination or almanac output.

Preserve profile creation, entitlement/payment states, and `huangli` pending-backend behavior.

Exit: a new user can complete the profile-free path without first opening profile creation.

### P3 — Mobile shell and context contract

Goal: restore reviewed mobile zones and fixed-chrome budgets.

Files: `AppFlowShell.vue`, `ParameterComposer.vue`, `FlowDomainStrip.vue`, `DomainFlowView.vue`; add a flow-shell-owned context menu only if Element Plus cannot express the reviewed overflow behavior without becoming a second navigation system.

Behavior:

1. Active session hides the four-domain strip; the composer domain control remains.
2. Draft/person-home may show the strip.
3. The action dock contains only optional follow/new-content, parameter row, and input/action row.
4. Move “我的”, “经典版”, history, orders, account, and role-gated partner entry into context overflow.
5. Mobile context status remains noninteractive except restore and overflow.
6. At 320px enforce active-session fixed chrome ≤136px and draft/home ≤180px before safe-area inset.
7. Restore the horizontal strip contract: ≥44px targets, selected chip scrolled into view, next chip peeks 16–32px, no page-level horizontal scroll.
8. Use manual tab activation: arrows move focus and honor RTL; Enter/Space activates; Home/End jump; Tab reaches the document.
9. Keyboard-open hides restore/strip as documented and preserves the reading anchor.

Tests: bounding-box budgets at 320px, active strip absence, overflow destinations and role gating, LTR/RTL activation, longest labels, 200% reflow, focus order, reduced motion.

Exit: mobile E2E asserts documented budgets, not merely visible controls.

### P4 — Desktop shell and artifact sidecar

Goal: restore the desktop three-layer hierarchy.

Files: `AppFlowShell.vue`, `DomainFlowView.vue`, `ArtifactSurface.vue`, `useDomainArtifacts.ts`, `ConsultationDocument.vue`.

Behavior:

1. At 1024px use approximately 168–208px context rail, ≥400px consultation document, and 320–440px artifact layer.
2. The rail persistently exposes four domains plus mine/history/partner/account actions.
3. Artifact sidecar supports closed, peek, and pinned.
4. Peek and pinned are explicit user actions.
5. Closing restores focus to the originating capsule.
6. Artifact state remains scoped to `(person, domain)`.
7. Reading text is constrained to approximately 720–760px.
8. Wider viewports grow margin and layers according to the reviewed hierarchy; do not create unreadably long text.

Tests: 1024/1280/1440 measurements, minimum document width with sidecar open, reading measure, closed/peek/pinned interaction, person/domain scoping, focus return.

Exit: sidecar no longer compresses the 1024px document below 400px.

### P5 — Artifact disclosure and turn anchoring

Goal: make artifacts reference layers anchored to the consultation document.

Files: `ConsultationDocument.vue`, `ArtifactCapsule.vue`, `ArtifactSurface.vue`, `useDomainArtifacts.ts`, `DomainFlowView.vue`.

Behavior:

1. Move capsules from the bottom dock into the relevant consultation turn or focused-form result.
2. Preserve deterministic payload mapping; never infer artifacts from prose.
3. Mobile sheet supports collapsed/medium/full; opening enters medium; explicit control or gesture reaches full.
4. Backdrop, Escape, explicit close, and system back close the sheet; close restores focus and scroll.
5. Sheet scrolling uses `overscroll-behavior: contain`.
6. Reduced motion avoids a large slide.
7. Streaming continues behind the sheet without stealing scroll.

Tests: capsule placement, detents, focus restoration, long content at 320px, deterministic payload requirement, no fake empty/failed artifacts.

### P6 — Focused domain forms

Goal: turn absorbed tools into slot-filling states.

Files: a new `DomainSlotForm` wrapper or the smallest shared form-shell extension, `web/app/src/components/flow/forms/**`, `DomainFlowView.vue`, and transient `step=form` query contract.

Behavior:

1. Focused form hides the composer.
2. Title/back remains visible; primary submit is stable.
3. One primary submit action per form.
4. Dirty state is guarded before back/close/navigation.
5. Refresh restores the dirty draft.
6. Submit returns the result to the consultation document/artifact layer.
7. Compatibility presents 本人 and 对方 explicitly.
8. Fengshui auto-submit is allowed only for an explicit marketing handoff; user edits require explicit submit.

Preserve shared validation, engine adapters, backend payloads, record-save behavior, and `huangli` pending-backend boundary.

Exit: no focused form leaves two primary consultation actions visible.

### P7 — Documentation and guardrail reset

Files:

1. `docs/frontend/app-flow-redesign-plan.md`
   - Record the two-template exception and accepted deviations.
2. `docs/frontend/ux-architecture.md`
   - Replace stale Classic-only route/navigation descriptions.
3. `FRONTEND.md`
   - Update App routes, Me layer, Consultation, and handoff rules.
4. `web/routes/app.contract.json`
5. `web/scripts/check-route-contracts.mjs`
6. `web/app/src/tests/i18n-contract.test.ts`
7. Playwright specs and visual baselines.

Rule: do not refresh a visual snapshot unless the reviewer can name the reviewed contract the new snapshot satisfies.

## Validation matrix

Per behavior-changing phase:

```bash
make lint
make typecheck
make test
cd web && npm run routes:check
cd web/app && VITE_FEATURE_APP_FLOW_REDESIGN=true npm run build
```

Focused E2E after P1–P6:

```bash
cd web
BASE_URL=<preview-url> npm run test:e2e -- \
  e2e/journeys/domain-flow.spec.ts \
  e2e/journeys/domain-flow-a11y.spec.ts \
  e2e/journeys/domain-flow-resilience.spec.ts \
  e2e/journeys/domain-flow-reflow.spec.ts \
  e2e/journeys/domain-flow-visual.spec.ts \
  e2e/journeys/domain-flow-rollout.spec.ts
```

Before merge to default: `make check && make test`.

Manual verification:

- 320 / 390 / 768 / 1024 / 1440px.
- `zh-Hans`, `zh-Hant`, `en`, including longest labels.
- LTR and RTL mirror; 200% zoom.
- Keyboard-only journey; reduced motion.
- Draft restore, failed send, offline, permission failure, payment return.
- No-profile, one profile, multiple profiles, personless divination.
- Active consultation, history restore, new consultation, partner-role account.

## Rollback

- Keep `VITE_FEATURE_APP_FLOW_REDESIGN=false` as the Classic rollback switch.
- Keep `/app/classic/**` and compatibility redirects until backend domain session/context migration completes.
- Keep `huangli` pending-backend and Classic-owned until backend ownership changes.
- Each phase must be independently revertible.

## Stop conditions

Stop and reopen the design contract if:

1. A fix requires a fifth visible domain.
2. A fix requires person or session IDs in the URL.
3. Session restore cannot be represented by current `(profile_id, expert, consultation context)`.
4. An artifact must be inferred from chat prose instead of typed deterministic payload.
5. Desktop cannot preserve a 400px document and readable artifact at 1024px.
6. Focused-form backend contracts cannot preserve dirty state or explicit submit semantics.
7. Product changes the active-session window, canonical route ownership, or template-switch policy.

## Design documentation

After acceptance:

1. Mark completed phases in this plan or the execution board.
2. Update the redesign plan with accepted exceptions.
3. Update UX architecture, frontend docs, route contract, i18n contract, and visual baselines in the acceptance PR.
4. Record restored Partner Center keys and the dynamic-key guard in the subtraction-review correction.
