# Frontend

`liki-web` has three frontend surfaces with different responsibilities.

## Surfaces

| Path | Purpose | Policy |
| --- | --- | --- |
| `web/src` | Astro marketing and locale pages | SEO-first static pages |
| `web/app` | Vue 3 authenticated product SPA | Primary product surface |
| `web/js` | retired static-page interaction runtime | No new product behavior |

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

## Routes

Authenticated routes live under `/app`:

```text
/app
/app/birth-profiles
/app/birth-profiles/new
/app/birth-profiles/:id
/app/synastry
/app/divination
/app/fengshui
/app/huangli
/app/chat
/app/chat/:id
/app/orders
/app/partners
/app/partners/kol
/app/partners/scout
/app/account
/app/admin
/app/admin/entitlements
/app/admin/partners
/app/admin/payouts
/app/admin/finance
/app/admin/audit
/app/admin/administrators
```

Router guards call `/api/me`, redirect unauthenticated users to `/login`, and preserve redirect targets.

### Scenario handoff

Marketing pages use `/app?scenario=naming|mingshu|hepan&locale=…`. The app keeps
the locale, authenticates the user, resolves the scenario after profile data has
loaded, and falls unknown scenarios back to the general workspace.

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
/api/chat/sessions/{id}/ag-ui
```

It does not connect directly to `liki-agents`.

The app consumes product-safe events. Backend persistence only uses the authoritative terminal result. `RUN_ERROR` is surfaced as a generation error, not stored as a formal assistant message.

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

`web/scripts/check-inline-scripts.cjs` enforces this in two ways:

1. source inventory rejects new executable inline scripts;
2. `scripts/build-web.sh` scans the built HTML and fails if executable inline
   scripts reappear after Astro/Vite processing.

Static-page browser behavior belongs in `web/src/scripts`, and Vue application
modules belong under `web/app/src`.

## Frontend module policy

`web/js` is not a product surface. Its remaining browser modules have two
distinct roles:

| Group | Modules | Policy |
| --- | --- | --- |
| static-page interaction | `web/src/scripts/*.ts` | Astro-processed, external, same-origin modules; this is the only supported way to add static-page behavior |
| admin console | Vue route `/app/admin` | capability-scoped views with a typed API client and E2E coverage |

Do not add product behavior to `web/js`. The vanilla admin console, client-side
report demo renderer, and vendored Markdown sanitizer dependencies are retired.

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

User-visible errors follow the same boundary. HTTP, engine, and stream clients
preserve stable machine codes such as `generation_failed` and
`engine_unavailable`; `toUserMessage` maps those codes through the active locale.
Server-provided diagnostic messages are not treated as UI translations.

## Partner journeys

`/app/partners` is the single role-discovery and state-dispatch surface. KOL and
Scout dashboard routes are intentionally role-specific; applications, review
states, role conflicts, frozen states, and CTAs are first represented in the
partner center. There is no separate vanilla-JS partner console.

## Retired product pages

The public `/naming`, `/mingshu`, and `/hepan` pages are Liki Chat CTA pages.
They explain a scenario and send the visitor to `/app?scenario=…`, just as the
home page sends visitors to the app.

The retired naming activation and chat URLs redirect to the Liki Chat scenario;
the retired commerce backends, client-side report demo renderer, and vendored
Markdown/DOMPurify scripts have been removed. The remaining public scripts are
small, static-page interactions; the frozen admin console is tracked separately
for Vue replacement.
