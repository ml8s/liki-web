<script setup lang="ts" generic="T extends string | number">
import { ElRadioButton, ElRadioGroup } from 'element-plus'
import FieldShell from '../FieldShell.vue'
import { useFieldA11y } from '../use-field'
import type { BaseFieldProps, ChoiceOption } from '../types'

interface ChoiceCardGroupProps extends Omit<BaseFieldProps, 'placeholder' | 'readonly'> {
  modelValue: T | null
  options: Array<ChoiceOption<T>>
  ariaLabel?: string
  ariaLabelledby?: string
}

const props = withDefaults(defineProps<ChoiceCardGroupProps>(), {
  label: '',
  prop: undefined,
  hint: undefined,
  error: undefined,
  ariaLabel: undefined,
  ariaLabelledby: undefined,
  disabled: false,
})

const { fieldId } = useFieldA11y(() => props.hint, () => props.error)

const emit = defineEmits<{
  'update:modelValue': [value: T]
  change: [value: T]
}>()

function isSelected(value: T) {
  return props.modelValue === value
}
</script>

<template>
  <FieldShell
    :field-id="fieldId"
    :label="label"
    :prop="prop"
    :hint="hint"
    :error="error"
    :disabled="disabled"
  >
    <el-radio-group
      class="choice-card"
      :class="{ 'choice-card--error': Boolean(error) }"
      :model-value="modelValue ?? undefined"
      :name="name"
      :aria-label="ariaLabel ?? label"
      :aria-labelledby="ariaLabelledby"
      :aria-invalid="Boolean(error) ? true : undefined"
      :disabled="disabled"
      @update:model-value="emit('update:modelValue', $event as T)"
      @change="emit('change', $event as T)"
    >
      <el-radio-button
        v-for="option in options"
        :key="option.value"
        :value="option.value"
        :disabled="disabled || option.disabled"
      >
        <span class="choice-card__body">
          <span class="choice-card__copy">
            <span class="choice-card__label">{{ option.label }}</span>
            <span
              v-if="option.description"
              class="choice-card__description"
            >{{ option.description }}</span>
          </span>
          <span
            class="choice-card__state"
            :class="{ 'choice-card__state--selected': isSelected(option.value) }"
            aria-hidden="true"
          >
            <svg
              v-if="isSelected(option.value)"
              viewBox="0 0 20 20"
              width="12"
              height="12"
              fill="none"
              stroke="currentColor"
              stroke-width="2.5"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M4 10.5 8.2 14.7 16 6.5" />
            </svg>
          </span>
        </span>
      </el-radio-button>
    </el-radio-group>
  </FieldShell>
</template>

<style scoped>
.choice-card {
  display: grid;
  gap: var(--space-2);
  width: 100%;
}

@media (min-width: 480px) {
  .choice-card {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

.choice-card :deep(.el-radio-button) {
  display: block;
  width: 100%;
}

.choice-card :deep(.el-radio-button__inner) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  width: 100%;
  min-height: var(--control-height-touch);
  padding: var(--space-2) var(--space-4);
  border: 1px solid var(--control-border);
  border-radius: var(--control-radius);
  outline: none;
  background: var(--choice-surface);
  color: var(--control-text);
  font: inherit;
  font-size: var(--control-font-md);
  font-weight: 500;
  line-height: 1.3;
  text-align: start;
  white-space: normal;
  transition: border-color var(--liki-dur) var(--liki-ease),
    background-color var(--liki-dur) var(--liki-ease),
    box-shadow var(--liki-dur) var(--liki-ease),
    color var(--liki-dur) var(--liki-ease);
}

.choice-card :deep(.el-radio-button:first-child .el-radio-button__inner),
.choice-card :deep(.el-radio-button:last-child .el-radio-button__inner) {
  border-radius: var(--control-radius);
  box-shadow: none;
}

@media (hover: hover) and (pointer: fine) {
  .choice-card :deep(.el-radio-button:not(.is-disabled) .el-radio-button__inner:hover) {
    border-color: var(--control-border-hover);
  }
}

.choice-card :deep(.el-radio-button__original-radio:focus-visible + .el-radio-button__inner) {
  border-color: var(--control-border-focus);
  box-shadow: 0 0 0 3px var(--control-focus-ring);
}

.choice-card :deep(.el-radio-button.is-active .el-radio-button__inner) {
  border-color: var(--choice-selected-border);
  background: var(--choice-selected-surface);
  color: var(--choice-selected-text);
  box-shadow: 0 0 0 1px var(--choice-selected-border) inset;
  font-weight: 650;
}

.choice-card :deep(.el-radio-button__original-radio:checked + .el-radio-button__inner) {
  border-color: var(--choice-selected-border);
  background: var(--choice-selected-surface);
  color: var(--choice-selected-text);
}

.choice-card :deep(.el-radio-button.is-disabled .el-radio-button__inner) {
  cursor: not-allowed;
  opacity: 0.65;
}

.choice-card--error :deep(.el-radio-button__inner) {
  border-color: var(--field-error);
}

.choice-card__body {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  width: 100%;
}

.choice-card__copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.choice-card__label {
  overflow-wrap: anywhere;
}

.choice-card__description {
  color: var(--field-hint);
  font-size: var(--font-size-label);
  font-weight: 400;
  overflow-wrap: anywhere;
}

.choice-card__state {
  display: inline-grid;
  flex-shrink: 0;
  place-items: center;
  width: 20px;
  height: 20px;
  border: 1.5px solid currentColor;
  border-radius: var(--radius-full);
}
</style>
