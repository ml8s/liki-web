<script setup lang="ts">
import { ElTimePicker } from 'element-plus'
import FieldShell from '../FieldShell.vue'
import { useFieldA11y } from '../use-field'
import type { BaseFieldProps } from '../types'

interface TimeFieldProps extends BaseFieldProps {
  modelValue: string
  format?: string
}

const props = withDefaults(defineProps<TimeFieldProps>(), {
  prop: undefined,
  hint: undefined,
  error: undefined,
  required: false,
  disabled: false,
  readonly: false,
  placeholder: undefined,
  format: 'HH:mm',
  size: 'md',
})

const { fieldId, describedBy, invalid } = useFieldA11y(() => props.hint, () => props.error)
const emit = defineEmits<{
  'update:modelValue': [value: string]
  change: [value: string]
}>()
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
    :readonly="readonly"
    :size="size"
  >
    <el-time-picker
      :id="fieldId"
      :model-value="modelValue || undefined"
      :format="format"
        value-format="HH:mm"
      :name="name"
      :placeholder="placeholder"
      :disabled="disabled"
      :readonly="readonly"
      :aria-describedby="describedBy"
      :aria-invalid="invalid ? true : undefined"
        class="lf-field__control"
      @update:model-value="emit('update:modelValue', $event ?? '')"
      @change="emit('change', $event ?? '')"
    />
  </FieldShell>
</template>
