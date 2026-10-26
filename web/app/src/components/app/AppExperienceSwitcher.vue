<template>
  <RouterLink
    :to="{ name: destination.routeName }"
    class="experience-switcher"
    :class="`experience-switcher--${variant}`"
  >
    <slot />
  </RouterLink>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import type { AppExperience } from '@/router/experience'

const props = defineProps<{
  target: AppExperience
  variant?: 'sidebar' | 'topbar' | 'dock'
}>()

const destinations = {
  classic: { routeName: 'dashboard' },
  consultation: { routeName: 'app-consultation' },
} satisfies Record<AppExperience, { routeName: string }>

const destination = computed(() => destinations[props.target])
</script>

<style scoped>
.experience-switcher {
  min-height: var(--control-height-touch);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  min-width: 0;
  font: inherit;
  text-decoration: none;
  cursor: pointer;
}

.experience-switcher:focus-visible {
  outline: 3px solid var(--liki-primary);
  outline-offset: 2px;
  border-radius: var(--radius-md);
}

.experience-switcher--sidebar {
  justify-content: flex-start;
  width: 100%;
  padding: 10px 12px;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--liki-on-ink-link);
  font-size: var(--font-size-body-lg);
}

.experience-switcher--sidebar:hover {
  background: var(--liki-on-ink-hover);
  color: var(--liki-on-ink);
}

.experience-switcher--topbar {
  padding: 6px 10px;
  border: 1px solid var(--liki-on-ink-line);
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--liki-on-ink);
  font-size: var(--font-size-label);
  white-space: nowrap;
}

.experience-switcher--topbar:hover {
  border-color: var(--liki-on-ink-hover);
  background: var(--liki-on-ink-hover);
}

.experience-switcher--dock {
  width: 100%;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-full);
  background: var(--liki-surface);
  color: var(--liki-text-2);
  font-size: var(--font-size-label);
  text-align: center;
}

.experience-switcher--dock:hover {
  border-color: var(--liki-primary-border);
  color: var(--liki-primary);
}
</style>
