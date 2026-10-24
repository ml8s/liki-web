<script setup lang="ts">
import { ElCheckbox } from 'element-plus'
import FieldShell from '../FieldShell.vue'
import { useFieldA11y } from '../use-field'
import type { BaseFieldProps } from '../types'

interface CheckboxFieldProps extends Omit<BaseFieldProps, 'placeholder' | 'readonly'> {
  modelValue: boolean
}

const props = withDefaults(defineProps<CheckboxFieldProps>(), {
  label: '',
  prop: undefined,
  hint: undefined,
  error: undefined,
  required: false,
  disabled: false,
  size: 'md',
})

const { fieldId, describedBy, invalid } = useFieldA11y(() => props.hint, () => props.error)
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  change: [value: boolean]
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
    <el-checkbox
      :id="fieldId"
      :model-value="modelValue"
      :name="name"
      :disabled="disabled"
      :aria-describedby="describedBy"
      :aria-invalid="invalid ? true : undefined"
      class="lf-field__control"
      @update:model-value="emit('update:modelValue', $event as boolean)"
      @change="emit('change', $event as boolean)"
    >
      <slot />
    </el-checkbox>
  </FieldShell>
</template>

<style scoped>
/* Long consent copy must wrap on narrow screens; Element Plus defaults the
   label to a non-wrapping inline box, which causes page-level overflow. */
:deep(.el-checkbox) {
  display: flex;
  align-items: flex-start;
  white-space: normal;
}

:deep(.el-checkbox__input) {
  flex: 0 0 auto;
  margin-top: 2px;
}

:deep(.el-checkbox__label) {
  white-space: normal;
  overflow-wrap: anywhere;
  line-height: 1.5;
}
</style>
