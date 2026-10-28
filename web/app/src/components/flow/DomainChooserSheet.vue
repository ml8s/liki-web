<template>
  <FlowSheet
    :model-value="modelValue"
    :title="$t('flow.domainTitle')"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <fieldset class="domain-list">
      <legend class="flow-visually-hidden">
        {{ $t('flow.domainTitle') }}
      </legend>
      <div
        v-for="domain in domains"
        :key="domain.id"
        class="domain-option"
        :class="{ 'domain-option--selected': domain.id === selectedDomainId }"
      >
        <input
          :id="`flow-domain-${domain.id}`"
          class="flow-visually-hidden"
          type="radio"
          name="flow-domain-selection"
          :value="domain.id"
          :checked="domain.id === selectedDomainId"
          @change="$emit('select', domain.id)"
        >
        <label :for="`flow-domain-${domain.id}`">
          <span>{{ $t(domain.labelKey) }}</span>
          <span
            v-if="domain.id === selectedDomainId"
            class="domain-check"
            aria-hidden="true"
          >✓</span>
        </label>
      </div>
    </fieldset>
    <button
      type="button"
      class="new-consultation"
      @click="$emit('new-consultation')"
    >
      {{ $t('flow.newConsultation') }}
    </button>
  </FlowSheet>
</template>

<script setup lang="ts">
import FlowSheet from './FlowSheet.vue'
import type { AppDomain, AppDomainId } from '@shared/domain/flow'

defineProps<{
  modelValue: boolean
  domains: readonly AppDomain[]
  selectedDomainId: AppDomainId
}>()

defineEmits<{
  'update:modelValue': [value: boolean]
  select: [domainId: AppDomainId]
  'new-consultation': []
}>()
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
.domain-list {
  display: grid;
  gap: var(--space-2);
  margin: 0 0 var(--space-4);
  padding: 0;
  border: 0;
}
.domain-option {
  position: relative;
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-md);
  background: var(--liki-surface);
  color: var(--liki-ink);
}
.domain-option:has(:focus-visible) {
  outline: 3px solid var(--liki-primary);
  outline-offset: 2px;
}
.domain-option--selected {
  border-color: var(--liki-primary-border);
  background: var(--liki-primary-tint);
}
.domain-option label {
  min-height: var(--control-height-touch);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: 12px;
  color: inherit;
  font: inherit;
  font-weight: 600;
  cursor: pointer;
}
.domain-check { color: var(--liki-primary); font-weight: 700; }
.new-consultation {
  width: 100%;
  min-height: var(--control-height-touch);
  border: 0;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--liki-text-2);
  font: inherit;
  cursor: pointer;
}
.new-consultation:focus-visible,
.domain-option :focus-visible {
  outline: 3px solid var(--liki-primary);
  outline-offset: 2px;
}
</style>
