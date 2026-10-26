<template>
  <section
    class="flow-naming-form"
    :aria-busy="loading"
  >
    <header class="flow-naming-form__header">
      <h2>{{ $t('services.naming.label') }}</h2>
      <p>{{ $t('services.naming.description') }}</p>
    </header>

    <dl class="flow-naming-form__person">
      <div>
        <dt>{{ $t('chat.selectBirthProfile') }}</dt>
        <dd>{{ personLabel }}</dd>
      </div>
      <div>
        <dt>{{ $t('profiles.form.birthDate') }}</dt>
        <dd>{{ birthDate }}</dd>
      </div>
      <div>
        <dt>{{ $t('profiles.form.birthTime') }}</dt>
        <dd>{{ birthTime }}</dd>
      </div>
    </dl>

    <FormShell
      :submit-label="$t('services.naming.label')"
      :loading="loading"
      :disabled="disabled"
      form-class="flow-naming-form__form"
      @submit="submit"
    />
  </section>
</template>

<script setup lang="ts">
import FormShell from '@shared/components/form/FormShell.vue'

const props = withDefaults(defineProps<{
  personLabel: string
  birthDate: string
  birthTime: string
  loading?: boolean
  disabled?: boolean
}>(), {
  loading: false,
  disabled: false,
})

const emit = defineEmits<{
  submit: []
}>()

function submit() {
  if (props.loading || props.disabled) return
  emit('submit')
}
</script>

<style scoped>
.flow-naming-form {
  min-width: 0;
  padding: var(--space-4);
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-card);
  background: var(--liki-surface);
}

.flow-naming-form__header {
  margin-bottom: var(--space-3);
}

.flow-naming-form__header h2 {
  margin: 0;
  color: var(--liki-ink);
  font-size: var(--font-size-title-xs);
  font-weight: 650;
  letter-spacing: -0.02em;
}

.flow-naming-form__header p {
  margin: var(--space-1) 0 0;
  color: var(--liki-text-2);
  font-size: var(--font-size-label);
}

.flow-naming-form__person {
  margin: 0 0 var(--space-3);
  display: grid;
  gap: var(--space-2);
  grid-template-columns: repeat(auto-fit, minmax(130px, 1fr));
}

.flow-naming-form__person > div { min-width: 0; }

.flow-naming-form__person dt {
  color: var(--liki-text-3);
  font-size: var(--font-size-caption);
}

.flow-naming-form__person dd {
  margin: 2px 0 0;
  color: var(--liki-ink);
  font-weight: 600;
  overflow-wrap: anywhere;
}

.flow-naming-form__form { min-width: 0; }
</style>
