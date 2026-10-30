# Frontend

`liki-web` has two frontend surfaces with different responsibilities. Both are
framework-rendered, and TypeScript is the source language for every
browser-facing module — the Astro interaction scripts and the Playwright
journeys included.

## Surfaces

| Path | Purpose | Policy |
| --- | --- | --- |
| `web/src` | Astro marketing and locale pages, page behaviour in `web/src/scripts` | SEO-first static pages |
| `web/app` | Vue 3 authenticated product SPA | Primary product surface |

New product functionality belongs in `web/app` and backing Go APIs.

## Vue stack

`web/app` uses:

- Vue 3;
- Vue Router;
- Pinia;
- Element Plus;
- Vite;
- TypeScript;
- Vitest;
- `@ag-ui/client`;
- `markdown-it` + DOMPurify.

Component rule: user-facing list views stay under ~600 lines; a view that
crosses the threshold is split during its next feature change, not as a
standalone refactor.

## Routes

When `VITE_FEATURE_APP_FLOW_REDESIGN=true`, authenticated product routes use two
templates. Consultation is the default four-domain surface; Classic remains the
maintenance and rollback surface:

```text
/app
/app/consultation
/app/me
/app/classic
/app/classic/birth-profiles
/app/classic/birth-profiles/new
/app/classic/birth-profiles/:id
/app/classic/hepan
/app/classic/divination
/app/classic/fengshui
/app/classic/huangli
/app/classic/chat
/app/classic/chat/:id
/app/classic/orders
/app/classic/partners
/app/classic/partners/kol
/app/classic/partners/scout
/app/account
/app/admin
/app/admin/entitlements
/app/admin/partners
/app/admin/payouts
/app/admin/finance
/app/admin/orders
/app/admin/subscriptions
/app/admin/audit
/app/admin/administrators
```

Legacy `/app/<tool>` URLs redirect to Consultation when the domain has reached
parity, or to `/app/classic/<tool>` while compatibility is still required.
A user-selected Classic preference may send `/app` to `/app/classic`.

Router guards call `/api/me`, redirect unauthenticated users to `/app/login`, and preserve redirect targets.

Route components stay lazy. Product and Admin navigation preloads one route on
`pointerenter` or `focusin`; small operators and save-data sessions skip this
speculative work. There is intentionally no idle preload of every route.

### Scenario handoff

Marketing pages hand off with the scenario id — one vocabulary across both
surfaces (`/{locale}/natal/` ↔ `/app?scenario=natal`):

- scenario ids: `natal | liunian | hepan | divination | huangli | fengshui | naming`
  (the seven user-facing scenarios). The internal expert deployment
  (`natal | divination | fengshui | naming`) is never presented to users;
- tool-route entries use the same ids through `/app?service=…`; Consultation
  normalizes this to `domain` plus transient `seed` state;
- `huangli` remains a compatibility entry until backend ownership moves to
  `natal`;
- the app keeps the locale, authenticates the user, resolves the scenario after
  profile data has loaded, and falls unknown values back to the general
  workspace.

## Shared frontend layers

Both surfaces consume the same function layers under `web/shared`:

```text
api/mcp       Streamable MCP transport only
api/engine/*  one wire adapter per engine domain
domain/*      explicit domain inputs and use-case functions
components/*  shared Element Plus form and chart components
stores/*      cross-surface persisted form state
```

The dependency direction is fixed:

```text
island/view/component -> domain function -> engine adapter -> MCP client
```

Views do not construct engine wire arguments. They create a complete domain
input (for example `BirthInput`), which is converted once to the canonical
`ChartInput` of `{standard_time, longitude}`.

## State

Pinia stores are the frontend server-state layer.

| Store | Responsibility |
| --- | --- |
| `auth` | current user and session operations |
| `birth-profiles` | birth profile list, detail, create, update |
| `entitlements` | active skills and purchase state |
| `chat` | sessions, messages, streaming, generation state |
| `sales` | products, checkout, payment return state |

Stores display server-verified state. They do not grant entitlement or decide price.

### Birth location and canonical time

The shared `RegionField` resolves free text through the built-in city index and
the engine `city_coords` fallback. It emits one atomic location value:

```text
{ place, longitude, timezone, source }
```

- Exactly one candidate auto-fills longitude and timezone; multiple candidates
  require an explicit pick.
- Manual entry requires both longitude and IANA timezone.
- Exact chart inputs require the complete location; longitude without timezone
  is rejected rather than guessed.
- Profiles persist `birth_longitude`, `birth_timezone`, and `birth_instant`.

### Chart time normalization

Birth input is complete Gregorian wall time (`YYYY-MM-DD` + `HH:mm`). The
shared time adapter rejects DST gaps/folds and projects the civil time into the
zone's standard-time offset; chart tools receive only
`{standard_time, longitude}`. Date-only tools use a stable midday projection in
the product calendar timezone. There is no approximate/unknown time state:
incomplete information cannot be charted. Lunar dates are not accepted as
browser input; lunar fields in panels are engine projections.

### LiuYao casting context

The divination form owns an explicit context: question, casting mode, and an
atomic casting location. The submit handler captures the ritual instant
before the first Engine MCP request and projects exactly that instant into the
casting timezone's standard_time for `liuyao_chart`; network completion time
never becomes the casting time. It also passes `yong_shen` explicitly.

Browser Geolocation may be requested only from the visible “use current
location” action. Success prefills place, longitude, and the device IANA
timezone, while denial or failure leaves the normal city-resolution and manual
location path. No hidden longitude or timezone default is sent, and chart
requests carry no raw place text.

### Tool-to-Chat consultation context

A paid consultation that starts from a tool result must create its chat session
with `consultation_context_v1`. The context is durable server-side state, not
URL state or prose. It carries the minimum deterministic inputs and, for
divination, the non-reproducible casting receipt and ritual instant. He Pan
requires an explicit third-party consent control before computation and sends
the affirmative domain fact with the second person's atomic birth facts. The BFF
forwards the authorized product context to the runtime.

Public tools use the same-origin session-storage handoff for at most thirty
minutes and never place personal facts in a URL. All seven scenarios carry the
minimum deterministic context; the matching App service view or Chat List
consumes it once, converts it into local form state, and never reconstructs it
from the URL. For birth-based tools, Chat List first selects an existing profile
whose atomic facts match the handoff. If no profile matches, it routes to profile
creation with the visible birth-form handoff rather than silently substituting
the most recently used profile.

## Dashboard states

The dashboard is state-driven.

| State | Primary action |
| --- | --- |
| no profile | create profile and see free chart |
| profile without entitlement | view chart and start paid AI interpretation |
| entitlement without session | start consultation |
| entitlement with session | continue most recent consultation |

Scenario entries remain available below the primary guidance.

## Chat

The Vue app uses the official AG-UI client against:

```text
/api/chat/sessions/{id}/ag-ui        (BFF)
```

The BFF reaches the runtime through the internal network. It owns authentication,
entitlement, authoritative history, profile context, and product persistence;
the runtime remains stateless with respect to product chat history.

The app consumes product-safe events. Progress uses the namespaced custom event
`hk.liki.phase`; its value is the raw tool-name fact and the App renders the
localized label. Unknown custom events are ignored. Backend persistence uses
only the authoritative BFF `RUN_FINISHED` result (`session`, `active_skill`,
`user_message`, and `assistant_message`). `RUN_ERROR` is surfaced as a
generation error, not stored as a formal assistant message.

Message history is cursor-paginated: `GET /api/chat/sessions/{id}/messages?before=&limit=`
returns `{messages, next_cursor}` (default page 200, cap 500; cursors are
`<created_at>|<id>` keyset strings). The session view offers “load earlier”, and
the agent context is capped to the latest 50 messages. Session lists are capped
at the recent 200. Keyset indexes must cover the full sort key — a query-plan
contract test guards this.

## Markdown

Assistant Markdown uses:

```text
markdown-it + DOMPurify + AppMarkdown.vue
```

Rules:

- raw HTML is disabled;
- output is sanitized;
- external links get `rel="noopener noreferrer nofollow"`;
- dangerous protocols are removed.

## Script and CSP policy

Marketing and app pages use external same-origin scripts. The production edge
CSP does not permit executable inline scripts. JSON-LD and declarative JSON
data blocks are allowed because the browser does not execute them as JavaScript.

Static styles are external. `style-src 'unsafe-inline'` remains an explicit
exception because Element Plus writes runtime style attributes for positioning.
Replacing or configuring that component library is the path to removing this
exception; do not add new hand-written inline styles.

`web/scripts/check-inline-scripts.js` enforces this in two ways:

1. source inventory rejects new executable inline scripts;
2. `scripts/build-web.sh` scans the built HTML and fails if executable inline
   scripts reappear after Astro/Vite processing.

Static-page browser behavior belongs in `web/src/scripts`, and Vue application
modules belong under `web/app/src`.

## Frontend module policy

Browser code has exactly two places to live:

| Group | Modules | Policy |
| --- | --- | --- |
| static-page interaction | `web/src/scripts/*.ts` | Astro-processed, external, same-origin modules; this is the only supported way to add static-page behavior |
| product surface | `web/app/src/**`, including the Vue admin at route `/app/admin` | capability-scoped views with a typed API client and E2E coverage |

The former `web/js` runtime — vanilla admin console, client-side report demo
renderer, vendored Markdown sanitizer — is retired. Do not reintroduce it.

## Checks

`make check` covers every layer:

| Layer | Lint | Types |
| --- | --- | --- |
| static pages (`web/src`) | `cd web && npm run lint` | `astro check`, `.astro` templates included |
| app (`web/app`) | `cd web/app && npm run lint` | `vue-tsc --noEmit` |
| journeys (`web/e2e`) | the same eslint config as `web/` | `tsc --noEmit -p e2e/tsconfig.json` |

eslint covers `.ts` and `.js` sources under `web/`; `.astro` templates are
covered by `astro check`. There is no `eslint-plugin-astro` because its 2.x/3.x
line requires a newer Node than the pinned `.nvmrc`, and 1.x carries an
unfixable high advisory that would fail the CI dependency audit.

`make check` also enforces cross-layer contracts: marketing↔App i18n key usage,
service-CTA parity, chart-message sync, BFF-only chat transport, and error-code
copy coverage — every Go code must carry App copy in all three locales, while
client/runtime-only codes live in the explicit frontend inventory
(`error-codes-frontend.json`). Go's generated inventory remains
`docs/error-codes.md`; retired paths cannot return
(`check-retired-chat-transports.mjs`).

## Payments

Commerce UI handles:

1. loading product and purchase state;
2. starting checkout;
3. redirecting to the provider;
4. polling entitlement after return;
5. entering chat after confirmed fulfillment.

## Mobile

The authenticated app is a first-class mobile experience.

Requirements:

- forms work on iOS Safari and Android Chrome;
- primary buttons have sufficient touch targets;
- chat output wraps safely;
- payment return restores product state;
- safe-area insets are respected.

The admin console is also responsive, but it uses a different density contract:
tables remain the desktop view and are replaced by record cards below 720px.
Horizontal admin navigation keeps the active route visible.

## Design tokens

`web/css/tokens.css` owns the canonical token layers:

| Layer | Prefix | Use |
| --- | --- | --- |
| Primitive | `--brand-*` | source palette and value definitions |
| Semantic | `--color-*`, `--liki-*` | marketing and product component CSS |
| Component bridge | `--el-*` | Element Plus theme variables |

Component code should consume a semantic token when one exists. Raw colors are
allowed only inside the primitive token source; raw spacing values should be
limited to genuine layout measurements rather than replacing spacing tokens.
`check-css-vars.js` rejects CSS custom properties that are consumed by source
but not defined in the shared source layers.

The full route inventory, page archetypes, state matrix, URL-state rules, and
component rules live in
[`docs/frontend/ux-architecture.md`](docs/frontend/ux-architecture.md).

### Page typography

Use the existing page-header component for normal product pages. It owns the
default responsive scale (`26–32px` title and `15px` subtitle). These are the
only intentional exceptions:

| Context | Title scale | Reason |
| --- | ---: | --- |
| Back-navigation flow pages | `24px` | Keep profile creation/detail subordinate to navigation |
| Chat session header | `20px` | Preserve a long, truncatable session title and maximize workspace height |
| Admin pages | `24px` | Dense operator surface, owned by `AdminPageHeader` |
| Long-form legal/static pages | `20px` | Reading-first document layout |

Do not add a new local heading scale for a whole product page. If a new context
does not fit, extend the shared header contract first.

### Layout contracts

Use the shared z-index tokens (`--z-header`, `--z-surface`, and
`--z-skip-link`) instead of numeric values. The standard responsive
breakpoints are `480 / 640 / 768 / 1024`. `720` is the Admin-only dense-surface
exception, and `1023.98` is the max-width edge paired with the `1024`
sidebar breakpoint.

Admin list state is split into draft filters and applied filters. The URL only
reflects applied filters, defaults and page 1 are omitted, and changing an
input does not query until the operator submits that filter block. Entitlement
email is intentionally excluded from the URL because it is customer PII.

## Internationalization

The Vue app uses `vue-i18n` and Element Plus locale data for:

```text
zh-Hans
zh-Hant
en
```

Locale resolution uses, in order:

```text
?locale=
liki_locale cookie
account locale
navigator.language
zh-Hant
```

Locale messages are loaded on demand and only the active locale is bundled into
the initial route chunk. The app waits for the initial message pack before
mounting so users do not see raw keys or a flash of the wrong language.

Locale is not part of the authenticated route path. Users can change it in
Account settings; the public language switcher writes the same cookie before
entering the app.

Admin console copy is deliberately Simplified Chinese for now: it is an
internal operator surface, while customer-facing Vue surfaces remain localized.
Revisit this only when there is an explicit requirement to localize operator
workflows.

User-visible errors follow the same boundary. HTTP, engine, and stream clients
preserve stable machine codes such as `generation_failed` and
`engine_unavailable`; `toUserMessage` maps those codes through the active locale.
Server-provided diagnostic messages are not treated as UI translations.

## Partner journeys

`/app/partners` is the single role-discovery and state-dispatch surface. KOL and
Scout dashboard routes are intentionally role-specific; applications, review
states, role conflicts, frozen states, and CTAs are first represented in the
partner center. Partner dashboards are routes inside `/app`, not a separate console.

Partner referral links use `/r/{partner_code}`. This entry point writes the
30-day attribution cookie and redirects to marketing. Private App URLs are not
referral targets. Referral redirects support only allowlisted same-origin
marketing paths or the App home. The detailed URL, sharing, and result-state
policy lives in
[`docs/frontend/ux-architecture.md`](docs/frontend/ux-architecture.md).

## Capability pages

The public scenario pages — `/natal`, `/liunian`, `/hepan`, `/divination`,
`/fengshui`, `/huangli`, `/naming` — are the SEO landing pages for the seven
user-facing scenarios. They share one component
(`web/src/components/ScenarioLanding.astro`), explain a single scenario with a
free tool, and hand off to `/app?scenario=…` (tool-route entries use
`service=…`); the home page service cards link to them. The raw App routes stay
private — these pages exist so each scenario has a public, indexable entry
point that ends in the app.

## Retired product pages

`/mingshu` and `/bazi` are retired and return 404; the life reading is the
deliverable of the natal scenario. `/naming` and `/hepan` are live scenario
pages.

The retired naming activation and chat URLs redirect to the Liki Chat scenario;
the retired commerce backends, client-side report demo renderer, and vendored
Markdown/DOMPurify scripts have been removed. The remaining public scripts are
small, static-page interactions; the frozen admin console is tracked separately
for Vue replacement.
