<script setup lang="ts" generic="T extends string | number">
import { ElOption, ElSelect } from 'element-plus'
import FieldShell from '../FieldShell.vue'
import { useFieldA11y } from '../use-field'
import type { BaseFieldProps, SelectOption } from '../types'

interface SelectFieldProps extends Omit<BaseFieldProps, 'readonly'> {
  modelValue: T | null
  options: Array<SelectOption<T>>
  filterable?: boolean
  clearable?: boolean
}

const props = withDefaults(defineProps<SelectFieldProps>(), {
  prop: undefined,
  hint: undefined,
  error: undefined,
  required: false,
  disabled: false,
  placeholder: undefined,
  filterable: false,
  clearable: false,
  size: 'md',
})

const { fieldId, describedBy, invalid } = useFieldA11y(() => props.hint, () => props.error)
const emit = defineEmits<{
  'update:modelValue': [value: T | null]
  change: [value: T | null]
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
    :size="size"
  >
    <el-select
      :id="fieldId"
      :model-value="modelValue ?? undefined"
      :name="name"
      :placeholder="placeholder"
      :filterable="filterable"
      :clearable="clearable"
      :disabled="disabled"
      :aria-describedby="describedBy"
      :aria-invalid="invalid ? true : undefined"
        class="lf-field__control"
      @update:model-value="emit('update:modelValue', ($event ?? null) as T | null)"
      @change="emit('change', ($event ?? null) as T | null)"
      >
        <el-option
        v-for="option in props.options"
          :key="option.value"
          :value="option.value"
          :label="option.label"
          :disabled="option.disabled"
        />
      </el-select>
  </FieldShell>
</template>
