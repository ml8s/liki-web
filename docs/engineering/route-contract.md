# Route contract

`web/shared/config/scenarios.json` is the single source of truth for business
scenarios and their legacy expert labels. `web/routes/marketing.contract.json`
is the single source of truth for public marketing routes.
`web/routes/app.contract.json` is the single source of truth for App SPA paths.

Runtime layers must consume or validate these contracts:

| Layer | Contract use |
| --- | --- |
| Astro pages | existence and scenario ID check |
| Vue App | scenario config, legacy expert label routing, service params |
| Go static server | canonical pages, bare redirects, retired 404s |
| Caddy | bare redirects |
| Playwright | generated page, redirect, and retired matrix |
| post-deploy verifier | production route matrix |

Generated artifacts are not editable:

```text
web/routes/generated
internal/http/routes.generated.json
liki-deploy/caddy generated redirect block
```

To change a route, update the contract, run:

```bash
cd web && npm run routes:generate
```

then update tests and documentation. Adding retired aliases or compatibility
redirects is prohibited.
