# Manual deep system check

`scripts/smoke.mjs` is a **manual system diagnostic**, not production
monitoring. It exercises browser, auth, intent and payment surfaces that the
HTTP deploy smoke cannot cover.

It is deliberately not scheduled and not part of deploy. The automated layers
are:

- `liki-deploy/scripts/deploy_smoke.py`: deploy gate, HTTP/system contract;
- `web/e2e`: repo-local browser tests (and system browser tests via `BASE_URL`);
- `/api/health` + `/api/ready`: monitoring signals.

## Run

```bash
cd web
npm run system:check
```

Useful flags:

```bash
node scripts/smoke.mjs --mode=synthetic --no-browser --no-dns
node scripts/smoke.mjs --mode=smoke --json
node scripts/smoke.mjs --intents=all
```

## Configuration

Environment-only; nothing is hardcoded.

```bash
export SMOKE_BASE_URL=https://liki.hk
export SMOKE_EMAIL=<test-account@example.com>          # identity assertion
export SMOKE_SESSION_COOKIE='liki_session=...'   # secret; rotate regularly
export SMOKE_DOMAIN=liki.hk
export SMOKE_SENDING_DOMAIN=notify.liki.hk
export SMOKE_REQUIRE_ENTITLEMENT=1
```

- `SMOKE_SESSION_COOKIE` enables authenticated checks. Without it, they are
  skipped.
- The script never requests a login code and never reads the mailbox. This
  avoids touching the shared admin mailbox.

## What it checks

- liveness `/api/health`, readiness `/api/ready`, version;
- security headers;
- three locales and seven marketing pages, raw i18n keys;
- robots/sitemap;
- engine MCP and counsel MCP tool calls;
- authenticated identity, entitlement, intent identity;
- payment product currency/options (read-only);
- context session create/delete;
- 404 localization, CSP console (browser);
- DNS/SPF/DMARC and TLS expiry.

## Boundary

Do not schedule this as the primary monitor. Monitoring should stay shallow:
health/ready/TLS/DNS. Functional problems belong in tests.

The following checks are candidates to move into `liki-deploy` system tests
before this file is retired:

- authenticated identity / entitlement;
- intent identity;
- context session + real chat send;
- payment product/checkout sandbox.
