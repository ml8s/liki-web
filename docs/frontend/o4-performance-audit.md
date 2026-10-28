# O4 Performance and Bundle Audit

## Appendix: O6 engine-result catalog audit (2026-10-27)

Finding: the Wave 4 catalog goal is already satisfied. Both layouts consume
the same canonical panels (`@shared/components/engine/*`) and the same typed
domain functions (`@shared/domain/*`); `ArtifactPanel` lazy-loads the same
13 panels that Classic views render directly. The only remaining per-view
code is ~30 lines of compute/loading/error orchestration whose inputs differ
by design (Classic uses full dual birth forms; Consultation uses the profile
plus focused-form). Forcing these into a shared adapter would be speculative
abstraction. O6 "page-local projections → typed adapters" is therefore
**closed as already satisfied**; new result families must extend
`catalog.ts`, `engine-contract.spec.ts`, and the lazy `ArtifactPanel`
branches in the same change.

Status: baseline recorded 2026-10-27 on branch `feat/app-flow-redesign`
at commit `c8c191a`, build flag `VITE_FEATURE_APP_FLOW_REDESIGN=true`,
Vite 6 production build (`vue-tsc --noEmit && vite build`).

## 1. Route-level baseline

Measured from the production build output graph (static import closure
per route entry plus the global CSS loaded by `index.html`).

| Route | JS raw | CSS raw | Files loaded |
| --- | --- | --- | --- |
| HTML shell (all routes) | 299.8 kB | 735.9 kB | 3 |
| `/app` (Consultation) | 895.7 kB | 803.2 kB | 45 JS + CSS |
| `/app/me` | 299.8 kB | 735.9 kB | 3 |
| `/classic` (Dashboard) | 312.8 kB | 739.9 kB | 7 JS + CSS |
| `/classic/chat/:id` | 477.6 kB | 743.4 kB | 10 JS + CSS |

Gzip sizes for the three HTML-loaded assets:

| Asset | Raw | Gzip |
| --- | --- | --- |
| `index-B3x6GZuO.js` (entry) | 195.7 kB | 67.4 kB |
| `vue-vendor-KroxMd0w.js` | 111.3 kB | 43.5 kB |
| `index-C-hYbDgu.css` | 753.5 kB | 101.8 kB |

The `chat` chunk (339.7 kB / 80.6 kB gz) is a lazy dynamic import inside
the entry chunk; it does not load on any route until the AG-UI transport
is first used.

## 2. Consultation route JS composition (largest contributors)

| Chunk | Size | Contents |
| --- | --- | --- |
| `index-B3x6GZuO.js` | 191.1 kB | Entry: main.ts, router, stores, Element Plus runtime core, vue-i18n |
| `AppMarkdown-D9KfhtQp.js` | 130.3 kB | markdown-it + highlight.js + dompurify |
| `vue-vendor-KroxMd0w.js` | 108.7 kB | vue + vue-router + pinia |
| `index-CY7_u-lw.js` | 101.9 kB | Element Plus DatePicker (dayjs) |
| `DomainFlowView-DpG0jsXL.js` | 58.1 kB | DomainFlowView + flow child components |
| `index-DvGPoOVB.js` | 46.7 kB | Element Plus Tooltip/Popper/Scrollbar |
| `index-B79eM09f.js` | 30.6 kB | Element Plus Select/TreeSelect/Tag/Icon |
| `index-D25-6Oar.js` | 27.1 kB | Element Plus Form/FormItem |
| `index-C8fvWWaj.js` | 24.4 kB | Element Plus Input |
| `RegionAutocompleteField` | 23.5 kB | region autocomplete (search + data) |
| `ZiweiDaXianPanel` | 21.1 kB | Ziwei chart panel (eager) |
| `LiuyaoChartPanel` | 15.3 kB | Liuyao chart panel (eager) |

Remaining ~65 kB is spread across ~33 small chunks (Element Plus
directives, lodash helpers, composables, form field components).

## 3. Findings

### F1 — Artifact panels load eagerly with the route

`ArtifactPanel.vue` statically imports 13 engine panels
(Bazi×5, Ziwei×2, Liuyao×2, Fengshui×3, Huangli×1). Only one renders at
a time, and only when the user expands an artifact capsule. Combined
weight in the consultation graph: ~52 kB (Ziwei 21 + Liuyao 15 + the
rest ~16), plus their transitive dependencies.

### F2 — AppMarkdown loads eagerly with the route

`ConsultationDocument.vue` and `ChatMessageItem.vue` both statically
import `AppMarkdown.vue`, which statically imports `markdown-it`,
`dompurify`, and `highlight.js`. Combined chunk: 130.3 kB. Markdown
rendering is needed only once the first assistant message arrives.

### F3 — Element Plus DatePicker loads eagerly with the route

The DatePicker chunk (101.9 kB including dayjs) is pulled into the
consultation graph through `DateField.vue` → `birth-fields` → form
components. It is only needed when a birth-date field is focused.

### F4 — Full Element Plus CSS loads globally

`@shared/styles/element-plus/index.css` imports the entire
`element-plus/theme-chalk/index.css` into `@layer element-plus`.
This is intentional (one cascade boundary, application tokens win),
but it makes the global CSS 753.5 kB regardless of route. No
per-component splitting is currently possible without changing the
cascade-layer strategy; recorded as an intentional trade-off.

### F5 — Locale messages are already lazy

`en` (41.1 kB), `zh-Hant` (36.4 kB), and `zh-Hans` (36.4 kB) load on
demand via dynamic import in `i18n/index.ts`. Only the active locale
loads. No action needed.

### F6 — Chat chunk is already lazy

The AG-UI transport (`@ag-ui/client` → proto/protobuf/zod) sits in a
339.7 kB chunk loaded via `import()` on first use. No action needed.

### F7 — No admin, partner, or classic views in the consultation graph

Route-level splitting is correct: admin, partner, classic views, and
their dependencies are not reachable from the consultation import
closure. No action needed.

## 4. Intentional trade-offs

1. Full Element Plus CSS (F4) is kept as a single cascade layer load.
   Tree-shaking it would require replacing the cascade-layer boundary
   with per-component style injection, which risks specificity regressions
   across Astro islands and Classic. Rejected for O4.
2. The `chat` chunk's 339.7 kB is accepted because it is lazy and only
   loads when the user sends a consultation message.
3. `RegionAutocompleteField` (23.5 kB) is accepted because it provides
   required form validation UX and cannot be meaningfully split further.

## 5. Implementation results

All four planned changes have been implemented.

| # | Change | Actual saving |
| --- | --- | --- |
| 1 | ArtifactPanel's 13 engine-panel imports → `defineAsyncComponent` | ~52 kB |
| 2 | AppMarkdown lazy-loaded in ChatMessageItem + ConsultationDocument | ~130 kB |
| 3 | DatePicker lazy-loaded in DateField + FengshuiLayoutFields | ~102 kB |
| 4 | Bundle budget + consultation route-graph checks in CI | — |

### Post-implementation baselines

| Route | Before JS | After JS | Δ |
| --- | --- | --- | --- |
| HTML entry | 299.8 kB | 302.4 kB | +2.6 kB (async wrappers) |
| `/app` consultation | 895.7 kB | 644.2 kB | **−251.5 kB (−28%)** |
| `/classic` dashboard | 312.8 kB | 323.1 kB | +10.3 kB |
| `/classic/chat/:id` | 477.6 kB | 359.5 kB | **−118.1 kB (−25%)** |

HTML entry brotli sizes (unchanged):

| Asset | Raw | Brotli |
| --- | --- | --- |
| Entry JS | 191.9 kB | 56.6 kB |
| vue-vendor | 110.5 kB | 38.9 kB |
| Global CSS | 735.9 kB | 40.3 kB |

### CI budgets added

`web/scripts/check-frontend-budget.js` now additionally verifies:

1. Consultation route-graph JS total ≤ 650 KiB.
2. `AppMarkdown` / chart-panel chunks are not in the consultation static
   import closure.
3. No static `ElDatePicker` import exists inside the consultation graph.

## 6. Verification

Full matrix run 2026-10-27, all green:

| Check | Result |
| --- | --- |
| `make check` (incl. budget, govulncheck) | ✓ |
| `make test` (Go + Python + JS 100 files / 441 tests) | ✓ |
| Flagged build `VITE_FEATURE_APP_FLOW_REDESIGN=true` | ✓ |
| 32 domain-flow E2E tests (11 spec files) | ✓ 32/32 |
| `check-frontend-budget.js` (incl. new route-graph checks) | ✓ |
