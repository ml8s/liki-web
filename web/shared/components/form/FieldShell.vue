<script setup lang="ts">
import { computed } from 'vue'
import { ElFormItem } from 'element-plus'
import ElementPlusProvider from './ElementPlusProvider.vue'
import type { BaseFieldProps } from './types'

interface FieldShellProps extends Omit<BaseFieldProps, 'name' | 'placeholder'> {
  fieldId: string
}

const props = withDefaults(defineProps<FieldShellProps>(), {
  prop: undefined,
  hint: undefined,
  error: undefined,
  required: false,
  disabled: false,
  readonly: false,
  size: 'md',
})

const state = computed(() => props.error
  ? 'error'
  : props.disabled
    ? 'disabled'
    : props.readonly ? 'readonly' : 'default')

const { fieldId } = props

</script>

<template>
  <ElementPlusProvider>
    <el-form-item
      class="lf-field"
      :data-size="size"
      :data-state="state"
      :label="label"
      :label-for="fieldId"
      :prop="prop"
      :required="required ? true : undefined"
    >
      <slot />
      <p
        v-if="error"
        :id="`${fieldId}-error`"
        class="lf-field__message lf-field__error"
        role="alert"
      >
        {{ error }}
      </p>
      <p
        v-else-if="hint"
        :id="`${fieldId}-hint`"
        class="lf-field__message lf-field__hint"
      >
        {{ hint }}
      </p>
    </el-form-item>
  </ElementPlusProvider>
</template>
