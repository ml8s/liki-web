# Subtraction review — 2026-10-27

Scope: app + shared frontend, e2e tree, scripts. Method: machine scans
(reference graphs for components/composables/utils/exports/dependencies,
locale-key usage incl. dynamic prefixes and shared/Astro corpora) plus manual
verdicts. Guiding rule: Classic and compatibility surfaces are live product
(two-layout decision), not subtraction targets.

## Removed in this pass (verified zero references)

1. Locale keys `partner.ruleKol1/ruleKol2/ruleScout1/ruleScout2/defaultRejectionReason`
   — no static or dynamic lookup anywhere (app, shared, Astro). Removed from
   all three locales symmetrically; `check-i18n` green.
2. `createUnavailableArtifact` / `createPendingBackendArtifact`
   (artifacts/types.ts) — dead factories, zero references incl. tests.
3. `useConsultationSubmit` returned `sendUsingActiveSession` but the view only
   uses the other members — removed from the public return object (the
   function stays private).
4. Internalized single-use exports: `appExperiences` (collapsed to a pure
   type union — the runtime array existed only to derive the type),
   `createLegacyAppRedirects`, `artifactsFromConsultation`.

## Investigated and explicitly KEPT (anti-removal record)

| Suspect | Verdict |
| --- | --- |
| 225 "dead" locale keys from the naive scan | False positives: `errors.code.*` is dynamic (`errors.code.${code}`), `services.*.shensha.*` is dynamic in `HepanPanel`, and shared engine panels consume the host app i18n instance (e.g. `common.listSeparator`). True dead set was only the five partner keys above. |
| Legacy-mode E2E specs (`app.spec`, `chat-session.spec`, `profile-journey.spec`, … asserting `/app/birth-profiles` etc.) | Not expired: `make e2e` (Docker) builds with the redesign flag OFF, so these are the rollback-mode contract. The a11y gate builds flag ON and runs preview-aware specs. Deleting them would remove rollback assurance. |
| `check-rtl-readiness.js` | Not wired into any gate. Expired wiring — flagged for a follow-up decision: wire it into `make check` or delete the script. Not removed in this pass. |
| `scripts/smoke.mjs` (`system:check`) | Documented manual ops tool (`SMOKE.md`); not CI-wired. Kept. |
| `/app/consultation` transition alias | No inbound links or E2E, but it is a documented compatibility surface for earlier preview bookmarks. Kept per the compatibility-surface rule. |
| `web/i18n/*.json` | Live: consumed by the Astro about/contact/disclaimer pages. |
| API response type exports with zero external imports (`OrderCheckoutResult`, `SSEError`, …) | Wire-contract documentation; removal is churn without runtime benefit. Kept. |
| Options/type exports on new composables (`ConsultationSubmitOptions`, …) | Public API documentation; kept. |

## Orthogonality observations (no action, recorded)

1. Two translation delivery systems exist by design: app vue-i18n locales
   (SPA + shared panels via host `t()`) and `chart-messages` JSON (Astro
   islands without the app i18n instance). Drift between the two for the same
   panel key would surface as app-vs-marketing text divergence; a sync gate
   could be added later if drift is ever observed (addition, not subtraction).
2. Error rendering has three intentional levels: page-level `AppStatusAlert`,
   spine/workspace failure block, and inline `reading__error`. Distinct
   semantics, not duplication.
3. `useToolRecords` owns the classic record loop while consultation composes
   the same save primitive — one save implementation, two consumers. Healthy.

## Verification after removals

467 unit tests, typecheck, eslint, locale parity gate, full pre-push matrix
(`make check` + `make test` incl. Go race + 38 a11y E2E) all green.
