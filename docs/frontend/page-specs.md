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
- Goal: list user-owned orders.
- States: loading, empty, list, error.
- P1: retry payment, invoice links.

## `/app/account`
- Goal: identity, auth identities, active sessions, logout.
- States: loading, loaded, error.
- P1: revoke individual sessions, remove identity, account deletion.

## `/app/chat`
- Goal: list chat sessions.
- States: loading, empty, list, no entitlement, error.

## `/app/chat/:sessionId`
- Goal: streaming conversation within one entitlement.
- States: loading, empty, streaming, completed, failed, unauthorized, no entitlement.
