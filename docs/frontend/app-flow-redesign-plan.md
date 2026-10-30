# App flow redesign plan

Status: active implementation plan (discussion outcome, revised after several
layout reviews; implementation is behind the app-flow redesign feature flag).

Accepted exception (2026-10-30): both templates ship together. Consultation
owns `/app` by default and Classic remains at `/app/classic` for rollback and
compatibility. A saved Classic preference guards the bare root.

This document records the converged information architecture, the concrete
layout details, and the phased migration plan for the logged-in product
surface (`/app`). It does not modify `ux-architecture.md`; that document is
updated as part of P1 implementation.

## 1. Why

The current navigation splits the product by page type (dashboard /
profiles / chat). Users think in one task: **ask something about someone**
(or ask something with no person attached). Domain analysis confirms:

- `BirthProfile` is the aggregate root. Personal natal, annual, compatibility,
  naming, personalized date-selection, and fengshui readings can all operate on
  it; divination and generic almanac lookup can remain person-optional.
- The seven public products are acquisition and explanation languages, not the
  in-app navigation taxonomy.
- Chat is a capability, not a domain concept; it should not own a
  top-level destination.

Conclusion: the product has **one verb (ask)** with an optional person, a
practice domain, and a free-form question. The app collapses to four domains;
it does not expose a seven-product task hierarchy.

## 2. Target information architecture

```text
ask(domain, person?, question)    // one flow; the question defines the task
context         person status · conditional restore · overflow / rail
consultation    person home · structured readings · follow-up questions
domain          four user-facing practice domains
domain control  four domain chips in a draft; collapsed domain control in a session
composer        typing + send = create session (no confirm page)

partner center (role-gated first-class entry)
mine (layer, not tab)  orders · entitlements · account · full history
                       · profile management
```

Key decisions already made during discussion:

| Decision | Outcome |
| --- | --- |
| Person switcher vs profile pages | person is a global context, not a destination |
| Primary domains | four fixed practice domains: 命理、起名、问卦、风水 |
| Marketing scenarios | the seven marketing islands remain unchanged, but they hand off to a domain rather than becoming App tasks |
| Bottom tabbar / sidebar | retired entirely |
| Open app | restore last context (person + running session), zero taps |
| Purchase | inline at domain entry, never a separate page |
| Chart focus ("边聊边亮") | client-side dictionary matching only; no model tool call |
| Free-form entry | the user selects person/domain and asks directly; missing facts become domain slot-filling, not a product-selection step |
| Person-session binding | every visible person surface is pinned to the session; switching person switches the view, never rebinds a session |
| Partner center | first-class entry for partner-role accounts, not buried inside mine |
| Chart computation | lazy (first artifact expansion) and cached per person; never part of boot |
| Open behavior | adaptive default: active session -> restore its latest reading; no active session -> person home (chart summary + suggested questions + ask) |
| Parameter placement | separate person and domain controls live with the composer, synced with the context surface |
| Chips semantics | four domain selectors; no seven-task strip or task chooser exists in the App |
| Chips placement | mobile drafts expose the four-domain strip above the composer; an active session collapses it into the composer domain control |
| Focused forms | a form is domain slot-filling triggered by the question or marketing handoff, represented as transient flow state; its result becomes a consultation artifact |
| Draft ownership | an unsent message belongs to one `(person, domain)` pair and is never carried invisibly into another domain |
| Main surface | a structured consultation document, not a generic chat transcript |
| Layout priority | consultation document spine is primary; chart/tool artifacts are reference layers, never co-equal main content panes |
| Responsive strategy | mobile-first: the 320px flow defines hierarchy, scrolling, and action safety; desktop adds persistent controls and an artifact sidecar |

### 2.1 Person-session binding (mandatory rule)

Sessions are bound to `profile_id` at creation and never rebind. The UI
rule follows from the data model:

- Every visible person surface is pinned to the visible session: if the screen
  shows a reading for 张三, both the context status and composer control must
  display 张三. A mismatched pair (李四 control + 张三 reading) is the most
  severe defect possible in this product — it reads one person's fate as
  another's.
- On mobile, the context surface is status, not a second switcher. Switching
  the composer person control switches the **view** to that person's latest active
  session (or their blank ask state). It never rewrites the previous
  session's `profile_id`.
- A domain question that needs a second person (for example compatibility)
  keeps the counterpart in focused slot-filling; the session binds to the
  context person (A) only.

### 2.2 Session lifecycle

- Restore on open = latest active session; typing there **continues** it.
- Selecting a domain for a person restores that `(person, domain)` pair's most
  recent active session immediately. Starting a different consultation is an
  explicit action in the domain chooser and consultation overflow menu; domain
  selection never opens a continue/new interrupt.
- Otherwise, typing starts a new session directly.
- An explicit "new consultation" action is always reachable from the domain
  chooser and the consultation overflow menu.
- The active window (how recent a session counts as active) is an open
  tuning item, not a structural one.

### 2.3 Multi-session selection

When a `(person, domain)` pair has several historical sessions, the flow shows
the most recent active one; all others are reachable from history.

P2 history replaces the current global chat list with filtering by person,
practice domain, and title. Full message-content search is not part of the
frontend-only migration: it requires a server-side search/pagination contract
and is deliberately deferred until that contract exists.

### 2.4 Domain-first routing with a legacy expert bridge

The app exposes only four practice domains. The user never selects one of the
seven public products, and the App has no seven-task strip or task chooser.
The free-form question supplies consultation content. It is never a hidden
routing signal. Routing is deterministic: selected domain, plus an explicit
marketing/slot seed only.

| User-facing domain | Domain contains | Legacy skills for projection |
| --- | --- | --- |
| 命理 | 本命、流年、合盘、黄历择日、个人流日 | `natal`, `liunian`, `hepan`, `huangli` |
| 起名 | 命名、改名、名字分析 | `naming` |
| 问卦 | 六爻问卜 | `divination` |
| 风水 | 八宅、飞星、布局 | `fengshui` |

Canonical naming rule:

- `natal` is the canonical ID for the 命理 domain.
- The exact legacy intent `natal` means 本命 and remains a seed inside that
  domain.
- Reusing `natal` across layers is intentional: the domain is the umbrella,
  while the seed is one task within it. Do not introduce `destiny`, `bazi`,
  or another synonym as a fifth concept.
- Type names distinguish the layers (`AppDomainId` vs
  `LegacyScenarioId`); query keys distinguish context (`domain` vs `seed`).

`/app` is the canonical Consultation route in preview mode; `/app/consultation`
is a transition alias. The runtime router registers this mode only when
`VITE_FEATURE_APP_FLOW_REDESIGN === "true"`. The App contract declares two
experiences: `classic` remains maintenance and moves to `/app/classic`, while
`consultation` owns `/app`. Legacy logged-in URLs redirect with query and hash.
The implementation-only `/app/flow` name is not part of the public route
contract. Its query contract is:

| Query | Contract |
| --- | --- |
| `domain` | optional canonical `AppDomainId`; `natal`, `divination`, `fengshui`, or `naming`; absent/invalid uses `natal` |
| `seed` | optional transient legacy scenario handoff; unknown values are ignored, and cross-domain/pending-backend decisions surface as flow errors |
| `service` | optional legacy marketing alias for `seed`; existing public URLs and handoffs remain unchanged |
| `step` | optional transient value `form`; represents focused domain slot-filling and is never persisted on a session |

`person` and `sessionId` are deliberately absent from the URL. Domain and seed
are safe context, while person and consultation identity remain application
state. The route contract checks the canonical domain set, label keys, feature
flag name, and this query vocabulary; generated marketing route artifacts do
not change.

The seven public marketing islands remain unchanged. Their handoffs translate
to a domain plus an internal intent seed:

```text
natal / liunian / hepan / huangli -> domain=natal
divination                        -> domain=divination
fengshui                          -> domain=fengshui
naming                            -> domain=naming
```

The intent seed is never rendered as an App task. Marketing handoffs for
`huangli` remain on the legacy route until the backend moves that skill to the
natal expert.

### 2.4.1 Frontend-only `LegacyIntentBridge`

Contract correction: the backend now has four experts — `natal`,
`divination`, `fengshui`, and `naming`. It does **not** have seven product
skills. Session creation also accepts seven consultation context scenarios;
they are structured context types for tools such as annual readings,
compatibility, casting, date selection, layout, and naming. The adapter name
`LegacyIntentBridge` is therefore historical; a later cleanup should rename it
to something such as `ConsultationContextResolver`.

At session creation, the chat API accepts an expert or a consultation context
scenario and derives the matching expert server-side. The frontend keeps a
small, deterministic compatibility adapter so old marketing seeds and focused
forms can create the right expert/session/context combination. It maps only
the four visible domains and their explicit scenario seeds; it does not
classify free-form text.

| Visible domain | Bridge result | Current expert |
| --- | --- | --- |
| 命理 | `natal` by default; `liunian` / `hepan` only when an explicit seed or slot result is present | `natal` |
| 起名 | `naming` | `naming` |
| 问卦 | `divination` | `divination` |
| 风水 | `fengshui` | `fengshui` |

Bridge rules:

- An explicit seed or focused-form result is authoritative. If it belongs to
  another domain, the bridge returns `cross-domain`; it never reroutes silently.
- Without an explicit seed, a new session uses the selected domain's base
  scenario and matching expert.
- `huangli` is not a bridge-eligible skill during the frontend-only phase,
  because its current `divination` expert would contradict the visible 命理
  domain. Existing huangli sessions remain viewable/restorable, but new
  personalized date selection waits for the backend migration.
- The bridge runs only when a new session must be created.
- Follow-ups continue the visible session; they do not silently rebind or split
  it. This favors conversational continuity during the frontend-only phase and
  is simplified only after the compatibility seed vocabulary and old session
  restore paths can be retired.
- The bridge logs resolved seed, domain, and reason for instrumentation, but
  does not expose those values as UI state.

### 2.4.2 Target backend model, deferred

The backend already has four domain experts, but its consultation context
contract still retains seven scenario types and maps `huangli` to
`divination`. The later backend phase should preserve the four-domain expert
model while simplifying scenario ownership and context selection. A session
already binds to an expert; later model work can make the domain-owned context
boundary explicit. It must also move `huangli` from the `divination` expert to
`natal`, because personalized date selection requires the natal chart, dayun,
liunian, and liuri artifacts.

| Parameter | Frontend-only role | Target role |
| --- | --- | --- |
| person | `session.profile_id`, optional | unchanged |
| domain | visible practice domain and draft key | immutable session expert/domain plus domain-owned context ownership |
| intent | explicit marketing seed or focused-form result only | consultation context scenario; later a per-turn tool/artifact label, never a navigation level or frontend classifier |
| expert | derived server-side from the consultation context scenario | domain deployment (`natal`, `divination`, `fengshui`, `naming`) |

Visible domain switching starts or restores another domain session. Person
switching changes the view and never rebinds a session.

### 2.5 ParameterComposer

The composer stops being an internal part of the session view and becomes
a flow-level component that exists **before any session does**.

| Mode | Parameter behavior | Send semantics |
| --- | --- | --- |
| Draft (no session) | person plus domain parameters; the free-form text is consultation content, not routing input | send = bridge the selected domain/explicit seed to legacy skill, create session, and send the first message in one step |
| Session-bound | reflects the session's binding (2.1) | send = continue; tapping the person control switches the view, not the binding |

Sanitized Markdown rendering, AG-UI streaming, and entitlement cards are reused
as-is; the chat store gains a small domain state and a draft map
keyed by `{ personId, domainId }`. Switching domains loads the destination
domain's draft and leaves the source draft stored there; text is never carried
invisibly across people or domains. No protocol or backend changes are required
for the visible model or bridge.

## 3. Layout details

### 3.1 Layout model: consultation document + artifact layer

The product is not a page containing a chat. The consultation document is the
product's spine: person-home guidance, each reading, follow-up questions, and
tool-to-ask handoff form one continuous document. Domain outputs attach to
that document as artifacts. This avoids both a transient chat transcript and a
chart/tool canvas competing with the reading for primary focus.

| Layer | Contents | Mobile | Desktop |
| --- | --- | --- | --- |
| Context | person, restore, mine, partner (role-gated) | fixed context bar with overflow | context rail |
| Consultation document | person home, structured readings, follow-up questions | the only main scroll region | central primary column |
| Artifact | chart, tool result, hexagram | inline capsule; expanded sheet | inline capsule; peek/pinned sidecar |
| Action | domain control + composer | fixed two-row dock above safe area | two-row dock at the bottom of the document column |

Artifact presentation is progressive:

1. **Inline capsule** — one or two lines in the consultation document, available
   at the relevant turn.
2. **Peek** — a compact reference view showing the active palace/section.
3. **Pinned/full** — the complete chart/tool artifact, opened only by an
   explicit user action.

Chart focus may update a capsule or an already open sidecar, but it never
changes the user's reading surface by itself.

Artifact state is domain-scoped. The desktop sidecar mode and selected artifact
belong to the visible `(person, domain)` pair; switching domains never
leaves 张三's chart open beside 李四's consultation. The safe
default is `closed`; `peek` and `pinned` are explicit user commitments.

### 3.2 Mobile — active session (base layout)

The mobile layout is the source of truth, not a compressed desktop layout.
It must be complete at 320px: one person control, one domain control, one
readable input, one send/stop action, and no clipped primary recovery action.
Desktop may add persistent controls and sidecar space, but it cannot introduce
a different task hierarchy.

The four-domain strip is hidden while an active consultation is visible. It
returns through the composer's domain chooser, preventing product-like controls
from competing with the answer being read.

```text
+-----------------------------------+
| 张三 · 命理       [restore] [⋯]   | <- status + conditional restore
+-----------------------------------+
| [命盘 · 夫妻宫 ▲]                 | <- artifact capsule in flow
+-----------------------------------+
|                                   |
|  (consultation document)          |
|                                   |
+-----------------------------------+
| [张三 ▾] [命理 ▾]                 | <- parameters
| [input .......................][↑]| <- input + send/stop
+-----------------------------------+
```

Zone contract:

| Zone | Responsibility | Behavior |
| --- | --- | --- |
| Context status | current person/domain identity | fixed outside the flow scroll region; noninteractive on mobile so the screen has one person switcher |
| Conditional restore | return to the latest active session | visible only when the visible document is not that session; hidden while the keyboard is open |
| Consultation document | person home, structured readings, follow-up questions, and inline artifact capsules | the single vertical scroll owner; the latest reading and follow-up action remain the visual center |
| Artifact capsule | current chart/tool/hexagram reference | one/two-line summary plus explicit open action; stays anchored to the turn that produced it |
| Context overflow | account, orders, history, partner, and settings | keeps the mobile context bar to status + conditional restore + one menu; partner remains role-gated and first in the menu |
| ParameterComposer | the single action point, displaying its domain | separate person and domain controls on one row; input and send/stop occupy a second row |

The top status identifies the document being browsed; the composer control is
the sole mobile person switcher and changes the send-time payload. The two
surfaces can never display different people.

Domain-control states:

| Flow state | Mobile presentation | Why |
| --- | --- | --- |
| No session / person home / blank draft | four domain chips in a horizontal strip above the composer | entry discovery happens at practice level, not product level |
| Active session | the domain strip collapses into the composer's domain control | protects reading focus and removes redundant controls |
| Slot-filling step | domain appears in the step header; chat composer is hidden | one primary submit action per step |
| Composer domain chooser | bottom sheet with the four fixed domains | predictable switching without permanent strip noise |

The domain chooser also shows the current active consultation and a separate
**new consultation** action. Choosing the same domain restores its active
session; the explicit action starts a fresh one.

Domain-home suggestions are natural questions, not product tiles. For example,
命理 may suggest “今年事业怎么样？” or “下个月哪天结婚好？”, but those lines do
not create a second-level product navigation or task selector. A marketing
handoff may preselect the domain and prefill such a question, but the visible
App state remains person + domain + free-form text.

Layout contract:

- The flow owns the viewport (`100dvh`, with `visualViewport` handling on
  keyboard platforms). Context bar and composer stay outside the scroll
  region; person-home content, document sections, and artifact capsules share one
  internal scroll region.
- The action dock is always two rows: parameters, then input/action.
  It does not collapse person, domain, textarea, and send into one 320px row.
- The base layout remains in effect through `1023px`: extra width increases
  reading measure and touch comfort, but does not prematurely add a rail or
  sidecar hierarchy.
- The action dock respects `env(safe-area-inset-bottom)`. Keyboard resize keeps
  the composer visible without moving the document scroll anchor.
- The artifact sheet is the only expanded secondary scroll surface. It
  overlays the flow and contains its own scrolling; mobile never has two
  independently scrolling vertical main panes.

Fixed-area budget at the 320px design floor:

| State | Fixed areas | Budget |
| --- | --- | --- |
| Active session | status + parameter row + input row | at most `136px` before safe-area inset |
| Draft/home | status + domain strip + parameter row + input row | at most `180px` before safe-area inset |
| Keyboard open | compressed status + parameter row + input row | at most `128px`; domain strip and restore are hidden |

With a typical small-phone visual viewport of about `240px` while the keyboard
is open, at least `96px` of the current reading remains visible. The input
never covers the reading, and the scroll anchor does not jump when the keyboard
dismisses.

Mobile gesture and touch rules:

- The domain strip scrolls horizontally if a locale requires it, with proximity
  snapping; the selected chip
  chip is scrolled into view and the next chip peeks by 16-32px.
- The artifact sheet may be dragged between detents, but close buttons and
  keyboard activation are always available.
- System back closes the topmost layer in order: keyboard/picker, sheet,
  reader, form step, and only then leaves the flow.
- Do not use swipe to switch person or domain; both change consultation
  context and are too easy to trigger accidentally.
- Do not use swipe-to-delete consultations, pull-to-refresh during streaming,
  or long-press as the only path to an action.
- Adjacent bordered/filled controls keep 8-12px spacing; touch targets are at
  least 44px and extended hit areas never overlap.

Open behavior remains adaptive: an active session restores the latest reading;
otherwise the spine becomes person home — chart capsule, recent
interpretations, suggested questions, expanded domain chips, and the same
composer.

### 3.3 Mobile — artifact sheet detents (not "half screen")

At 390 pt width a readable ziwei grid needs roughly 520-560 px of height
(palace cells grow taller than wide once stars render). A fixed 50% sheet
cannot fit it. The sheet is full-width and content-sized, with detents:

The mobile artifact is focus-first, not a shrunken desktop grid. At 320-359px,
the default full state is a focused palace card plus a vertical palace list;
a complete grid is optional only when it can retain its minimum cell metrics.
At 390px and above, the full state may show the complete grid.

| Detent | Height | Entry | Use |
| --- | --- | --- | --- |
| Collapsed | inline capsule / one-line control | drag down | default; reading at full speed |
| Medium | content height needed for the focused section or whole grid, capped by viewport and safe areas | tap capsule or focused token | focused section and enough grid context visible |
| Full | ~95% | push up from medium | palace-by-palace deep reading |

Chart focus changes the highlighted palace and artifact control but never
changes the detent by itself; the user taps the control or token to open medium.
Streaming continues behind the sheet; dismissing restores scroll position.

Artifact sheet interaction:

- The drag handle and close control have at least 44px hit areas.
- Drag down moves full -> medium -> collapsed; drag up moves medium -> full.
- Backdrop tap and Escape close the current detent; an explicit close button is
  always visible.
- Sheet scrolling uses `overscroll-behavior: contain` and never scrolls the
  consultation document behind it.
- Closing returns focus to the originating capsule or token.
- Under `prefers-reduced-motion`, detent changes use a fade or instant position
  change rather than a large slide.

### 3.4 Mobile — no-profile state (new user)

```text
| 未建档              [restore] [⋯]    |
+-------------------------------------+
| 先问一卦，无需建档                    |
| [ 问一卦 ]                           |
| 次级：查黄历择日                     |
|                                     |
| 建档解锁：个人命盘 · 流年 · 合盘      |
| [ 创建档案 ]                         |
| 命理* | 问卦 | 风水 | 起名            |
+---------------------------------------+
```

The no-profile spine leads with one primary profile-free CTA (`问一卦`) and
suggested questions; generic 黄历择日 is a secondary path on the same screen.
Profile creation is the later upgrade path that unlocks personal charts, annual
readings, compatibility, personalized liuri, naming, and fengshui, not a gate
in front of the product. The 命理 domain is partially available rather than
disabled: generic almanac lookup works, while personal flow-day analysis
requests profile creation. The default selected domain is 问卦; partially
available domains show the available capability and the unlock reason.

### 3.5 Mobile — divination state (person not passed)

The person status shows "not applicable" with a one-line hint
“本场景不涉及个人”; the composer person control remains the path back to
张三. After a hexagram the context survives, and choosing 命理 keeps 张三
selected. Opacity alone is not the state cue. The hexagram becomes an artifact
capsule in the consultation document and must not display the person's name —
"parameter not passed" must not read as "parameter belongs to this person".

### 3.6 Mobile — profile switcher sheet

```text
| 最近使用 ------------------            |
| [张三] [李四] [王五] [+新建]           |
| 管理全部档案 ->                        |
```

Full profile management (list/new/edit/delete) lives behind this sheet and
in mine; it is not a navigation destination.

The switcher is a bottom sheet on mobile. Recent people, new-profile, and
manage-all rows are at least 44px tall; the selected person has a visible check
and text cue, not color alone. Backdrop, Escape, and system back close it, and
focus returns to the composer person control.

### 3.7 Desktop (>= 1024px)

The expanded layout is content-thresholded: it appears only when the context
rail, minimum consultation document, minimum artifact panel, gaps, and shell
padding all fit. With the current metrics that threshold is 1024px; a future
denser artifact must raise the threshold rather than squeeze the base flow.

```text
+----------+--------------------------+----------------+
| (person) |                          | [命盘·夫妻宫 ▲] |
|----------| consultation document    |----------------|
| 命理      | person home / readings   |   sidecar      |
| 问卦      |                          |                |
| 风水 ...  |                          |                |
|----------|                          |                |
| 我的      | [张三 ▾][命理 ▾]       |                |
|          | [input.............][↑]|                |
+----------+--------------------------+----------------+
  context rail      primary column       reference layer
```

The rail is a flow-control region rather than a page destination, but it
still needs navigation/tab semantics, visible focus, and a stable reading
order. It persistently exposes the four practice domains on desktop because
vertical space is inexpensive there and muscle memory is useful.

The central consultation document is primary. Use logical grid columns of
approximately `168-208px | minmax(400px, 1fr) | 320-440px`, with 16px gaps and
no more than 12px shell inline padding at 1024px. Rail labels may wrap to two
lines instead of truncating.

The canonical 1024px split is `168px rail | 400px document | 384px pinned
artifact`, plus 32px of gaps and 24px of shell inline padding. Wider viewports
grow the document column first, then the artifact panel.

The grid column may grow, but long-form reading text does not. Keep the
consultation text block at `max-width: 720-760px` (approximately 68-76ch,
tuned by locale). Artifact capsules and follow-up indentation may align to the
same measure. The remaining column space becomes stable margin, not unreadably
wide text.

The artifact sidecar has three states:

| State | Width | Use |
| --- | --- | --- |
| Closed | `0` | artifact exists only as an inline capsule |
| Peek | `320-360px` | focused palace/section plus compact context |
| Pinned | `384-440px` | full chart/tool reference, compact density at 1024px |

At 1024px, pinned mode must not compress the consultation document below its
minimum.
If an artifact cannot be readable at compact density, it opens as the overlay
sheet instead of narrowing the spine.

### 3.8 Long answers — structured sections and focus reading

| Layer | Surface | Entry |
| --- | --- | --- |
| Reading section | consultation document | default |
| Artifact sidecar/sheet | reference layer | capsule / focused token (3.3/3.7) |
| Focus reader | full-screen current-section/document view | "专注阅读" on long sections |

Long assistant turns render as document sections from P2, not as undifferentiated
chat bubbles: conclusion first, then key evidence, detailed interpretation,
artifact references, and follow-up questions. The UI can derive the initial
structure from Markdown headings; prompt conventions later make the structure
more reliable. The focus reader is therefore a deliberate reading mode, not an
escape hatch from an unusable stream.

Focus-reader details: no composer; adjustable font size; section jump list;
close returns to the exact document scroll position. Generation-side prompt
conventions (conclusion first, sectioned output, per-turn length budget) remain
a P3 prerequisite recorded in section 7.

Streaming behavior: if the user scrolls up during generation, auto-follow
pauses and a "new content" pill appears; tap resumes follow.

Mobile reading rules: the current section may use a one-line sticky header, but
it never covers the composer; conclusion and key evidence precede detail;
sections do not collapse while streaming; and the new-content pill sits directly
above the action dock.

### 3.9 Domain slot-filling and focused forms

A free-form question can reveal missing deterministic facts. Compatibility,
date selection, fengshui layout, and naming do not become product destinations:
they are slot-filling states inside their domain. When the input is too
structured for conversational typing, the domain opens a focused near-full-screen
form step (clearly titled, with a visible way back). The chat composer is
hidden during that step so form submit is the only primary action.

Mobile form rules:

- The form scrolls internally; title/back stays above it and the primary submit
  stays attached to the keyboard-aware bottom area.
- Fields are grouped by decision, with inline errors beside their inputs.
- System back first dismisses the keyboard or picker, then asks the dirty-state
  guard before leaving the step.
- Compatibility presents the two people as explicit segments (`本人` / `对方`)
  instead of one long mixed form.
- Date selection is planned as a 命理 slot-filling state, but its personal
  流日 implementation waits for the backend `huangli -> natal` ownership
  contract; the current legacy tool remains on its legacy route.
- Mobile-native date, time, and place controls are used where available.

#### Date-selection artifact target state

择日 belongs to the 命理 domain and is not a standalone App product. The target
artifact has two levels:

1. **Candidate list** — generic 黄历 rows with date, day pillar, event
   suitability, and the primary almanac reason. Do not eagerly calculate a flow
   chart for every candidate.
2. **Selected day** — an explicit selected date can load:
   - 黄历：宜忌、建除、黄道/黑道、星宿、彭祖百忌、时辰；
   - 八字流日：流日干支、十神、与命局/大运/流年的关系、神煞；
   - 紫微流日：流日命宫、四化、十二宫、星曜。

This is **not part of the frontend-only phase**. Personalized liuri requires the
natal chart/dayun/liunian model, and the current backend still routes `huangli`
to `divination`. The frontend must not display a person's name beside generic
almanac facts or pretend those facts are personalized. The full artifact ships
only after the backend contract moves ownership to `natal`.

When domain slot-filling is implemented, it will be addressed as user-visible
domain state plus a transient internal handoff seed (for example
`domain=natal&seed=hepan`); the seed will never be rendered as a task selector.
Back will close the step, refresh will restore the dirty-form draft, and
submission will return the result to the consultation document. The URL must
contain only opaque IDs/seeds — never birth facts or other personal input. The
focused-form query contract is intentionally deferred until that state exists.

### 3.10 Adaptivity and state coverage

Implementation and E2E coverage must include the design floor and the first
expanded layout: 320px, 390px, 768px, 1024px, and 1440px. At each width,
test 200% zoom, the RTL mirror, and the longest supported locale labels.
The 320px layout is the acceptance baseline; desktop is an additive layout and
must never require a narrower mobile action or hide a recovery action to work.
The required state matrix is:

| State | Required check |
| --- | --- |
| No profile | profile-free success path is primary; profile-bound action opens creation |
| Person home | chart capsule, suggested questions, expanded chips, and composer fit without an empty document void |
| Active session | status, composer payload, and session binding agree; two-row composer remains reachable |
| Fixed chrome | active-session and keyboard budgets hold at 320px |
| Artifact | inline capsule, mobile focus-first sheet, desktop closed/peek/pinned states |
| Domain control | expanded draft strip, active-session chooser, and form-step header all select the same domain |
| Streaming | follow-pause pill and stop action remain reachable |
| Slot-filling | back, refresh/dirty draft, submit, and follow-up composer |
| Artifact sheet | collapsed/medium/full detents, focused palace, scroll restore |
| Keyboard open | composer and at least 96px of current reading remain visible; scroll position is stable |
| Gestures | allowed and prohibited mobile gestures do not create an accidental context switch |
| Loading/error/partial | primary recovery action is not covered by context bar or sheet |

### 3.11 Component and interaction contract

| Component | Contract |
| --- | --- |
| `AppFlowShell` | owns viewport height, context bar/rail, main scroll region, and action dock; no other component writes global scroll behavior |
| `ConsultationSpine` | the only main vertical scroller; renders person-home states, structured reading sections, follow-ups, and inline artifact capsules in DOM/reading order |
| `ArtifactCapsule` | a real button with a visible open affordance; opens the domain-scoped sheet/sidecar and returns focus to the capsule on close |
| `ArtifactSurface` | one implementation with mobile focus-first sheet and desktop sidecar presentations; preserves scroll and focus separately per domain |
| `ParameterComposer` | owns separate person/domain controls, the domain-keyed textarea, and the send/stop action; parameters and input always render as two rows |
| `DomainSlotForm` | one primary submit action, visible back affordance, dirty-state guard, and no chat composer until the result enters the spine |
| `LegacyIntentBridge` | isolated user-invisible adapter; receives person/domain/question/seed and returns the current skill/expert contract |

The target-state draft domain strip uses tab semantics with **manual
activation**: arrows move focus between domains in the APG direction model
(including RTL), Enter/Space activates the focused domain, Home/End jump to
the ends, and Tab reaches the consultation document. The mobile strip keeps a
16-32px peek of the next chip when a locale requires overflow. All touch
targets are at least 44px; icon-only controls have accessible names.

## 4. Chart focus (client-side dictionary, P3)

- No model involvement, no runtime involvement (direct transport keeps the
  runtime a dumb pipe per contract C3).
- Dictionary is **derived from the loaded chart DTO** (palace names, dayun
  pillars, liunian terms), so person-specific terms match naturally;
  locale-keyed.
- Matching runs on accumulated message text (incremental cursor, ~200 ms
  debounce), sets a `chartFocus` state consumed by `ZiweiPanel` (`focused`
  class), the capsule, and the open sidecar/sheet.
- Passive mention -> a small pill plus a clickable token on the palace
  term in the rendered answer; tapping opens that domain's artifact surface at
  the focused palace. A surface never auto-opens from a passive mention — the
  user, not the match, commits the view change. Latest hit wins the highlight;
  any pulse runs only under `prefers-reduced-motion: no-preference`.
- Known limit: non-standard references ("你的婚姻位") are not matched;
  prompt wording conventions may nudge standard terms, but this is soft
  guidance, never a contract.

## 5. Implementation principles

### 5.1 Component architecture and reuse

The implementation is component-first and deliberately boring:

| Boundary | Contract |
| --- | --- |
| `AppFlowShell` | owns viewport height, context status/rail, one main scroll region, and action dock; contains no metaphysics rules |
| `ConsultationSpine` | renders person-home, structured reading sections, follow-ups, and artifact capsules in reading order |
| `ParameterComposer` | owns person/domain controls, domain-keyed drafts, input, and send/stop states |
| `PersonSwitchSheet` / `DomainChooserSheet` | reusable bottom-sheet controls; dispatch selection events only |
| `ArtifactCapsule` / `ArtifactSurface` | progressive disclosure and domain-scoped artifact state; no session mutation |
| `DomainSlotForm` | orchestrates existing shared form primitives and emits a typed result payload |
| `LegacyIntentBridge` | isolated adapter from visible person/domain/question to the current four-expert/session-context API; no UI coupling |

Rules:

- Prefer already-present libraries before adding anything: Vue, Pinia, Vue
  Router, Element Plus, VueUse, markdown-it, DOMPurify, and the official
  AG-UI client.
- Reuse the existing shared form and chart components in
  `web/shared/components`; marketing islands and the App must render the same
  domain result from the same component and locale catalogue.
- Shell/layout components are presentational and typed. Domain projection,
  routing, API calls, and session semantics live in composables/stores/adapters.
- Every new prop, emitted event, API payload, and persisted state field has a
  TypeScript contract. Avoid duplicate view-model shapes when the engine wire
  type is already sufficient.
- Use existing standards/contracts rather than private parallels: AG-UI events,
  engine MCP wire types, `consultation_context_v1`, generated route artifacts,
  shared scenario/domain config, locale catalogues, design tokens, and ARIA APG
  interaction patterns.
- Do not build a custom message protocol, form system, chart framework, modal
  system, or state library. Element Plus/VueUse cover common primitives; a new
  dependency needs a written tradeoff and bundle-size justification.

### 5.2 Engine result component catalog and theme composition

The existing refactor is not discarded. The four-domain IA, flow shell,
consultation document, artifact shell, forms, adapters, and tests remain the
foundation. This section adds the missing systematic component layer: **one
standard result-component family per deterministic engine method**, then
composes those families into product themes.

#### 5.2.1 Layer contract

```text
MCP / engine method
    ↓
typed API adapter and DTO
    ↓
standard engine result component family
    ↓
theme/template composition
```

Themes do not own domain calculations:

| Layer | Owns | Must never own |
| --- | --- | --- |
| API adapter | MCP tool name, request shape, DTO, transport | layout or session semantics |
| Result component | deterministic projection of one DTO | fetching, routing, stores, session mutation |
| Result composer | several deterministic results for one task | engine calls or protocol variants |
| Theme | placement, navigation, responsive shell, interaction container | metaphysics calculation or private DTO reshaping |
| Flow/state adapter | person/domain/session context and lifecycle | display projection |

“Old frontend” and “new frontend” are therefore not two implementations. They
are two themes over the same result-component catalog:

```text
engine result components
  ├── legacy tool-page theme
  ├── new consultation-flow theme
  └── marketing-island theme
```

#### 5.2.2 Canonical component catalog

A method does not have to become one monolithic component. It maps to one
**result-component family**: a public panel plus optional private sections.

| Method / contract | DTO | Target result family | Existing asset to preserve | Notes |
| --- | --- | --- | --- | --- |
| `city_coords` | city coordinate result | input-support adapter only | `citySearch` / region field | not a chart panel |
| `bazi_chart` | `BaziChartData` | `BaziChartPanel` | `BaziPillars` | basic four-pillar projection |
| `bazi_fullchart` | `BaziFullChartData` | `BaziFullChartPanel` | `BaziPanel`, `BaziRelations`, `BaziCycles`, `BaziYongShen` | full deterministic chart composition |
| `bazi_liunian` | `BaziLiuNianOutput` | `BaziLiuNianPanel` | `BaziLiunianDetail` | annual projection |
| `bazi_yongshen` | `BaziYongShenResult` | `BaziYongShenPanel` | `YongShenPanel` | counsel result |
| `bazi_bond` | `BaziBondResult` | `BaziBondPanel` | `HepanPanel` | rename concept from page name to engine result |
| `ziwei_chart` | `ZiweiChart` | `ZiweiChartPanel` | `ZiweiPanel` | palace grid projection |
| `ziwei_fullchart` | `ZiweiChart` | `ZiweiFullChartPanel` | `ZiweiPanel` + pattern sections | grid, patterns, sanfang, kongjie; compose daxian separately |
| `ziwei_daxian` | `ZiweiDaXianItem[]` | `ZiweiDaXianPanel` | section inside `ZiweiPanel` | standalone result family |
| `ziwei_liunian` | `ZiweiLiuNianData` | `ZiweiLiuNianPanel` | currently host-orchestrated | extract reusable panel |
| `bazhai_chart` | `BazhaiChart` | `BazhaiChartPanel` | `MingGuaSection` | natal-Gua/eight-house base |
| `bazhai_layout` | `BazhaiLayout` | `BazhaiLayoutPanel` | `BazhaiLayoutPanel` | preserve current component |
| `xuankong_chart` | `XuankongChart` | `XuankongChartPanel` | `XuankongPanel` | flying-stars panel |
| `liuyao_qigua` | `LiuyaoCasting` | `LiuyaoCastingPanel` | `DivinationCastPanel` | receipt-only projection |
| `liuyao_chart` | `LiuyaoChartResult` | `LiuyaoChartPanel` | large UI currently inside `DivinationView` | highest-priority extraction |
| `huangli_days` | `HuangliDay[]` | `HuangliDaysPanel` + `HuangliDayDetail` | current panels | candidate list and day detail |
| `bazi_liuri` | `BaziLiuRiOutput` | `BaziLiuRiPanel` | backend contract aligned | selected-day date selection |
| `ziwei_liuri` | `ZiweiLiuRiData` | `ZiweiLiuRiPanel` | backend contract aligned | selected-day date selection |

Naming rules:

- Component names describe the engine result, not the page: `BaziBondPanel`,
  not `HepanPageChart`.
- `Panel` is a deterministic result projection.
- `Theme` is a composed presentation template.
- `Form` owns input collection only.
- `use*Result` owns loading a DTO, never rendering it.
- Existing components remain compatibility facades while consumers migrate.

Target directory:

```text
web/shared/components/engine/
  bazi/
    BaziChartPanel.vue
    BaziFullChartPanel.vue
    BaziLiuNianPanel.vue
    BaziYongShenPanel.vue
    BaziBondPanel.vue
  ziwei/
    ZiweiChartPanel.vue
    ZiweiFullChartPanel.vue
    ZiweiDaXianPanel.vue
    ZiweiLiuNianPanel.vue
  liuyao/
    LiuyaoCastingPanel.vue
    LiuyaoChartPanel.vue
  huangli/
    HuangliDaysPanel.vue
    HuangliDayDetail.vue
  fengshui/
    BazhaiChartPanel.vue
    BazhaiLayoutPanel.vue
    XuankongChartPanel.vue
```

#### 5.2.3 Result component contract

Every public result panel is typed, presentational, and deterministic:

```ts
defineProps<{
  result: BaziFullChartData
  focused?: string
  density?: 'compact' | 'full'
}>()
```

Rules:

- Props use engine DTO names where practical; local projection is allowed, but
  private duplicate DTO shapes are not.
- No API call, router access, Pinia store, session mutation, or global event.
- No hard-coded user-facing string; locale keys remain shared.
- All text and grid layouts wrap at 320px.
- Focusable domain cells expose real names and states.
- Every state that can be absent has an explicit branch.
- Charts use semantic HTML/CSS/SVG, not ECharts.
- Existing Element Plus and shared form primitives are reused.

A typical loader is separate:

```ts
const result = await engineBaziFullChart(chart)
```

The loader owns transport and errors; the panel only renders a successfully
typed result.

#### 5.2.4 Theme composition

Themes compose result families; they do not copy them.

##### Legacy tool-page theme

```text
header
shared input form
engine result panels
record list
consultation CTA
```

##### New consultation-flow theme

```text
context/domain controls
domain slot form
consultation document
artifact capsule
ArtifactSurface -> same engine result panel
composer
```

##### Marketing-island theme

```text
scenario explanation
free shared input form
engine result panel
consultation CTA
```

The same panel may appear in all three themes. Theme CSS may change placement
and density, but must not fork the panel implementation.

#### 5.2.5 Date-selection composition example

择日 is a composition, not a single engine result:

```text
DateSelectionTheme
├── event/range form
├── HuangliDaysPanel
├── selected HuangliDayDetail
├── BaziLiuRiPanel
├── optional ZiweiLiuRiPanel
└── decision summary
```

Current frontend-only scope:

- generic 黄历 can use `huangli_days`;
- personalized Bazi/Ziwei 流日 waits for exposed web contracts and correct
  natal-expert ownership;
- no component may present generic almanac facts as personalized.

#### 5.2.6 Catalog migration plan

1. **Inventory and freeze contracts** — create a typed catalog manifest,
   current-component inventory, visual baselines, and public props.
2. **Extract result panels** — move deterministic display out of page views
   into `web/shared/components/engine/**`; old paths become compatibility
   facades.
3. **Add typed loaders** — separate API adapters from panels and cover loading,
   error, empty, and retry states.
4. **Adopt in legacy themes** — replace old page-local chart implementations
   with the shared panels while preserving current URLs and behavior.
5. **Adopt in the flow theme** — map `ArtifactDescriptor.payload` to the same
   result families.
6. **Compose cross-method themes** — build date selection and other compound
   presentations from existing panels.
7. **Remove compatibility facades** — delete old paths only after route and
   E2E parity pass.

#### 5.2.8 Engine result lifecycle contract

All deterministic engine panels share one lifecycle protocol. The panel remains
presentational; the lifecycle lives in a shared composable and a generic state
boundary.

```text
typed input
    ↓
canonical cache key
    ↓
typed MCP/domain loader
    ↓
EngineResultBoundary
    ↓
typed result panel
```

The shared contract is:

```ts
const resource = useEngineResult({
  input: () => baziFullChartInput,
  load: input => engineBaziFullChart(input),
  staleTime: 5 * 60_000,
})

const { data, status, error, refreshing, reload } = resource
```

The composable owns:

- canonical input serialization;
- in-memory cache entries;
- loading / success / error / idle status;
- stale-while-revalidate for an existing result;
- concurrent request de-duplication by cache key;
- explicit invalidation and forced reload;
- stale-result protection when input changes.

The boundary owns only generic presentation:

- loading state;
- error/retry state;
- result slot;
- `aria-busy`;
- no metaphysics rules.

Panels continue to own domain projection and receive typed engine DTOs. They do
not fetch data, access stores, or know whether the result came from cache.

##### Phase 3 lifecycle slice

1. Add `useEngineResult` as a shared, dependency-free composable.
2. Add `EngineResultBoundary` as the generic state shell.
3. Cover cache hit, stale revalidate, input change, error retry, invalidation,
   concurrent de-duplication, and boundary accessibility with tests.
4. Existing panels and themes may adopt the boundary incrementally; this slice
   does not migrate all engine methods at once.

##### Phase 3 adoption status

Reuse boundary:

- Result panels (`BaziChartPanel`, `ZiweiChartPanel`, `HuangliDaysPanel`, and the
  rest of `web/shared/components/engine/**`) stay presentational and
  self-contained: props in, markup out, no fetch, no cache, no MCP import.
- The DTO contract stays in the shared type layer (`api/engine/*`,
  `EngineResultDtoMap`, `contract.ts`, `catalog.ts`). Panels import it with
  `import type` only.
- `EngineQueryPanel` (`web/shared/components/engine/EngineQueryPanel.vue`) is the
  single query wrapper: it owns the resource lifecycle, code-to-locale error
  text, retry, and the boundary. It is composed at the page/island shell, never
  inside a result panel.
- No per-product result-wrapper components. App views and marketing islands
  compose the same `EngineQueryPanel` + shared forms + shared panels, keeping
  only their own input state, loaders, and CTA.
- `useEngineResult` takes a required resource `namespace`; the cache key is
  `<namespace>:<canonical input>`.
- Adopted resources: `natal.bazi`, `natal.ziwei`, `natal.bazhai`,
  `natal.bazi.liunian`, `natal.ziwei.liunian`, `liunian.bazi`,
  `liunian.bazi.liunian`, `fengshui.bazhai`, `fengshui.xuankong`,
  `fengshui.layout`, `hepan.bond`, `huangli.days`, `naming.assessment`,
  `divination.chart`, `tool.hepan`, `tool.fengshui`, `tool.naming`.
- Marketing islands are thin shells over the same components: `NatalTool`,
  `LiunianTool`, `HepanTool`, `FengshuiTool`, `HuangliTool`, `NamingTool`.
  The homepage quick-chart reuses `NatalTool` instead of redirecting to the App.
- A free-tool submit never navigates. Only the interpretation CTA (LikiChatCta
  or the tool's in-result CTA) enters the App. The homepage hero and service
  card "try it" links stay on the marketing site and target the on-page free
  tool (`#chart-tool` / `#free-tool`); natal's zodiac and birth-time links do
  the same. Scenario landing hero/bottom CTAs remain Liki Chat handoffs.
- Shared tool surfaces are fixed: `ToolCta` for interpretation handoffs,
  `EngineQueryPanel`/`EngineResultBoundary` for query loading/error/retry, and
  `ToolError` for mutation or validation failures that are not queries. Islands
  do not hand-roll these three states.
- `liuyao_qigua` remains an explicit mutation with `ToolError` handling; the
  derived chart query owns the resource boundary and supports retrying the same
  casting receipt. `city_coords` remains input-support and is not a result
  resource.

#### 5.2.7 Extraction provenance policy

Wave 4 is an **extraction refactor**, not a redesign. The current production
implementation is the source of truth for deterministic projections. A new
engine result panel is created by moving or wrapping existing markup,
computed projections, styles, locale keys, and tests—not by inventing a new
interpretation of the DTO.

Every catalog entry must record its source:

```ts
interface EngineResultCatalogEntry {
  method: keyof typeof ENGINE_TOOLS | 'bazi_yongshen'
  dto: unknown
  target: string
  sourcePaths: readonly string[]
  sourceProjection: string
  compatibilityAliases?: readonly string[]
  baselineTest: string
  knownGaps?: readonly string[]
}
```

Rules:

- `sourcePaths` is required. A target component with no production source is a
  catalog gap, not an implementation ticket.
- `sourceProjection` names the exact current behavior being extracted, such as
  “six lines bottom-up with moving-line cue” or “Ziwei twelve-palace grid in
  configured ZHI order.”
- The first panel PR is mechanical: move/wrap existing projection, replace
  page-local state with typed `result` props, and preserve locale keys and
  design tokens.
- Visual or domain-behavior changes require a separate reviewed PR with their
  own rationale and tests; they cannot ride inside an extraction PR.
- If current behavior looks wrong, record a known limitation and preserve it
  unless it blocks accessibility, privacy, or a supported journey. Fix it in a
  separate bug-fix PR.
- Existing fixtures, engine contract tests, tool E2E, and visual baselines are
  the parity oracle. Do not replace them with synthetic “expected” data.
- A new panel may add only the minimal wrapper needed for density/focus props.
  It may not silently reorder columns, rename domain fields, reinterpret
  missing values, or add new conclusions.
- Tests must mention the source path in the test description or fixture name,
  making provenance auditable.

### 5.3 Chart rendering policy

The current domain “charts” are semantic domain layouts, not Cartesian plots:

| Artifact | Correct presentation |
| --- | --- |
| Ziwei natal/liuri grid | accessible CSS grid/SVG palaces with focusable palace cells |
| Bazi pillars and liuri relations | semantic table/cards, not a plotted graph |
| Bazhai/fengshui layout | nine-palace grid plus semantic table |
| Liuyao hexagram | DOM/SVG lines with text labels |
| Huangli candidate range | date cards/list with expandable details |

Therefore **do not introduce ECharts in P1/P2 merely to draw these domain
charts**. A general chart library would add bundle weight and make text,
localization, keyboard focus, chart-focus highlighting, and testing harder. The
existing shared Vue components already use data-driven semantic HTML/CSS and
should be refactored/reused instead of replaced.

ECharts is allowed only when a concrete quantitative visualization exists — for
example a future 五行分布 bar, 大运 timeline, annual-trend line, or radar chart.
If that happens:

1. add `echarts` through `echarts/core` with only required renderers/charts;
2. lazy-load it outside the initial consultation payload;
3. use design-token colors and project locale text;
4. give the visualization a fixed responsive height and reduced-motion mode;
5. provide a semantic table or text alternative for the same data;
6. verify bundle-size and runtime budgets in `make check`.

No chart library is a P1/P2 prerequisite.

## 6. Migration and implementation plan

### 6.0 Delivery and engineering governance

This is a full frontend refactor, but it must not become a long-lived parallel
rewrite. Work lands in trunk behind the `APP_FLOW_REDESIGN` feature flag as a
sequence of independently testable increments.

#### Plan-first work rule

No implementation task starts without a written, reviewable plan. The plan can
be lean, but it must be explicit and traceable. A phase or ticket is not ready
for implementation until it records:

| Planning input | Required content |
| --- | --- |
| User outcome | the observable journey and supported states |
| Scope | files/modules, reused components, APIs, routes, stores, and locale keys |
| Contracts | TypeScript types, query state, API payload, route contract, and persisted shape |
| Design evidence | mobile/desktop layout, reading order, focus behavior, and artifact behavior |
| Test plan | unit/component/contract/E2E/a11y cases, fixtures, and unsupported cases |
| Migration impact | redirects, feature flag, rollback, and old-session projection |
| Risk assessment | person/session safety, privacy, performance, i18n, and accessibility risks |
| Acceptance criteria | binary checks that define done |

The implementation plan is updated when scope changes. Silent scope growth,
private compatibility branches, and “implement first, design later” patches are
not accepted.

#### Clean replacement policy

“New first, delete old later” means additive implementation, not compatibility
hacks:

1. Build the new flow behind a feature flag with its own contracts and tests.
2. Keep the legacy flow fully usable until the replacement reaches parity.
3. Delete a legacy route/component only after its redirect and replacement E2E
   pass.
4. Never introduce a transitional design that hides a missing contract, fakes a
   routing decision, or becomes a permanent dependency.
5. If a capability requires a backend change, stop at an explicit “pending
   backend” state. Do not approximate it in frontend routing.

For this refactor, the four domains route only to their correct current
experts. A free-form question is consultation content; it is not a hidden
classifier input. `huangli` remains a legacy skill and is not promoted into the
new natal domain flow until the backend owns it under `natal`.

#### Engineering workflow

Every slice follows the same sequence:

1. **Discover** — inspect the existing implementation, tests, routes, locale
   keys, API contract, and analytics.
2. **Plan** — write the slice plan and acceptance criteria.
3. **Review the plan** — reach agreement on contracts, reuse, boundaries, and
   risks before coding.
4. **Write failing tests/contracts first** — cover adapters, routes, state
   transitions, accessibility, and regression cases where practical.
5. **Implement the smallest reviewable slice** — keep PRs cohesive and
   independently revertible.
6. **Verify locally** — run lint, typecheck, focused tests, route checks, and
   affected E2E.
7. **Review code against the plan** — reviewers check behavior, contracts,
   reuse, accessibility, tests, and documentation; style-only review is not
   sufficient.
8. **Validate in CI** — run the required gate before merge.
9. **Document and instrument** — update route/component docs, state diagrams,
   locale catalogues, telemetry, and rollback notes.
10. **Release and observe** — use the feature flag, monitor errors and product
    metrics, and roll back without database or protocol cleanup if needed.

#### Definition of ready

A work item is ready only when:

- the desired user state and unsupported states are named;
- current reusable assets are inventoried;
- the data/route/session contract is defined;
- test and accessibility acceptance criteria are explicit;
- migration, rollback, and telemetry are understood;
- no open question blocks a safe partial implementation.

A capability that needs a backend contract can be planned and tracked, but it
is not implementation-ready until that contract exists.

Questions that do not block the first slice are recorded as follow-ups with
owners and phase assignments.

Rollout rules:

- The legacy App remains usable until the replacement reaches functional and
  accessibility parity.
- Every increment keeps `make lint`, typecheck, unit/component tests, route
  contracts, and affected E2E passing.
- No increment ships a half-working hybrid as the default experience.
- Public marketing URLs and current API contracts remain stable through the
  frontend-only phases.
- New UI code is additive until the final redirect/cleanup phase; deprecated
  components are removed only after their replacements pass the relevant E2E
  suites.

Feature rollout:

1. Set the build/deploy environment variable
   `VITE_FEATURE_APP_FLOW_REDESIGN=true`. Because this is a Vite build-time
   flag, restart local development or rebuild/deploy after changing it; do not
   treat a browser-only environment edit as a rollout.
2. Keep the legacy App as the safe default. An unset, malformed, or
   `false` value keeps Classic at `/app` and disables Consultation registration and leaves every existing
   route unchanged.
3. Roll back by removing the variable or redeploying with
   `VITE_FEATURE_APP_FLOW_REDESIGN=false`. No database, session, URL, or API
   cleanup is required because the first increments add no durable domain
   session protocol.

Standard definition of done for every implementation slice:

1. TypeScript contracts and pure adapters are tested.
2. All affected locale catalogues are complete.
3. Keyboard-only and screen-reader paths work.
4. The 320px, 390px, 768px, 1024px, and 1440px states pass.
5. Route contracts and redirects are updated.
6. Loading, empty, error, partial, offline/unreachable, and permission states
   are implemented.
7. No new dependency is added without a written bundle/runtime tradeoff.

#### Engineering standards

| Standard | Expectation |
| --- | --- |
| Change control | Plan, scope, contracts, tests, migration, and rollback are reviewed before implementation; deviations are recorded |
| Configuration management | Shared config/types are the source of truth; generated route artifacts and lockfiles are committed reproducibly |
| Requirements traceability | Each phase maps to user journeys, tests, telemetry, and acceptance criteria |
| Testing strategy | Use the smallest reliable level: pure unit → component → contract → integration/E2E → visual/a11y/performance |
| Accessibility | WCAG 2.2 AA intent, ARIA APG patterns, keyboard-first implementation, and axe/keyboard E2E coverage |
| Internationalization | No hard-coded user-facing strings; test zh-Hans, zh-Hant, and English growth/wrapping |
| Security/privacy | Preserve CSP, DOMPurify, auth redirects, ownership checks, and no personal facts in URLs/logs |
| Performance | Initial route budget, lazy heavy artifacts, stable streaming rendering, and no unnecessary dependencies |
| Observability | User-safe metrics for bridge decisions, route handoffs, restore, errors, and artifact usage |
| Documentation | Keep this plan, `ux-architecture.md`, route inventory, component contracts, and ADR-like decisions current |
| Review practice | At least one substantive reviewer; author does not merge unresolved contract/accessibility questions |

Tests are not limited to the cases that already exist. When a behavior is
high-risk — person/session binding, bridge routing, payment return, dirty form
protection, focus restoration, or mobile keyboard handling — the test must fail
for the known bad implementation, not merely document the happy path.

### 6.0.1 Parallel execution model and anti-drift controls

Parallel work starts only after the Phase 0 gate. Each agent owns one lane, one
directory, and one integration responsibility. Agents do not redesign
architecture or edit shared contracts.

#### Preconditions for opening parallel lanes

1. Phase 0 contracts are merged into the integration branch.
2. `LegacyIntentBridge` has deterministic fixtures and tests.
3. Route-contract validation covers legacy and new query state.
4. Component props/events and store API are frozen for the wave.
5. Locale keys are preallocated so agents do not edit all locale files freely.
6. Design tokens and component directory boundaries are published.
7. Feature flag, rollback, and branch naming are agreed.

#### Branch and merge train

Use short-lived lane branches based on the integration branch:

```text
integration:
  feat/app-flow-redesign

lane branches:
  feat/flow-shell-a
  feat/flow-composer-b
  feat/flow-sheets-c
  feat/flow-spine-d
  feat/flow-qa-e
```

Merge order is deterministic:

1. Contract PR (serial).
2. Shell/composer/sheets/spine PRs (parallel).
3. Integration wiring PR (serial).
4. Contract/E2E repair PRs (small, targeted).
5. Cleanup PR (serial).

A lane must rebase at least daily and before requesting review. If a shared
contract must change, the agent stops and opens a contract-change request; it
does not patch shared files locally to make its lane compile.

#### Parallel Wave 2 — artifact integration, QA, and hardening

Wave 2 starts only after the Phase 0/1 integration gate. The artifact shell
contracts are now frozen, so implementation can split into four non-overlapping
lanes.

| Lane | Owned paths | Scope | Must not do |
| --- | --- | --- | --- |
| A — Artifact integration | `web/app/src/views/DomainFlowView.vue`, new `useDomainArtifacts` if needed | Wire `ArtifactCapsule` / `ArtifactSurface` into the visible flow; manage selected/open state; pass only real descriptors | Do not fabricate payloads or parse chat prose into charts |
| B — Mobile/a11y document QA | `AppFlowShell.vue`, `ConsultationDocument.vue`, their tests | Harden 320px, touch targets, focus, reduced motion, long-content behavior | Do not change session/API contracts |
| C — Flow E2E | `web/e2e/journeys/domain-flow.spec.ts`, new fixtures only if isolated | Cover restore, free-form submit, person/domain drafts, `huangli` pending-backend | Do not modify app implementation to make a test pass |
| D — Docs/i18n/route contract | docs, locale catalogues, route contract scripts | Sync artifact/domain vocabulary, feature-flag instructions, route inventory | Do not alter runtime behavior |

Shared files remain with the integration owner. Locale additions are submitted
to Lane D or attached as a contract-change request; implementation lanes must
not edit all locale catalogues independently.

Merge order:

1. Artifact integration PR.
2. Mobile/a11y and E2E PRs in parallel.
3. Docs/i18n/route contract PR.
4. Full integration gate.

#### Component interface freeze

Before parallel implementation, publish these contracts as TypeScript types:

```ts
interface FlowShellProps {
  personLabel?: string
  domainLabel: string
  restoreVisible?: boolean
}

interface FlowShellEmits {
  restore: []
}

interface ParameterComposerProps {
  modelValue: string
  domains: readonly AppDomain[]
  domainId: AppDomainId
  personLabel?: string
  domainLabel: string
  sending?: boolean
  showDomainStrip?: boolean
}

interface ParameterComposerEmits {
  'update:modelValue': [value: string]
  'update:domainId': [domainId: AppDomainId]
  'select-person': []
  'select-domain': []
  submit: []
  stop: []
}

interface PersonSwitchSheetProps {
  modelValue: boolean
  profiles: readonly BirthProfile[]
  selectedPersonId: string | null
  loading?: boolean
  error?: string
}

interface DomainChooserSheetProps {
  modelValue: boolean
  domains: readonly AppDomain[]
  selectedDomainId: AppDomainId
}
```

These interfaces are the integration contract. Agents may refine internal CSS,
tests, accessibility, and local computed state, but must not change public
props/events without a contract-change request.

#### Anti-drift gates

Every lane PR must pass the same gate:

```text
make lint-js
make typecheck
make test-js
route-contract checks
focused lane tests
```

Reviewers reject a lane PR when any of the following occurs:

- it edits files outside its owned paths;
- it changes a shared contract without a contract-change request;
- it adds a dependency, CSS utility pattern, chart library, or private protocol;
- it hard-codes user-facing text or expert IDs;
- it creates a second component that duplicates an existing shared component;
- it changes IA, session semantics, or expert ownership;
- it adds loading-only UI without error/empty/disabled handling;
- it introduces layout that creates a second mobile scroll owner;
- it leaves focus, keyboard, RTL, reduced-motion, or safe-area behavior broken.

#### Integration review checklist

The integration owner checks each merge in this order:

1. Scope matches the lane contract.
2. Shared files were not modified outside Lane E.
3. TypeScript props/events match the frozen interface.
4. Tests cover behavior and the known bad implementation.
5. No component imports another lane’s internals.
6. State changes remain limited to person/domain selection and drafts.
7. Locale keys exist in zh-Hans, zh-Hant, and English.
8. Accessibility and viewport checks are documented or reproduced.
9. No public route or default experience changes without an explicit rollout PR.
10. The lane can be reverted independently.

#### Completion criteria for the parallel wave

The parallel wave is complete only when:

- all lane tests and integration tests pass;
- the shell mounts behind the feature flag without altering the default App;
- person/domain drafts are isolated;
- sheets restore focus and close through Escape/backdrop/button;
- the spine has loading, empty, error, and profile states;
- no seven-task hierarchy or legacy product route is exposed in the new shell;
- the integration branch passes the full frontend gate.

### 6.1 Phase 0 — contracts, adapters, and fixtures

**Goal:** make the domain model executable before changing the visible shell.

Work items:

- Add a shared four-domain contract, for example
  `web/shared/config/domains.json`, containing:
  - stable domain IDs (`natal`, `divination`, `fengshui`, `naming`);
  - locale label keys;
  - base legacy skill;
  - accepted legacy seeds for projection;
  - bridge-eligible seeds for new sessions;
  - default marketing/tool route during the hybrid phase;
  - profile policy (`required`, `optional`, `partial`).
- Keep `scenarios.json` as the public seven-island contract and add the mapping
  from each marketing scenario to `domain + seed`.
- Implement `LegacyIntentBridge` as a pure deterministic adapter:

```text
input:
  domainId
  personId?
  seed?
  slotFormResult?

output:
  skill
  expert
  fallbackUsed
  reason
```

- Add exhaustive bridge fixtures for:
  - each domain base fallback;
  - explicit marketing seeds;
  - explicit focused-form results;
  - cross-domain seeds;
  - pending-backend `huangli`;
  - locale variants and empty input.
- Add domain-to-legacy-session projection for restoring old sessions.
- Extend route generation/check scripts for `/app`, its `/app/consultation` preview alias, `/app/classic`, canonical `domain`,
  transient `seed` and legacy `service` handoffs, and transient `step` query
  state while preserving public marketing paths.
- Add locale keys and test fixtures for four domains before introducing UI.
- Create an implementation inventory mapping current components to reuse:

| Current asset | Reuse |
| --- | --- |
| `ChatThread`, `ChatMessageItem`, `ChatComposer`, `ChatStreamStatus` | consultation stream and structured reading sections |
| `AppMarkdown` | sanitized domain document rendering |
| shared chart panels | artifact surfaces |
| shared form primitives | domain slot-filling |
| Element Plus dialogs/sheets/popovers | person/domain/overflow sheets where suitable |
| Pinia stores | session, profile, entitlement, and domain flow state |
| AG-UI client | unchanged transport contract |

Exit criteria:

- Domain config and bridge are covered by unit tests.
- No component yet depends on the bridge; it is a pure, typed adapter.
- Route contract scripts understand old and new handoffs.
- Design/mobile acceptance criteria are agreed before shell implementation.

### 6.2 Phase 1 — mobile-first shell behind the flag

**Goal:** introduce the new physical shell without absorbing chat or tools yet.

Work items:

- Build `AppFlowShell`:
  - fixed person/domain status;
  - conditional restore;
  - context overflow;
  - one main consultation scroll region;
  - two-row action dock.
- Build the four-domain draft strip and `DomainChooserSheet`.
- Build `PersonSwitchSheet`, reusing the current birth-profile store.
- Build `ParameterComposer`:
  - person control;
  - domain control;
  - domain-keyed textarea;
  - send/stop control;
  - draft map keyed by `{personId, domainId}`.
- Build `ConsultationSpine` skeleton with person-home, loading, empty, error,
  and permission states.
- Keep all current routes reachable. In this phase, a domain chip may navigate
  to that domain's current default page; no seven-product hierarchy is shown.
- Apply the 320px fixed-area and keyboard budgets from section 3.2.
- Replace shell-level accessibility behavior without weakening the old shell.

Primary test additions:

- Shell layout at 320, 390, 768, 1024, and 1440.
- Keyboard route focus and skip behavior.
- Person/domain sheet focus trap, Escape, backdrop, and restore focus.
- Draft isolation by person/domain.
- Conditional restore visibility.
- Mobile keyboard and safe-area behavior.
- Route-contract coverage for legacy paths plus new domain query state.

Exit criteria:

- Flagged shell can wrap legacy routes without losing navigation.
- No viewport has two main vertical scroll owners on mobile.
- Shell accessibility tests pass.
- No user-visible seven-task hierarchy exists in the flagged shell.

#### Phase 1 implementation slice

The first shell PR is intentionally limited to contract-bearing, testable
components; it does not register a public route or claim consultation delivery:

| Component | Contract |
| --- | --- |
| `FlowDomainStrip` | four-domain tab strip; manual activation; emits `update:modelValue` |
| `FlowSheet` | Element Plus bottom drawer wrapper with title, close, focus, and mobile sizing |
| `PersonSwitchSheet` | profile list, selected cue, create/manage/retry actions; emits selection only |
| `DomainChooserSheet` | four fixed domains and explicit new-consultation action |
| `ParameterComposer` | person/domain row + input/action row; Enter submits, Shift+Enter newlines, IME-safe |
| `ConsultationSpine` | one scroll region; person-home and loading/error/empty states |
| `useDomainFlowStore` | person/domain selection and domain-keyed drafts only; no chat API mutation |

The slice mounts no route by default. Integration and chat delivery remain
Phase 2 work. This avoids shipping a composer that appears functional before
the consultation contract exists.

### 6.3 Phase 2 — free-form consultation MVP

**Goal:** make the four-domain consultation flow usable with the existing chat
API and existing session model.

Work items:

- Add the domain flow view behind the same flag and connect it to the current
  chat store.
- Reuse AG-UI streaming, message persistence, stop-generation, retry, and
  entitlement behavior.
- Apply `LegacyIntentBridge` only when creating a new legacy session.
- Project restored legacy sessions to visible domains:
  - `natal/liunian/hepan/huangli -> 命理`;
  - `divination -> 问卦`;
  - `fengshui -> 风水`;
  - `naming -> 起名`.
- Preserve mandatory person/session binding; switching person switches views
  and never rebinds.
- Render assistant output as structured `ReadingSection` blocks using current
  sanitized Markdown output.
- Add follow-pause and jump-to-latest behavior.
- Keep the old chat list available during this phase.

Primary test additions:

- Free-form send creates a correctly bridged session.
- Marketing `?service=` values become `domain + seed`.
- Ambiguous 命理 input falls back to `natal`.
- `huangli` handoffs remain on the legacy route until backend ownership moves
  to `natal`; the new flow returns a pending-backend state instead of faking a
  route.
- Existing sessions restore into the correct visible domain.
- Person switching never mutates `profile_id`.
- Stop/retry/error paths preserve drafts.
- Streaming does not steal scrolling or cover the composer.

Exit criteria:

- A user can select person/domain, ask a free-form question, stream a response,
  stop/retry, and resume under the flag.
- No old chat behavior is lost for legacy routes.
- Bridge routing decisions and pending-backend outcomes are instrumented.

#### Phase 2 implementation slice

The first consultation integration is deliberately limited to one vertical slice:

- Mount the flag-gated Consultation route at `/app` and move Classic to `/app/classic`.
- Load profiles, entitlements, and existing sessions.
- Restore the latest `(person, domain)` session; otherwise show person home.
- Submit a new free-form question through `LegacyIntentBridge` and the existing
  chat store.
- Reuse `ChatThread`, `ChatStreamStatus`, and the existing AG-UI transport.
- Keep `huangli` explicit seeds in `pending-backend`; do not fake a route.

This slice does not yet absorb tool forms, add composite date artifacts, or
replace the chat list. It proves the visible domain model and session lifecycle
with the current four-expert/session-context API.

#### Phase 2 completion slice

Phase 2 is not complete until the reused chat thread is replaced by the domain
consultation document:

- Build `ConsultationDocument` as the structured reading surface.
- Render user turns as compact questions and assistant turns as sanitized
  Markdown reading sections.
- Track main-scroll distance in `AppFlowShell`.
- Pause auto-follow when the user scrolls up and expose a new-content pill.
- Restore auto-follow only through the explicit pill.
- Keep stop/status outside the scroll region and never cover the composer.

This slice uses the existing sanitized Markdown pipeline and AG-UI streaming.
It does not add chart artifacts, backend intent routing, or full-text reader
mode.

### 6.4 Phase 3 — artifact layer

#### Phase 3 implementation slice

The first artifact slice is UI infrastructure only; it does not fabricate chart
data or imply that a chat answer produced a deterministic chart.

Contracts:

```ts
type ArtifactKind =
  | 'ziwei' | 'bazi' | 'hepan' | 'liuyao'
  | 'huangli' | 'fengshui' | 'naming'

interface ArtifactDescriptor {
  id: string
  domainId: AppDomainId
  kind: ArtifactKind
  titleKey: string
  state: 'ready' | 'pending-backend' | 'unavailable'
  payload?: ArtifactPayload
}
```

Components:

- `ArtifactCapsule`: progressive disclosure trigger, `aria-expanded`, and state.
- `ArtifactSurface`: one semantic surface with mobile bottom sheet and desktop
  sidecar presentation; explicit close; scroll containment.
- `artifactPanel`: typed rendering map that reuses existing shared chart panels.
- `buildArtifactDescriptors`: pure adapter; returns descriptors only for data
  that actually exists. An empty result must yield no fake artifact.

The first integration passes zero descriptors into `DomainFlowView` until a
backend artifact contract exists. This proves capsule, surface, responsive
states, a11y, and lifecycle without pretending an AG-UI answer produced a
domain chart.

### 6.5 Phase 4 — absorb forms and tool pages

**Goal:** replace product/tool destinations with domain slot-filling.

Implement in this order:

1. **问卦 casting** — profile-free, validates the flow with a non-chart artifact.
2. **合盘** — second-person consent and shared birth form.
3. **风水** — layout and birth/year inputs.
4. **起名** — naming inputs and result rendering.

黄历择日 and personalized Bazi/Ziwei 流日 are intentionally **not absorbed in
the frontend-only phase**. Their correct semantic owner is the natal expert, and
the current backend routes `huangli` to `divination`. Implementing that flow now
would either use the wrong expert or fake the missing backend contract.

For each tool:

- Reuse the current shared form component and validation contract.
- Wrap it in `DomainSlotForm`.
- Keep one primary submit action.
- Convert `peekConsultationHandoff` into in-flow state transfer.
- Render the result as an artifact capsule plus consultation section.
- Preserve record save/open/rename/delete behavior where supported.
- Add the old route redirect only after the new state reaches parity.
- Keep the public marketing island and its free tool unchanged.

Date-selection requirements are deferred with the backend contract:

- Keep the existing public `/huangli` island and legacy App route until P4.
- Existing huangli sessions are viewable/restorable as the 命理 domain.
- A new 命理 question without a `huangli` seed routes to the natal base skill.
- A new explicit `huangli` seed returns `pending-backend`; it must not route to
  `divination` while displayed as 命理.
- Candidate lists, Bazi 流日, and Ziwei 流日 move into the domain flow only after
  the backend expert/context contract exists.

Exit criteria:

- All bridge-eligible marketing handoffs enter a four-domain App state.
- Legacy `huangli` remains available without a fake new-domain route.
- No App surface exposes a seven-product task selector.
- Each absorbed tool passes its existing domain/E2E suite in the new flow.
- Redirect maps cover `scenarios.json`, `chatUrl`, `?scenario=`, and `?service=`.

#### Phase 4 implementation slice 1 — 问卦 casting

##### Canonical vocabulary

The divination slice keeps one name at each layer:

| Concept | Canonical value | Rule |
| --- | --- | --- |
| Practice domain | `divination` | The visible domain is 问卦. Do not call the domain `liuyao`, `qigua`, or `hexagram`. |
| Casting method and record kind | `liuyao` | 六爻 is the method inside the 问卦 domain; it is not a fifth domain. |
| Casting receipt | `LiuyaoCasting` | The engine's non-reproducible receipt returned by casting. It is deterministic input to charting, not chat prose. |
| Casting instant | `casting_instant` | The canonical persisted field name. It is captured locally as ISO-8601 before the engine call. Existing locale keys may translate it as 起卦时间, but no alternate data name is introduced. |
| Casting location | `BirthLocation` | The visible form label is 起卦地点. Its durable consultation projection contains longitude and timezone. Do not reuse birth-place copy for this field. |
| Chart result | `LiuyaoChartResult` | The deterministic装卦 result derived from the receipt plus the casting instant/location. |
| Artifact | `ArtifactKind = 'liuyao'` | One domain-scoped, deterministic chart artifact. Its readiness follows the slice rules below. |
| Follow-up consultation | same `divination` session/context | A follow-up is a free-form turn that carries the existing consultation context; it neither re-casts nor changes person/domain. |

Do not add aliases such as `divination_domain`, `liuyao_domain`, `hexagram`,
or `casting_time` to routes, state, API payloads, locale keys, or documentation.

##### Locale contract

The wrapped flow form reuses the shared form's existing
`services.divination.*` labels rather than creating a duplicate
`flow.casting.*` vocabulary. The Wave 3 audit found all three catalogues at
exact leaf-key parity (`zh-Hans`, `zh-Hant`, and `en`), including 63
`services.divination` leaf keys. This includes the question, casting mode,
casting location, longitude, geolocation, validation, and submit labels used by
`DivinationCastingForm`.

Rules:

- Add a locale key only for new flow-only chrome. Add it to all three
  catalogues in the same change.
- Do not copy an existing service label into a second key or hardcode it in a
  flow component.
- Keep existing keys used by the old tool route and shared chart messages;
  they are compatibility-owned until Phase 6.
- Do not encode domain or field semantics in English key names that contradict
  the canonical identifiers above.

The first absorbed tool is limited to the deterministic casting receipt and its
follow-up context:

- Show the `FlowDivinationForm` wrapper, which reuses `DivinationCastingForm`,
  inside the 问卦 domain only while no casting artifact exists.
- Reuse the existing shared form, casting API, chart API, and result artifact
  panel.
- Capture the non-reproducible casting instant locally before the engine call.
- Emit a real `liuyao` artifact only after both casting and chart succeed.
- Allow the following free-form question to create the session with
  `ConsultationContext` carrying `question`, `casting`, `casting_instant`, and
  timezone/longitude.
- Do not parse chat prose, invent a chart, or claim personalization without the
  casting instant/location receipt.
- Keep the old `/app/divination` route authoritative for records/full history
  until record migration is implemented.

This slice does not yet migrate tool records or absorb hepan, huangli, fengshui,
and naming forms.

#### Parallel Wave 3 — Phase 4 slice 1 and quality lanes

Wave 3 starts after the Phase 2/3 integration gate. The divination form
contract is frozen first; then implementation splits into non-overlapping
lanes.

Canonical divination slice state:

```ts
interface DivinationCastingState {
  form: DivinationCastingFormValue
  loading: boolean
  error: string
  casting: LiuyaoCasting | null
  castingInstant: string
  location: BirthLocation | null
  chart: LiuyaoChartResult | null
}
```

Slice rules:

- `DivinationCastingForm` is reused through `FlowDivinationForm`; do not copy
  its fields.
- A successful cast plus chart emits one deterministic `liuyao` artifact in the
  `divination` domain. A chart failure remains an in-form retryable error and
  must not emit a ready artifact. Retrying charting reuses the same casting
  receipt, casting instant, and location; it does not call casting again.
- The first follow-up question after casting creates the session with durable
  `ConsultationContext`.
- Follow-up turns keep the original casting receipt, casting instant, and
  location; they do not create a second casting.
- No free-form text classifier is introduced.
- Existing `/app/divination` remains authoritative for tool records until the
  new record surface reaches parity.

Slice rollout/rollback remains governed by the build-time
`VITE_FEATURE_APP_FLOW_REDESIGN` flag: enable it by rebuilding/redeploying with
`true`; roll back by rebuilding/redeploying with `false` or unsetting it. The
legacy App and `/app/divination` stay available, so rollback requires no
session, record, URL, or protocol cleanup. The new form must not write a
domain-session protocol or claim record migration before those contracts exist.

| Lane | Owned paths | Scope | Must not do |
| --- | --- | --- | --- |
| A — Casting integration | `DomainFlowView.vue`, `useDomainArtifacts.ts`, view tests | Complete partial state: show form only for 问卦, call casting/chart adapters, emit artifact, attach context to next session | Do not edit shared form, routes, locales |
| B — Casting form UX | new `FlowDivinationForm.vue`, its tests | Wrap/reuse `DivinationCastingForm`; mobile 320px layout, focus, errors, loading, 44px targets | Do not duplicate form fields or API calls |
| C — Casting E2E | `web/e2e/journeys/domain-flow-casting.spec.ts`, isolated helper/fixtures | Cover cast → artifact → question context, validation/error, no seven-task hierarchy | Do not modify app implementation to force a pass |
| D — Docs/i18n | docs, locale catalogues | Add divination flow keys and canonical vocabulary | Do not change runtime behavior |
| E — Integration QA | no direct implementation ownership | Run integration gates after A–D merge; file targeted repairs or contract changes | Do not widen scope |

Merge order:

1. B form component PR and D docs/i18n PR can proceed in parallel.
2. A casting integration lands after B contract is available.
3. C E2E lands after A.
4. E integration gate runs the full frontend check.

#### Parallel Wave 4 — engine catalog extraction and theme reuse

Wave 4 does **not** restart the refactor. The four-domain flow, consultation
document, artifact shell, casting slice, forms, and tests remain the current
integration baseline. Wave 4 standardizes the result-component layer beneath
them so legacy pages, marketing islands, and the new flow reuse one catalog.

##### Wave 4 prerequisites

1. Wave 3 integration gate passes.
2. Current chart panels are captured with component/E2E baselines.
3. A catalog manifest lists every engine method, DTO, current component,
   target component, public props, and consumer.
4. Target directories and TypeScript props are frozen.
5. Existing components remain in place as compatibility facades during
   extraction.
6. Locale ownership remains centralized; extraction lanes request keys rather
   than editing all catalogues.

##### Wave 4A — serial catalog contract PR

Owner: integration/component-contract owner.

Deliverables:

- Create `web/shared/components/engine/catalog.ts`.
- Create `web/shared/components/engine/contract.ts` as the executable props
  registry. A panel derives its public props from
  its concrete component alias in `contract.ts`; it must not redeclare a
  private copy of the engine DTO. `EngineResultPanelPropsFor<Method>` remains
  the pure TypeScript integration assertion, but Vue SFC `defineProps` uses
  the concrete alias because the compiler cannot lower a generic conditional
  type.
- Define a static, typed inventory of result families. This manifest is an
  inventory and type registry, not a dynamic generic renderer.
- Record the required provenance fields from section 5.2.7 for every entry:
  source paths, exact source projection, baseline test, aliases, and known
  gaps.
- Freeze each canonical `targetPath`, optional same-family
  `supportTargetPaths`, and human-auditable `publicProps` in the catalog. The
  catalog contract test verifies unique methods, existing source paths and
  baselines, canonical path shape, standard/bond props, and that pending
  methods remain unimplemented.
- Define common presentational options:

```ts
interface EngineResultPanelProps<Result> {
  result: Result
  density?: 'compact' | 'full'
  focused?: string
}
```

The documented context extensions are `BaziBondPanelContextProps`
(`chartA`, `chartB`, optional display names), because `bazi_bond` returns the
relationship result while its deterministic pillar projection needs the two
already-loaded source charts; and `BazhaiChartPanelContextProps` (optional
`layout`), because the `bazhai_chart` direction grid is annotated with
door/master/stove badges only after `bazhai_layout` returns.
`HuangliDayDetailProps` is a private member of the `huangli_days` family, not
another engine method.

- Record compatibility aliases:

```text
HepanPanel        -> BaziBondPanel
DivinationCastPanel -> LiuyaoCastingPanel
BaziPanel         -> BaziFullChartPanel composition
ZiweiPanel        -> ZiweiChartPanel / ZiweiFullChartPanel
```

- Add visual/baseline tests or Playwright snapshots for current tool pages.
- Publish lane ownership and no-go rules.
- Reject any catalog target whose source path cannot be named. It becomes a
  documented gap until a backend/web contract and product owner approve it.

Exit criteria:

- Catalog contract, directory boundaries, aliases, and props are reviewable.
- `catalog-contract.test.ts`, web typecheck, and focused ESLint pass.
- Every target component has a named current production source and baseline.
- No behavior changes are included in this PR.
- No Wave 4B implementation lane may start until this gate is green.

##### Wave 4B — parallel result-panel extraction

Wave 4B is not limited to four lanes. After the serial catalog contract lands,
up to **ten implementation lanes** can run in parallel. Concurrency is bounded
by disjoint file ownership and DTO boundaries, not by an arbitrary lane count.

| Lane | Owned target files | Required source paths | Extract, without reinterpretation |
| --- | --- | --- | --- |
| B1 — Bazi basic | `engine/bazi/BaziChartPanel.vue`, `engine/bazi/BaziFullChartPanel.vue`, matching tests | `BaziPanel.vue`, `BaziPillars.vue`, `BaziRelations.vue`, `BaziCycles.vue` | existing four-pillar, relation, and cycle projections |
| B2 — Bazi annual | `engine/bazi/BaziLiuNianPanel.vue`, matching tests | `BaziLiunianDetail.vue`, current profile/chart composition | existing annual interaction and omitted-field behavior |
| B3 — Bazi yongshen | `engine/bazi/BaziYongShenPanel.vue`, matching tests | `YongShenPanel.vue`, current naming result composition | existing favorable/unfavorable evidence and unresolved states |
| B4 — Bazi bond | `engine/bazi/BaziBondPanel.vue`, matching tests | `HepanPanel.vue`, `HepanView.vue` | existing day-master, branch, nayin, and cross-star projections |
| B5 — Ziwei basic | `engine/ziwei/ZiweiChartPanel.vue`, matching tests | `ZiweiPanel.vue` palace-grid section | current palace order, center panel, and star wrapping |
| B6 — Ziwei full | `engine/ziwei/ZiweiFullChartPanel.vue`, matching tests | `ZiweiPanel.vue` pattern/sanfang/kongjie sections | existing full-chart sections and ordering |
| B7 — Ziwei derived | `engine/ziwei/ZiweiDaXianPanel.vue`, `ZiweiLiuNianPanel.vue`, matching tests | `ZiweiPanel.vue`, `ChartPreview.vue` current dayun/liunian composition | current year/dayun projections and focus state |
| B8 — Liuyao | `engine/liuyao/LiuyaoCastingPanel.vue`, `LiuyaoChartPanel.vue`, matching tests | `DivinationCastPanel.vue`, complete result blocks in `DivinationView.vue` | existing receipt, 六亲、纳甲、世应、用神、卦辞、动爻、应期 |
| B9 — Huangli | `engine/huangli/HuangliDaysPanel.vue`, `HuangliDayDetail.vue`, matching tests | current `HuangliDaysPanel.vue`, `HuangliDayDetail.vue` | existing candidate grouping, day details, and shichen rendering |
| B10 — Fengshui | `engine/fengshui/BazhaiChartPanel.vue`, `BazhaiLayoutPanel.vue`, `XuankongChartPanel.vue`, matching tests | `MingGuaSection.vue`, `BazhaiLayoutPanel.vue`, `XuankongPanel.vue`, `LiuNianFlySection.vue` | existing natal-Gua, door/main/stove, flying-star, and annual-star projections |

Two support lanes may run alongside them without touching implementation
files:

| Lane | Owned paths | Deliver | Must not do |
| --- | --- | --- | --- |
| B-QA — catalog QA | new isolated QA fixtures/tests only | shared minimal DTO fixtures, visual-parity harness notes, coverage matrix | Do not edit implementation lanes |
| B-Docs — catalog docs/i18n | docs and locale catalogues only | catalog glossary, approved keys, three-locale parity | Do not alter runtime components |

##### Wave 4B concurrency policy

- Maximum concurrent implementation lanes: ten, plus QA/docs support.
- A lane may start only after 4A0 freezes its public props and target path.
- Lanes B1–B10 own exact files; they do not edit sibling files even when the
  domain is related.
- Shared private sections are pre-created by 4A0 or deferred to integration;
  an extraction lane cannot create a shared private module unilaterally.
- Each lane rebases daily and must remain independently revertible.
- If a lane discovers a contract issue, it stops and files a contract-change
  request rather than patching the shared DTO or catalog.
- Integration owner may reduce active concurrency when conflicts, review
  latency, or test runtime outweigh throughput.

Lane rules:

- Extraction is mechanical and source-backed. A lane must begin from the source
  paths named in its catalog entry; it may not design a new panel from memory.
- Preserve current ordering, labels, locale keys, calculations, empty-value
  handling, and visual hierarchy unless a separate behavior-change PR approves
  the difference.
- A missing source implementation is recorded as a catalog gap. The lane does
  not synthesize a replacement.
- New panels are presentational and receive typed `result` props.
- No lane fetches data, imports a router/store, creates a session, or parses
  assistant prose.
- Existing shared components and page views are read-only source material;
  lanes must not edit them.
- No lane introduces ECharts, CSS utilities, or private DTO duplicates.
- Every panel handles absent optional fields without crashing.
- Every panel has 320px, long-locale, keyboard/focus, reduced-motion, and
  empty/partial checks appropriate to its result.
- Locale additions go through B-Docs or a contract-change request.

Tests per lane:

1. Typed result rendering.
2. A provenance assertion or test name identifying the exact source path.
3. Minimal DTO / omitted optional fields, using existing fixtures where they
   exist.
4. Long locale text and 320px wrapping.
5. Focusable cells and accessible names where applicable.
6. No API/session/router imports.
7. Visual parity against the legacy panel where practical.
8. Domain-specific projection checks, e.g. moving line, palace focus, branch
   relation, flying-star palace.

Merge order:

1. Lanes B1–B10 merge independently when their focused tests pass.
2. B-Docs may merge before or after implementation PRs, but locale parity must
   pass before integration.
3. B-QA merges the shared coverage matrix after all intended panel PRs are
   visible.
4. Integration owner resolves only catalog-level conflicts.

##### Wave 4C — parallel theme adoption

Theme adoption starts after the relevant Wave 4B panels pass focused tests. It
can also run wider than four lanes; each consumer boundary is independent.

| Lane | Owned paths | Scope | Depends on | Must not do |
| --- | --- | --- | --- | --- |
| C1 — Legacy natal | birth-profile chart views/tests | profile chart pages use Bazi/Ziwei panels | B1, B2, B5–B7 | Do not alter routes, records, or API behavior |
| C2 — Legacy divination | `DivinationView` and its tests | complete liuyao result uses `LiuyaoChartPanel` | B8 | Do not remove records or old route |
| C3 — Legacy hepan | `HepanView` and its tests | bond result uses `BaziBondPanel` | B4 | Do not weaken second-person consent |
| C4 — Legacy fengshui | `FengshuiView` and its tests | uses Bazhai/Xuankong panels | B10 | Do not alter input contracts |
| C5 — Legacy huangli | `HuangliView` and its tests | uses Huangli panels | B9 | Do not add personalized liuri claims |
| C6 — Flow artifacts | `ArtifactPanel`, artifact types/tests | maps typed payload to catalog panels | relevant B lanes | Do not fabricate artifacts or parse chat text |
| C7 — Marketing islands | `web/src/islands/**`, shared wrappers | public tools use catalog panels | relevant B lanes | Do not fork locale or result logic |
| C8 — Docs/i18n | docs and three locale catalogues | consumer vocabulary and migration docs | catalog contract | Do not change runtime behavior |
| C9 — QA/a11y | new QA specs / visual snapshots | cross-theme parity and a11y matrix | partial C lanes | Do not modify implementation to force passes |

Adoption rules:

- One consumer domain per PR.
- Existing paths remain redirect-compatible.
- Old components become thin re-exports only after consumer tests pass.
- Visual snapshots compare before/after output.
- E2E covers the legacy URL and, where applicable, the new `/app` consultation default
  artifact surface.
- C1–C7 can run concurrently only while their owned paths remain disjoint.
- C9 can begin with partial consumers and expand the matrix as lanes merge.

##### Wave 4B / 4C6 trunk status

As of this plan revision, the serial contract gate and all source-backed Wave 4B
implementation lanes have landed:

- `contract.ts` contains the DTO map, shared panel contract, and Vue-safe
  concrete props aliases. SFC `defineProps` must use the concrete aliases; the
  conditional helper is retained for pure TypeScript assertions.
- All 15 source-backed result panels are implemented, plus the private
  `HuangliDayDetail` member of the `huangli_days` family. `bazi_liuri` and
  `ziwei_liuri` remain explicit catalog gaps.
- `catalog-contract.test.ts` freezes method uniqueness, provenance, baselines,
  target paths, public props, and pending-contract behavior.
- `ArtifactPanel` is the first theme consumer: its typed payloads now map to
  catalog panels rather than the compatibility chart implementations. The
  liuyao artifact renders the complete chart when present and falls back to the
  casting receipt only when no chart exists.
- Integration repairs included Vue-safe prop aliases, engine shared-component
  ESLint coverage, a missing moving-line locale key, form-kit reuse in the
  flow composer, and preservation of the composer's accessible name.
- Legacy App consumers C1–C5 now use catalog panels: `ChartPreview`,
  `DivinationView`, `HepanView`, `FengshuiView`, and `HuangliView`. They retain
  their forms, orchestration, records, errors, consent, and URLs. The Bazi full
  panel exposes a named composition point so an independently loaded annual
  panel remains in the legacy position before 用神.
- Focused adoption tests cover panel selection, context props, loading/empty
  branches, consent/records boundaries, and host-owned year pickers.
- Marketing islands C7 now use catalog panels as well. Public divination
  remains a casting-receipt result; no chart call or fabricated chart was added.
  The Bazi annual panel preserves the legacy annual-pillar summary in addition
  to shensha, fuyin/fanyin, and interaction groups.

Compatibility cleanup is deliberately deferred until route/E2E parity remains
green for all consumers.

##### Wave 4D — compound themes

After single-result adoption, build cross-method compositions:

1. **Full natal reading**
   - `BaziFullChartPanel`
   - `ZiweiFullChartPanel`
   - `BaziLiuNianPanel` / `ZiweiLiuNianPanel`
   - focused palace coordination
2. **Compatibility reading**
   - two basic chart panels
   - `BaziBondPanel`
   - consent and second-person form state
3. **Fengshui reading**
   - `BazhaiChartPanel`
   - `BazhaiLayoutPanel`
   - `XuankongChartPanel`
4. **Date selection**
   - `HuangliDaysPanel`
   - `HuangliDayDetail`
   - future `BaziLiuRiPanel`
   - future `ZiweiLiuRiPanel`
   - decision summary

Date-selection Bazi/Ziwei 流日 remains blocked until web contracts and natal
expert ownership exist.

##### Wave 4E — integration QA gate

Run:

```text
make lint-js
make typecheck
make test-js
make lint-docs
route-contract checks
affected tool-page E2E
/app consultation artifact E2E
a11y E2E
320 / 390 / 768 / 1024 / 1440 manual or visual checks
```

Acceptance criteria:

- Every engine result has one canonical panel family or an approved gap entry.
- Every implemented panel has catalog provenance naming its production source,
  source projection, and parity baseline.
- Every extraction PR is behavior-neutral unless an explicitly linked
  behavior-change PR is included in the review description.
- Legacy pages and new flow render the same result component.
- Marketing islands still work without private chart forks.
- No page owns a deterministic chart projection that belongs in the catalog.
- No dynamic “render any JSON” framework exists.
- No chart library was added.
- Compatibility aliases are either still tested or removed with route/E2E
  coverage.
- Catalog documentation matches implemented components.

##### Wave 4 PR-level execution board

| PR | Name | Depends on | Owner | Primary output | Review gate |
| --- | --- | --- | --- | --- | --- |
| 4A0 | Catalog contract | Wave 3 gate | Integration owner | typed manifest, aliases, directories, props, baselines | no behavior change |
| 4B1 | Bazi basic family | 4A0 | Bazi-basic lane | `BaziChartPanel`, `BaziFullChartPanel` | DTO parity and 320px |
| 4B2 | Bazi annual panel | 4A0 | Bazi-annual lane | `BaziLiuNianPanel` | annual projection and omitted-field handling |
| 4B3 | Bazi yongshen panel | 4A0 | Bazi-yongshen lane | `BaziYongShenPanel` | evidence and unresolved-state parity |
| 4B4 | Bazi bond panel | 4A0 | Bazi-bond lane | `BaziBondPanel` | two-chart and relation parity |
| 4B5 | Ziwei basic panel | 4A0 | Ziwei-basic lane | `ZiweiChartPanel` | palace order, star wrapping, focus |
| 4B6 | Ziwei full panel | 4A0 | Ziwei-full lane | `ZiweiFullChartPanel` | patterns/sanfang/kongjie parity |
| 4B7 | Ziwei derived panels | 4A0 | Ziwei-derived lane | `ZiweiDaXianPanel`, `ZiweiLiuNianPanel` | year/dayun DTO parity |
| 4B8 | Liuyao family | 4A0 | Liuyao lane | casting and full chart panels | receipt immutability and complete chart |
| 4B9 | Huangli family | 4A0 | Huangli lane | day list/detail panels | almanac fields and time sections |
| 4B10 | Fengshui family | 4A0 | Fengshui lane | Bazhai/layout/Xuankong panels | grid/table parity and accessible cells |
| 4B-QA | Coverage matrix | partial 4B PRs | QA owner | shared fixtures, parity/coverage matrix | no implementation edits |
| 4B-Docs | Catalog docs/i18n | 4A0 | Docs owner | glossary and three-locale parity | no runtime edits |
| 4C1 | Legacy natal adoption | 4B1/2/5/6/7 | Legacy natal owner | profile charts use catalog panels | route/E2E unchanged |
| 4C2 | Legacy divination adoption | 4B8 | Legacy divination owner | old tool page uses catalog panel | records and route unchanged |
| 4C3 | Legacy hepan adoption | 4B4 | Legacy hepan owner | old tool page uses bond panel | consent preserved |
| 4C4 | Legacy fengshui adoption | 4B10 | Legacy fengshui owner | old tool page uses Fengshui panels | input contract unchanged |
| 4C5 | Legacy huangli adoption | 4B9 | Legacy huangli owner | old tool page uses Huangli panels | no personalized claims |
| 4C6 | Flow artifact adoption | relevant 4B PRs | Flow owner | typed payload maps to catalog panels | no fake artifacts |
| 4C7 | Marketing adoption | relevant 4B PRs | Marketing owner | islands use catalog panels | SEO/public tools unchanged |
| 4C8 | Consumer docs | 4C lanes as they merge | Docs owner | adoption state and aliases | docs match implementation |
| 4C9 | Cross-theme QA | partial 4C lanes | QA owner | parity, a11y, visual matrix | no forced implementation changes |
| 4D1 | Compound natal theme | 4C1, 4C6 | Flow owner | full reading composition | document remains primary |
| 4D2 | Compatibility theme | 4C3, 4C6 | Flow owner | two-chart + bond composition | consent/context preserved |
| 4D3 | Fengshui theme | 4C4, 4C6 | Flow owner | three-result composition | one mobile scroll owner |
| 4D4 | Date-selection theme | backend contract | Flow owner | almanac + liuri composition | no personalization without natal context |
| 4E0 | Integration gate | all intended PRs | Integration owner | full quality gate and catalog docs | release-ready or explicit gaps |

##### Wave 4 RACI

| Activity | Responsible | Accountable | Consulted | Informed |
| --- | --- | --- | --- | --- |
| Catalog contract | Component-contract owner | Integration owner | result-panel lane owners | all lanes |
| Result extraction | each result lane | integration owner | legacy page owners | QA |
| Legacy adoption | legacy theme owner | integration owner | result-panel lane | QA |
| Flow adoption | flow owner | integration owner | artifact owner | QA |
| Locale additions | docs/i18n owner | integration owner | requesting lane | all lanes |
| Visual parity | QA owner | integration owner | theme owners | lane owners |
| Route/rollback | integration owner | product owner | theme owners | all lanes |

##### Wave 4 anti-drift rules

- A lane cannot rename an engine method or invent a DTO.
- A lane cannot invent a result projection or implement a panel without a
  production source path and parity baseline.
- A lane cannot make a result panel fetch data or mutate a store.
- A theme cannot copy a result projection to customize one label.
- A compatibility facade cannot regain business logic.
- A missing result remains a catalog gap; it is not replaced by sample data.
- An old route is removed only after its shared-panel consumer and redirect
  E2E pass.
- Compound themes cannot combine results that lack explicit input provenance.
- The `ArtifactDescriptor` remains domain-scoped and payload-typed; it does not
  become a generic JSON envelope.

### 6.6 Phase 5 — history, mine, commerce, and hardening

**Goal:** complete the product surface around the consultation flow.

Work items:

- Move global chat history into person/domain history.
- Preserve rename/delete and payment-alert actions.
- Add person, domain, and title filtering.
- Keep full message-content search deferred until the server contract exists.
- Move orders, entitlements, account, full history, and profile management into
  the mine layer.
- Keep partner center role-gated and first-class.
- Inline purchase/unlock at domain entry while preserving payment-return flows.
- Add complete unread/restoration behavior for active sessions.
- Run performance and accessibility hardening:
  - keyboard-only full journey;
  - screen-reader labels and live regions;
  - 200% zoom and reflow;
  - RTL mirror;
  - reduced motion;
  - color contrast;
  - offline/unreachable and permission states;
  - bundle-size and initial-JS budgets;
  - Playwright visual snapshots for shell and artifact states.

Exit criteria:

- The flagged flow can serve the complete logged-in product without falling
  back to the old shell for primary tasks.
- `make check`, affected unit suites, and E2E/a11y suites pass.
- No primary action is hidden or clipped at any supported viewport.

### 6.7 Phase 6 — rollout, redirects, and cleanup

**Goal:** make the domain flow the default and remove dead code.

Rollout:

1. Internal dogfood.
2. Small percentage flag rollout.
3. Metric and error-rate review.
4. Full rollout.
5. Legacy route redirects.
6. Cleanup.

Cleanup items:

- Remove `AppSidebar` and `AppTabbar`.
- Remove old chat-list navigation and obsolete tool page shells after redirects.
- Remove duplicated form/chart projections.
- Remove dead locale keys and CSS.
- Remove temporary feature branches/flags after stability.
- Keep the consultation context resolver until `huangli` ownership migrates
  to `natal` and compatibility seeds are retired.

Final acceptance:

- Default `/app` is the four-domain consultation flow.
- All public marketing URLs still convert correctly.
- No route, payment-return, handoff, or entitlement regression remains.
- Documentation (`ux-architecture.md`, route docs, component docs) matches the
  implementation.

### 6.8 Deferred backend phase — domain-owned consultation contexts

The frontend compatibility resolver remains temporary. The later backend
phase:

- Keeps four domain experts and clarifies domain-owned consultation contexts.
- Keeps sessions bound to `(person, domain/expert)` while simplifying the
  scenario vocabulary and ownership rules.
- Infers intent per turn for tool/artifact selection.
- Moves `huangli` from the `divination` expert to `natal`.
- Extends natal expert capabilities for 黄历 + Bazi 流日 + Ziwei 流日.
- Extends durable consultation context for personalized date selection.
- Migrates legacy sessions by domain.
- Retires or renames the compatibility resolver once old seeds and restore
  paths no longer require it.

Until that phase ships, the frontend must not fake domain-level session
rebinding. Follow-ups continue the visible session; only an explicit new
consultation may create another legacy session.

### 6.9 Risk controls

| Risk | Control |
| --- | --- |
| Hidden text classifier creates unpredictable expert routing | The frontend has no text classifier; routing is domain base plus explicit seed/form only |
| Same visible domain needs multiple legacy sessions | Continue the visible session; no silent split; measure and design domain-owned consultation contexts |
| Mobile viewport regressions | 320px budget tests, keyboard E2E, one-scroll-owner contract |
| Route/payment regressions | Generated route contracts and redirect map before removing old paths |
| Accessibility regressions | APG patterns, focus restore, axe E2E, keyboard-only journeys |
| Bundle growth | Reuse current libraries/components; no ECharts in P1/P2; lazy artifact surfaces |
| Rewrite branch drift | Trunk-based feature-flagged increments with tests at every slice |
| Marketing/App taxonomy divergence | Seven public marketing URLs map through one shared contract, not ad-hoc UI links |
| Result-component fragmentation | Wave 4 catalog maps each method to one canonical result family; aliases are temporary and tested |
| Generic dynamic renderer creep | The catalog is a typed inventory, not a `renderEngineJson` switch or schema-driven UI framework |
| Visual drift between old/new themes | Legacy and flow consumers use the same result panels plus visual parity snapshots |
| Extraction regression | Baseline snapshots and existing tool E2E must pass before old page-local projections are removed |
| Cross-method composition overreach | Compound themes are built only from implemented result families; missing contracts remain explicit gaps |

### 6.10 Delivery impact summary

| Phase | Primary impact | User-visible result |
| --- | --- | --- |
| 0 | Contracts, bridge, fixtures | None |
| 1 | Shell, composer, sheets | Flagged mobile-first shell around legacy routes |
| 2 | Free-form consultation | Person/domain + question works with current chat API |
| 3 | Artifacts | Charts/tools attach progressively to the consultation |
| 3.5 / Wave 4 | Engine result catalog and theme reuse | Legacy pages, marketing islands, and flow consume one typed result-component catalog |
| 4 | Tool absorption | Product/tool pages become domain slot-filling |
| 5 | History/mine/commerce | Complete product surface |
| 6 | Rollout/cleanup | New flow becomes default |
| Deferred backend | Domain sessions | Bridge retirement and correct huangli expert ownership |

## 7. Open items

1. Reader threshold (lines/chars) and whether section detection reuses the
   Markdown headings already rendered.
2. Prompt-side length budget numbers per domain.
3. `LegacyIntentBridge` product thresholds: which explicit seeds/slot forms
   should enter the new flow, and which must remain legacy pending backend.
4. Restore rule when the last session ended more than N days ago (restore
   vs fresh ask).
5. Session active window value (2.2): how recent a session counts as
   "active" for chip selection and restore.
6. Server-side history contract if message-content search or per-profile
   pagination becomes necessary.
7. Person home composition (which chart summary and suggested questions
   show when there is no active session).
8. Date-selection artifact copy and ranking model: how 黄历 suitability and
   Bazi 流日 interactions are combined without overclaiming certainty.
9. Web contract exposure for `bazi_liuri` and `ziwei_liuri`, including DTO
   names, input shape, locale-specific day boundaries, and error codes.
10. Catalog inventory sign-off: whether every current shared chart component is
    preserved, split, renamed, or left as a compatibility facade.
11. Whether `bazi_yongshen` belongs in the engine catalog or a separate counsel
    result catalog, based on its `/counsel/mcp` transport boundary.

Phase 0-6 follow-up work is sequenced in the
[unified optimization plan](./app-flow-optimization-plan.md).
