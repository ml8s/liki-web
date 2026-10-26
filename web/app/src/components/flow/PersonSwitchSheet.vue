<template>
  <FlowSheet
    :model-value="modelValue"
    :title="$t('flow.personTitle')"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div
      v-if="loading"
      class="person-state"
      role="status"
    >
      <el-skeleton
        :rows="3"
        animated
        aria-busy="true"
      />
    </div>

    <div
      v-else-if="error"
      class="person-state"
    >
      <p
        class="person-error"
        role="alert"
      >
        {{ error }}
      </p>
      <button
        type="button"
        class="person-action"
        @click="$emit('retry')"
      >
        {{ $t('common.retry') }}
      </button>
    </div>

    <template v-else>
      <p
        v-if="!profiles.length"
        class="person-empty"
        role="status"
      >
        {{ $t('flow.noProfiles') }}
      </p>
      <fieldset
        v-else
        class="person-list"
      >
        <legend class="flow-visually-hidden">
          {{ $t('flow.personTitle') }}
        </legend>
        <div
          v-for="profile in profiles"
          :key="profile.id"
          class="person-option"
          :class="{ 'person-option--selected': profile.id === selectedPersonId }"
        >
          <input
            :id="`flow-person-${profile.id}`"
            class="flow-visually-hidden"
            type="radio"
            name="flow-person-selection"
            :value="profile.id"
            :checked="profile.id === selectedPersonId"
            @change="$emit('select', profile.id)"
          >
          <label :for="`flow-person-${profile.id}`">
            <span class="person-name">{{ profile.nickname }}</span>
            <span class="person-summary">{{ birthSummary(profile) }}</span>
            <span
              v-if="profile.id === selectedPersonId"
              class="person-check"
              aria-hidden="true"
            >✓</span>
          </label>
        </div>
      </fieldset>

      <div class="person-footer">
        <button
          type="button"
          class="person-action"
          @click="$emit('create')"
        >
          {{ $t('flow.createProfile') }}
        </button>
        <button
          type="button"
          class="person-action person-action--secondary"
          @click="$emit('manage')"
        >
          {{ $t('flow.manageProfiles') }}
        </button>
      </div>
    </template>
  </FlowSheet>
</template>

<script setup lang="ts">
import FlowSheet from './FlowSheet.vue'
import { birthSummary } from '@/api/birth-profiles'
import type { BirthProfile } from '@/api/birth-profiles'

defineProps<{
  modelValue: boolean
  profiles: readonly BirthProfile[]
  selectedPersonId: string | null
  loading?: boolean
  error?: string
}>()

defineEmits<{
  'update:modelValue': [value: boolean]
  select: [personId: string]
  create: []
  manage: []
  retry: []
}>()
</script>

<style scoped>
.person-state { padding: var(--space-2) 0 var(--space-4); }
.person-empty { margin: var(--space-2) 0 var(--space-4); color: var(--liki-text-2); }
.person-list {
  display: grid;
  gap: var(--space-2);
  margin: 0 0 var(--space-4);
  padding: 0;
  border: 0;
  list-style: none;
}
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
.person-option {
  position: relative;
  width: 100%;
  min-height: var(--control-height-touch);
  display: grid;
  gap: 2px;
  padding: 10px 38px 10px 12px;
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-md);
  background: var(--liki-surface);
  color: var(--liki-ink);
  font: inherit;
  text-align: start;
  cursor: pointer;
}
.person-option:focus-visible {
  outline: none;
}
.person-option:has(:focus-visible) {
  outline: 3px solid var(--liki-primary);
  outline-offset: 2px;
}
.person-option--selected {
  border-color: var(--liki-primary-border);
  background: var(--liki-primary-tint);
}
.person-name { font-weight: 600; }
.person-summary { color: var(--liki-text-2); font-size: var(--font-size-label); }
.person-check {
  position: absolute;
  inset-inline-end: 12px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--liki-primary);
  font-weight: 700;
}
.person-option label {
  min-height: var(--control-height-touch);
  display: grid;
  align-content: center;
  gap: 2px;
  cursor: pointer;
}
.person-footer {
  display: grid;
  gap: var(--space-2);
}
.person-action {
  min-height: var(--control-height-touch);
  padding: 10px 12px;
  border: 1px solid var(--liki-primary-border);
  border-radius: var(--radius-md);
  background: var(--liki-primary-tint);
  color: var(--liki-primary);
  font: inherit;
  font-weight: 600;
  cursor: pointer;
}
.person-action--secondary {
  border-color: var(--liki-line);
  background: var(--liki-surface);
  color: var(--liki-text-2);
  font-weight: 500;
}
.person-action:focus-visible {
  outline: 2px solid var(--liki-primary);
  outline-offset: 2px;
}
</style>
