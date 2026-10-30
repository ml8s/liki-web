<template>
  <FlowSheet
    v-if="!isDesktop"
    :model-value="open"
    :title="title"
    :size="mobileSize"
    @update:model-value="close"
  >
    <template #header-actions>
      <button
        type="button"
        class="artifact-sheet__action"
        :class="{ 'is-active': mobileDetent === 'medium' }"
        :aria-pressed="mobileDetent === 'medium'"
        @click="mobileDetent = 'medium'"
      >
        {{ $t('artifact.detent.medium') }}
      </button>
      <button
        type="button"
        class="artifact-sheet__action"
        :class="{ 'is-active': mobileDetent === 'full' }"
        :aria-pressed="mobileDetent === 'full'"
        @click="mobileDetent = 'full'"
      >
        {{ $t('artifact.detent.full') }}
      </button>
    </template>
    <ArtifactPanel :descriptor="descriptor" />
  </FlowSheet>

  <aside
    v-else
    v-show="open"
    class="artifact-sidecar"
    :class="[`artifact-sidecar--${desktopMode}`]"
    role="region"
    :aria-label="title"
  >
    <header class="artifact-sidecar__header">
      <div>
        <p class="artifact-sidecar__kind">
          {{ stateLabel }}
        </p>
        <h2 class="artifact-sidecar__title">
          {{ title }}
        </h2>
      </div>
      <div class="artifact-sidecar__controls">
        <button
          type="button"
          class="artifact-sidecar__mode"
          :class="{ 'is-active': desktopMode === 'peek' }"
          :aria-pressed="desktopMode === 'peek'"
          @click="emit('desktop-mode', 'peek')"
        >
          {{ $t('artifact.detent.peek') }}
        </button>
        <button
          type="button"
          class="artifact-sidecar__mode"
          :class="{ 'is-active': desktopMode === 'pinned' }"
          :aria-pressed="desktopMode === 'pinned'"
          @click="emit('desktop-mode', 'pinned')"
        >
          {{ $t('artifact.detent.pinned') }}
        </button>
        <button
          type="button"
          class="artifact-sidecar__close"
          :aria-label="$t('flow.close')"
          @click="close"
        >
          <span aria-hidden="true">×</span>
        </button>
      </div>
    </header>

    <div class="artifact-sidecar__body">
      <ArtifactPanel :descriptor="descriptor" />
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useMediaQuery } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import FlowSheet from '../FlowSheet.vue'
import ArtifactPanel from './ArtifactPanel.vue'
import type { AnyArtifactDescriptor } from './types'

const props = defineProps<{
  descriptor: AnyArtifactDescriptor
  open: boolean
  mode?: 'peek' | 'pinned'
}>()

const emit = defineEmits<{
  close: []
  'desktop-mode': [mode: 'peek' | 'pinned']
}>()

function close() {
  emit('close')
}

const isDesktop = useMediaQuery('(min-width: 1024px)')
const desktopMode = computed(() => props.mode ?? 'peek')
const mobileDetent = ref<'medium' | 'full'>('medium')
const mobileSize = computed(() => mobileDetent.value === 'full' ? '95dvh' : 'min(72dvh, 520px)')
watch([() => props.open, () => props.descriptor?.id], () => {
  if (props.open) mobileDetent.value = 'medium'
})
const { t } = useI18n()

const title = computed(() => {
  if (!props.descriptor) return ''
  return t(props.descriptor.titleKey)
})
const stateLabel = computed(() => props.descriptor
  ? t(`artifact.state.${props.descriptor.state}`)
  : '')
</script>

<style scoped>
.artifact-sidecar {
  min-width: 0;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-lg);
  background: var(--liki-surface);
  overflow: hidden;
}
.artifact-sidecar--peek { width: 320px; }
.artifact-sidecar--pinned { width: 384px; }
.artifact-sidecar__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-2);
  padding: var(--space-3);
  border-bottom: 1px solid var(--liki-line);
}
.artifact-sidecar__controls,
.artifact-sheet__action {
  flex: 0 0 auto;
}
.artifact-sidecar__controls {
  display: flex;
  align-items: center;
  gap: var(--space-1);
}
.artifact-sidecar__mode,
.artifact-sheet__action {
  min-height: var(--control-height-touch);
  padding: 6px 10px;
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--liki-text-2);
  font: inherit;
  font-size: var(--font-size-caption);
  cursor: pointer;
}
.artifact-sidecar__mode.is-active,
.artifact-sheet__action.is-active {
  border-color: var(--liki-primary-border);
  background: var(--liki-primary-tint);
  color: var(--liki-primary);
  font-weight: 600;
}
.artifact-sidecar__mode:focus-visible,
.artifact-sheet__action:focus-visible {
  outline: 3px solid var(--liki-primary);
  outline-offset: 2px;
}
.artifact-sidecar__kind {
  margin: 0 0 2px;
  color: var(--liki-text-3);
  font-size: var(--font-size-micro);
  font-weight: 650;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}
.artifact-sidecar__title {
  margin: 0;
  color: var(--liki-ink);
  font-size: var(--font-size-title-sm);
  font-weight: 650;
  letter-spacing: -0.02em;
}
.artifact-sidecar__close {
  flex: 0 0 auto;
  width: var(--control-height-touch);
  height: var(--control-height-touch);
  border: 0;
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--liki-text-2);
  font-size: var(--font-size-title-md);
  line-height: 1;
  cursor: pointer;
}
.artifact-sidecar__close:hover {
  background: var(--liki-primary-tint);
  color: var(--liki-primary);
}
.artifact-sidecar__close:focus-visible {
  outline: 3px solid var(--liki-primary);
  outline-offset: 2px;
}
.artifact-sidecar__body {
  min-height: 0;
  overflow: auto;
  overscroll-behavior: contain;
  padding: var(--space-3);
}
@media (prefers-reduced-motion: reduce) {
  .artifact-sidecar,
  .artifact-sidecar__body {
    transition: none;
    scroll-behavior: auto;
  }
}
</style>
