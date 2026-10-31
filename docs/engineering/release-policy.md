# Release policy

## Environments

This project intentionally has no staging deployment. The only complete
pre-production verification environment is the local assembly stack.

## Release gate

A release candidate may be deployed only after all of the following pass:

1. every source repository is clean and pushed;
2. liki-web runs its full Go, frontend, static, i18n, and unit gates;
3. route and generated-artifact checks pass;
4. liki-deploy runs `make gate`;
5. liki-deploy runs `make test-local`;
6. liki-deploy assembly `@system` journeys cover Caddy, BFF AG-UI, expert,
   engine MCP, and persistence;
7. the release manifest records the exact image SHAs.

## Production deployment

1. back up production SQLite data;
2. deploy the assembly-verified local images by SHA;
3. run standard production smoke;
4. run post-deploy route, database, and image verification;
5. retain the smoke and verifier reports with the release record.

The web repository's normal browser E2E explicitly excludes `@system`. The
canonical live-system journey is owned by liki-deploy's assembly suite.

## Compatibility

Retired product URLs, scenario aliases, old locale paths, and legacy DB schema
fallbacks are not supported. Runtime code reads only the canonical contract and
canonical schema. The retired Consultation `/classic/**` App prefix is the
limited exception: it redirects to canonical `/app/**` paths while bookmark
traffic decays.

## Rollback

Rollback means redeploying the previous assembly-verified image. The database is
not automatically rolled back. Destructive migrations require a fresh backup and
are one-way.
