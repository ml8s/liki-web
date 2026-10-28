<template>
  <button
    :id="buttonId"
    type="button"
    class="artifact-capsule"
    :class="[`artifact-capsule--${descriptor.state}`, { 'is-open': open }]"
    :aria-expanded="open"
    :aria-controls="surfaceId"
    :aria-describedby="descriptionId"
    @click="$emit('toggle')"
  >
    <span class="artifact-capsule__title">
      {{ $t(descriptor.titleKey) }}
    </span>
    <span class="artifact-capsule__state">
      {{ stateLabel }}
    </span>
    <span
      class="artifact-capsule__chevron"
      aria-hidden="true"
    />
  </button>

  <p
    :id="descriptionId"
    class="flow-visually-hidden"
  >
    {{ $t('artifact.toggleHint') }}
  </p>
</template>

<script setup lang="ts">
import { computed, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AnyArtifactDescriptor } from './types'

const props = defineProps<{
  descriptor: AnyArtifactDescriptor
  open: boolean
  surfaceId: string
}>()

defineEmits<{ toggle: [] }>()

const { t } = useI18n()
const buttonId = useId()
const descriptionId = useId()

const stateLabel = computed(() => t(`artifact.state.${props.descriptor.state}`))
</script>

<style scoped>
.flow-visually-hidden {
  position: absolute;
  width: 1px;
  height: 1px;
  margin: -1px;
  padding: 0;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  clip-path: inset(50%);
  white-space: nowrap;
  border: 0;
}
.artifact-capsule {
  width: 100%;
  min-height: var(--control-height-touch);
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: 10px var(--space-3);
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-full);
  background: var(--liki-surface);
  color: var(--liki-ink);
  font: inherit;
  font-size: var(--font-size-label);
  cursor: pointer;
}
.artifact-capsule:focus-visible {
  outline: 3px solid var(--liki-primary);
  outline-offset: 2px;
}
.artifact-capsule--pending-backend,
.artifact-capsule--unavailable {
  border-style: dashed;
  color: var(--liki-text-2);
}
.artifact-capsule.is-open {
  border-color: var(--liki-primary-border);
  background: var(--liki-primary-tint);
}
.artifact-capsule__title {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 600;
}
.artifact-capsule__state {
  margin-inline-start: auto;
  flex: 0 0 auto;
  color: var(--liki-text-2);
  font-size: var(--font-size-caption);
}
.artifact-capsule__chevron {
  flex: 0 0 auto;
  width: 7px;
  height: 7px;
  border-inline-end: 1.5px solid currentColor;
  border-bottom: 1.5px solid currentColor;
  transform: rotate(45deg);
  transition: transform var(--liki-dur) var(--liki-ease);
}
.artifact-capsule.is-open .artifact-capsule__chevron {
  transform: rotate(-135deg);
}
@media (prefers-reduced-motion: reduce) {
  .artifact-capsule__chevron {
    transition: none;
  }
}
</style>
