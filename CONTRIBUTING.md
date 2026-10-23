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

Before adding or changing tests, read [`docs/testing.md`](./docs/testing.md).
Choose the one layer that owns the failure question; do not add parallel
assertions for a contract already owned by another layer. A defect found in
system testing, release, or production must map to the owning regression guard.

Changed areas must at least pass:

```bash
make lint-go
make lint-js
make typecheck
make test-go
make test-js
make check-web-static
make test-python
```

Cross-layer contracts enforced by `make check`: every Go error code carries App
copy in all three locales (`web/scripts/check-error-codes.js`; the generated
inventory lives in `docs/error-codes.md` — refresh with `--update` after adding
or removing codes).

## Known flaky tests

偶发用例记录在此（发现即登记，修复后删除条目），避免「偶发=没发生」：

| 用例 | 现象 | 处置 |
| --- | --- | --- |
| `web/e2e/journeys/purchase-unlock.spec.ts`（purchased state） | 曾在全量运行中偶发失败一次（重试通过），未复现 | 观察；连续两次复现即按根因修复并在此记录 |

约定：重试通过的失败也算 flaky，必须登记；同一用例连续两次进入台账即提升为修复项。

## Toolchain versions

Versions are declared once and shared by CI and local runs; `make check` fails
when local runtimes drift:

- Node: `.nvmrc`; Python: `.python-version`; Go: `go.mod`
- golangci-lint: `deps/golangci-lint.version`, shared by `make lint-go` and CI;
  `scripts/tests/test_toolchain_parity.py` pins all sides together

GitHub Actions are SHA-pinned; Dependabot opens one grouped weekly PR to follow
the official latest. Read the release notes before merging major bumps.

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
- New static-page behavior belongs in `web/src/scripts`.

## Documentation

Documentation must describe stable behavior and invariants. Do not add implementation plans, review reports, handoff notes, or temporary checklists to `docs/`.

Important stable documents are:

```text
ARCHITECTURE.md
DOMAIN_MODEL.md
API.md
FRONTEND.md
BRAND.md
```

## Security

Never commit secrets, private keys, customer data, or production environment files. Report security issues as described in [`SECURITY.md`](./SECURITY.md).

## Documentation naming

- Stable repository-level guides use uppercase names: `README.md`, `API.md`, `ARCHITECTURE.md`.
- Detailed documents live by domain under `docs/` and use lowercase kebab-case file names.
- Organize detailed documents by bounded context: `docs/database/`, `docs/payments/`, `docs/integrations/`, `docs/architecture/`.
- Name files after the domain concept, not the historical implementation.
