# Liki Web

`liki-web` is the web and BFF service for [Liki](https://liki.hk). It owns the authenticated product experience, birth profiles, orders, payments, entitlements, Liki Chat sessions, and account governance.

It is not the metaphysics engine and not a generic agent runtime. Deterministic domain computation comes from `liki`; generic AG-UI execution can come from `liki-agents`; production deployment belongs to `liki-deploy`.

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
liki engine  /  liki-agents  /  payment providers
```

| Repository | Responsibility |
| --- | --- |
| `liki-web` | Product BFF, platform identity, profiles, commerce, entitlements, Liki Chat |
| `liki` | deterministic engine, skills, Engine MCP, Counsel MCP source |
| `liki-agents` | generic agent runtime, AG-UI/A2A protocol adapters, execution audit |
| `liki-deploy` | images, Compose, Caddy, release and rollback |

## Quick start

Requirements are documented by:

- `.nvmrc` for Node.js;
- `go.mod` for Go;
- a prepared immutable `liki-web-skill-bundle.tar.gz` under `web/skills`.

```bash
make hooks
make build
make dev
```

The Docker E2E uses `LIKI_ENGINE_IMAGE` and does not build the sibling liki
repository. The non-Docker `make dev` path accepts a pre-extracted engine MCP
binary through `ENGINE_MCP_BIN`; set `ENGINE_RPC_BIN` only when testing the
legacy public JSON-RPC compatibility endpoint.

In development, the login page shows a local verification code when the email service is unavailable. Production does not expose this code.

## Commands

| Command | Purpose |
| --- | --- |
| `make dev` | Build and start the local stack |
| `make test` | Unit, integration, and skill tests (Go/JS/Python) |
| `make gate` | `check` + `test` + candidate image smoke |
| `make build` | Verify the prepared skill artifact, then build static assets and Go service |
| `make clean` | Remove local build artifacts |

## Configuration

`.env.example` is the complete template. Important variables:

| Variable | Purpose |
| --- | --- |
| `JWT_SECRET` | Platform session signing secret |
| `REFERRAL_PAYOUT_SECRET` | Encrypts partner payout accounts; must be at least 32 bytes |
| `LISTEN_ADDR` | Internal HTTP listen address |
| `LIKI_DATA_DIR` | SQLite data directory |
| `RETURN_URL` | Public site base URL used for payment returns |
| `PAYMENTS_WEBHOOK_URL` | Provider webhook URL |
| `DODO_API_KEY`, `DODO_WEBHOOK_KEY`, `DODO_PRODUCT_LIKI_CHAT` | Dodo configuration |
| `XUNHU_APPID`, `XUNHU_APPSECRET` | XunHuPay configuration |
| `LIKI_AGENTS_URL`, `LIKI_AGENTS_INTERNAL_TOKEN` | Mandatory AG-UI runtime integration |
| `TRUSTED_PROXY_HOPS` | Number of trusted reverse-proxy hops before the public client |
| `LIKI_CURRENCY_SOURCE`, `LIKI_BILLING_REGION` | Server-owned billing policy; explicit mode ignores spoofable geo headers |
| `METRICS_TOKEN` | Protects the internal metrics endpoint |
| `FEEDBACK_API_KEY` | Reads and updates feedback records |

Never commit real environment files or secrets.

CI consumes an exact liki release through repository variables:

```text
LIKI_SKILLS_RELEASE_TAG
LIKI_SKILLS_BUNDLE_SHA256
```

The composite action downloads that release asset, verifies SHA256 and the
embedded archive digest, and prepares `web/skills`. It never checks out or
rebuilds the liki source repository.

## Payment

CNY checkout uses XunHuPay. USD and other supported non-CNY currencies use Dodo.

The server owns:

- currency selection;
- amount selection from the catalog;
- provider routing;
- webhook verification;
- reconciliation;
- fulfillment and entitlement.

USD checkout requires `DODO_PRODUCT_LIKI_CHAT`. If the value is absent, USD checkout fails closed.

## Quality gate

```bash
make gate
```

This runs Go lint/vet, frontend lint/typecheck, JS and Python tests, static HTML/i18n checks, and Go race tests. Docker E2E runs separately via `make e2e`; packaging is verified by `make check-packaging`.

## Documentation

| Document | Purpose |
| --- | --- |
| [`ARCHITECTURE.md`](./ARCHITECTURE.md) | Components, boundaries, and request flows |
| [`DOMAIN_MODEL.md`](./DOMAIN_MODEL.md) | Stable entities, relationships, and invariants |
| [`API.md`](./API.md) | Stable HTTP API contract |
| [`FRONTEND.md`](./FRONTEND.md) | Astro marketing pages, Vue App, and frontend state |
| [`LEGACY_PRODUCTS.md`](./LEGACY_PRODUCTS.md) | retired-product compatibility boundary |
| [`docs/database/overview.md`](./docs/database/overview.md) | SQLite bounded-context inventory |
| [`docs/mcp.md`](./docs/mcp.md) | Engine/Counsel MCP compatibility boundary |

## Deployment

Production deployment lives in [`liki-deploy`](../liki-deploy). This repository owns application behavior, not server lifecycle.

## License

[MIT](./LICENSE)
