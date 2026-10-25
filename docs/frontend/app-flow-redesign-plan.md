# App flow redesign plan

Status: proposal (discussion outcome; no implementation yet).

This document records the converged information architecture, the concrete
layout details, and the phased migration plan for the logged-in product
surface (`/app`). It does not modify `ux-architecture.md`; that document is
updated as part of P1 implementation.

## 1. Why

The current navigation splits the product by page type (dashboard /
profiles / chat). Users think in one task: **ask something about someone**
(or ask something with no person attached). Domain analysis confirms:

- `BirthProfile` is the aggregate root; natal, liunian, hepan, naming, and
  fengshui all operate on it (fengshui requires a profile today).
- Divination and huangli need no profile. In the data model this is already
  an **optional parameter** (`chat_sessions.profile_id DEFAULT ''`), not a
  separate category.
- Chat is a capability, not a domain concept; it should not own a
  top-level destination.

Conclusion: the product has **one verb (ask) with two parameters
(scenario, optional person)**. The UI collapses to a single flow;
navigation destinations disappear.

## 2. Target information architecture

```text
ask(scenario, person?)            // one flow, person is optional
context bar     person chip (switcher) · restore pill · mine
content area    current person's chart / tool content
scenario chips  all 7 scenarios in one row (no grouping)
composer        typing + send = create session (no confirm page)

mine (layer, not tab)  orders · entitlements · account · full history
                       · profile management · partner center
```

Key decisions already made during discussion:

| Decision | Outcome |
| --- | --- |
| Person switcher vs profile pages | person is a global context, not a destination |
| Tools (divination/huangli) | same flow; chips are not grouped; person chip only dims |
| Scenario chip order | fixed product order (muscle memory), no per-user reordering |
| Bottom tabbar / sidebar | retired entirely |
| Open app | restore last context (person + running session), zero taps |
| Purchase | inline at scenario entry, never a separate page |
| Chart focus ("边聊边亮") | client-side dictionary matching only; no model tool call |
| Scenario entry modes | type-first (divination, liunian) vs form-first (hepan, fengshui, huangli, naming); the content area hosts the form, the composer handles follow-up |
| Person-session binding | the visible session pins the person chip; switching the chip switches the view, never rebinds a session |
| Partner center | first-class entry for partner-role accounts, not buried inside mine |
| Chart computation | lazy (first sheet open) and cached per person; never part of boot |
| Open behavior | adaptive default: active session -> restore the conversation; no active session -> person home (chart summary + suggested questions + ask) |
| Parameter placement | person·scenario pill lives on the composer (at the point of action), synced with the context bar |
| Chips semantics | channel selectors — tapping a chip switches the view to that (person, scenario) channel immediately; the pill displays the current channel at the action point |
| Chips placement | anchored directly above the composer (the channel the composer speaks into), not mid-screen |
| Form-first entry | focused near-full-screen form step; results flow back into the content area |

### 2.1 Person-session binding (mandatory rule)

Sessions are bound to `profile_id` at creation and never rebind. The UI
rule follows from the data model:

- The **visible session pins the person chip**: if the screen shows a
  session for 张三, the chip must display 张三. A mismatched pair (李四
  chip + 张三 conversation) is the most severe defect possible in this
  product — it reads one person's fate as another's.
- Switching the chip switches the **view** to that person's latest active
  session (or their blank ask state). It never rewrites the previous
  session's `profile_id`.
- Form-first scenarios that need a second person (hepan) keep the
  counterpart in the content-area form; the session binds to the context
  person (A) only.

### 2.2 Session lifecycle

- Restore on open = latest active session; typing there **continues** it.
- Selecting a chip for a (person, scenario) pair that already has an
  active session offers continue vs new; otherwise typing starts a new
  session directly.
- An explicit "new consultation" action is always reachable from the
  conversation overflow menu.
- The active window (how recent a session counts as active) is an open
  tuning item, not a structural one.

### 2.3 Multi-session selection

When a (person, scenario) pair has several historical sessions, the flow
shows the most recent active one; all others are reachable from history.
History (in mine) must support search by person, scenario, and message
content — replacing the global list the current chat page provides.

### 2.4 Parameter semantics: person is payload, scenario is routing

Seven scenarios map to four experts through `scenarios.json`; each expert
is a separately deployed runtime. In direct transport the frontend itself
resolves the run URL from the session's bound expert (`/rt/{expert}/ag-ui`).

| Parameter | Architectural role | Where it lives |
| --- | --- | --- |
| person | context payload — the client fetches the profile and attaches `AgentProfile` per run (C3) | `session.profile_id`, optional |
| scenario | routing key — selects which of the four agent deployments answers | `session.skill` / `session.expert`, fixed at creation |

Scenario-to-expert mapping: natal, liunian, hepan -> natal;
divination, huangli -> divination; fengshui -> fengshui; naming -> naming.

Implications:

- Switching scenario may switch experts, and expert binding is immutable
  per session — so a scenario change is necessarily a **new session**. The
  channel model (chips as channels) is structural, not stylistic.
- The composer pill has two independently tappable halves: the **person
  half** switches the view to that person's session (never rebinds,
  per 2.1); the **scenario half** starts or switches to a session routed
  to the mapped expert.
- Draft send sequence (frontend-only): scenario -> expert lookup ->
  `createChatSession(person, skill, expert)` -> connect
  `/rt/{expert}/ag-ui` -> first message.

### 2.5 ParameterComposer

The composer stops being an internal part of the session view and becomes
a flow-level component that exists **before any session does**.

| Mode | Pill behavior | Send semantics |
| --- | --- | --- |
| Draft (no session) | editable parameters | send = create session (person, skill, expert) + send the first message, one step |
| Session-bound | reflects the session's binding (2.1) | send = continue; tapping the person half switches the view, not the binding |

Message rendering, AG-UI streaming, and entitlement cards are reused
as-is; the chat store gains only a small draft intent state
(`{ personId, scenarioId }`). No protocol or backend changes.

## 3. Layout details

### 3.1 Mobile — main state (person + running session)

```text
+-----------------------------------+
| (person chip) (restore pill) (mine)| <- context bar (sticky)
+-----------------------------------+
| +-------------------------------+ |
| | chart / tool / hexagram       | | <- content area
| +-------------------------------+ |
+-----------------------------------+
|                                   |
|  (messages)                       | <- conversation grows in place
|                                   |
+-----------------------------------+
| 命理|流年|合盘|起名|风水|问卦|择日 | <- scenario chips (h-scroll)
+-----------------------------------+
| [张三·流年 v] [input.........] [up]| <- ParameterComposer
+-----------------------------------+
```

Zone contract:

| Zone | Responsibility | Behavior |
| --- | --- | --- |
| Context bar | current person, restore entry, mine | tap chip -> switcher sheet; restore pill -> jump to last session |
| Content area | the "thing" of current context | chart, tool result, or hexagram; collapses to one-line summary during active reading |
| Scenario chips | select the channel the composer speaks into | tapping switches the view immediately (active session of that pair, or the scenario's draft/form state); anchored directly above the composer; thumb-reachable |
| ParameterComposer | the single action point, displaying its channel | person·scenario pill on the composer (2.5); the **person half** is the thumb-reachable person switch at the action point (the top chip remains for identity and browsing), the **scenario half** switches to the channel routed to the mapped expert |

Open behavior (adaptive default): with an active session the app restores
the conversation; without one it opens the person home — chart summary,
recent interpretations, suggested questions, and the same composer. First
time users land on a visual, guided surface instead of an empty input.

### 3.2 Mobile — chart sheet detents (not "half screen")

At 390 pt width a readable ziwei grid needs roughly 520-560 px of height
(palace cells grow taller than wide once stars render). A fixed 50% sheet
cannot fit it. The sheet is full-width and content-sized, with detents:

| Detent | Height | Entry | Use |
| --- | --- | --- | --- |
| Collapsed | one-line pill | drag down | default; chat at full speed |
| Medium | ~70% | tap pill, or chart-focus auto-open | whole grid visible; focused palace pulses |
| Full | ~95% | push up from medium | palace-by-palace deep reading |

Chart focus auto-opens **medium only**, never full. Streaming continues
behind the sheet; dismissing restores scroll position.

### 3.3 Mobile — no-profile state (new user)

```text
| (建立第一个档案 chip)            (mine) |
| +-----------------------------------+ |
| | 先建一份命盘档案                    | |
| | [ 开始建档 ]                       | |
| +-----------------------------------+ |
| 问卦 | 择日 | 命理 | ... (rest dimmed) |
+---------------------------------------+
```

Profile-free scenarios stay fully usable; profile-bound chips are dimmed
with a hint, and selecting one starts the profile-creation flow.

### 3.4 Mobile — divination state (person not passed)

Person chip dims (e.g. 50% opacity + one-line hint "本场景不涉及个人") but
stays interactive: after a hexagram the context survives, tapping 流年
keeps 张三 selected. Content area swaps to the tool/hexagram surface. The
result card must not display the person's name — "parameter not passed"
must not read as "parameter belongs to this person".

### 3.5 Mobile — profile switcher sheet

```text
| 最近使用 ------------------            |
| [张三] [李四] [王五] [+新建]           |
| 管理全部档案 ->                        |
```

Full profile management (list/new/edit/delete) lives behind this sheet and
in mine; it is not a navigation destination.

### 3.6 Desktop (>= 1024px)

```text
+----------+--------------------+--------------------+
| (person) |                    |                    |
|----------|  chart / tool      |  conversation      |
| 命理      |  content           |                    |
| 流年      |                    |                    |
| 合盘 ...  |                    |                    |
|----------|                    | [ input..... ] [up]|
| 我的      |                    |                    |
+----------+--------------------+--------------------+
   rail         content pane        conversation pane
```

The rail is not navigation; it is the same flow controls (person, chips,
mine) stacked vertically. All 7 chips fit without "more". Chart and
conversation are permanently side-by-side; no collapse behavior on desktop.

### 3.7 Long answers — three reading layers

| Layer | Surface | Entry |
| --- | --- | --- |
| Stream | conversation column | default |
| Chart sheet | reference overlay | pill / focus event (3.2) |
| Full-text reader | full-screen document view | "全文阅读" on messages over ~8-10 lines / 500 chars |

Reader details: no composer; adjustable font size; section jump list when
the answer has section headings; close returns to the exact conversation
scroll position. Generation-side prompt conventions (conclusion first,
sectioned output, per-turn length budget) are a P3 prerequisite recorded in
section 6.

Streaming behavior: if the user scrolls up during generation, auto-follow
pauses and a "new content" pill appears; tap resumes follow.

### 3.8 Form-first scenarios (focused step)

Form-first scenarios (hepan, fengshui, huangli, naming) do not squeeze
their forms into the content area. Selecting the chip opens a focused
near-full-screen form step (clearly titled, with a visible way back), and
submitting returns to the flow with results in the content area and the
composer ready for follow-up. The step belongs to the flow; it is not a
separate destination.

## 4. Chart focus (client-side dictionary, P3)

- No model involvement, no runtime involvement (direct transport keeps the
  runtime a dumb pipe per contract C3).
- Dictionary is **derived from the loaded chart DTO** (palace names, dayun
  pillars, liunian terms), so person-specific terms match naturally;
  locale-keyed.
- Matching runs on accumulated message text (incremental cursor, ~200 ms
  debounce), sets a `chartFocus` state consumed by `ZiweiPanel` (`focused`
  class) and the sheet pill.
- Passive mention -> a small pill plus a clickable token on the palace
  term in the rendered answer; tapping opens the sheet at the focused
  palace. The sheet never auto-opens from a passive mention — the user,
  not the match, commits the view change. Latest hit wins with a pulse
  animation.
- Known limit: non-standard references ("你的婚姻位") are not matched;
  prompt wording conventions may nudge standard terms, but this is soft
  guidance, never a contract.

## 5. Migration plan

Scope: **frontend-only**. In direct transport the runtime is stateless and
the client already carries profile context (contract C3: `chat.ts` fetches
the profile and attaches an `AgentProfile` before each run).
`chat_sessions.profile_id` is already an optional parameter, and session
create/list/rename/delete endpoints are reused as-is. No backend or
runtime changes are in scope; backend work reappears only if server-side
per-profile history pagination becomes necessary at scale.

### P1 — new shell (routes preserved)

- Build `AppShell` (context bar, switcher sheet, mine sheet), retire
  `AppSidebar` / `AppTabbar`.
- Build the flow view skeleton: content area, `ParameterComposer` with
  draft intent state and the person·scenario pill (2.4/2.5), scenario
  chips anchored above it. Chips **navigate to the existing pages** (chat
  list, tool pages) — no route changes, marketing deep links untouched.
- Adaptive open on boot: restore the latest active session; otherwise
  show the person home (3.1).
- P1 ships behind a **feature flag**. The hybrid state (new shell + chips
  jumping to old full pages) must not become the default production
  experience; the user-visible minimum is the completed P2 flow.
- Rewrite shell-level tests: `app.spec`, `app-a11y.spec`,
  `focus-ring.spec`, `chrome-consistency.spec` (~530 lines), plus
  `route-contract` additions; add basic flow journeys (~200-300 lines, net
  new).

### Gate — P1 to P2 (instrumentation, one release cycle)

- Scenario usage share (divination/huangli vs profile-bound scenarios) to
  validate chip ordering and the shared-flow model.
- Person-switch frequency and restore-vs-new-ask ratio, to tune the
  session active window (2.2).
- If practitioner usage dominates expectations, promote multi-context
  support from P3 into P2 **before** absorbing tools.

### P2 — absorb tools and chat into the flow

- Tool pages become content-area states; cross-page handoff
  (`peekConsultationHandoff`) becomes in-flow state transfer. Reuse the
  existing form components as content-area states — selectors then survive
  incidentally; tests never constrain the design.
- Chat list dissolves: sessions belong to person context; global history
  (rename/delete/payment alerts) moves to mine/history, with search per
  2.3.
- The `viewportBound` / `visualViewport` keyboard handling migrates from
  `ChatSessionView` into the flow view and is re-tested against the chart
  sheet (keyboard + sheet must not fight for viewport height).
- Route removals require a redirect map for `scenarios.json` appPaths,
  `chatUrl`, and `?scenario=` / `?service=` params; extend
  `route-contract.spec`.
- Affected suites (adaptation, not rewrite): `tool-handoff`, 7 tool specs,
  `records`, `chat-list-context`, `scenario`, `purchase-unlock`
  (about 1000 lines).

### P3 — enhancements

- Chart focus dictionary + `ZiweiPanel.focused` + sheet pill.
- Full-text reader layer + streaming follow-pause.
- Prompt conventions: conclusion-first, sectioned answers, per-turn length
  budget.
- Optional: multi-context support for practitioner workflows.

### E2E impact summary

| Phase | Lines touched | Lines untouched |
| --- | --- | --- |
| P1 | ~530 shell-level + new flow journeys | ~5100 (incl. 2044-line engine contract) |
| P2 | ~1000 tool/chat absorption | remaining suites stable |
| P3 | grows with features | - |

## 6. Open items

1. Reader threshold (lines/chars) and whether section detection reuses the
   Markdown headings already rendered.
2. Prompt-side length budget numbers per expert/scenario.
3. Whether divination/huangli should accept optional profile enrichment in
   the future (e.g. huangli avoiding year-branch clashes) — the optional
   parameter model already supports it without structural change.
4. Restore rule when the last session ended more than N days ago (restore
   vs fresh ask).
5. Session active window value (2.2): how recent a session counts as
   "active" for chip selection and restore.
6. History search scope for the mine/history surface (fields, matching,
   server-side pagination trigger).
7. Person home composition (which chart summary and suggested questions
   show when there is no active session).
