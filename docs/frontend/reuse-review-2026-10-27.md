# Resource-reuse review — 2026-10-27

Companion to the subtraction review: instead of removing dead code, this pass
hunts duplicated LIVE code and consolidates it onto one shared implementation.

## Consolidated in this pass

1. **Birth-year parsing** (4 sites → 1): `birthYearFromDate(date, min, max)`
   in `@shared/domain/birth.ts` replaces the inline
   `parseInt(date.slice(0,4))` + range checks in `useConsultationTools`,
   `FengshuiView` (validator + compute), and `ChartPreview`. Behavior note:
   the two unguarded sites now adopt the same 1900–2100 engine bounds as the
   consultation path (standardization, not a regression — profiles outside
   that range are outside the engines' supported domain).
2. **Profile → engine chart-input projection** (4 sites → 1):
   `chartInputFromProfileBirth(profile)` replaces the inline
   `{instant, timezone, longitude, gender}` builds + completeness guards in
   `useConsultationTools.profileChartInput`, `HepanView` (loadA + compute),
   and `ChartPreview`. The shared helper throws on incomplete facts; callers
   with soft-failure UX keep their guards. Nullable wire-shape fields are
   accepted and guarded.

Both helpers are unit-tested (`birth-helpers.test.ts`); the full matrix
(467→471 tests) stays green.

## Verified reuse strengths (no action)

| Resource | State |
| --- | --- |
| Domain functions (`@shared/domain/*`) | Single source for both layouts and marketing islands |
| Engine result panels | Same `@shared/components/engine/*` panels in Classic, Consultation, Astro islands |
| `RegionAutocompleteField` | One shared field (23.5 kB chunk) across app + marketing |
| Design tokens | `tokens.css` single source; `check-design-tokens`/`check-frontend-tokens`/`check-css-vars` gates |
| Birth-fields form kit | `BirthFields`/`DateField`/… shared by Classic, Consultation, marketing islands |
| profiles store | Single-flight + staleness cache; no double-fetch across views |
| Assets | No duplicate images/fonts (md5 scan); single favicon |
| Dependencies | No duplicate-purpose libraries (no second date lib, no second sanitizer) |
| E2E fixtures/helpers | `mockFlowWorkspace`, `flowProfile`, `fillBirthFields`, engine fixtures shared across specs |

## Known coexistence costs (by design, recorded)

1. **Two Vue runtimes ship**: `vue-vendor` (app SPA) and
   `runtime-dom.esm-bundler` (Astro islands). Different entry modes; merging
   would couple the marketing site to the app's Vue build. Accepted.
2. **Two translation delivery systems** (app vue-i18n locales vs
   chart-messages JSON for islands) — same finding as the subtraction review;
   sync gate only if drift is observed.
3. `profileBirthInput` (BirthInput with place metadata for
   `baziChartFor`) remains a separate small adapter from
   `chartInputFromProfileBirth` — different target contracts (full
   `BirthLocation` with place/source vs bare engine input). Not worth merging.
4. Classic saves stay silent on success (errors surface via `recordsError`),
   while consultation saves announce — a consistency gap, noted for a future
   a11y parity pass rather than forced into the current batch.
