# Make LiuYao casting context explicit

Written against: `d02d8fe` plus the current uncommitted working tree

Status: implemented in the current working tree on 2026-10-20.

## Execution record

- Implemented the schema-derived qigua contract, explicit LiuYao form state, visible casting place/longitude, browser-location prefill, explicit submit instant, and expanded tool-record input.
- Removed the hidden `116.4` longitude and the `engineLiuyaoQigua` wrapper default.
- Extracted the authenticated casting form into `LiuYaoCastingForm.vue`; the page now owns chart orchestration, consultation, and records.
- Kept the repository-standard Element Plus form contract: `el-form @submit.prevent` plus `el-button native-type="submit"`; no click-only workaround.
- Aligned the longitude input's native `step` with its four-decimal precision so browser constraint validation does not silently block form submission.
- Kept Cloudflare IP location out of scope; existing country headers remain currency-only.
- Updated local Caddy to allow same-origin Geolocation. Production edge policy still needs a deployment check before release.
- Validation passed: `make lint-js`, `make typecheck`, `make test-js`, `make check-web-static`, `make build-web`, targeted records/divination Playwright journeys, and the divination App accessibility audit.

## Problem

The LiuYao flow still sends implicit context to Engine MCP:

- `engineLiuyaoQigua(mode = 'auto')` supplies a wrapper default.
- `DivinationView.vue` resolves the casting time with a hidden `116.4` longitude.
- The time is captured only after the qigua request returns, so slow network latency can move the ritual instant.
- The saved tool record contains only the question and cannot reproduce or audit the submitted casting context.

This violates the agreed Engine MCP rule: the engine must not infer parameters. Every default must be materialized by the frontend and sent explicitly.

## Product decision

LiuYao is a current-event divination method. Its place means the place where the question is cast, not the user's birthplace and not the interpreted property.

The casting context has four explicit parts:

1. question
2. casting mode
3. casting instant
4. casting longitude, represented in the UI by a visible place label

No chart request is valid until the longitude is present in visible form state. A future product default may prefill that state, but it must remain visible, localized, editable, and recorded as a default source. It must not be injected at the API wrapper or engine call site.

## Evidence and current owners

- MCP wrapper: `web/shared/api/engine.ts`
  - `engineLiuyaoQigua` is the remaining wrapper with a default argument.
  - `engineLiuyaoChart` already requires `gregorianTime` and `longitude`.
- Authenticated LiuYao surface: `web/app/src/views/DivinationView.vue`
  - Form state is currently only `question`.
  - Current call sequence is `engineLiuyaoQigua('auto')`, then `resolveNowPaipanContext(116.4)`, then `engineLiuyaoChart`.
- Public LiuYao island: `web/src/islands/DivinationTool.vue`
  - Also calls `engineLiuyaoQigua('auto')`; audit it in the same contract slice or file a separate follow-up before shipping the wrapper change.
- Time helper: `web/shared/utils/paipan.ts`
  - `resolveNowPaipanContext(longitude)` creates `new Date()` inside the helper, which is too late for the casting ritual.
- Existing place selectors:
  - App/Element Plus: `web/app/src/components/birth-profiles/PlaceLongitudeField.vue`
  - Marketing island: `web/shared/components/tools/RegionField.vue`
  - Both already implement `place → city_coords(candidates: true) → confirmed longitude`.
- Saved records: `useToolRecords('liuyao')` in `DivinationView.vue`.

## Phase 1 — wrapper and state contract

### 1. Derive the qigua mode from Engine schema

Before changing the public type:

1. Inspect the authoritative `liuyao_qigua` schema from the running Engine MCP or generated engine contract.
2. Define only the modes that schema actually accepts.
3. Confirm whether `liuyao_chart.yong_shen` is optional and preserve that exact optionality.

Do not invent a manual/automatic mode merely to populate a select box. If the engine accepts only `auto`, model that single valid value rather than a broad string.

### 2. Remove the wrapper default

Change the wrapper to required object input:

```ts
export type LiuyaoQiguaMode = /* values copied from authoritative Engine schema */

export function engineLiuyaoQigua(
  input: { mode: LiuyaoQiguaMode },
): Promise<{ casting: LiuyaoCasting }> {
  return engineTool(ENGINE_TOOLS.LiuyaoQigua, { mode: input.mode })
}
```

Update every caller to pass the mode explicitly. Compilation, not a runtime fallback, must catch a missing mode.

### 3. Make form state explicit

Use one stable reactive form object in `DivinationView.vue`:

```ts
type CastingLocationSource =
  | 'user-selected'
  | 'browser'
  | 'saved'
  | 'product-default'

const form = reactive({
  question: '',
  castingMode: 'auto' as LiuyaoQiguaMode,
  castingPlace: '',
  castingLongitude: null as number | null,
  castingLocationSource: 'user-selected' as CastingLocationSource,
})
```

The initial `castingMode` is allowed only if `auto` is a schema-valid engine mode. It is a frontend-authored default, not an engine default. If the UI exposes no alternative, render it as explanatory text such as “系统自动起卦” rather than pretending to offer a choice.

### 4. Capture the ritual instant at submit

Split time creation from time resolution:

```ts
const castingTime = new Date().toISOString()
```

Capture it once after frontend validation succeeds and before the first Engine MCP request. Use that exact string for `tianwen_resolve` and store it in the record. Never replace it with the time at which qigua or chart networking completes.

Prefer a helper shaped like:

```ts
resolveInstantPaipanContext({
  instant: castingTime,
  longitude: form.castingLongitude,
})
```

Keep `resolveNowPaipanContext` only if another caller genuinely needs “now”; it must delegate to the explicit-instant helper.

## Phase 2 — visible place and longitude UI

### Reuse decision

Generalize the existing place selector rather than creating a parallel geocoder:

- Reuse the `city_coords(candidates: true)` behavior.
- One candidate fills the longitude automatically.
- Multiple candidates require an explicit choice.
- No match leaves longitude empty and permits manual longitude entry.
- The place string remains display/record context; chart tools receive only numeric longitude.

Two implementation options, in preference order:

1. Add semantic label props and a safe `namePrefix` to `PlaceLongitudeField.vue`, then use it with LiuYao-specific labels.
2. Add a scoped `CastingPlaceField.vue` only if birth-profile validation or layout cannot be parameterized without weakening the existing owner.

Use “起卦地点” and related LiuYao copy. Do not reuse “出生地”.

### Location acquisition sources

#### Primary: user-selected city

This is the authoritative source. Manual typing, candidate selection, and manual longitude must always be available.

#### Secondary: browser Geolocation

Use `navigator.geolocation.getCurrentPosition` only from a user-triggered control such as “使用当前定位”. Do not prompt automatically on page load.

On success:

- copy `coords.longitude` into the visible longitude field;
- use a visible non-geocoded label such as “当前设备位置”; the browser API returns coordinates, not a canonical city name;
- mark source as `browser`;
- display accuracy as an approximate hint when available;
- let the user replace the result by editing place/longitude;
- do not persist precise coordinates without the user seeing and submitting them.

On denial, timeout, unavailable API, or insecure context, keep fields empty and show recoverable guidance plus the normal manual path. Do not make the error a dead end.

Local development currently blocks this feature in `dev/Caddyfile.local`:

```text
Permissions-Policy "camera=(), microphone=(), geolocation=()"
```

The executing slice must change it to allow same-origin geolocation, e.g. `geolocation=(self)`, and verify the production edge policy separately.

#### Excluded: Cloudflare IP location

Do not add IP-based place prefill to this flow.

The existing `CF-IPCountry` / `X-Geo-Country` use in `internal/sales/currency.go` remains scoped to billing-currency detection. It is not a location source for LiuYao and must not be expanded into longitude.

Rationale:

- Country-only data cannot produce a responsible true-solar-time longitude.
- City/latitude/longitude headers would add a trusted-ingress and spoofing-review surface.
- IP geolocation is approximate and has no accuracy SLA.
- Browser location plus the normal manual/city-selector path already covers the product need with less privacy and security risk.

#### Comparison

| Source | Better for | Limitations |
| --- | --- | --- |
| User-selected `city_coords` | Reproducibility and user intent | Requires user effort |
| Browser Geolocation | Actual current device longitude | Permission prompt; can be unavailable or imprecise; gives no canonical city label |

Recommended order: explicit user choice, then user-triggered browser location, then empty required fields. No source silently wins over visible user input.

## Phase 3 — record contract

Save the submitted context, not post-hoc derived context:

```json
{
  "question": "…",
  "casting_mode": "auto",
  "casting_time": "2026-10-20T04:00:00.000Z",
  "casting_place": "上海市",
  "casting_longitude": 121.4737,
  "casting_location_source": "user-selected"
}
```

Legacy records with only `question` must continue to open. Render unknown legacy fields as unavailable; do not fabricate longitude, place, or mode when displaying them.

## Phase 4 — accessibility and locale copy

Add `zh-Hans`, `zh-Hant`, and `en` keys for:

- casting mode
- casting place
- longitude
- use current location
- locating
- browser permission denied / unavailable / timeout
- location source labels
- required longitude error
- ambiguous city choice
- casting-time helper

Accessibility requirements:

- Associate labels, helper text, and field errors with their controls.
- Announce location resolution and permission failures through a polite status region.
- Keep controls at the app's existing mobile touch-target size.
- Keep the “use current location” action operable by keyboard.
- Do not rely on color alone to distinguish approximate versus confirmed location.

## Validation

### Contract and unit tests

- `liuyao_qigua` receives exactly the frontend-authored mode and no implicit default.
- Missing mode fails TypeScript compilation.
- No `liuyao_chart` or `tianwen_resolve` request is sent when longitude is `null`.
- The captured submit instant, not qigua completion time, reaches `tianwen_resolve`.
- `liuyao_chart` receives the resolver's `candidate.instant` and numeric longitude.
- No raw place string is sent to `liuyao_chart`.
- Legacy `question`-only records open without invented fields.

### E2E tests

- City entry with one candidate fills longitude.
- Ambiguous city requires an explicit choice.
- Missing longitude blocks submission and shows a field-level error.
- Browser location success fills the visible longitude.
- Browser permission denial leaves a usable manual path.
- Submit saves and reopens the explicit casting context.

### Repository gates

- `make lint-js`
- `make typecheck`
- `make test-js`
- `make check-web-static`
- `make build-web`

## Implementation order

1. Confirm Engine schema and production ingress facts.
2. Harden the wrapper and all callers.
3. Add explicit LiuYao form state and remove hidden `116.4`.
4. Generalize/reuse the place selector with LiuYao semantics.
5. Add explicit-instant capture and record persistence.
6. Enable and test browser location.
7. Run the contract, accessibility, locale, and E2E gates.

## Stop conditions

- Stop if the authoritative Engine schema does not define `auto`; renegotiate the mode contract instead of broadening the frontend type.
- Stop if place-selector generalization would change birth-profile validation; use a scoped LiuYao component instead.
- Stop if precise location cannot be kept out of logs and URLs; remove the location feature rather than weakening privacy.
