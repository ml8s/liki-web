# Business continuity and compliance implementation plan

Status: implemented and verified on 2026-10-24  
Written against: `677451d`

## Business invariants

1. **Deterministic before AI.** A paid consultation that starts from a tool
   result must carry the same atomic facts and non-reproducible receipts used
   by that tool. Display labels such as `location.place` never replace
   longitude, timezone, and canonical instant.
2. **One session, one authoritative context.** Context is attached to the
   durable chat session and consumed by both relay and direct transports. It
   is not reconstructed from prose, a URL, or browser-only memory.
3. **Fail closed on required facts.** If a profile-bound or tool-bound run
   cannot load its required context, generation stops with a retryable error.
4. **Minimum necessary personal data.** Tool context contains only the facts
   needed by the expert, never database IDs, contact data, notes, or unrelated
   results.
5. **Erasure is complete and resumable.** Every durable store that can contain
   user-authored questions, birth facts, locations, or AI answers participates
   in the account-erasure checkpoint sequence.
6. **Consent is an enforced domain fact.** Collecting another person's birth
   facts requires an explicit confirmation; the server rejects missing
   confirmation rather than relying only on UI state.
7. **One locale contract.** Referral and normal marketing negotiation use the
   same locale resolver and default.
8. **Records replay the business state.** A saved tool record restores every
   deterministic input needed to recreate the result shown to the user.

## Phase 1 — Close compliance and deterministic-mode failures

These are independent of the larger context API and ship first.

### 1.1 Erase tool records

Add an owner-scoped `AnonymizeUser` operation to `internal/records` that clears
`title`, `input`, and `output`, and soft-deletes the rows. Add it as a durable
erasure step named `tool_records`.

Acceptance:

- erasing an account clears all four record kinds;
- resumable erasure records the new checkpoint;
- a regression test covers He Pan's embedded second-person birth facts.

### 1.2 Erase direct-run answers

Add a `runcontrol` erasure operation that clears `answer` and `error` while
retaining pseudonymous metrics and run identifiers for operational audit. Add
it as a durable erasure step named `run_records`.

Acceptance:

- direct-run answers cannot survive account erasure;
- usage counters and timestamps remain available for audit;
- erasure response/test reports the cleared count.

### 1.3 Make direct profile loading fail closed

If the direct transport cannot load a required birth profile, abort the run and
show a retryable error. Do not send a profile-less run for a profile-bound
session.

Acceptance:

- relay and direct both fail when required profile facts are unavailable;
- no request is sent to `/rt/...` after profile lookup fails;
- the UI offers retry rather than silently producing generic advice.

### 1.4 Enforce third-party consent

Require explicit confirmation whenever a profile or He Pan form stores another
person's birth facts. Validate on both client and server.

Acceptance:

- unchecked profile submission is rejected;
- He Pan submission includes a third-party confirmation field;
- server tests reject missing consent independently of the UI.

### 1.5 Restore Feng Shui records completely

Persist sitting/facing period, date, and door/master/stove inputs with each
Bazhai record. Restore all inputs when reopening the record.

Acceptance:

- reopening a record restores the exact deterministic layout;
- legacy records lacking layout fields fall back to current defaults without
  pretending those fields were persisted.

### 1.6 Share locale negotiation

Move marketing locale negotiation into a reusable HTTP helper and use it for
both unprefixed marketing paths and referral landing.

Acceptance:

- `/` and `/r/{code}` use the same default (`zh-Hant`) when language is
  unknown;
- tests cover zh-Hans, zh-Hant, English, and unknown locales.

## Phase 2 — Durable tool consultation context

### 2.1 Contract

Create a versioned, discriminated `consultation_context_v1` object. It is
attached at chat-session creation and persisted as JSON.

Initial members:

```text
version = 1
scenario = natal | liunian | hepan | divination | huangli | fengshui | naming
birth?             # atomic birth facts when the tool is birth-based
second_birth?      # He Pan only
year?              # liunian only
question?          # divination only
casting?           # non-reproducible liuyao receipt
casting_instant?   # divination only
location?          # longitude + timezone, especially divination
event?             # huangli only
start_date?        # huangli only
end_date?          # huangli only
layout?            # fengshui only
third_party_consent? # hepan only; must be true
```

Exclusions:

- database IDs;
- email, identity, contact details, and notes;
- nickname/place-only substitutions for atomic facts;
- unrelated tool outputs.

### 2.2 API and storage

Extend `POST /api/chat/sessions` with optional `context`. Validate the
discriminated union, reject unknown members, and store it in a new nullable
`context_json` column. Session reads return context only to the owner while
entitled.

Relay generation reads the persisted context and places it under:

```json
forwardedProps.consultation_context
```

Direct generation fetches the same context from the session response before it
calls `/rt/{expert}/ag-ui`.

Acceptance:

- relay and direct forwarded props are equivalent;
- context survives refresh;
- context is cleared by account erasure;
- unknown, irrelevant, scenario-mismatched, incomplete, or
  persisted-but-corrupt context shapes fail closed.

### 2.3 App tool integration

Each service page constructs context from its current deterministic state:

| Service | Required context |
| --- | --- |
| He Pan | selected profile identity is already session-bound; include second birth facts and affirmative consent |
| Divination | question, casting receipt, casting instant, longitude, timezone |
| Huangli | event, start/end dates |
| Feng Shui | birth year/gender when public, plus layout inputs |
| Liunian | birth facts and selected year |
| Naming | birth facts |

The Chat API receives context at session creation. Intent text remains a
natural-language request and never becomes the source of chart facts.

### 2.4 Public tool handoff

Public tools save a short-lived, same-origin session-storage handoff and route
to the matching App service. The handoff:

- is read once;
- expires after thirty minutes;
- never appears in a URL;
- is reduced to the Phase 2 contract;
- is deleted after the App converts it to a durable session or explicit form
  state.

Acceptance:

- all seven public tool CTAs preserve deterministic facts into the
  authenticated surface;
- no personal data is placed in a URL;
- opening the CTA in a new tab does not leak the current tab's tool data;
- Chat List does not replace public-tool birth facts with an unrelated default
  profile.

## Phase 3 — Domain boundary confirmation

### 3.1 Bazhai year semantics

Obtain or add an engine contract test that defines whether `birth_year` is a
Gregorian calendar year or a Chinese solar-year/year-life year. If it is not
already boundary-aware, pass canonical `instant` and timezone to the engine or
compute the year-life boundary in the shared domain layer.

Acceptance:

- the engine contract fixture returns
  `year_boundary: "gregorian_calendar_year"` for both life Gua and annual
  flying stars;
- the engine API accepts `birth_year` rather than an instant/date, so the
  frontend must not invent a Lichun conversion;
- public form copy states that January before Lichun still uses the Gregorian
  year, and E2E checks that copy and the engine fixture field.

### 3.2 Documentation synchronization

Update `DOMAIN_MODEL.md`, `API.md`, `FRONTEND.md`, and `llms.txt` after code
acceptance:

- context contract;
- erasure domains;
- consent rule;
- public-tool access without login;
- Bazhai year semantics.

## Verification matrix

| Concern | Tests |
| --- | --- |
| Context parity | relay/direct unit tests + AG-UI contract test |
| Public handoff | Playwright marketing → login → service → chat journey |
| Erasure | Go integration test over every PII store |
| Consent | component + HTTP validation tests |
| Record replay | Playwright save/refresh/reopen for every tool |
| Locale | HTTP negotiation table |
| Security | unknown/irrelevant-member JSON, owner/entitlement, no URL PII |
| Domain boundary | engine fixture for pre-Lichun birth |

## Rollout order

1. Phase 1, in separate reviewable commits.
2. Phase 2 storage/API behind strict validation.
3. App tool context migration.
4. Public handoff migration.
5. Phase 3 domain confirmation and documentation.

No phase may widen scope into unrelated UI restyling or payment behavior.

## Verification record — 2026-10-24

- `go test ./...`
- `make lint-go && make fmt-check`
- `make lint-js && make typecheck && make lint-docs`
- `make check-web-static`
- `scripts/build-web.sh`
- `web/app`: lint, typecheck, and 195 unit tests
- `web`: lint, typecheck, static tests, and i18n/chart-message/error-code checks
- Playwright: 7 public-tool handoffs, 3 Chat List context/profile-selection
  tests, 3 shared-input tests, 2 Feng Shui year-basis tests, and 17 all-page
  console/page-error smoke tests
