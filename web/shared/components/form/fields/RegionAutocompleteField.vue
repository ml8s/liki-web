<script setup lang="ts">
// 统一出生/起卦地点：单字段候选 + 手动经度/IANA 时区逃生口，输出原子 location。
import { computed, nextTick, onMounted, ref, useId, watch } from 'vue'
import { ElAutocomplete } from 'element-plus'
import FieldShell from '../FieldShell.vue'
import NumberField from './NumberField.vue'
import TextField from './TextField.vue'
import { locationCandidatesFor } from '@shared/domain/location'
import type { BirthLocation } from '@shared/domain/location'
import { loadCityIndex, searchCities, type CityMatch } from '@shared/utils/citySearch'
import type { RegionFieldLabels } from '../region-field'
import '@shared/styles/element-plus/index.css'

interface RegionSuggestion {
  value: string
  title: string
  subtitle: string
  kindLabel: string
  candidate: CityMatch
}

const props = withDefaults(defineProps<{
  location: BirthLocation | null
  labels: RegionFieldLabels
  namePrefix?: string
  initialPlace?: string
}>(), {
  namePrefix: 'birth',
  initialPlace: '',
})

const emit = defineEmits<{
  'update:location': [value: BirthLocation | null]
}>()

interface AutocompleteExposed {
	highlightedIndex?: { value: number }
	inputRef?: { ref?: HTMLInputElement }
}

const autocompleteRef = ref<AutocompleteExposed>()
const query = ref(props.location?.place ?? props.initialPlace)
const manualTimezone = ref(props.location?.timezone ?? '')
const suggestions = ref<RegionSuggestion[]>([])
const resolving = ref(false)
const failed = ref(false)
const manualOpen = ref(false)
let searchSequence = 0

function sanitizeActiveDescendant() {
	const input = autocompleteRef.value?.inputRef?.ref
	const highlightedIndex = autocompleteRef.value?.highlightedIndex?.value ?? -1
	if (input && highlightedIndex < 0) input.removeAttribute('aria-activedescendant')
}

const fieldId = useId()
const statusId = `${fieldId}-status`

const longitude = computed(() => props.location?.longitude ?? null)
const timezone = computed(() => props.location?.timezone ?? manualTimezone.value)
const showManual = computed(() =>
  manualOpen.value
  || failed.value
  || props.location?.source === 'manual')

const statusText = computed(() => {
  if (resolving.value) return props.labels.placeResolving
  if (failed.value) return props.labels.placeNotFound
  if (!props.location && suggestions.value.length > 1) return props.labels.placeAmbiguous
  if (
    props.location
    && (props.location.source === 'city' || props.location.source === 'browser')
    && query.value.trim() === props.location.place
  ) {
    return props.labels.placeMatched.replace('{lng}', props.location.longitude.toFixed(4))
  }
  return ''
})

function kindLabel(candidate: CityMatch) {
  if (candidate.level === 'city') return props.labels.kindCity
  if (candidate.level === 'county') return props.labels.kindCounty
  return props.labels.kindWorld
}

function canonicalLabel(candidate: CityMatch) {
  return [...new Set([candidate.region, candidate.city, candidate.name])]
    .filter(Boolean)
    .join('')
}

function toSuggestion(candidate: CityMatch): RegionSuggestion {
  const value = canonicalLabel(candidate)
  const subtitle = [candidate.city, candidate.region]
    .filter(Boolean)
    .filter(part => part !== candidate.name)
  return {
    value,
    title: candidate.name,
    subtitle: [...new Set(subtitle)].join(' · '),
    kindLabel: kindLabel(candidate),
    candidate,
  }
}

async function localSuggestions(text: string): Promise<RegionSuggestion[]> {
  const index = await loadCityIndex()
  return searchCities(index, text, 12).map(toSuggestion)
}

async function fetchSuggestions(
  text: string,
  callback: (items: RegionSuggestion[]) => void,
) {
  const normalized = text.trim()
  const sequence = ++searchSequence
  if (!normalized) {
    suggestions.value = []
    failed.value = false
    callback([])
    return
  }

  try {
    const items = await localSuggestions(normalized)
    if (sequence !== searchSequence) return
    suggestions.value = items
    failed.value = false
    callback(items)
  } catch {
    if (sequence !== searchSequence) return
    suggestions.value = []
    callback([])
  }
}

function emitManualLocation(nextLongitude: number | null, nextTimezone: string) {
  manualTimezone.value = nextTimezone
  if (nextLongitude == null || !/^[A-Za-z][A-Za-z0-9_+-]*(\/[A-Za-z0-9_+-]+)+$|^UTC$/.test(nextTimezone)) {
    emit('update:location', null)
    return
  }
  emit('update:location', {
    place: query.value.trim() || `${nextLongitude},${nextTimezone}`,
    longitude: nextLongitude,
    timezone: nextTimezone,
    source: 'manual',
  })
}

function applyCandidate(candidate: CityMatch) {
  const value = canonicalLabel(candidate)
  query.value = value
  manualTimezone.value = candidate.tz ?? ''
  failed.value = false
  suggestions.value = []
  if (!candidate.tz) {
    emit('update:location', null)
    return
  }
  const longitudeRange = candidate.longitude_range
    ?? (candidate.bbox ? { west: candidate.bbox.west, east: candidate.bbox.east } : undefined)
  emit('update:location', {
    place: value,
    longitude: Math.round(candidate.lng * 10000) / 10000,
    timezone: candidate.tz,
    source: 'city',
    ...(longitudeRange ? { longitude_range: longitudeRange } : {}),
  })
}

function selectCandidate(item: RegionSuggestion) {
  applyCandidate(item.candidate)
}

function isRegionSuggestion(value: unknown): value is RegionSuggestion {
  if (typeof value !== 'object' || value === null) return false
  const candidate = (value as { candidate?: unknown }).candidate
  return typeof (value as { value?: unknown }).value === 'string'
    && typeof candidate === 'object'
    && candidate !== null
}

function onPlaceInput(value: string) {
  query.value = value
  failed.value = false
  if (props.location?.source === 'city' || props.location?.source === 'browser') {
    manualTimezone.value = ''
  }
  emit('update:location', null)
}

async function localMatches(text: string): Promise<CityMatch[]> {
  try {
    return await loadCityIndex().then(index => searchCities(index, text, 12))
  } catch {
    return []
  }
}

async function resolvePlace() {
  const text = query.value.trim()
  if (!text) {
    suggestions.value = []
    failed.value = false
    if (props.location) emit('update:location', null)
    return
  }
  if (props.location?.place === text && props.location.source !== 'manual') return

  const sequence = ++searchSequence
  resolving.value = true
  failed.value = false
  try {
    let matches = await localMatches(text)
    if (sequence !== searchSequence) return

    if (!matches.length) {
      const fallback = await locationCandidatesFor(text)
      if (sequence !== searchSequence) return
      matches = fallback.candidates.map(item => ({
        name: item.name,
        region: item.region,
        city: item.city,
        lng: item.longitude,
        lat: item.latitude,
        tz: item.timezone,
        longitude_range: item.longitude_range,
        level: 'world' as const,
      }))
    }

    if (matches.length === 1) {
      applyCandidate(matches[0]!)
      return
    }

    suggestions.value = matches.map(toSuggestion)
    emit('update:location', null)
    if (!matches.length) failed.value = true
  } catch {
    if (sequence !== searchSequence) return
    suggestions.value = []
    emit('update:location', null)
    failed.value = true
  } finally {
    if (sequence === searchSequence) resolving.value = false
  }
}

function onPlaceEnter(event: KeyboardEvent) {
	void nextTick(sanitizeActiveDescendant)
	if (event.isComposing || event.keyCode === 229) return
  const highlighted = autocompleteRef.value?.highlightedIndex?.value ?? -1
  if (highlighted >= 0) return
  window.setTimeout(() => { void resolvePlace() }, 0)
}

function onLongitudeChange(value: number | null) {
  emitManualLocation(value, timezone.value)
}

function onTimezoneChange(value: string) {
  emitManualLocation(longitude.value, value.trim())
}

function onAutocompleteInput(value: string | number) {
  onPlaceInput(String(value))
}

function onAutocompleteSelect(item: Record<string, unknown>) {
  if (isRegionSuggestion(item)) selectCandidate(item)
}

watch(() => props.location, (value) => {
  if (value?.source === 'manual') manualOpen.value = true
  if (value?.place === query.value && value.timezone === manualTimezone.value) return
  if (value) {
    query.value = value.place
    manualTimezone.value = value.timezone
    suggestions.value = []
    failed.value = false
  }
})

watch(
	() => autocompleteRef.value?.highlightedIndex?.value,
	() => void nextTick(sanitizeActiveDescendant),
)

onMounted(() => {
	void nextTick(sanitizeActiveDescendant)
	if (query.value.trim() && !props.location) void resolvePlace()
})
</script>

<template>
  <FieldShell
    :field-id="fieldId"
    :label="labels.place"
    prop="location"
  >
    <el-autocomplete
      ref="autocompleteRef"
      :id="fieldId"
      v-model="query"
      :name="`${namePrefix}-place`"
      :fetch-suggestions="fetchSuggestions"
      :debounce="120"
      trigger-on-focus
      fit-input-width
      clearable
      hide-loading
      value-key="value"
      autocomplete="off"
      spellcheck="false"
      popper-class="lf-region-autocomplete"
      :aria-label="labels.place"
      :aria-describedby="statusId"
      @input="onAutocompleteInput"
      @select="onAutocompleteSelect"
      @blur="resolvePlace"
      @keydown.capture="onPlaceEnter"
    >
      <template #default="{ item }">
        <span class="region-option">
          <span class="region-option__copy">
            <span class="region-option__title">{{ item.title }}</span>
            <span
              v-if="item.subtitle"
              class="region-option__subtitle"
            >{{ item.subtitle }}</span>
          </span>
          <span class="region-option__kind">{{ item.kindLabel }}</span>
        </span>
      </template>
    </el-autocomplete>

    <p
      :id="statusId"
      class="region-status"
      aria-live="polite"
    >
      {{ statusText }}
    </p>
    <button
      type="button"
      class="region-manual-toggle"
      :aria-expanded="showManual"
      @click="manualOpen = !manualOpen"
    >
      {{ labels.manualToggle }}
    </button>
  </FieldShell>

  <div
    v-if="showManual"
    class="region-manual-fields"
  >
    <NumberField
      :model-value="longitude"
      :name="`${namePrefix}-longitude`"
      :label="labels.longitude"
      :hint="labels.longitudeHint"
      :min="-180"
      :max="180"
      :step="0.0001"
      :precision="4"
      @update:model-value="onLongitudeChange"
    />
    <TextField
      :model-value="timezone"
      :name="`${namePrefix}-timezone`"
      :label="labels.timezone"
      :hint="labels.timezoneHint"
      :placeholder="labels.timezonePlaceholder"
      autocomplete="off"
      :spellcheck="false"
      @update:model-value="onTimezoneChange"
    />
  </div>
</template>

<style scoped>
.region-status {
  width: 100%;
  margin: var(--space-1) 0 0;
  color: var(--field-hint);
  font-size: var(--font-size-label);
  line-height: 1.4;
}

.region-manual-toggle {
  display: inline-flex;
  align-items: center;
  align-self: flex-start;
  min-height: 36px;
  padding: 4px 0;
  border: 0;
  background: transparent;
  color: var(--liki-primary);
  font: inherit;
  font-size: var(--font-size-label);
  cursor: pointer;
}

.region-manual-fields {
  display: grid;
  gap: var(--space-3);
  width: 100%;
}

@media (min-width: 640px) {
  .region-manual-fields {
    grid-template-columns: minmax(0, 180px) minmax(0, 1fr);
  }
}
</style>

<style>
.lf-region-autocomplete .el-autocomplete-suggestion__wrap {
  max-height: min(50vh, 320px);
}

.lf-region-autocomplete li[role='option'] {
  display: block;
  min-height: 44px;
  padding: var(--space-2) var(--space-3);
  line-height: 1.3;
  white-space: normal;
}

.lf-region-autocomplete .region-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  width: 100%;
}

.lf-region-autocomplete .region-option__copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.lf-region-autocomplete .region-option__title {
  overflow-wrap: anywhere;
  font-weight: 500;
}

.lf-region-autocomplete .region-option__subtitle {
  color: var(--el-text-color-secondary);
  font-size: var(--font-size-label);
  overflow-wrap: anywhere;
}

.lf-region-autocomplete .region-option__kind {
  flex-shrink: 0;
  padding: 2px 8px;
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-full);
  color: var(--liki-text-2);
  font-size: var(--font-size-caption);
}
</style>
