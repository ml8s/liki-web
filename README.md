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
- `../liki` for local skill assets.

```bash
make hooks
make build
make dev
```

In development, the login page shows a local verification code when the email service is unavailable. Production does not expose this code.

## Commands

| Command | Purpose |
| --- | --- |
| `make dev` | Build and start the local stack |
| `make build` | Build skills, static assets, and Go service |
| `make check` | Static checks and project validation |
| `make test` | Full tests, build, and packaging |
| `make gate` | `check` + `test`; pre-release gate |
| `make clean` | Remove local build artifacts |

## Configuration

`.env.example` is the complete template. Important variables:

| Variable | Purpose |
| --- | --- |
| `JWT_SECRET` | Platform session signing secret |
| `LIKI_DATA_DIR` | SQLite data directory |
| `RETURN_URL` | Public site base URL used for payment returns |
| `PAYMENTS_WEBHOOK_URL` | Provider webhook URL |
| `DODO_API_KEY`, `DODO_WEBHOOK_KEY`, `DODO_PRODUCT_LIKI_CHAT` | Dodo configuration |
| `XUNHU_APPID`, `XUNHU_APPSECRET` | XunHuPay configuration |
| `LIKI_LLM_BASE_URL`, `LIKI_LLM_API_KEY`, `LIKI_LLM_MODEL` | In-process LLM runner |
| `TRUSTED_PROXY_HOPS` | Number of trusted reverse-proxy hops before the public client |
| `LIKI_AGENTS_MODE` | `local` or `agent` |
| `LIKI_AGENTS_URL`, `LIKI_AGENTS_INTERNAL_TOKEN` | Agent integration |
| `METRICS_TOKEN` | Protects the internal metrics endpoint |
| `FEEDBACK_API_KEY` | Reads and updates feedback records |

Never commit real environment files or secrets.

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

This runs Go lint/vet, frontend lint/typecheck, JS and Python tests, static HTML/i18n checks, Go race tests, E2E tests, and packaging checks.

## Documentation

| Document | Purpose |
| --- | --- |
| [`ARCHITECTURE.md`](./ARCHITECTURE.md) | Components, boundaries, and request flows |
| [`DOMAIN_MODEL.md`](./DOMAIN_MODEL.md) | Stable entities, relationships, and invariants |
| [`API.md`](./API.md) | Stable HTTP API contract |
| [`FRONTEND.md`](./FRONTEND.md) | Astro, Vue App, legacy JS, and frontend state |
| [`LEGACY_PRODUCTS.md`](./LEGACY_PRODUCTS.md) | compatibility boundary for legacy products |
| [`docs/database/overview.md`](./docs/database/overview.md) | SQLite bounded-context inventory |
| [`docs/mcp.md`](./docs/mcp.md) | Engine/Counsel MCP compatibility boundary |

## Deployment

Production deployment lives in [`liki-deploy`](../liki-deploy). This repository owns application behavior, not server lifecycle.

## License

[MIT](./LICENSE)
