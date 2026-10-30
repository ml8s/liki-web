# Marketing free-tool depth audit — 2026-10-30

Scope: all seven public scenario pages and their deterministic, login-free
outputs. AI interpretation, saved history, records, and account features remain
App-side. The goal is to make every public tool show the full deterministic
chart evidence without faking expertise or exposing private data.

## Findings

| Scenario | Before | App/reference depth | Decision |
| --- | --- | --- | --- |
| `/natal` | Bazi full chart + Bazi annual + favorable elements; Ziwei natal chart + major-limit chart. | App profile preview also has Ziwei annual palace chart. | Add Ziwei annual and an independent year control. **Implemented.** |
| `/liunian` | Bazi natal/full chart and Bazi annual only. | “流年” should expose both deterministic annual systems over a comparable horizon. | Add Ziwei base chart + a merged ten-year Bazi/Zi Wei overview with selected-year evidence. **Implemented.** |
| `/divination` | Casting receipt only; full hexagram intentionally deferred. | App immediately derives the deterministic six-yao chart: six relatives, NaYin, useful god, patterns, and timing. | Keep AI reading in App, but publish the complete deterministic chart. **Implemented.** |
| `/hepan` | Two-person natal charts, relationship summary, key branch relations, NaYin, mutual Shen Sha, and ten-god interactions through `BaziBondPanel`. | App uses the same deterministic panel. | Already at parity; keep AI follow-up as the upgrade. |
| `/huangli` | Date range, suitable/possible/unsuitable days, day pillar, suitability reason, and expandable detail through `HuangliDaysPanel`. | App uses the same deterministic panel; a formal product page should later add selected-day evidence and Bazi/Ziwei flow-day planning. | Current deterministic output is at parity. Full product-page and flow-day roadmap: [`huangli-product-page-plan.md`](./huangli-product-page-plan.md). |
| `/fengshui` | Ming Gua/Bazhai chart with layout badges, Xuan Kong annual chart, and door/master/stove layout. | App uses the same three deterministic outputs. | Already at parity. |
| `/naming` | Yong Shen, Xi Shen, Ji Shen, evidence, conflicts, and review warning. | Flow artifact uses the same deterministic panel. | Already at parity; name generation stays App-side by design. |

## Design rules

1. Public tools may show every deterministic fact produced by the engine.
2. They must not fabricate AI interpretation, predictions beyond engine data,
   saved history, or account-specific recommendations.
3. Marketing and App consume the same shared panels and domain adapters.
4. Every added section remains lazy, error-isolated, and independently retryable.
5. Birth facts are not persisted by public tools; handoff stays client-side until
   the user explicitly continues into the App.

## Delivered changes

1. `NatalTool.vue`
   - Adds Ziwei annual palace chart.
   - Adds a separate Ziwei annual-year control so Bazi and Ziwei years can be
     compared independently.
2. `LiunianTool.vue`
   - Adds Ziwei base chart and Ziwei annual palace chart.
   - Loads Bazi and Ziwei chains independently so one failure does not erase the
     other result.
   - Adds a ten-year overview from the selected start year and selected-year
     Bazi/Zi Wei detail.
   - Marks a year partial when one deterministic system is unavailable; the
     frontend never fabricates the missing chart.
3. `DivinationTool.vue`
   - Keeps the casting receipt.
   - Adds the complete deterministic six-yao chart from the submitted instant,
     longitude/timezone, and casting receipt.
   - Does not add AI text.

### P1 result-explanation layer

Implemented after the depth parity pass:

1. Every result section now has a one-line explanation that names the evidence
   shown and what the user can check.
2. Result sections use a consistent separated title/intro hierarchy without a
   second design system or new dependency.
3. Copy is synchronized across Simplified Chinese, Traditional Chinese, and
   English.
4. Bazi and Zi Wei annual years can be compared independently on the natal page.

## Follow-up plan

Page-by-page relayout details are now tracked in
[`marketing-page-relayout-plan.md`](./marketing-page-relayout-plan.md).

### Dedicated date-selection product plan

See
[`huangli-product-page-plan.md`](./huangli-product-page-plan.md). It upgrades
`/huangli/` into a formal date-selection product page and plans a layered
model: almanac candidates, selected-day evidence, Bazi flow-day, Ziwei flow-day,
and App deep reading. This is planning-only until backend contracts are
confirmed.

| Priority | Outcome | Work | Acceptance |
| --- | --- | --- | --- |
| P1 | Explain the evidence | Add concise section intros above Bazi/Ziwei/LiuYao outputs; reuse existing copy where possible. | Each chart has a one-line “what am I reading” explanation in all locales. |
| P1 | Visual hierarchy | Add a shared tool-results layout for section titles, source badges, and CTA separation. | No new page-local chart styles; sections have consistent spacing and headings. |
| P2 | Comparable-free teaser | For each result, show two or three next-step questions derived from the scenario, not model output. | Each page has a deterministic “continue in App” pathway with handoff intact. |
| P2 | Public-tool QA matrix | Add marketing-island E2E coverage for result depth at representative widths. | Ziwei annual, dual annual, and six-yao chart assertions exist for public pages. |
| P3 | Progressive detail | Allow compact/full density and section anchors for long panels. | 360px has readable charts; 1024px can show detail without hiding evidence. |
