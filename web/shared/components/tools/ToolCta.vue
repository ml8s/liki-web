<template>
  <a
    class="btn-purple tool-cta"
    :href="href"
    @click="$emit('click', $event)"
  ><slot /></a>
  <p
    v-if="privacy"
    class="tool-privacy"
  >{{ privacy }}</p>
</template>

<script setup lang="ts">
defineProps<{
  href: string
  /** Optional privacy note rendered under the CTA. */
  privacy?: string
}>()

defineEmits<{ click: [MouseEvent] }>()
</script>
