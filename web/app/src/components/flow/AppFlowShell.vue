<template>
  <div class="app-flow-shell">
    <a
      :href="`#${mainId}`"
      class="flow-skip-link"
    >
      {{ $t('a11y.skipToContent') }}
    </a>

    <header class="flow-status">
      <div class="flow-status__context">
        <span class="flow-status__person">{{ personLabel || $t('flow.personEmpty') }}</span>
        <span
          class="flow-status__separator"
          aria-hidden="true"
        >·</span>
        <span class="flow-status__domain">{{ domainLabel }}</span>
      </div>

      <button
        v-if="restoreVisible"
        type="button"
        class="flow-restore"
        @click="emit('restore')"
      >
        {{ $t('flow.restore') }}
      </button>
    </header>

    <div class="flow-body">
      <main
        :id="mainId"
        ref="mainElement"
        class="flow-main"
        tabindex="-1"
      >
        <slot />
      </main>

      <div class="flow-action-dock">
        <slot name="actions" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref } from 'vue'
import { useId } from 'vue'

defineProps<{
  personLabel?: string
  domainLabel: string
  restoreVisible?: boolean
}>()

const emit = defineEmits<{
  restore: []
}>()

const mainId = useId()
const mainElement = ref<HTMLElement | null>(null)

function getMainElement() {
  return mainElement.value
}

async function scrollToBottom({ focusMain = false }: { focusMain?: boolean } = {}) {
  await nextTick()
  const element = mainElement.value
  if (!element) return
  if (focusMain) element.focus({ preventScroll: true })
  if (typeof element.scrollTo === 'function') {
    const reducedMotion = window.matchMedia?.('(prefers-reduced-motion: reduce)')?.matches ?? false
    element.scrollTo({
      top: element.scrollHeight,
      behavior: reducedMotion ? 'auto' : 'smooth',
    })
  } else {
    element.scrollTop = element.scrollHeight
  }
}

defineExpose({
  getMainElement,
  scrollToBottom,
})
</script>

<style scoped>
.app-flow-shell {
  /* The shell—not the document—owns the viewport, keeping the dock visible
     and making the consultation document the only primary scroll region. */
  height: 100vh;
  height: 100dvh;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  background: var(--liki-paper);
}

.flow-skip-link {
  position: absolute;
  inset-inline-start: -9999px;
  z-index: var(--z-skip-link);
  padding: 10px var(--space-4);
  border-radius: 0 0 var(--radius-md) 0;
  background: var(--liki-primary);
  color: var(--liki-on-primary);
  font: inherit;
  font-size: var(--font-size-label);
  text-decoration: none;
}

.flow-skip-link:focus-visible {
  inset-inline-start: 0;
  outline: 3px solid var(--liki-ink);
  outline-offset: -2px;
}

.flow-status {
  min-width: 0;
  min-height: 44px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  flex-wrap: wrap;
  background: var(--liki-ink);
  color: var(--liki-on-ink);
}

@supports (padding: max(0px)) {
  .flow-status {
    padding-top: max(var(--space-2), env(safe-area-inset-top));
    padding-inline: max(var(--space-3), env(safe-area-inset-left))
      max(var(--space-3), env(safe-area-inset-right));
  }
}

.flow-status__context {
  flex: 1 1 12rem;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: var(--font-size-label);
}

.flow-status__person,
.flow-status__domain {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.flow-status__person { font-weight: 600; }
.flow-status__domain { color: var(--liki-on-ink-subtle); }
.flow-status__separator { flex: 0 0 auto; color: var(--liki-on-ink-muted); }

.flow-restore {
  flex: 0 1 auto;
  min-width: 0;
  max-width: 100%;
  min-height: var(--control-height-touch);
  padding-inline: var(--space-3);
  border: 1px solid var(--liki-on-ink-line);
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--liki-on-ink);
  font: inherit;
  font-size: var(--font-size-label);
  overflow-wrap: anywhere;
  white-space: normal;
  text-wrap: balance;
  cursor: pointer;
}

.flow-restore:focus-visible {
  outline: 3px solid var(--liki-primary);
  outline-offset: 2px;
}

.flow-body {
  min-height: 0;
  display: grid;
  grid-template-rows: minmax(0, 1fr) auto;
}

.flow-main {
  min-width: 0;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  overscroll-behavior: contain;
  scrollbar-gutter: stable;
  padding: var(--space-4);
}

.flow-main:focus-visible {
  outline: 3px solid var(--liki-primary);
  outline-offset: -2px;
}

.flow-action-dock {
  min-width: 0;
  display: grid;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3) calc(var(--space-3) + env(safe-area-inset-bottom));
  border-top: 1px solid var(--liki-line);
  background: var(--liki-surface);
}

/* Expand the dock's interactive safety area over device curves without
   pushing visible content off screen. */
@supports (padding: max(0px)) {
  .flow-action-dock {
    padding-inline: max(var(--space-3), env(safe-area-inset-left))
      max(var(--space-3), env(safe-area-inset-right));
  }
}

@media (min-width: 1024px) {
  .app-flow-shell {
    grid-template-columns: 248px minmax(0, 1fr);
    grid-template-rows: minmax(0, 1fr);
  }

  .flow-status {
    grid-column: 1;
    min-height: 0;
    align-items: stretch;
    justify-content: flex-start;
    flex-direction: column;
    gap: var(--space-4);
    padding: var(--space-5) var(--space-4);
    border-right: 1px solid var(--liki-line);
    background: var(--liki-surface);
    color: var(--liki-ink);
  }

  .flow-status__context {
    position: relative;
    align-items: flex-start;
    flex-direction: column;
    gap: var(--space-1);
    min-height: 0;
  }

  /* A rail reads as two context rows; the compact-bar separator is redundant. */
  .flow-status__separator { display: none; }

  .flow-status__person,
  .flow-status__domain {
    overflow-wrap: anywhere;
    white-space: normal;
  }

  .flow-status__domain { color: var(--liki-text-2); }

  .flow-restore {
    align-self: flex-start;
    border-color: var(--liki-line);
    color: var(--liki-ink);
  }

  .flow-restore:hover { border-color: var(--liki-primary-border); }

  .flow-body {
    grid-column: 2;
    width: min(100%, 920px);
    margin-inline: auto;
  }

  .flow-main,
  .flow-action-dock {
    padding-inline: var(--space-6);
  }
}
</style>
