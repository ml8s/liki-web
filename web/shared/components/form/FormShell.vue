<script setup lang="ts">
// 统一 Element Plus 表单壳：唯一负责 submit.prevent + validate 的入口。
import { ref } from 'vue'
import { ElButton, ElForm, ElFormItem, type FormInstance, type FormRules } from 'element-plus'
import ElementPlusProvider from './ElementPlusProvider.vue'
import '@shared/styles/element-plus/index.css'

const props = withDefaults(defineProps<{
  model?: Record<string, unknown>
  rules?: FormRules
  labelPosition?: 'top' | 'left' | 'right'
  formClass?: string
  submitLabel: string
  loading?: boolean
  disabled?: boolean
}>(), {
  model: () => ({}),
  rules: () => ({}),
  labelPosition: 'top',
  formClass: '',
  loading: false,
  disabled: false,
})

const emit = defineEmits<{ submit: [] }>()

const formRef = ref<FormInstance>()

async function handleSubmit() {
  if (props.disabled) return
  if (formRef.value) {
    const valid = await formRef.value.validate().then(() => true).catch(() => false)
    if (!valid) return
  }
  emit('submit')
}

function validate() {
  return formRef.value?.validate()
}

defineExpose({ validate })
</script>

<template>
  <ElementPlusProvider>
    <el-form
      ref="formRef"
      :model="model"
      :rules="rules"
      :label-position="labelPosition"
      :class="formClass"
      @submit.prevent="handleSubmit"
    >
      <slot />

      <el-form-item>
        <slot name="actions">
          <el-button
            type="primary"
            native-type="submit"
            :loading="loading"
            :disabled="disabled"
          >
            {{ submitLabel }}
          </el-button>
        </slot>
      </el-form-item>
    </el-form>
  </ElementPlusProvider>
</template>