# Retired Products

`liki-chat` is the only commercial product. The former naming, Ming Shu, and
He Pan products are retired.

| Former ID | Former product | Current role |
| --- | --- | --- |
| `naming` | custom naming | `naming` Liki Chat scenario |
| `mingshu` | lifebook report | `mingshu` Liki Chat scenario |
| `hepan` | synastry report | `hepan` Liki Chat scenario |

## Public pages

`/naming`, `/mingshu`, and `/hepan` are marketing/CTA pages for Liki Chat.
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
capabilities (naming is a current Liki Chat scenario; mingshu/hepan report
templates serve the CTA page demos). These are skill content, not product
backends.
