# Retired Products

`liki-chat` is the only commercial product. The former naming, Ming Shu, and
He Pan products are retired.

| Former ID | Former product | Current role |
| --- | --- | --- |
| `naming` | custom naming | `naming` Liki Chat scenario and historical-order compatibility |
| `mingshu` | lifebook report | `mingshu` Liki Chat scenario and historical-order compatibility |
| `hepan` | synastry report | `hepan` Liki Chat scenario and historical-order compatibility |

## Public pages

`/naming`, `/mingshu`, and `/hepan` are marketing/CTA pages for Liki Chat.
Like the home page, their primary action opens the app with a scenario handoff.
They do not present a separate price, checkout, activation form, or product SKU.

## Compatibility rules

- Retired IDs remain stable for historical records and fulfillment reconciliation.
- Retired orders remain auditable.
- Previously paid reports and naming-chat APIs remain readable through their old bounded authorization rules.
- Email-only naming login and email/order-ID selection are disabled and return `410 product_retired`.
- Retired identifiers cannot create a new order, checkout, activation code, or entitlement.
- Retired products never implicitly create a Liki Chat entitlement.
- A Liki Chat entitlement never implicitly creates a retired-product entitlement.
- New functionality is built in the Vue app and Liki Chat APIs, not the legacy vanilla-JS surfaces.

## Fulfillment compatibility

Existing fulfillment flows remain available to drain previously created payment
intents and replay webhooks. Duplicate callbacks do not repeat fulfillment.

## Reports

Historical `mingshu` and `hepan` reports use the old renderer and structured
report JSON. New assistant output belongs to Liki Chat and the Vue App Markdown
pipeline.
