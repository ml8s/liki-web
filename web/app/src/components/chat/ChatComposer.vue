<template>
  <form class="composer" @submit.prevent="submit">
    <el-input
      :model-value="modelValue"
      type="textarea"
      :rows="4"
      resize="none"
      :disabled="sending"
      aria-label="咨询输入框"
      placeholder="输入问题，Enter 发送，Shift + Enter 换行"
      @update:model-value="$emit('update:modelValue', $event)"
      @keydown.enter.exact.prevent="submit"
    />
    <div class="actions">
      <span>由 AI 生成，命理内容仅供参考。</span>
      <el-button
        v-if="sending"
        native-type="button"
        @click="$emit('stop')"
      >停止</el-button>
      <el-button
        v-else
        native-type="submit"
        type="primary"
        :disabled="!modelValue.trim()"
      >发送</el-button>
    </div>
  </form>
</template>

<script setup lang="ts">
const props = defineProps<{
  modelValue: string
  sending?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  submit: []
  stop: []
}>()

function submit() {
  if (!props.modelValue.trim() || props.sending) return
  emit('submit')
}
</script>

<style scoped>
.composer { padding-top: 12px; border-top: 1px solid #e5e7eb; }
.actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 10px;
  color: var(--liki-text-2);
  font-size: 12px;
}
</style>
