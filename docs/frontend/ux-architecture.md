# Frontend UX architecture

This document is the working contract for page planning, page states, and the
shared UI system. `FRONTEND.md` records the stack and product behavior; this
document records how pages are classified and how they should behave.

## Principles

1. **One page has one primary task.** Secondary actions may support it, but
   cannot compete with it.
2. **Domain rendering belongs to the page.** Shared shells own structure,
   state placement, and spacing; they do not own metaphysics business rules.
3. **Use Element Plus first.** Add a local component only when the same domain
   structure repeats or Element Plus cannot express the interaction.
4. **State is part of the design.** Loading, error, empty, and permission states
   are not afterthoughts.
5. **Customer and operator surfaces are different densities.** Product and
   marketing are reading/interaction first; Admin is dense operator UI.
6. **Tests validate current user-visible behavior.** They do not enforce source
   layout or private formatting contracts.

## Surfaces

| Surface | Path | Density | Primary goal |
| --- | --- | --- | --- |
| Marketing | Astro `/[locale]` | reading/landing | explain the product and enter the app |
| Product | Vue `/app` | comfortable product | complete the user's metaphysics task |
| Admin | Vue `/app/admin` | dense operator | review and operate safely |

## Page archetypes

Every route belongs to one archetype. A new page must declare its archetype
before implementation.

| Archetype | Primary task | Structure | Examples |
| --- | --- | --- | --- |
| Marketing landing | convert visitor to app | hero, proof, capability grid, primary CTA | home |
| Scenario SEO | qualify a scenario and hand off to chat | scenario explanation, examples, FAQ, CTA | naming, mingshu, hepan |
| Developer tool | make integration/installation easy | install commands, skill contract, links | skills |
| Legal / reading | make policy readable | title, scoped sections | terms, privacy, disclaimer, about |
| Product dashboard | continue the user's next task | state-driven hero, services, recent activity | `/app` |
| Profile management | manage birth profiles | list/detail/new/edit, confirmation for delete | birth profiles |
| Form + result | input data and receive a domain result | header, error, input, result, consultation CTA | synastry, divination, fengshui, huangli |
| Conversation | conduct a consultation | context, stream, composer, status | chat |
| Commerce | show ownership and billing | subscription, one-time orders, history | orders |
| Partner workspace | show partner performance and earnings | role state, stats, lists, settlement | partner center/KOL/Scout |
| Account & security | manage identity and sessions | profile, locale, sessions, danger zone | account |
| Operator list | filter, inspect, and act | filter form, table/cards, pagination, dialogs | admin pages |

## State matrix

Every page must account for these states. A state may be intentionally absent
only when the domain cannot produce it.

| State | Required presentation |
| --- | --- |
| Default | primary task is visible |
| Loading | `el-skeleton` or `v-loading`; list/container has `aria-busy` |
| Ready | primary task and data are visible |
| Empty | `el-empty` or equivalent; one clear next action |
| Page error | `el-alert type="error"`; `el-alert` already has `role="alert"` |
| Field error | message beside the field; associated through Element Plus validation |
| Unreachable | recovery action such as retry or sign in again |
| Permission denied | explain the missing capability and provide next action |
| Partial success | preserve user input and explain what failed |
| Mobile fallback | card/list equivalent with the same critical fields as desktop |

Forms that create or edit durable data must track dirty state. A router guard asks
the user to confirm in-app navigation, and the browser unload guard protects
refresh/close.

Domain-specific states are listed in the route inventory below.

## Route inventory

### Marketing

| Route | Archetype | Primary action | Notes |
| --- | --- | --- | --- |
| `/{locale}/` | marketing landing | start free chart | direct app CTA is intentional |
| `/{locale}/naming` | scenario SEO | open naming scenario | CTA sends `/app?scenario=naming` |
| `/{locale}/mingshu` | scenario SEO | open mingshu scenario | CTA sends `/app?scenario=mingshu` |
| `/{locale}/hepan` | scenario SEO | open hepan scenario | CTA sends `/app?scenario=hepan` |
| `/{locale}/skills` | developer tool | install skill | developer surface, not C-end navigation |
| `/{locale}/partners` | partner landing | learn/apply | shares public footer |
| `/{locale}/about` | legal / reading | learn about Liki | no primary conversion |
| `/{locale}/contact` | legal / reading | contact | contains external links |
| `/{locale}/privacy` | legal / reading | read policy | noindex is not applied; legal is public |
| `/{locale}/terms` | legal / reading | read policy | legal tone |
| `/{locale}/disclaimer` | legal / reading | read policy | avoids professional-advice misunderstanding |

### Product

| Route | Archetype | Primary action | Required domain states |
| --- | --- | --- | --- |
| `/app/login` | auth form | request code, verify code | code sent, resend cooldown, invalid code, unreachable |
| `/app` | product dashboard | continue next task | no profile, profile without entitlement, entitlement without session, existing session, scenario intent |
| `/app/birth-profiles` | profile management | open/create profile | loading, empty, list ready, load more, delete confirmation |
| `/app/birth-profiles/new` | profile management | save profile | draft restored, validation failed, save failed |
| `/app/birth-profiles/:id` | profile management | inspect/update profile | loading, error, ready, update failed, free chart, paid interpretation |
| `/app/synastry` | form + result | select profiles and interpret bond | profiles missing, validation, computing, error, result, locked |
| `/app/divination` | form + result | cast/read divination | empty question, validation, casting, error, result |
| `/app/fengshui` | form + result | select profile and inspect house | no profile, loading, error, result |
| `/app/huangli` | form + result | select date range and inspect days | invalid range, range too long, loading, error, result |
| `/app/chat` | conversation entry | select profile/skill or open session | sessions loading, empty, entitlement locked, payment return |
| `/app/chat/:id` | conversation | send and read consultation | messages loading, send/stream, stopped, failed, profile context |
| `/app/orders` | commerce | inspect orders/subscription | loading, empty, paid, pending, subscription active/cancelled |
| `/app/partners` | partner workspace | choose/enter role | unapplied, pending, approved, rejected, frozen, retired |
| `/app/partners/kol` | partner workspace | inspect KOL earnings | loading, error, active, commission and payout states |
| `/app/partners/scout` | partner workspace | inspect Scout channel | loading, error, active, recruits and payout states |
| `/app/account` | account & security | manage identity and sessions | loading, empty sessions, revoke, revoke all, erase |

### Admin

All Admin routes use the operator-list archetype unless noted.

| Route | Primary action | Capability gate |
| --- | --- | --- |
| `/app/admin` | inspect enabled capabilities | any admin capability |
| `/app/admin/entitlements` | search, compensate, expire, revoke | `entitlement.read` / `entitlement.manage` |
| `/app/admin/partners` | review applications and partners | `partner.read`, approve/freeze capabilities |
| `/app/admin/payouts` | review and act on payouts | `payout.read`, action capabilities |
| `/app/admin/finance` | inspect finance and resolve tasks | `commission.read` / `commission.void` |
| `/app/admin/audit` | inspect partner actions | `audit.read` |
| `/app/admin/administrators` | manage admin accounts | `admin.user.manage` |

Admin query and pagination state remains in the URL. Defaults and page 1 are
omitted. Sensitive search values such as customer email are intentionally not
written to the URL.

## Navigation model

### Product

Desktop sidebar and mobile tabbar expose the same core destinations:

```text
Workspace
Profiles
Chat
Orders / Account
Partners
```

Service routes are entered from Dashboard and cards, not from global navigation.
This keeps the global model simple while service pages remain deep-linkable.

### Admin

Admin navigation is capability-scoped and independent from customer navigation.
It intentionally uses Simplified Chinese because it is an internal operator
surface.

## Design tokens

`web/css/tokens.css` is the canonical source.

| Layer | Prefix | Rule |
| --- | --- | --- |
| Primitive | `--brand-*` | value source only; components consume semantic aliases |
| Cross-surface semantic | `--color-*` | marketing and shared UI |
| Product semantic | `--liki-*` | Vue product/Admin components |
| Component bridge | `--el-*` | Element Plus variables |

Typography uses explicit semantic sizes:

| Token | Size | Use |
| --- | ---: | --- |
| `--font-size-micro` | 11px | badges and very dense metadata |
| `--font-size-caption` | 12px | timestamps, helper text, dense labels |
| `--font-size-label` | 13px | labels, captions, secondary body |
| `--font-size-body` | 14px | default product body |
| `--font-size-body-lg` | 15px | emphasized body and subtitles |
| `--font-size-title-xs` | 18px | card title |
| `--font-size-title-sm` | 16px | compact section title |
| `--font-size-title-md` | 20px | section title / chat title |
| `--font-size-title-lg` | 24px | admin/product page title |
| `--font-size-display-sm` | 27px | mobile product page display |
| `--font-size-display` | 32px | desktop product page display |

Radii use:

```text
--radius-sm       4px
--radius-small    6px
--radius-md       8px
--radius-control 10px
--radius-lg      12px
--radius-card    16px
--radius-full   9999px
```

Breakpoints are `480 / 640 / 768 / 1024`. Admin may use `720` for its dense
operator breakpoint.

## URL state rule

Put state in the URL only when the user needs one of these abilities:

1. return to the same result with back/forward;
2. share the state;
3. recover a list or filter after refresh;
4. preserve a scenario or payment return.

Do not put transient typing, open dialogs, or one-time generated domain results
in the URL merely because they are state.

| Example | In URL |
| --- | --- |
| Admin filters and page | yes |
| Marketing locale | yes, through path/query handoff |
| Scenario entry | yes |
| Payment return | yes, then cleaned after handling |
| Birth form while typing | no |
| Chat composer draft | no |
| Service result after calculation | no unless sharing becomes a product requirement |

## Component rules

Use Element Plus before building a control.

Shared components are allowed when they own real structure or prevent domain
drift:

| Component | Owns |
| --- | --- |
| `AppPageHeader` | product title/subtitle/action contract |
| `AdminPageHeader` | dense operator title contract |
| `AdminRecordCard` | mobile representation of admin table records |
| `AdminStatusTag` | admin status label and semantic color |
| `PartnerStatGrid` | partner metric structure |
| `PartnerCardHeader` | partner card title/action layout |
| `PartnerSettlement` | shared KOL/Scout settlement behavior |
| `PayoutDialog` | payout method/account form |
| `ProductPurchaseCard` | purchase options and billing mode |

Do not create a universal page component. Page-specific rendering stays in the
page.

## Copy rules

1. Buttons use the current action: “Save profile”, “Delete”, “Approve”,
   “Reject”, “Mark paid”.
2. Destructive confirm buttons never use generic “Submit”.
3. Empty states contain one next action.
4. Errors explain what failed and, where possible, the next step.
5. Customer surfaces remain localized in `zh-Hans`, `zh-Hant`, and `en`.
6. Admin remains Simplified Chinese until operator localization is required.

## Validation

Automated validation checks current behavior:

```bash
make gate
cd web/app && npm run test && npx eslint src/ && npx vue-tsc --noEmit
cd web && npm run build
```

Playwright journeys cover current user behavior, including static pages, login,
profiles, services, chat, commerce, Admin, accessibility, and mobile smoke.

Manual validation is required for:

1. keyboard-only signup through profile creation;
2. screen-reader feedback for validation and streaming;
3. mobile Chat with the keyboard open;
4. Admin dialogs and table/card parity;
5. payment return after refresh or browser back;
6. Partner role states after approval, freeze, and rejection.
