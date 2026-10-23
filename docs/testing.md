# Testing

This document defines the test architecture and the decision rules for adding
tests. It is a stable guide, not a task list.

## Principles

1. **One test owns one question.** A test that repeats an assertion from another
   layer adds maintenance cost without catching a new failure mode.
2. **Choose the layer before writing the test.** If an existing layer can catch
   the regression, extend that focused test instead of adding a parallel suite.
3. **Contract failures are data and transport failures.** They belong to the
   boundary that owns the contract, not to every page that consumes it.
4. **Browser journeys validate integration and user-visible behavior.** They do
   not restate pure domain rules or request serialization rules.
5. **Coverage is diagnostic, not a goal.** A new branch still needs a failure
   question; coverage alone does not justify a test.

## Layers

| Layer | Primary question | Current home | Do not use it for |
| --- | --- | --- | --- |
| Static contracts | Do naming, routes, error codes, locale keys, tokens, and packaging agree? | `make check` and focused scripts under `web/scripts` | Runtime behavior |
| Domain and state | Does a pure rule or state transition work? | Package-local Go tests and `web/app/src/tests` | HTTP shape or browser integration |
| First-party HTTP contract | Does the BFF accept and return the precise JSON, status, error, and transport contract? | Package handler tests plus `internal/httputil` transport tests | Visual rendering |
| Engine/Counsel MCP contract | Does the frontend send the exact tool request and consume the exact envelope/result? | `web/app/src/tests/engine-contract.test.ts` | Domain correctness behind the adapter |
| Browser integration | Does a production-built page assemble and complete the user task? | `web/e2e/journeys` | Restating every unit-level branch |
| Release system | Does the built image start and expose the expected surface? | `make smoke` | Product-policy edge cases |

## Contract ownership

Before adding a contract assertion, assign it to exactly one canonical home:

| Contract | Canonical home |
| --- | --- |
| JSON body size, content type, unknown fields, trailing data | `internal/httputil/json_test.go` |
| Endpoint-specific request, status, and error mapping | The handler package's HTTP tests |
| RFC 9457 `problem+json` normalization | `web/app/src/tests/api-client.test.ts` |
| MCP request arguments, route, protocol metadata, and error mapping | `web/app/src/tests/engine-contract.test.ts` |
| Engine data visible in a real page | The relevant Playwright journey using `web/e2e/engine-fixtures.ts` |
| Build/runtime loading, CSS chunks, console errors | Browser smoke journeys over a production build |

If two layers appear to need the same assertion, separate the questions. For
example, an engine adapter test asserts that the frontend sends
`instant`, `timezone`, and `longitude` exactly; a browser journey asserts that a
resolved profile can reach and render the chart. The browser journey should not
repeat the full adapter payload assertion.

## Regression intake

A failure found outside the repository must be classified before fixing:

| Failure class | Required regression guard |
| --- | --- |
| Domain/state bug | A package or frontend unit test at the decision point |
| BFF HTTP contract bug | A handler or shared HTTP transport test |
| Engine/Counsel protocol bug | An MCP adapter contract test |
| Marketing/App integration or build bug | A production-build browser smoke test |
| Deployment/toolchain bug | The release/system check that owns the missing invariant |

The fix PR must name the guard and the layer. If no existing layer can observe
the failure, first add the smallest missing observation point, then document it
in the layer table above. Do not scatter duplicate assertions across unrelated
tests.

If a defect was caught only by manual testing and all existing layers are green,
the missing layer is part of the defect. Fix the architecture, not just the
symptom.

## Stability

- Retries may reduce noise, but a retry-only failure is still flaky and follows
  the flaky-test ledger in `CONTRIBUTING.md`.
- Journey tests wait for domain state such as a resolved place, enabled action,
  rendered result, or completed request; they must not use arbitrary sleeps.
- Fixtures represent explicit engine/Counsel versions and shapes. When an
  external contract changes, update the fixture and the owning contract test
  together.
- CI failures must retain enough evidence to diagnose without a local rerun:
  trace, screenshot/video, server log, or release job log as appropriate.
