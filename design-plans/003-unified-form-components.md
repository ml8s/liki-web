# Unified form components

Status: Phase 1 + Phase 2 implemented in the current working tree. Later phases are planned and must not be mixed into unrelated changes.

## Goal

One functional module has one shared field/form implementation and one shared result implementation. App and marketing islands are hosts with different permissions and page chrome, not two independent implementations.

## Rules

- App forms use Element Plus through shared components.
- Marketing islands may use Element Plus through the same shared components.
- Element Plus is imported per component; full-library registration is forbidden.
- Shared form components emit values; MCP/API orchestration stays in the host page or feature composable.
- Form submit handling lives in `FormShell` via `el-form @submit.prevent` and `el-button native-type="submit"`.
- Paipan result components remain in `web/shared/components/paipan`.

## Phase 1 — foundation (done)

- Added `element-plus` to the marketing `web` package with the same major version as App.
- Added `element-plus` to Vite dedupe for both Astro and App builds.
- Added `web/shared/components/form/FormShell.vue`.
- `FormShell` owns `@submit.prevent`, Element Plus validation, loading, and the standard native submit button.
- Import Element Plus styles at the shared component boundary; no full-library import.

## Phase 2 — LiuYao pilot (done)

Added:

```text
web/shared/components/liuyao/types.ts
web/shared/components/liuyao/CastingPlaceField.vue
web/shared/components/liuyao/LiuYaoCastingForm.vue
```

`LiuYaoCastingForm` variants:

- `public-qigua`: question + submit; host calls `liuyao_qigua`.
- `full`: question + casting mode + casting place/longitude + optional browser location; host calls `liuyao_qigua -> tianwen_resolve -> liuyao_chart`.

Integrated hosts:

- `web/app/src/views/DivinationView.vue` uses `variant="full"`.
- `web/src/islands/DivinationTool.vue` uses `variant="public-qigua"`.

Removed the duplicate App-only `LiuYaoCastingForm`.

Added shared i18n keys for casting context to all three chart-message locales.

## Validation (done)

- `make lint-js`
- `make typecheck`
- `make test-js`
- `make check-web-static`
- `make build-web`
- App LiuYao E2E: save/replay, missing-longitude block, browser-location prefill
- Marketing LiuYao E2E: public qigua render and engine failure
- Component tests: public payload, required question, full restore/context

## Phase 3 — RegionField (done)

Added:

```text
web/shared/components/form/RegionField.vue
web/shared/components/form/region-field.ts
```

`RegionField` is now the single Element Plus implementation for place + longitude. It preserves:

- one-candidate autofill;
- ambiguous candidate choice;
- no-match manual longitude;
- externally restored longitude;
- host-provided localized labels.

Integrated hosts:

- App `BirthForm`
- App `SynastryView`
- shared LiuYao form
- marketing `BirthInputFields`

Removed:

- App `PlaceLongitudeField.vue`
- marketing lightweight `RegionField.vue`
- LiuYao-specific `CastingPlaceField.vue`

The legacy marketing label contract remains only as the temporary props shape accepted by `BirthInputFields`; it will disappear when date/time/gender fields move to the shared form kit.

## Next phases


### Phase 4 — BirthFields

Create an Element Plus `BirthFields` component around date, time, gender, and `RegionField`. Migrate:

- `NatalTool`
- `NamingTool`
- `LiunianTool`
- `HepanTool`
- App `BirthForm`
- App `SynastryView`

Support App-only modes incrementally: solar/lunar, exact/approximate/unknown time.

### Phase 5 — Synastry

Create `SynastryForm.vue` composing two `BirthFields`. Hosts provide profile selection and API orchestration.

### Phase 6 — Huangli

Create `HuangliRangeForm.vue` for event + date range and use it on both surfaces.

### Phase 7 — Fengshui

Create `FengshuiBirthFields.vue` and `FengshuiLayoutFields.vue`; both hosts use the shared result panels already present under `paipan`.

### Phase 8 — cleanup

Remove legacy lightweight duplicated field implementations once all hosts use the shared form kit. Update `FRONTEND.md` and UX architecture docs so new forms must use the shared kit.

## Phase 4 — BirthFields implementation plan

Goal: replace the remaining duplicated birth inputs with one shared Element Plus component.

### Component contract

```text
web/shared/components/form/BirthFields.vue
```

Controlled value:

```ts
interface BirthFieldsValue {
  gender: string
  calendar: 'solar' | 'lunar'
  birth_date: string
  time_known: 'known' | 'approximate' | 'unknown'
  birth_time: string
  birth_place: string
  birth_longitude: number | null
}
```

Props:

- `modelValue: BirthFieldsValue`
- `namePrefix?: string`
- `calendarEditable?: boolean` (default true)
- `timeKnownEditable?: boolean` (default true)
- `showRegion?: boolean` (default `timeKnown === 'known'`)

The component owns field layout and labels through shared i18n. It does not call MCP.

### Behavior

- Solar date uses `el-date-picker`.
- Lunar date uses `el-input` with the existing lunar format.
- `time_known=known` uses `el-time-picker` and shows `RegionField`.
- `time_known=approximate` renders the twelve shichen buttons and stores the shichen start time in `birth_time`.
- `time_known=unknown` renders the existing explanatory hint and hides region inputs.
- Gender uses `el-select` so App and marketing have the same compact field behavior.
- App `BirthForm` continues to own nickname, note, consent, draft, dirty tracking, submit validation, and API payload.

### Migration

1. Add shared i18n labels under `profiles.form` to all chart-message locales.
2. Add `BirthFields.vue` and component tests.
3. Replace `BirthInputFields` in Natal / Naming / Liunian / Hepan islands.
4. Replace duplicated gender/date/timeKnown/time/place blocks in App `BirthForm`.
5. Remove `BirthInputFields` and the temporary marketing region-label contract after no references remain.

### Validation

- Component tests: solar known, lunar known, approximate shichen, unknown time, v-model updates.
- App tests: existing BirthForm validation and payload tests must remain green.
- Marketing E2E: natal, naming, liunian, hepan.
- App E2E: birth profile create/edit and birth-place ambiguous candidate flow.
- Gates: lint, typecheck, unit, static checks, build.

## Phase 4 status

BirthFields migration completed and validated.
