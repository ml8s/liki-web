<template>
  <EngineResultBoundary
    :status="status"
    :data="data"
    :error="errorMessage"
    :refreshing="refreshing"
    @retry="reload"
  >
    <template
      v-if="$slots.loading"
      #loading
    >
      <slot name="loading" />
    </template>
    <template
      v-if="$slots.empty"
      #empty
    >
      <slot name="empty" />
    </template>
    <template #default="{ result }">
      <slot
        :result="asResult(result)"
        :data="data"
      />
    </template>
  </EngineResultBoundary>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import EngineResultBoundary from '@shared/components/engine/EngineResultBoundary.vue'
import { useEngineResult, type EngineResultStatus } from '@shared/composables/useEngineResult'

const props = withDefaults(defineProps<{
  /** Stable resource id; also the cache namespace. */
  resource: string
  input: unknown
  load: (input: any) => Promise<any>
  enabled?: boolean
  staleTime?: number
  fallback: string
  refreshToken?: number
}>(), {
  enabled: true,
  staleTime: undefined,
  refreshToken: 0,
})

const emit = defineEmits<{
  'update:data': [unknown]
  'update:status': [EngineResultStatus]
}>()

const { t, te } = useI18n()

const engine = useEngineResult<any, any>({
  namespace: props.resource,
  input: () => props.input,
  load: input => props.load(input),
  enabled: () => props.enabled,
  staleTime: props.staleTime,
})

const status = computed(() => engine.status.value)
const data = computed(() => engine.data.value)
const refreshing = computed(() => engine.refreshing.value)
const reload = engine.reload
const errorMessage = computed(() => {
  const failure = engine.error.value
  if (!failure) return ''
  if (failure.code) {
    const key = `errors.code.${failure.code}`
    if (te(key)) return t(key)
  }
  return props.fallback
})

// The generic boundary exposes its slot payload as `unknown`; the caller's
// loader already defines the concrete result type.
function asResult(value: unknown): any {
  return value
}

watch(() => engine.data.value, value => {
  emit('update:data', value)
}, { immediate: true })

watch(status, value => {
  emit('update:status', value)
}, { immediate: true })

watch(() => props.refreshToken, () => {
  void reload()
})

defineExpose({
  status,
  data,
  reload,
  refresh: engine.refresh,
})
</script>
