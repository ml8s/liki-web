<template>
  <div
    class="flow-domain-strip"
    role="tablist"
    :aria-label="$t('flow.domainStrip')"
    @keydown="onKeydown"
  >
    <button
      v-for="domain in domains"
      :key="domain.id"
      ref="tabElements"
      type="button"
      role="tab"
      class="flow-domain"
      :class="{ selected: domain.id === modelValue }"
      :aria-selected="domain.id === modelValue"
      :tabindex="domain.id === modelValue ? 0 : -1"
      @click="$emit('update:modelValue', domain.id)"
    >
      <span class="flow-domain__label">{{ $t(domain.labelKey) }}</span>
      <span
        v-if="domain.id === modelValue"
        class="flow-domain__cue"
        aria-hidden="true"
      />
    </button>
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import type { AppDomain, AppDomainId } from '@shared/domain/flow'

const props = defineProps<{
  domains: readonly AppDomain[]
  modelValue: AppDomainId
}>()

const emit = defineEmits<{
  'update:modelValue': [domainId: AppDomainId]
}>()

const tabElements = ref<HTMLButtonElement[]>([])

function onKeydown(event: KeyboardEvent) {
  const focusedIndex = tabElements.value.findIndex(element => element === document.activeElement)
  const currentIndex = focusedIndex >= 0
    ? focusedIndex
    : props.domains.findIndex(domain => domain.id === props.modelValue)
  if (currentIndex < 0) return

  const direction = getComputedStyle(event.target as Element).direction
  const forward = direction === 'rtl'
    ? event.key === 'ArrowLeft'
    : event.key === 'ArrowRight'
  const backward = direction === 'rtl'
    ? event.key === 'ArrowRight'
    : event.key === 'ArrowLeft'
  let nextIndex: number | null = null
  if (forward) nextIndex = (currentIndex + 1) % props.domains.length
  if (backward) nextIndex = (currentIndex - 1 + props.domains.length) % props.domains.length
  if (event.key === 'Home') nextIndex = 0
  if (event.key === 'End') nextIndex = props.domains.length - 1
  if (event.key === 'Enter' || event.key === ' ') {
    if (focusedIndex >= 0) {
      event.preventDefault()
      emit('update:modelValue', props.domains[focusedIndex].id)
    }
    return
  }
  if (nextIndex === null) return

  event.preventDefault()
  tabElements.value[nextIndex]?.focus()
}

watch(() => props.modelValue, async () => {
  await nextTick()
  const currentIndex = props.domains.findIndex(domain => domain.id === props.modelValue)
  const selected = tabElements.value[currentIndex]
  const reducedMotion = window.matchMedia?.('(prefers-reduced-motion: reduce)')?.matches ?? false
  selected?.scrollIntoView?.({
    block: 'nearest',
    inline: 'nearest',
    behavior: reducedMotion ? 'auto' : 'smooth',
  })
}, { immediate: true })
</script>

<style scoped>
.flow-domain-strip {
  display: grid;
  grid-auto-flow: column;
  grid-auto-columns: minmax(7.5rem, 1fr);
  gap: var(--space-2);
  overflow-x: auto;
  scroll-snap-type: x proximity;
  padding-bottom: var(--space-1);
  scrollbar-width: thin;
}
.flow-domain {
  position: relative;
  min-height: var(--control-height-touch);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 8px 12px;
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-full);
  background: var(--liki-surface);
  color: var(--liki-text-2);
  font: inherit;
  font-size: var(--font-size-label);
  white-space: nowrap;
  scroll-snap-align: start;
  cursor: pointer;
}
.flow-domain.selected {
  border-color: var(--liki-primary);
  background: var(--liki-primary-tint);
  color: var(--liki-primary);
  font-weight: 600;
}
.flow-domain__cue {
  position: absolute;
  inset-inline-start: 50%;
  bottom: 4px;
  width: 14px;
  height: 2px;
  border-radius: 2px;
  background: currentColor;
  transform: translateX(-50%);
}

@media (max-width: 479.98px) {
  .flow-domain-strip {
    gap: 0;
    padding-bottom: 0;
  }
}
</style>
