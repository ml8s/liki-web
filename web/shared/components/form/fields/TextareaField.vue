<script setup lang="ts">
import { ElInput } from 'element-plus'
import FieldShell from '../FieldShell.vue'
import { useFieldA11y } from '../use-field'
import type { BaseFieldProps } from '../types'

interface TextareaFieldProps extends BaseFieldProps {
  modelValue: string
  maxlength?: number
  rows?: number
  autosize?: boolean | { minRows?: number; maxRows?: number }
  resize?: 'none' | 'both' | 'horizontal' | 'vertical'
  showWordLimit?: boolean
  enterkeyhint?: string
}

const props = withDefaults(defineProps<TextareaFieldProps>(), {
  prop: undefined,
  hint: undefined,
  error: undefined,
  required: false,
  disabled: false,
  readonly: false,
  placeholder: undefined,
  maxlength: undefined,
  rows: 3,
  autosize: false,
  resize: 'vertical',
  showWordLimit: false,
  enterkeyhint: undefined,
  size: 'md',
})

const { fieldId, describedBy, invalid } = useFieldA11y(() => props.hint, () => props.error)
const emit = defineEmits<{
  'update:modelValue': [value: string]
  change: [value: string]
  blur: [event: FocusEvent]
  keydown: [event: KeyboardEvent]
}>()

function onKeydown(event: Event) {
  emit('keydown', event as KeyboardEvent)
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
    :readonly="readonly"
    :size="size"
  >
    <el-input
      :id="fieldId"
      :model-value="modelValue"
      :name="name"
        type="textarea"
      :rows="rows"
      :autosize="autosize"
      :resize="resize"
      :maxlength="maxlength"
      :show-word-limit="showWordLimit"
      :placeholder="placeholder"
      :disabled="disabled"
      :readonly="readonly"
      :enterkeyhint="enterkeyhint"
      :aria-describedby="describedBy"
      :aria-invalid="invalid ? true : undefined"
      @update:model-value="emit('update:modelValue', $event ?? '')"
      @change="emit('change', $event ?? '')"
      @blur="emit('blur', $event)"
      @keydown="onKeydown"
    />
  </FieldShell>
</template>
