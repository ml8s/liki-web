# Legacy Products

`liki-web` retains three legacy commercial products for compatibility.

| ID | Product | Status |
| --- | --- | --- |
| `naming` | custom naming | legacy chat/order flow |
| `mingshu` | lifebook report | legacy generated report |
| `hepan` | synastry report | legacy generated report |

## Compatibility rules

- Legacy IDs remain stable.
- Legacy orders remain auditable.
- Legacy reports remain accessible through existing authorization rules.
- Legacy products never implicitly create `liki-chat` entitlement.
- `liki-chat` never implicitly creates legacy report entitlement.
- New functionality is not built around legacy vanilla JS pages.

## Fulfillment

Legacy products use the shared payment runtime and separate fulfillment flows.

After payment, the provider callback is captured, a durable fulfillment task is queued, and the legacy flow marks its own order paid. Duplicate callbacks do not repeat fulfillment.

## Reports

`mingshu` and `hepan` reports use the legacy report renderer and structured report JSON.

New rendering belongs in the Vue App Markdown pipeline.

## Future

Legacy products are technical assets, not the current product narrative. They remain accessible until existing business records expire or are migrated deliberately.
