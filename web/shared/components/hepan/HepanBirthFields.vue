<script setup lang="ts">
import BirthFields from '@shared/components/form/BirthFields.vue'
import type { BirthFieldLabels, BirthFieldsValue } from '@shared/components/form/birth-fields'

export interface HepanBirthValues {
  personA: BirthFieldsValue
  personB: BirthFieldsValue
}

defineProps<{
  value: HepanBirthValues
  labels: BirthFieldLabels
  groupLabels: {
    personA: string
    personB: string
  }
}>()

const emit = defineEmits<{
  'update:patch': [patch: Partial<HepanBirthValues>]
}>()

function patchA(patch: Partial<BirthFieldsValue>) {
  emit('update:patch', { personA: patch })
}

function patchB(patch: Partial<BirthFieldsValue>) {
  emit('update:patch', { personB: patch })
}
</script>

<template>
  <div class="hepan-birth-fields">
    <fieldset>
      <legend>{{ groupLabels.personA }}</legend>
      <BirthFields
        :value="value.personA"
        :labels="labels"
        name-prefix="person-a"
        @update:patch="patchA"
      >
        <slot name="personA" />
      </BirthFields>
    </fieldset>
    <fieldset>
      <legend>{{ groupLabels.personB }}</legend>
      <BirthFields
        :value="value.personB"
        :labels="labels"
        name-prefix="person-b"
        @update:patch="patchB"
      >
        <slot name="personB" />
      </BirthFields>
    </fieldset>
  </div>
</template>

<style scoped>
.hepan-birth-fields {
  display: grid;
  gap: var(--space-4);
}
</style>
