<template>
  <section
    class="group-slot-form"
    :aria-labelledby="headingId"
    @input.capture="dirty = true"
    @change.capture="dirty = true"
  >
    <header class="group-slot-form__header">
      <button
        type="button"
        class="group-slot-form__back"
        @click="leave"
      >
        <span
          class="group-slot-form__chevron"
          aria-hidden="true"
        />
        {{ $t('flow.form.back') }}
      </button>
      <div>
        <p class="group-slot-form__intent">
          {{ intentGroupLabel }}
        </p>
        <h2
          :id="headingId"
          class="group-slot-form__title"
        >
          {{ title }}
        </h2>
        <p
          v-if="description"
          class="group-slot-form__description"
        >
          {{ description }}
        </p>
      </div>
    </header>

    <div class="group-slot-form__body">
      <slot />
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import { confirmAction } from '@/utils/dialogs'

defineProps<{
  intentGroupLabel: string
  title: string
  description?: string
}>()

const emit = defineEmits<{ back: [] }>()

const { t } = useI18n()
const headingId = useId()
const dirty = ref(false)

async function leave() {
  if (dirty.value) {
    const discard = await confirmAction({
      title: t('flow.form.unsavedTitle'),
      message: t('flow.form.unsavedMessage'),
      confirmText: t('flow.form.discard'),
    })
    if (!discard) return
  }
  dirty.value = false
  emit('back')
}

defineExpose({ dirty })
</script>

<style scoped>
.group-slot-form {
  min-width: 0;
  padding: var(--space-4);
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-card);
  background: var(--liki-surface);
}

.group-slot-form__header {
  display: grid;
  gap: var(--space-3);
  margin-bottom: var(--space-4);
}

.group-slot-form__back {
  justify-self: start;
  min-height: var(--control-height-touch);
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  padding: 8px var(--space-3);
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-full);
  background: var(--liki-surface);
  color: var(--liki-text-2);
  font: inherit;
  font-size: var(--font-size-label);
  cursor: pointer;
}

.group-slot-form__back:hover {
  border-color: var(--liki-primary-border);
  color: var(--liki-primary);
}

.group-slot-form__back:focus-visible {
  outline: 3px solid var(--liki-primary);
  outline-offset: 2px;
}

.group-slot-form__chevron {
  width: 7px;
  height: 7px;
  border-inline-start: 1.5px solid currentColor;
  border-bottom: 1.5px solid currentColor;
  transform: rotate(45deg);
}

.group-slot-form__intent {
  margin: 0;
  color: var(--liki-text-3);
  font-size: var(--font-size-micro);
  font-weight: 650;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.group-slot-form__title {
  margin: var(--space-1) 0 0;
  color: var(--liki-ink);
  font-size: var(--font-size-title-md);
  font-weight: 650;
  line-height: 1.25;
  letter-spacing: -0.02em;
}

.group-slot-form__description {
  margin: var(--space-1) 0 0;
  color: var(--liki-text-2);
  font-size: var(--font-size-label);
  line-height: 1.6;
}

.group-slot-form__body {
  min-width: 0;
  display: grid;
  gap: var(--space-3);
}

@media (max-width: 479.98px) {
  .group-slot-form {
    padding: var(--space-3);
  }
}
</style>
