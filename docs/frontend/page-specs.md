# Liki App Page Specs v0

## `/app/login`
- Goal: passwordless email login.
- States: request form, code form, sending, verifying, error, success.
- Success: redirect to `redirect` query or `/app`.
- Security: never expose code; session cookie is HttpOnly.

## `/app`
- Goal: authenticated workbench.
- Data: `/api/me`.
- States: loading, authenticated, unauthorized, API error.
- P1 additions: recent reports, recent sessions, entitlement summary.

## `/app/reports`
- Goal: list user-owned reports.
- States: loading, empty, list, error.
- P1: generation forms and deterministic Engine chart reports.

## `/app/reports/:id`
- Goal: render one user-owned report.
- Data: `GET /api/reports/:id`.
- States: loading, not found, forbidden, deterministic chart, premium report, generation in progress, failed.

## `/app/orders`
- Goal: list unified Chat orders.
- Data: `GET /api/chat/orders`.
- States: loading, empty, list, pending, paid, canceled, error.

## `/app/account`
- Goal: identity, auth identities, active sessions, logout.
- States: loading, loaded, error.
- P1: revoke individual sessions, remove identity, account deletion.

## `/app/chat`
- Goal: purchase Liki Chat, choose an internal skill, and list chat sessions.
- Data: `GET /api/chat/products`, `GET /api/me/entitlements`, `GET /api/chat/sessions`.
- Payment return: recognize `order_id`, `status`, `payment_intent_id`, `status_token`; poll entitlement briefly after `status=paid`.
- States: loading, empty sessions, product purchase, purchased, four internal skills, no entitlement, payment verifying, payment confirmed, payment unpaid, error.

## `/app/chat/:sessionId`
- Goal: conversation within one internal skill.
- Data: `GET /api/chat/sessions/{id}`, messages API, SSE generation API.
- States: loading, empty, thinking, streaming, completed, failed, stopped, unauthorized, expired entitlement, not found.
- Behavior: restore a direct link after refresh; append both persisted messages after the SSE done event; aipay stop; scroll to the latest message.
