<script setup lang="ts">
import { computed } from 'vue'
import { formatInstant } from '../../utils/paipan'

const props = withDefaults(defineProps<{
  instant: string
  timezone: string
  locale?: string
  options?: Intl.DateTimeFormatOptions
}>(), { locale: undefined, options: undefined })

const text = computed(() => formatInstant(props.instant, props.timezone, props.locale, props.options))
</script>

<template>
  <time :datetime="props.instant" :data-timezone="props.timezone">{{ text }}</time>
</template>
