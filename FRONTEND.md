# Frontend

`liki-web` has three frontend surfaces with different responsibilities.

## Surfaces

| Path | Purpose | Policy |
| --- | --- | --- |
| `web/src` | Astro marketing and locale pages | SEO-first static pages |
| `web/app` | Vue 3 authenticated product SPA | Primary product surface |
| `web/js` | legacy vanilla JS product pages | Frozen compatibility surface |

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
/app/profiles
/app/profiles/new
/app/profiles/:id
/app/synastry
/app/naming
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
| `profiles` | birth profile list, detail, create, update |
| `entitlements` | active skills and purchase state |
| `chat` | sessions, messages, streaming, generation state |
| `commerce` | products, checkout, payment return state |

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

## Legacy

`web/js` supports legacy naming, mingshu, and hepan compatibility pages.

Legacy pages do not receive new product features and do not implicitly interact with unified chat entitlements.
