<script setup lang="ts">
// App flow adapter for the canonical divination casting form.
import { ref } from 'vue'
import DivinationCastingForm from '@shared/components/divination/DivinationCastingForm.vue'
import {
  createDivinationCastingFormValue,
  type DivinationCastingFormValue,
} from '@shared/components/divination/types'

const props = withDefaults(defineProps<{
  loading?: boolean
}>(), {
  loading: false,
})

const emit = defineEmits<{
  submit: [value: DivinationCastingFormValue]
}>()

const castingForm = ref<InstanceType<typeof DivinationCastingForm>>()

function onSubmit(value: DivinationCastingFormValue) {
  if (props.loading) return
  emit('submit', value)
}

function restore(input: unknown) {
  castingForm.value?.restore(input)
}

function getContext(): DivinationCastingFormValue {
  return castingForm.value?.getContext() ?? createDivinationCastingFormValue()
}

defineExpose({ getContext, restore })
</script>

<template>
  <div
    class="flow-divination-form"
    :class="{ 'flow-divination-form--loading': loading }"
    :aria-busy="loading"
    :inert="loading ? true : undefined"
  >
    <DivinationCastingForm
      ref="castingForm"
      variant="full"
      :loading="loading"
      @submit="onSubmit"
    />
  </div>
</template>

<style scoped>
.flow-divination-form {
  min-width: 0;
}

.flow-divination-form[aria-busy='true'] {
  pointer-events: none;
}

.flow-divination-form form {
  width: 100%;
}

/* The shared form owns field layout and validation. This adapter only makes
   its single action suitable for the mobile flow dock. */
.flow-divination-form form :deep(.el-button--primary) {
  width: 100%;
  min-height: var(--control-height-touch);
  border-radius: var(--radius-full);
}

.flow-divination-form form :deep(.el-button--primary:focus-visible) {
  outline: 3px solid var(--liki-primary);
  outline-offset: 2px;
}

@media (min-width: 768px) {
  .flow-divination-form form :deep(.el-button--primary) {
    width: auto;
    min-width: 160px;
  }
}
</style>
