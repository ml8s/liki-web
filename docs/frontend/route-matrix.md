# Liki App Route Matrix

Scope: authenticated Vue 3 SPA under `/app`.

| Route | Page | Auth | Primary APIs | Status |
|---|---|---|---|---|
| `/app/login` | Email OTP login | public | `POST /api/auth/email/code`, `POST /api/auth/email/verify` | P0 |
| `/app` | Workbench dashboard | required | `GET /api/me` | P0 |
| `/app/reports` | Reports index | required | `GET /api/reports` | P1 |
| `/app/reports/:id` | Report detail | required + owner | `GET /api/reports/:id` | P1 |
| `/app/orders` | Chat orders | required | `GET /api/chat/orders` | P0 |
| `/app/account` | Account, identities, sessions | required | `GET /api/me`, identity/session APIs | P0 |
| `/app/chat` | Liki Chat product, skills, sessions | required; entitlement controls start/purchase | `GET /api/chat/products`, `GET /api/me/entitlements`, Chat APIs | P0 |
| `/app/chat/:sessionId` | Chat conversation | required + owner + entitlement | Chat APIs | P0 |

Every protected API endpoint performs owner and entitlement checks. Client route guards improve UX only.
