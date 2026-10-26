<script setup lang="ts">
import { defineAsyncComponent } from 'vue'
import FieldShell from '../FieldShell.vue'
import { useFieldA11y } from '../use-field'
import type { BaseFieldProps } from '../types'

// Element Plus DatePicker (with dayjs) weighs ~100 kB. Lazy-loading it keeps
// the picker chunk out of the initial route graph; it loads when the field
// first renders and the user interacts with a birth-date input.
const ElDatePickerAsync = defineAsyncComponent(() =>
  import('element-plus/es/components/date-picker/index.mjs').then(module => module.ElDatePicker),
)

interface DateFieldProps extends BaseFieldProps {
  modelValue: string
  disabledDate?: (date: Date) => boolean
}

const props = withDefaults(defineProps<DateFieldProps>(), {
  prop: undefined,
  hint: undefined,
  error: undefined,
  required: false,
  disabled: false,
  readonly: false,
  placeholder: undefined,
  disabledDate: undefined,
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
    <ElDatePickerAsync
      :id="fieldId"
      :model-value="modelValue || undefined"
        type="date"
        value-format="YYYY-MM-DD"
      :name="name"
      :placeholder="placeholder"
      :disabled-date="disabledDate"
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
