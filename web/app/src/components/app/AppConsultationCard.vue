<template>
  <el-card
    class="app-card app-card--highlight"
    shadow="never"
  >
    <template #header>
      <div class="app-card__header">
        <h2 class="app-card__title">
          {{ title }}
        </h2>
      </div>
    </template>
    <p class="description">
      {{ purchased ? description : lockedDescription }}
    </p>
    <slot />

    <div class="action">
      <el-button
        v-if="purchased"
        type="primary"
        size="large"
        @click="$emit('action')"
      >
        {{ actionLabel }}
      </el-button>
      <RouterLink
        v-else
        class="locked-link"
        :to="to"
      >
        {{ lockedActionLabel }}
      </RouterLink>
    </div>

    <p class="disclaimer">
      {{ $t('commerce.aiDisclaimer') }}
    </p>
  </el-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  title?: string
  description: string
  lockedDescription: string
  purchased: boolean
  actionLabel?: string
  lockedActionLabel?: string
  to?: string
}>()

const { t } = useI18n()
const title = computed(() => props.title ?? t('consultation.defaultTitle'))
const actionLabel = computed(() => props.actionLabel ?? t('consultation.defaultAction'))
const lockedActionLabel = computed(() => props.lockedActionLabel ?? t('commerce.activateNow'))
const to = computed(() => props.to ?? '/chat')

defineEmits<{ action: [] }>()
</script>

<style scoped>
.description {
  margin: 0;
  color: var(--liki-text-2);
  line-height: 1.7;
  text-wrap: pretty;
}
.action {
  display: flex;
  margin-top: var(--space-5);
}
.locked-link {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 44px;
  padding: 0 22px;
  border-radius: var(--radius-full);
  background: var(--liki-primary);
  color: var(--liki-on-primary);
  font-size: var(--font-size-title-sm);
  font-weight: 500;
  text-decoration: none;
  transition: background var(--liki-dur) var(--liki-ease), transform var(--liki-dur) var(--liki-ease);
}
.locked-link:hover {
  background: var(--liki-primary-hover);
}
.locked-link:active {
  transform: scale(0.96);
}
.disclaimer {
  margin: var(--space-4) 0 0;
  color: var(--liki-text-3);
  font-size: var(--font-size-caption);
}
</style>
