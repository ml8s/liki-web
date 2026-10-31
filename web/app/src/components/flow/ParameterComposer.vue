<template>
  <form
    class="flow-composer"
    @submit.prevent="submit"
  >
    <div class="flow-parameters">
      <button
        type="button"
        class="flow-parameter"
        aria-haspopup="dialog"
        :title="personLabel || $t('flow.personEmpty')"
        @click="$emit('select-person')"
      >
        {{ personLabel || $t('flow.personEmpty') }}
      </button>
    </div>

    <div class="flow-input-row">
      <div class="flow-question-field">
        <TextareaField
          :model-value="modelValue"
          :autosize="{ minRows: 1, maxRows: 5 }"
          resize="none"
          name="flow-question"
          autocomplete="off"
          enterkeyhint="send"
          :label="$t('flow.askPlaceholder')"
          :disabled="sending"
          :placeholder="$t('flow.askPlaceholder')"
          @update:model-value="$emit('update:modelValue', $event)"
          @keydown="onKeydown"
        />
      </div>
      <button
        v-if="sending"
        type="button"
        class="flow-send"
        @click="$emit('stop')"
      >
        {{ $t('flow.stop') }}
      </button>
      <button
        v-else
        type="submit"
        class="flow-send"
        :disabled="!canSubmit"
      >
        {{ $t('flow.send') }}
      </button>
    </div>
  </form>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import TextareaField from '@shared/components/form/fields/TextareaField.vue'

const props = defineProps<{
  modelValue: string
  personLabel?: string
  sending?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'select-person': []
  submit: []
  stop: []
}>()

const canSubmit = computed(() => !props.sending && props.modelValue.trim().length > 0)

function submit() {
  if (!canSubmit.value) return
  emit('submit')
}

function onKeydown(event: Event) {
  const keyboardEvent = event as KeyboardEvent
  if (keyboardEvent.key !== 'Enter') return
  if (keyboardEvent.isComposing || keyboardEvent.keyCode === 229) return
  if (keyboardEvent.shiftKey) return
  keyboardEvent.preventDefault()
  submit()
}
</script>

<style scoped>
.flow-composer {
  display: grid;
  gap: var(--space-2);
}
.flow-parameters {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: var(--space-2);
}
.flow-parameter {
  min-height: var(--control-height-touch);
  padding: 8px 12px;
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-full);
  background: var(--liki-surface);
  color: var(--liki-ink);
  font: inherit;
  font-size: var(--font-size-label);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: pointer;
}
.flow-parameter:focus-visible,
.flow-send:focus-visible {
  outline: 3px solid var(--liki-primary);
  outline-offset: 2px;
}
.flow-input-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: var(--space-2);
  align-items: end;
}
.flow-question-field :deep(.el-form-item__label) {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}
.flow-send {
  min-width: 52px;
  min-height: var(--control-height-touch);
  border: 0;
  border-radius: var(--radius-full);
  background: var(--liki-primary);
  color: var(--liki-on-primary);
  font: inherit;
  font-weight: 600;
  cursor: pointer;
}
.flow-send:disabled {
  background: var(--liki-line);
  color: var(--liki-text-3);
  cursor: not-allowed;
}

@media (max-width: 479.98px) {
  .flow-composer,
  .flow-parameters,
  .flow-input-row {
    gap: 0;
  }

  .flow-parameter,
  .flow-send {
    min-height: 44px;
    height: 44px;
  }

  .flow-question-field :deep(.el-textarea__inner) {
    height: 44px !important;
    min-height: 44px !important;
    padding-block: 10px;
  }

  .flow-question-field,
  .flow-question-field :deep(.el-form-item) {
    margin-bottom: 0;
  }
}
</style>
