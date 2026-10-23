<script setup lang="ts">
import { ElInputNumber } from 'element-plus'
import FieldShell from '../FieldShell.vue'
import { useFieldA11y } from '../use-field'
import type { BaseFieldProps } from '../types'

interface NumberFieldProps extends Omit<BaseFieldProps, 'readonly'> {
  modelValue: number | null
  min?: number
  max?: number
  step?: number
  precision?: number
}

const props = withDefaults(defineProps<NumberFieldProps>(), {
  prop: undefined,
  hint: undefined,
  error: undefined,
  required: false,
  disabled: false,
  placeholder: undefined,
  min: -Infinity,
  max: Infinity,
  step: 1,
  precision: undefined,
  size: 'md',
})

const { fieldId, describedBy, invalid } = useFieldA11y(() => props.hint, () => props.error)
const emit = defineEmits<{
  'update:modelValue': [value: number | null]
  change: [value: number | null]
  blur: [event: FocusEvent]
}>()

function toModel(value: number | undefined) {
  return typeof value === 'number' && Number.isFinite(value) ? value : null
}
</script>

<template>
  <FieldShell
    :field-id="fieldId"
    :label="label"
    :prop="prop"
    :hint="hint"
    :error="error"
    :required="required"
    :disabled="disabled"
    :size="size"
  >
    <el-input-number
      :id="fieldId"
      :model-value="modelValue ?? undefined"
      :name="name"
      :placeholder="placeholder"
      :min="min"
      :max="max"
      :step="step"
      :precision="precision"
        controls-position="right"
      :disabled="disabled"
      :aria-describedby="describedBy"
      :aria-invalid="invalid ? true : undefined"
      @update:model-value="emit('update:modelValue', toModel($event))"
      @change="emit('change', toModel($event))"
      @blur="emit('blur', $event)"
    />
  </FieldShell>
</template>
