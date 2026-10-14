# Contributing

## Branches

`main` is the only long-lived branch. Do not create `dev`, `master`, or long-lived
release branches. Maintainers commit focused changes directly to `main`; external
contributors may use a short-lived branch in a fork and target `main`.

Release tags are created only from tested `main` commits.

## Commits

Use Conventional Commits:

```text
feat(chat): ...
fix(payment): ...
refactor(profile): ...
docs(domain): ...
chore(deploy): ...
```

Keep unrelated changes in separate commits.

## Quality gate

Before pushing:

```bash
make check
make test
```

For release review:

```bash
make gate
```

Changed areas must at least pass:

```bash
make lint-go
make lint-js
make typecheck
make test-go
make test-js
make check-html
make test-python
```

## Code conventions

### Error wrapping

Wrap errors at operation boundaries so logs can locate the failure:
`fmt.Errorf("domain: operation: %w", err)` (e.g. `payment: enqueue
notification: ...`). Keep sentinel errors comparable with `errors.Is`.
Handlers may return bare `err` only when the call chain is one level deep.

- Go modules and binaries use the `liki-web` service identity.
- New authenticated UI belongs in `web/app`.
- New server state belongs in Pinia stores.
- New raw HTTP calls belong in `web/app/src/api`.
- New Markdown rendering must use `renderSafeMarkdown`.
- Legacy vanilla JS pages are frozen compatibility surfaces.

## Documentation

Documentation must describe stable behavior and invariants. Do not add implementation plans, review reports, handoff notes, or temporary checklists to `docs/`.

Important stable documents are:

```text
ARCHITECTURE.md
DOMAIN_MODEL.md
API.md
FRONTEND.md
LEGACY_PRODUCTS.md
```

## Security

Never commit secrets, private keys, customer data, or production environment files. Report security issues as described in [`SECURITY.md`](./SECURITY.md).

## Documentation naming

- Stable repository-level guides use uppercase names: `README.md`, `API.md`, `ARCHITECTURE.md`.
- Detailed documents live by domain under `docs/` and use lowercase kebab-case file names.
- Organize detailed documents by bounded context: `docs/database/`, `docs/payments/`, `docs/integrations/`, `docs/architecture/`.
- Name files after the domain concept, not the historical implementation.
