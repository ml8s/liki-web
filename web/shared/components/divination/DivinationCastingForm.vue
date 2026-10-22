<script setup lang="ts">
// 六爻起卦表单：App 完整上下文和营销站免费起卦共用同一字段与提交契约。
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElButton, ElInput, ElTag } from 'element-plus'
import FormShell from '@shared/components/form/FormShell.vue'
import RegionField from '@shared/components/form/RegionField.vue'
import type { RegionFieldLabels } from '@shared/components/form/region-field'
import {
  createDivinationCastingFormValue,
  readStoredDivinationCastingValue,
  type DivinationCastingFormValue,
} from './types'
import type { BirthLocation } from '@shared/domain/location'
import '@shared/styles/element-plus/index.css'

const props = withDefaults(defineProps<{
  variant?: 'full' | 'public-qigua'
  loading?: boolean
}>(), {
  variant: 'public-qigua',
  loading: false,
})

const emit = defineEmits<{
  submit: [value: DivinationCastingFormValue]
}>()

const { t } = useI18n()

const form = reactive(createDivinationCastingFormValue())
const locating = ref(false)
const browserLocationError = ref<'denied' | 'unavailable' | 'timeout' | 'invalid' | null>(null)
const browserAccuracy = ref<number | null>(null)

const rules = computed(() => ({
  question: [{ required: true, whitespace: true, message: t('services.divination.questionRequired'), trigger: 'blur' }],
  ...(props.variant === 'full'
    ? {
        location: [{ required: true, message: t('services.divination.placeRequired'), trigger: 'change' }],
      }
    : {}),
}))

const regionLabels = computed<RegionFieldLabels>(() => ({
  place: t('services.divination.placeLabel'),
  placePlaceholder: t('services.divination.placePlaceholder'),
  placeResolving: t('services.divination.placeResolving'),
  placeAmbiguous: t('services.divination.placeAmbiguous'),
  placePick: t('services.divination.placePick'),
  placeNotFound: t('services.divination.placeNotFound'),
  placeMatched: t('services.divination.placeMatched'),
  longitude: t('services.divination.longitudeLabel'),
  longitudeHint: t('services.divination.longitudeHint'),
  timezone: t('profiles.form.timezone'),
  timezonePlaceholder: t('profiles.form.timezonePlaceholder'),
  timezoneHint: t('profiles.form.timezoneHint'),
}))

const locationStatus = computed(() => {
  if (locating.value) return t('services.divination.locating')
  if (browserLocationError.value) {
    return t(`services.divination.locationError.${browserLocationError.value}`)
  }
  if (form.location?.source === 'browser') {
    return browserAccuracy.value == null
      ? t('services.divination.browserLocationUsed')
      : t('services.divination.browserLocationAccuracy', { meters: Math.round(browserAccuracy.value) })
  }
  return t('services.divination.locationManualHint')
})

function onSubmit() {
  emit('submit', { ...form })
}

function markManualLocation() {
  browserLocationError.value = null
  browserAccuracy.value = null
}

function onLocationChange(value: BirthLocation | null) {
  form.location = value
  markManualLocation()
}

function useCurrentLocation() {
  if (locating.value) return
  if (typeof navigator === 'undefined' || !navigator.geolocation) {
    browserLocationError.value = 'unavailable'
    return
  }
  locating.value = true
  browserLocationError.value = null
  navigator.geolocation.getCurrentPosition(
    position => {
      const longitude = Number(position.coords.longitude)
      if (!Number.isFinite(longitude) || longitude < -180 || longitude > 180) {
        browserLocationError.value = 'invalid'
        browserAccuracy.value = null
        locating.value = false
        return
      }
      const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone
      if (!timezone) {
        browserLocationError.value = 'unavailable'
        browserAccuracy.value = null
        locating.value = false
        return
      }
      form.location = {
        place: t('services.divination.currentDevicePlace'),
        longitude: Math.round(longitude * 10000) / 10000,
        timezone,
        source: 'browser',
      }
      browserAccuracy.value = Number.isFinite(position.coords.accuracy) ? position.coords.accuracy : null
      locating.value = false
    },
    err => {
      browserLocationError.value = err.code === 1
        ? 'denied'
        : err.code === 3 ? 'timeout' : 'unavailable'
      browserAccuracy.value = null
      locating.value = false
    },
    { timeout: 10000, maximumAge: 60000, enableHighAccuracy: false },
  )
}

function restore(input: unknown) {
  Object.assign(form, readStoredDivinationCastingValue(input))
  browserLocationError.value = null
  browserAccuracy.value = null
}

function getContext(): DivinationCastingFormValue {
  return { ...form }
}

defineExpose({ getContext, restore })
</script>

<template>
  <FormShell
    :model="form"
    :rules="rules"
    :loading="loading"
    :submit-label="t('services.divination.cast')"
    @submit="onSubmit"
  >
    <el-form-item
      prop="question"
      :label="t('services.divination.questionLabel')"
    >
      <el-input
        v-model="form.question"
        name="divination-question"
        autocomplete="off"
        maxlength="200"
        :placeholder="t('services.divination.questionPlaceholder')"
        show-word-limit
      />
    </el-form-item>

    <el-form-item
      v-if="variant === 'full'"
      :label="t('services.divination.castingModeLabel')"
    >
      <div class="casting-mode">
        <el-tag type="info">
          {{ t('services.divination.castingModeAuto') }}
        </el-tag>
        <span class="mode-hint">
          {{ t('services.divination.castingModeHint') }}
        </span>
      </div>
    </el-form-item>

    <template v-if="variant === 'full'">
      <RegionField
        :location="form.location"
        :labels="regionLabels"
        name-prefix="casting"
        @update:location="onLocationChange"
      />

      <el-form-item>
        <div class="location-actions">
          <el-button
            type="primary"
            plain
            :loading="locating"
            @click="useCurrentLocation"
          >
            {{ t('services.divination.useCurrentLocation') }}
          </el-button>
          <p
            class="location-status"
            role="status"
            aria-live="polite"
          >
            {{ locationStatus }}
          </p>
          <p class="casting-time-hint">
            {{ t('services.divination.castingTimeHint') }}
          </p>
        </div>
      </el-form-item>
    </template>
  </FormShell>
</template>

<style scoped>
.casting-mode { display: flex; align-items: center; gap: var(--space-2); min-height: 32px; }
.mode-hint { color: var(--liki-text-2); font-size: var(--font-size-label); }
.location-actions { display: flex; align-items: center; gap: var(--space-3); width: 100%; }
.location-status { margin: 0; color: var(--liki-text-2); font-size: var(--font-size-label); }
.casting-time-hint { margin: 0; color: var(--liki-text-3); font-size: var(--font-size-caption); }

@media (max-width: 768px) {
  .casting-mode { align-items: flex-start; flex-direction: column; gap: var(--space-1); }
  .location-actions { align-items: flex-start; flex-direction: column; gap: var(--space-2); }
}
</style>