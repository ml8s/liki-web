# State / API Mapping

## Server state: TanStack Query
- `['me']` → `GET /api/me`
- `['me','entitlements']` → `GET /api/me/entitlements`
- `['orders']` / `['orders', id]`
- `['reports']` / `['reports', id]`
- `['chat','sessions']`
- `['chat','messages', sessionId]`

## Client state: Pinia
- `auth`: current user and auth lifecycle
- `chat`: active session, drafts, streaming buffer
- `ui`: layout, theme, toasts

## Rules
1. Components call composables; no direct `fetch`.
2. Mutations invalidate related query keys.
3. Backend remains authoritative for auth, ownership, entitlement, and skill/tool access.
4. Do not persist auth tokens in Pinia or localStorage; use HttpOnly cookies.
