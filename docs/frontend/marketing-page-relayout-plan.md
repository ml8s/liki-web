# Marketing page relayout plan

Status: implemented for the common shell, scenario pages, and the ten-year
annual tool; formal date-selection product-page work remains contract-blocked.

Implementation status 2026-10-30: the reusable marketing section components and
the common tool-first order are implemented. `ScenarioLanding` now composes the
shared sections, and `/natal/` is migrated to the same component contract. All
scenario pages routed through `ScenarioLanding` inherit the new order. Focused
marketing tool and chrome E2E suites pass.

Implementation status 2026-10-30 (second pass): the `/liunian/` free tool now
renders a ten-year Bazi + Zi Wei overview and selected-year evidence. The
common-shell focused suites and the full browser regression pass. Remaining
product work is the explicitly deferred formal date-selection product page;
backend `bazi_liuri` / `ziwei_liuri` contracts must not be inferred.

## 1. Why

The previous free-tool plan covered result depth and a generic IA principle, but
did not specify the relayout for every marketing page. This plan adds the
page-by-page migration.

The unified conversion order is:

```txt
Hero
→ Free tool
→ Result evidence
→ Professional method / use cases
→ How it works
→ Topics
→ Example questions
→ FAQ
→ Final CTA
```

The tool is the first major section after the hero. Long education content is
not allowed to interrupt the tool.

## 2. Common page template

### 2.1 Hero

- One H1 with the page’s primary search intent.
- One-sentence positioning.
- Primary CTA: use the free tool (`#free-tool`).
- Secondary CTA: App deep reading or `#how-it-works`.
- Do not make App interpretation the first CTA before the user has produced a
  chart.

### 2.2 Free tool

- Directly below the hero.
- One H2.
- One-line input/output explanation.
- Tool island.
- Privacy note.
- No unrelated cards before the form.

### 2.3 Result evidence

Explains the deterministic fields the tool produces.

Required pattern:

```txt
H2: What the result shows
H3 / cards: each evidence group
```

This section must not invent AI conclusions.

### 2.4 Method and use cases

Explains how professionals use the same evidence.

Required pattern:

```txt
H2: Method or use cases
H3: scenario / evidence group
Cards or ordered steps
```

### 2.5 Supporting content

After method/use cases:

```txt
How it works
Topics
Example questions
FAQ
Final CTA
```

These remain useful for SEO and user education, but have lower priority than the
tool.

## 3. Component architecture

Refactor the marketing page shell into explicit section components rather than
keeping one large template.

### 3.1 Astro section components

Proposed components under `web/src/components/marketing/`:

| Component | Responsibility |
| --- | --- |
| `ScenarioPageHero.astro` | H1, subtitle, primary free-tool CTA, secondary CTA |
| `FreeToolSection.astro` | Section frame, H2, intro, tool slot, privacy |
| `ResultEvidenceSection.astro` | H2 and evidence cards for deterministic output |
| `UseCaseSection.astro` | Scenario-specific use cases and reading guidance |
| `HowItWorksSection.astro` | Step cards |
| `TopicLinksSection.astro` | Existing SEO topics |
| `ExampleQuestionsSection.astro` | User question examples |
| `ScenarioFaqSection.astro` | FAQ + FAQ schema input |
| `FinalCtaSection.astro` | Bottom App conversion |

`ScenarioLanding.astro` can become a compatibility composition around these
components. New pages should compose the section components directly.

### 3.2 Vue island boundaries

The tool remains a Vue island under the Astro `FreeToolSection`.

For each tool, separate:

| Concern | Location |
| --- | --- |
| Query lifecycle | composable |
| Form fields | shared form component |
| Result grouping | shared result panel component |
| Day/item detail | shared detail component |
| App handoff | island-level CTA |

Example:

```txt
LiunianTool.vue
  useAnnualDecadeTool()
  AnnualRangeFields
  AnnualOverview
  BaziLiuNianPanel
  ZiweiLiuNianPanel
```

Huangli should similarly keep form, result grouping, and day detail separated.

## 4. Page-by-page relayout

### 4.1 `/natal/`

#### Current issues

- Bazi education appears before the tool.
- Ziwei education appears after the tool.
- The tool calculates both systems, so the split feels accidental.
- Supporting topics such as zodiac and true-solar-time are mixed into the main
  conversion path.

#### Target order

```txt
Hero
→ Free Bazi + Ziwei tool
→ Combined result evidence
  - Bazi evidence
  - Ziwei evidence
  - Bazi + Ziwei annual evidence
→ Why professional reading helps
→ True solar time / zodiac / method
→ How it works
→ Example questions
→ FAQ
→ Related scenarios
→ Final CTA
```

#### Content rules

1. The tool owns both Bazi and Ziwei output.
2. Bazi and Ziwei education should be grouped as evidence, not as separate funnels.
3. Keep `#bazi`, `#ziwei`, and `#reading` anchors for compatibility.
4. Zodiac and true-solar-time sections remain, but after the core tool and
   evidence explanation.

#### Components

- `ScenarioPageHero`
- `FreeToolSection`
- `ResultEvidenceSection` with Bazi and Ziwei groups
- `UseCaseSection` for AI reading
- Existing supporting sections

---

### 4.2 `/liunian/`

#### Current issues

- Annual fortune is framed as one year.
- Bazi result is richer than Ziwei result.
- Education content appears after the tool but before the full result story is
  clear.

#### Target order

```txt
Hero
→ Free annual tool
→ Annual decade overview
→ Selected year Bazi detail
→ Selected year Ziwei detail
→ How to read annual rhythm
→ Method / use cases
→ How it works
→ Topics
→ Questions
→ FAQ
→ Final CTA
```

#### Content rules

1. Default range: current year plus the following nine years.
2. User selects one focus year for detailed Bazi and Ziwei evidence.
3. Do not describe the decade as one fused “ten-year fortune”; it is ten annual
   observations.
4. Bazi and Ziwei remain separate deterministic systems.
5. App CTA should pass the selected focus year.

#### Dependencies

The decade tool is a separate feature plan. Page relayout can define the section
structure first, but must not fake missing decade data.

---

### 4.3 `/hepan/`

#### Current issues

- The compatibility result is professional, but the page explains it as one
  generic result.
- Relationship use cases are not explicit enough.
- Consent is critical but appears as a form detail rather than part of the method.

#### Target order

```txt
Hero
→ Free compatibility tool
→ Compatibility result evidence
  - Side-by-side pillars
  - Day-master relation
  - Branch relations
  - NaYin
  - Shen Sha
  - Ten-god interaction
→ Relationship use cases
  - Marriage matching
  - Partnership fit
  - Family/work dynamics
→ Consent and ethics
→ How it works
→ Topics
→ Questions
→ FAQ
→ Final CTA
```

#### Content rules

1. Always keep third-party consent before calculation.
2. Explain that the deterministic result is evidence, not a verdict.
3. Encourage App reading for context, timing, and communication advice.
4. Do not put either person’s birth facts in the URL.

---

### 4.4 `/divination/`

#### Current issues

- Casting and full chart are now both shown, but the page still reads like a
  simple casting demo.
- The difference between receipt, chart, useful god, patterns, and timing is not
  explained clearly enough.

#### Target order

```txt
Hero
→ Free divination tool
→ Casting receipt evidence
→ Full Liu Yao chart evidence
→ How to read useful god / patterns / timing
→ Question quality guidance
→ How it works
→ Topics
→ Questions
→ FAQ
→ Final CTA
```

#### Content rules

1. Casting receipt and chart are deterministic evidence.
2. AI interpretation remains in App.
3. Explain why time and location matter.
4. Show examples of a focused question versus a vague question.
5. Keep engine failure and retry states visible.

---

### 4.5 `/huangli/`

This page has a dedicated product-page plan in
`huangli-product-page-plan.md`.

#### Target order

```txt
Hero
→ Free date-selection tool
→ Candidate results: recommended / possible / unsuitable
→ Selected-day evidence
→ Date-selection method
→ Event use cases
→ Personal flow-day roadmap
→ How it works
→ Topics
→ Questions
→ FAQ
→ Final CTA
```

#### Content rules

1. Event first: wedding, opening, moving, signing, groundbreaking.
2. Almanac evidence second: Yi/Ji, Day Officer, clash, spirits, day pillar.
3. Personal flow-day is planned, not faked.
4. App CTA should preserve event and date range context.
5. Birth facts must not be placed in the URL.

---

### 4.6 `/fengshui/`

#### Current issues

- Three result families are useful but appear as technical modules.
- Users may not understand how Ming Gua, Xuan Kong, and layout relate.

#### Target order

```txt
Hero
→ Free Feng Shui tool
→ Result evidence
  - Ming Gua / Eight Mansions
  - Xuan Kong annual stars
  - Door / master / stove layout
→ How to use the three views together
→ Home and business use cases
→ Method
→ How it works
→ Topics
→ Questions
→ FAQ
→ Final CTA
```

#### Content rules

1. Each module keeps its own loading and retry state.
2. One module failing must not hide the others.
3. Explain the intended reading order: person → period → layout.
4. Do not give remedial claims beyond the deterministic fields.

---

### 4.7 `/naming/`

#### Current issues

- The tool intentionally provides only Yong Shen evidence, but the page does not
  make the boundary strong enough.
- Naming segments are useful, but appear too far from the result.

#### Target order

```txt
Hero
→ Free Yong Shen tool
→ Yong Shen evidence
  - Ruling element
  - Favorable elements
  - Unfavorable elements
  - Adjudication evidence
  - Conflicts / review warning
→ Naming principles
→ Naming segments
→ How App naming works
→ Topics
→ Questions
→ FAQ
→ Final CTA
```

#### Content rules

1. Public tool does not generate names.
2. App naming uses the deterministic evidence with user preferences and cultural
   constraints.
3. Do not promise score, rank, or guaranteed luck.
4. Low-confidence results must show review warning.

---

## 5. Copy and SEO rules

### 5.1 Page-level SEO

Each page needs:

1. One unique H1.
2. One primary search intent.
3. Unique title and meta description.
4. Canonical and hreflang.
5. Service / FAQ / breadcrumb schema where appropriate.
6. One primary free-tool section.
7. Internal links to related scenarios and App.

### 5.2 Keyword separation

| Page | Primary theme | Must not cannibalize |
| --- | --- | --- |
| `/natal/` | Bazi + Ziwei natal chart | Generic annual browsing |
| `/liunian/` | Annual rhythm and selected-year detail | Full natal report |
| `/hepan/` | Two-person compatibility | Individual destiny reading |
| `/divination/` | Liu Yao question casting | Almanac date selection |
| `/huangli/` | Event date selection | Divination or annual report |
| `/fengshui/` | Ming Gua, Xuan Kong, layout | Destiny reading |
| `/naming/` | Yong Shen for naming | Full life reading |

### 5.3 Copy pattern

Each result section should follow:

```txt
Title: what this panel shows
Intro: how to read it
Evidence: fields / cards
Boundary: what still needs App interpretation
```

## 6. i18n plan

Every page relayout must update:

1. `web/i18n/zh-Hans.json`
2. `web/i18n/zh-Hant.json`
3. `web/i18n/en.json`

Required key families:

```txt
<page>.product.hero.*
<page>.product.tool.*
<page>.product.evidence.*
<page>.product.method.*
<page>.product.useCases.*
<page>.product.app.*
```

For existing pages, reuse current keys when the meaning is identical. Do not
create duplicate keys.

Rules:

1. No raw i18n keys in rendered pages.
2. No machine-translated traditional Chinese.
3. Professional terms remain consistent:
   - Bazi / 八字 / 八字
   - Zi Wei Dou Shu / 紫微斗数 / 紫微斗數
   - Liu Yao / 六爻 / 六爻
   - Yong Shen / 用神 / 用神
4. No guaranteed-outcome language.

## 7. Migration order

### Phase 1 — component shell

1. Add marketing section components.
2. Keep existing URLs and schema.
3. Migrate `ScenarioLanding` internally.
4. Verify all current pages visually and functionally.

### Phase 2 — high-traffic page relayout

1. `/natal/`
2. `/liunian/`
3. `/huangli/`
4. `/hepan/`

### Phase 3 — remaining pages

1. `/divination/`
2. `/fengshui/`
3. `/naming/`

### Phase 4 — visual polish

Status: covered for primary controls and consultation reflow at 320 / 640 /
768 / 1024 / 1440 by `domain-flow-reflow.spec.ts`, and for marketing smoke at
375 by `pages.spec.ts`. A dedicated screenshot matrix for every marketing page
remains optional follow-up, not a release blocker for this relayout.

1. 320px mobile tool layout.
2. 375px card spacing.
3. 768px two-column evidence.
4. 1024px desktop hierarchy.
5. Focus order and touch targets.

### Phase 5 — content and SEO QA

1. Three-language copy review.
2. Raw-key smoke.
3. Canonical / hreflang.
4. Structured data.
5. Internal anchor compatibility.

## 8. Testing plan

### 8.1 Existing tests

Update:

1. `pages.spec.ts`
2. `chrome-consistency.spec.ts`
3. Page-specific tool specs
4. `tool-handoff.spec.ts`

### 8.2 New assertions

For every relayout page:

1. Tool section appears before supporting education sections.
2. Primary hero CTA targets `#free-tool`.
3. H1 is unique.
4. Evidence section exists.
5. FAQ renders actual copy, not raw keys.
6. App handoff preserves required context.
7. Birth facts do not appear in URL.
8. No console errors or warnings.

### 8.3 Visual checks

Minimum widths:

```txt
320px
375px
768px
1024px
1440px
```

Check:

1. Tool controls fit.
2. Evidence cards reflow.
3. Headings do not clip.
4. CTA remains reachable.
5. No page-level horizontal scroll.

## 9. Acceptance criteria

A page is complete only when:

1. Its order matches the common template.
2. Tool appears before long education content.
3. Result evidence is explicit and grouped.
4. App value is explained after deterministic evidence.
5. SEO metadata and headings are unique.
6. Three-language copy is complete.
7. Existing handoffs still work.
8. Page-specific E2E passes.
9. No new dependency or second design system is introduced.

## 10. Non-goals

1. Do not introduce a CMS.
2. Do not add a new CSS framework.
3. Do not merge all scenario pages into one generic template at the cost of SEO.
4. Do not invent AI readings in marketing pages.
5. Do not remove educational content merely to make the page shorter.
