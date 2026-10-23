# Liki Web

`liki-web` is the web and BFF service for [Liki](https://liki.hk). It owns the authenticated product experience, birth profiles, orders, payments, entitlements, Liki Chat sessions, and account governance.

It is not the metaphysics engine and not a generic agent runtime. Deterministic domain computation comes from `liki`; generic AG-UI execution comes from `liki-agent-runtime`; production deployment belongs to `liki-deploy`.

## Architecture

```text
Browser
  ↓
Astro marketing pages  /  Vue 3 App
  ↓
liki-web HTTP/BFF service
  ↓
Bounded SQLite contexts
  ↓
liki engine  /  liki-agent-runtime  /  payment providers
```

| Repository | Responsibility |
| --- | --- |
| `liki-web` | Product BFF, platform identity, profiles, commerce, entitlements, Liki Chat |
| `liki` | deterministic engine, skills, Engine MCP, Counsel MCP source |
| `liki-agent-runtime` | generic agent runtime, AG-UI/A2A protocol adapters, execution audit |
| `liki-deploy` | local images, Compose, Caddy, deploy and rollback |

Liki Chat has two transports: the BFF relay (`/api/chat/sessions/<id>/ag-ui`)
and, behind the build-time `VITE_CHAT_TRANSPORT=direct` flag, a direct route
(`/rt/<profile>/ag-ui`) where Caddy performs ForwardAuth introspection and the
runtime connection is mutual TLS. Direct runs persist history through the
signed run-completed callback. See `liki-deploy`'s
`docs/DIRECT_CONNECT.md`; the relay remains the rollback lever until P5b.

## Quick start

Requirements are documented by:

- `.nvmrc` for Node.js;
- `.python-version` for Python;
- `go.mod` for Go;
- `deps/golangci-lint.version` for golangci-lint;
- a prepared immutable `liki-web-skill-bundle.tar.gz` under `web/skills`.

`make check` fails when your local runtimes differ from those declarations.

```bash
make hooks
make build
make dev
```

Docker E2E requires an explicit engine image reference and does not build the
sibling liki repository:

```bash
LIKI_ENGINE_IMAGE=liki-local/liki-engine:<sha> make e2e
```

`liki-web` deliberately keeps no default engine matrix; the production
combination is owned by `liki-deploy`. The non-Docker `make dev` path accepts
a pre-extracted engine MCP binary through `ENGINE_MCP_BIN`.

In development, the login page shows a local verification code when the email service is unavailable. Production does not expose this code.

## Commands

| Command | Purpose |
| --- | --- |
| `make dev` | Build and start the local stack |
| `make test` | Unit, integration, and skill tests (Go/JS/Python) |
| `make gate` | `check` + `test` |
| `make build` | Verify the prepared skill artifact, then build static assets and Go service |
| `make clean` | Remove local build artifacts |

## Configuration

`.env.example` is the complete template. Important variables:

| Variable | Purpose |
| --- | --- |
| `JWT_SECRET` | Platform session signing secret |
| `LIKI_REFERRAL_PAYOUT_ENCRYPTION_KEY` | Encrypts partner payout accounts; must be at least 32 bytes |
| `LIKI_HTTP_LISTEN_ADDRESS` | Internal HTTP listen address |
| `LIKI_DATA_DIR` | SQLite data directory |
| `LIKI_PUBLIC_BASE_URL` | Public site base URL used for payment returns |
| `LIKI_PAYMENT_WEBHOOK_URL` | Provider webhook URL |
| `DODO_API_KEY`, `DODO_WEBHOOK_KEY`, `DODO_PRODUCT_LIKI_CHAT_ONE_TIME`, `DODO_PRODUCT_LIKI_CHAT_SUBSCRIPTION` | Dodo configuration |
| `XUNHU_APPID`, `XUNHU_APPSECRET` | XunHuPay configuration |
| `LIKI_AGENTS_{NATAL,DIVINATION,FENGSHUI,NAMING}_URL`, `LIKI_AGENTS_INTERNAL_TOKEN` | AG-UI runtime per expert deployment (`LIKI_AGENTS_URL` = dev fallback) |
| `LIKI_AGENTS_OPS_PORT` | Expert ops surface port (`/readyz` / `/version` / `/metrics`) used by readiness probes |
| `LIKI_RUN_CALLBACK_SECRET` | Run-completed callback secret (Standard Webhooks); required once the direct transport is deployed — direct history is written by this callback |
| `TRUSTED_PROXY_HOPS` | Number of trusted reverse-proxy hops before the public client |
| `LIKI_BILLING_CURRENCY_SOURCE`, `LIKI_BILLING_REGION` | Server-owned billing policy; explicit mode ignores spoofable geo headers |
| `LIKI_METRICS_TOKEN` | Protects the internal metrics endpoint |
| `LIKI_FEEDBACK_API_TOKEN` | Reads and updates feedback records |

Never commit real environment files or secrets.

CI runs source gates only; it does not build images. Docker E2E runs locally
via `make e2e`, and the cross-repository flow runs via `liki-deploy`.

## Payment

CNY checkout uses XunHuPay. USD and other supported non-CNY currencies use Dodo.

The server owns:

- currency selection;
- amount selection from the catalog;
- provider routing;
- webhook verification;
- reconciliation;
- fulfillment and entitlement.

USD one-time checkout requires `DODO_PRODUCT_LIKI_CHAT_ONE_TIME`. Its Dodo price must match
`LIKI_CHAT_ONE_TIME_PRICE_USD_MINOR_UNITS`; the application grants `LIKI_CHAT_ONE_TIME_ACCESS_DAYS`.

USD monthly subscription requires `DODO_PRODUCT_LIKI_CHAT_SUBSCRIPTION`. The Dodo recurring product owns the charged
price; `LIKI_CHAT_SUBSCRIPTION_PRICE_USD_MINOR_UNITS` is the catalog/display price and must match it. Subscription
lifecycle webhooks update local state and entitlement. CNY subscription is intentionally unavailable: the catalog holds no
CNY subscription price, so subscription options are never listed for CNY buyers and CNY subscription checkout is rejected.

## Quality gate

```bash
make gate
```

This runs Go lint/vet, frontend lint/typecheck, JS and Python tests, static frontend/i18n checks, and Go race tests. Docker E2E runs separately via `make e2e`; packaging is verified by `make check-packaging`.

Images are built locally; CI never builds them. The image tag is the commit
sha, and `liki-deploy` assembles, tests, and deploys that exact set. See
[Release runbook](https://github.com/ml8s/liki-deploy/blob/main/docs/RELEASE_RUNBOOK.md).

## Documentation

| Document | Purpose |
| --- | --- |
| [`ARCHITECTURE.md`](./ARCHITECTURE.md) | Components, boundaries, and request flows |
| [`DOMAIN_MODEL.md`](./DOMAIN_MODEL.md) | Stable entities, relationships, and invariants |
| [`API.md`](./API.md) | Stable HTTP API |
| [`FRONTEND.md`](./FRONTEND.md) | Astro marketing pages, Vue App, and frontend state |
| [`BRAND.md`](./BRAND.md) | Brand definition: positioning, language, and principles |
| [`docs/database/overview.md`](./docs/database/overview.md) | SQLite bounded-context inventory |
| [`docs/database/partner.md`](./docs/database/partner.md) | Partner database schema |
| [`docs/database/platform-admin.md`](./docs/database/platform-admin.md) | Platform admin auth & audit schema |
| [`docs/integrations/mcp.md`](./docs/integrations/mcp.md) | Engine/Counsel MCP compatibility boundary |
| [`docs/payments/aipay.md`](./docs/payments/aipay.md) | AI Pay boundary and configuration |

## Deployment

Production deployment lives in [`liki-deploy`](../liki-deploy). This repository owns application behavior, not server lifecycle.

## License

[MIT](./LICENSE)
