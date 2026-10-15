<template>
  <form
    class="composer"
    @submit.prevent="submit"
  >
    <el-input
      :model-value="modelValue"
      name="chat-message"
      autocomplete="off"
      type="textarea"
      :autosize="{ minRows: 1, maxRows: 5 }"
      resize="none"
      :disabled="sending"
      :aria-label="$t('chat.composer.label')"
      :placeholder="$t('chat.composer.placeholder')"
      @update:model-value="$emit('update:modelValue', $event)"
      @keydown="onKeydown"
    />
    <div class="actions">
      <span>{{ $t('commerce.aiDisclaimer') }}</span>
      <el-button
        v-if="sending"
        native-type="button"
        @click="$emit('stop')"
      >
        {{ $t('chat.composer.stop') }}
      </el-button>
      <el-button
        v-else
        native-type="submit"
        type="primary"
        :disabled="!modelValue.trim()"
      >
        {{ $t('chat.composer.send') }}
      </el-button>
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

function onKeydown(event: Event) {
  const e = event as KeyboardEvent
  if (e.key !== 'Enter') return
  // 输入法组合态（keyCode 229 / isComposing）时 Enter 是上屏确认，不可误发。
  if (e.isComposing || e.keyCode === 229) return
  if (e.shiftKey) return
  e.preventDefault()
  submit()
}
</script>

<style scoped>
.composer { padding-top: 12px; border-top: 1px solid var(--liki-line); }
.actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 10px;
  color: var(--liki-text-2);
  font-size: var(--font-size-caption);
}
</style>
