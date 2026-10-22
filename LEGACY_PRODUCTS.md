# Retired Products

`liki-chat` is the only commercial product. The former naming, Ming Shu, and
He Pan products are retired.

| Former ID | Former product | Current role |
| --- | --- | --- |
| `naming` | custom naming | `naming` scenario page + Liki Chat scenario |
| `mingshu` | lifebook report | retired; `/mingshu` 301s to `/natal` (life reading is the natal scenario's deliverable) |
| `hepan` | synastry report | `hepan` scenario page + Liki Chat scenario |

## Public pages

`/naming` and `/hepan` are live scenario pages for Liki Chat; `/mingshu` is
retired and 301s to `/natal`.
Like the home page, their primary action opens the app with a scenario handoff
(`/app?scenario=…&locale=…`). They do not present a separate price, checkout,
activation form, or product SKU.

## Removed backend

The retired products shipped before public launch and hold no historical data.
Their commerce backends — orders, checkout, activation codes, email-only login,
report readers, and fulfillment compatibility — have been fully removed from
`liki-web`. Requests to the retired `/api/naming`, `/api/mingshu`, and
`/api/hepan` routes return `404`.

`web/skills/liki` retains the naming / mingshu report templates as skill
capabilities (naming is a current scenario page; the mingshu template no longer
backs a public page). These are skill content, not product backends.

## Admin console

The standalone `admin.html` and its vanilla scripts were replaced by the
capability-scoped Vue admin at `/app/admin`. Requests to `/admin.html` redirect
there. Authorization remains server-side capability checks; the SPA navigation
is only a rendering convenience.
