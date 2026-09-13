# Liki App Route Matrix

Scope: authenticated Vue 3 SPA under `/app`.

| Route | Page | Auth | Primary APIs | Status |
|---|---|---|---|---|
| `/app/login` | Email OTP login | public | `POST /api/auth/email/code`, `POST /api/auth/email/verify` | P0 |
| `/app` | Workbench dashboard | required | `GET /api/me` | P0 |
| `/app/reports` | Reports index | required | `GET /api/reports` | P1 |
| `/app/reports/:id` | Report detail | required + owner | `GET /api/reports/:id` | P1 |
| `/app/orders` | Orders index | required | `GET /api/me/orders` | P1 |
| `/app/account` | Account, identities, sessions | required | `GET /api/me`, identity/session APIs | P0 |
| `/app/chat` | Chat sessions | required + chat entitlement | Chat APIs | P1 |
| `/app/chat/:sessionId` | Chat conversation | required + owner | Chat APIs / SSE | P1 |

Every protected API endpoint performs owner and entitlement checks. Client route guards improve UX only.
