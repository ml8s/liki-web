<script setup lang="ts">
import { ElInput } from 'element-plus'
import FieldShell from '../FieldShell.vue'
import { useFieldA11y } from '../use-field'
import type { BaseFieldProps } from '../types'

interface TextFieldProps extends BaseFieldProps {
  modelValue: string
  type?: 'text' | 'email' | 'search' | 'password' | 'tel' | 'url'
  autocomplete?: string
  inputmode?: 'none' | 'text' | 'decimal' | 'numeric' | 'tel' | 'search' | 'email' | 'url'
  spellcheck?: boolean
  maxlength?: number
}

const props = withDefaults(defineProps<TextFieldProps>(), {
  prop: undefined,
  type: 'text',
  hint: undefined,
  error: undefined,
  required: false,
  disabled: false,
  readonly: false,
  placeholder: undefined,
  autocomplete: 'off',
  inputmode: undefined,
  spellcheck: undefined,
  maxlength: undefined,
  size: 'md',
})

const { fieldId, describedBy, invalid } = useFieldA11y(() => props.hint, () => props.error)
const emit = defineEmits<{
  'update:modelValue': [value: string]
  change: [value: string]
  blur: [event: FocusEvent]
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
    <el-input
      :id="fieldId"
      :model-value="modelValue"
      :name="name"
      :type="type"
      :autocomplete="autocomplete"
      :inputmode="inputmode"
      :spellcheck="spellcheck"
      :maxlength="maxlength"
      :placeholder="placeholder"
      :disabled="disabled"
      :readonly="readonly"
      :aria-describedby="describedBy"
      :aria-invalid="invalid ? true : undefined"
      @update:model-value="emit('update:modelValue', $event ?? '')"
      @change="emit('change', $event ?? '')"
      @blur="emit('blur', $event)"
    />
  </FieldShell>
</template>
