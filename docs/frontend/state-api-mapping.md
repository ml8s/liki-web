# State / API Mapping

## Current implementation: Pinia stores
- `auth` → `GET /api/me`, email-code login, logout, sessions
- `entitlements` → `GET /api/me/entitlements`
- `chat` → Chat APIs below
- `commerce` product/checkout state is currently page-local; orders use `src/api/orders.ts`.
- Future report pages may adopt TanStack Query when their backend APIs land.

## Commerce APIs
- `GET /api/chat/products`
- `POST /api/chat/products/{id}/checkout`
- `GET /api/chat/orders`

## Chat APIs
- `GET /api/chat/sessions`
- `POST /api/chat/sessions`
- `GET /api/chat/sessions/{id}`
- `GET /api/chat/sessions/{id}/messages`
- `POST /api/chat/sessions/{id}/messages` (SSE)

## Rules
1. Components call Pinia stores or API clients; browser SSE/stream parsing belongs in `src/api/sse.ts`.
2. Mutations invalidate related query keys.
3. Backend remains authoritative for auth, ownership, entitlement, and skill/tool access.
4. Do not persist auth tokens in Pinia or localStorage; use HttpOnly cookies.
