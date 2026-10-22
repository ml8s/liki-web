<script setup lang="ts">
// 统一出生/起卦地点：城市、手动经度、IANA 时区作为原子 location 契约输出。
import { computed, onMounted, ref, watch } from 'vue'
import {
  ElFormItem, ElInput, ElInputNumber, ElOption, ElSelect,
} from 'element-plus'
import { locationCandidatesFor } from '@shared/domain/location'
import type { BirthLocation } from '@shared/domain/location'
import { loadCityIndex, searchCities, type CityMatch } from '@shared/utils/citySearch'
import type { RegionFieldLabels } from './region-field'
import '@shared/styles/element-plus/index.css'

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

const query = ref(props.location?.place ?? props.initialPlace)
const manualTimezone = ref(props.location?.timezone ?? '')
const candidates = ref<CityMatch[]>([])
const resolving = ref(false)
const failed = ref(false)
const dirty = ref(false)
const pickedCandidateKey = ref('')
const autoMatched = ref(false)

function candidateLabel(candidate: CityMatch): string {
  // 直辖市 region/name 相同（如 "上海市"+"上海市"）：去重避免显示重复。
  return [...new Set([candidate.region, candidate.city, candidate.name])].filter(Boolean).join('')
}

const candidateOptions = computed(() => candidates.value.map((candidate, index) => ({
  key: String(index),
  label: candidateLabel(candidate),
})))

const longitude = computed(() => props.location?.longitude ?? null)
const timezone = computed(() => props.location?.timezone ?? manualTimezone.value)

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
  const label = candidateLabel(candidate)
  query.value = label
  manualTimezone.value = candidate.tz ?? ''
  autoMatched.value = true
  dirty.value = false
  candidates.value = []
  failed.value = false
  pickedCandidateKey.value = ''
  if (!candidate.tz) {
    emit('update:location', null)
    return
  }
  emit('update:location', {
    place: label,
    longitude: Math.round(candidate.lng * 10000) / 10000,
    timezone: candidate.tz,
    source: 'city',
    ...(candidate.bbox ? { bbox: candidate.bbox } : {}),
  })
}

function pickCandidate(key: string) {
  const candidate = candidates.value[Number(key)]
  if (candidate) applyCandidate(candidate)
}

async function resolvePlace() {
  const text = query.value.trim()
  if (!text) {
    candidates.value = []
    failed.value = false
    dirty.value = false
    if (autoMatched.value) emit('update:location', null)
    return
  }
  if (!dirty.value && props.location?.place === text) return

  resolving.value = true
  failed.value = false
  candidates.value = []
  pickedCandidateKey.value = ''
  try {
    let list: CityMatch[] = []
    try {
      list = searchCities(await loadCityIndex(), text)
    } catch {
      list = []
    }
    if (!list.length) {
      const fallback = await locationCandidatesFor(text)
      list = fallback.candidates.map(item => ({
        name: item.name,
        region: item.region,
        city: item.city,
        lng: item.longitude,
        lat: item.latitude,
        tz: item.timezone,
        bbox: item.bbox,
        level: 'world' as const,
      }))
    }
    if (list.length === 1) {
      applyCandidate(list[0])
      return
    }
    dirty.value = false
    emit('update:location', null)
    if (list.length > 1) candidates.value = list
    else failed.value = true
  } catch {
    emit('update:location', null)
    failed.value = true
  } finally {
    resolving.value = false
  }
}

function onPlaceInput(value: string) {
  if (props.location?.source === 'city' || props.location?.source === 'browser') {
    manualTimezone.value = ''
  }
  query.value = value
  dirty.value = true
  autoMatched.value = false
  candidates.value = []
  failed.value = false
  pickedCandidateKey.value = ''
  emit('update:location', null)
}

function onLongitudeChange(value: number | undefined) {
  autoMatched.value = false
  emitManualLocation(value ?? null, timezone.value)
}

function onTimezoneChange(value: string) {
  autoMatched.value = false
  emitManualLocation(longitude.value, value.trim())
}

watch(() => props.location, (value, previous) => {
  if (value?.place === query.value && value.timezone === manualTimezone.value) return
  if (value) {
    query.value = value.place
    manualTimezone.value = value.timezone
    candidates.value = []
    failed.value = false
    dirty.value = false
  } else if (previous?.place === query.value) {
    // 用户主动清空时保留输入文本；外部清空不覆盖手动草稿。
  }
})

onMounted(() => {
  if (query.value.trim() && !props.location) void resolvePlace()
})
</script>

<template>
  <el-form-item
    :label="labels.place"
    prop="location"
  >
    <div class="region-field">
      <el-input
        v-model="query"
        :name="`${namePrefix}-place`"
        autocomplete="off"
        spellcheck="false"
        :placeholder="labels.placePlaceholder"
        @update:model-value="onPlaceInput"
        @blur="resolvePlace"
        @keyup.enter="resolvePlace"
      />
      <p
        v-if="resolving"
        class="hint"
      >
        {{ labels.placeResolving }}
      </p>
      <template v-else-if="candidates.length">
        <p class="hint">
          {{ labels.placeAmbiguous }}
        </p>
        <el-select
          v-model="pickedCandidateKey"
          :placeholder="labels.placePick"
          class="w-full"
          @change="pickCandidate"
        >
          <el-option
            v-for="option in candidateOptions"
            :key="option.key"
            :value="option.key"
            :label="option.label"
          />
        </el-select>
      </template>
      <p
        v-else-if="failed"
        class="hint error"
      >
        {{ labels.placeNotFound }}
      </p>
      <p
        v-else-if="location && autoMatched"
        class="hint matched"
      >
        {{ labels.placeMatched.replace('{lng}', location.longitude.toFixed(4)) }}
      </p>
    </div>
  </el-form-item>

  <el-form-item :label="labels.longitude">
    <div class="location-row">
      <el-input-number
        :model-value="longitude"
        :precision="4"
        :step="0.0001"
        :min="-180"
        :max="180"
        controls-position="right"
        :name="`${namePrefix}-longitude`"
        @update:model-value="onLongitudeChange"
      />
      <span class="unit">°E</span>
    </div>
    <p class="hint">
      {{ labels.longitudeHint }}
    </p>
  </el-form-item>

  <el-form-item :label="labels.timezone">
    <el-input
      :model-value="timezone"
      :name="`${namePrefix}-timezone`"
      autocomplete="off"
      spellcheck="false"
      :placeholder="labels.timezonePlaceholder"
      @update:model-value="onTimezoneChange"
    />
    <p class="hint">
      {{ labels.timezoneHint }}
    </p>
  </el-form-item>
</template>

<style scoped>
.region-field { width: 100%; }
.region-field .hint { margin: 6px 0 0; }
.region-field .error { color: var(--color-error); }
.region-field .matched { color: var(--liki-primary); }
.location-row { display: flex; align-items: center; gap: 8px; width: 100%; }
.location-row :deep(.el-input-number) { width: 180px; }
.location-row + .hint { margin: 6px 0 0; }
.unit { color: var(--liki-text-2); font-size: var(--font-size-label); }
.hint { color: var(--liki-text-2); font-size: var(--font-size-label); margin: 0; }
</style>